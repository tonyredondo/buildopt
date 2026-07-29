//go:build linux

package protoconformance

import (
	"bytes"
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

const (
	maxFrameBytes = 1 << 20

	payloadProducerHello = 10
	payloadCapability    = 11
	payloadCachePut      = 12

	ackAccepted       = 1
	ackAttemptAborted = 2

	capabilityExact       = 1
	capabilityUnavailable = 2

	abortUnattributedPut = 1
)

type wireField struct {
	number  int
	wire    int
	varint  uint64
	payload []byte
}

type taskEvent struct {
	attemptID         string
	producerInstance  string
	sequence          uint64
	payloadField      int
	payload           []byte
	abortImmediately  bool
	schemaDescription []string
}

type taskEventAck struct {
	attemptID        string
	producerInstance string
	sequence         uint64
	status           uint64
}

func TestJavaToGoExactRoundTrip(t *testing.T) {
	requireCrossLanguageEnvironment(t)
	socketPath := filepath.Join(t.TempDir(), "java-to-go.sock")
	listener, err := net.ListenUnix("unix", &net.UnixAddr{Name: socketPath, Net: "unix"})
	if err != nil {
		t.Fatalf("listen on Unix socket: %v", err)
	}
	defer listener.Close()
	if err := listener.SetDeadline(time.Now().Add(10 * time.Second)); err != nil {
		t.Fatalf("set listener deadline: %v", err)
	}

	serverResult := make(chan error, 1)
	go func() {
		serverResult <- serveJavaExactSequence(listener)
	}()

	result := runJava(t, "client", socketPath)
	if result.err != nil {
		t.Fatalf("Java client failed: %v\n%s", result.err, result.output)
	}
	if err := <-serverResult; err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(result.output, "java-client-exact-ok") {
		t.Fatalf("Java client did not report success: %s", result.output)
	}
}

func TestGoToJavaUnattributedRoundTrip(t *testing.T) {
	requireCrossLanguageEnvironment(t)
	testRoot := t.TempDir()
	socketPath := filepath.Join(testRoot, "go-to-java.sock")
	readyPath := filepath.Join(testRoot, "java.ready")

	command, output := startJava(t, "server", socketPath, readyPath)
	waitForFile(t, readyPath, command, output)

	connection, err := net.DialTimeout("unix", socketPath, 5*time.Second)
	if err != nil {
		command.Process.Kill()
		command.Wait()
		t.Fatalf("dial Java Unix socket: %v\n%s", err, output.String())
	}
	defer connection.Close()
	if err := connection.SetDeadline(time.Now().Add(10 * time.Second)); err != nil {
		t.Fatalf("set connection deadline: %v", err)
	}

	events := []struct {
		bytes          []byte
		expectedStatus uint64
		schemaWants    []string
	}{
		{
			bytes: marshalTaskEvent(
				"attempt-go-unattributed",
				"go-producer",
				1,
				payloadProducerHello,
				marshalProducerHello(1, 0, 1, "go-conformance-v1"),
			),
			expectedStatus: ackAccepted,
			schemaWants: []string{
				`attempt_id: "attempt-go-unattributed"`,
				"producer_hello {",
				"major: 1",
				"PRODUCER_KIND_GRADLE_PLUGIN",
			},
		},
		{
			bytes: marshalTaskEvent(
				"attempt-go-unattributed",
				"go-producer",
				2,
				payloadCachePut,
				marshalUnattributedPut(
					"put-go-2",
					"c0111bcb4ba8ba492a6cb273f724a55b",
					1,
					1,
					abortUnattributedPut,
					"no task ancestor owns the observed PUT",
				),
			),
			expectedStatus: ackAttemptAborted,
			schemaWants: []string{
				"cache_put_observed {",
				`put_operation_id: "put-go-2"`,
				"UNATTRIBUTED_REASON_NO_TASK_ANCESTOR",
				"ATTEMPT_ABORT_REASON_UNATTRIBUTED_CACHE_PUT",
			},
		},
		{
			bytes: marshalTaskEvent(
				"attempt-go-unattributed",
				"go-producer",
				3,
				payloadCapability,
				marshalCapability(
					capabilityUnavailable,
					"9.6.1",
					"cold build contained a non-task PUT",
					abortUnattributedPut,
					"discard every pending write",
				),
			),
			expectedStatus: ackAttemptAborted,
			schemaWants: []string{
				"correlation_capability_declared {",
				"CORRELATION_CAPABILITY_UNAVAILABLE",
				"ATTEMPT_ABORT_REASON_UNATTRIBUTED_CACHE_PUT",
			},
		},
	}

	for index, expected := range events {
		assertProtocDecodes(t, expected.bytes, expected.schemaWants...)
		if err := writeDelimited(connection, expected.bytes); err != nil {
			t.Fatalf("write event %d: %v", index+1, err)
		}
		ackBytes, err := readDelimited(connection)
		if err != nil {
			t.Fatalf("read acknowledgement %d: %v", index+1, err)
		}
		ack, err := decodeTaskEventAck(ackBytes)
		if err != nil {
			t.Fatalf("decode acknowledgement %d: %v", index+1, err)
		}
		if ack.attemptID != "attempt-go-unattributed" ||
			ack.producerInstance != "go-producer" ||
			ack.sequence != uint64(index+1) ||
			ack.status != expected.expectedStatus {
			t.Fatalf("unexpected acknowledgement %d: %+v", index+1, ack)
		}
	}
	if err := connection.Close(); err != nil {
		t.Fatalf("close Go client connection: %v", err)
	}
	if err := command.Wait(); err != nil {
		t.Fatalf("Java server failed: %v\n%s", err, output.String())
	}
	if !strings.Contains(output.String(), "java-server-unattributed-ok") {
		t.Fatalf("Java server did not report success: %s", output.String())
	}
}

func TestJavaSemanticSelfTest(t *testing.T) {
	requireCrossLanguageEnvironment(t)
	result := runJava(t, "self-test")
	if result.err != nil {
		t.Fatalf("Java semantic self-test failed: %v\n%s", result.err, result.output)
	}
	if !strings.Contains(result.output, "java-semantic-self-test-ok") {
		t.Fatalf("Java semantic self-test did not report success: %s", result.output)
	}
}

func TestGoRejectsUnsafeSemanticCombinations(t *testing.T) {
	t.Run("cache put without attribution", func(t *testing.T) {
		payload := appendStringField(nil, 1, "put-missing-attribution")
		payload = appendStringField(payload, 2, "cache-key")
		payload = appendVarintField(payload, 3, 1)
		event := marshalTaskEvent("attempt", "producer", 2, payloadCachePut, payload)
		if _, err := decodeTaskEvent(event); err == nil ||
			!strings.Contains(err.Error(), "exactly one attribution") {
			t.Fatalf("expected attribution error, got %v", err)
		}
	})

	t.Run("unattributed put without whole-attempt abort", func(t *testing.T) {
		unattributed := appendVarintField(nil, 1, 1)
		payload := appendStringField(nil, 1, "put-no-abort")
		payload = appendStringField(payload, 2, "cache-key")
		payload = appendVarintField(payload, 3, 1)
		payload = appendMessageField(payload, 11, unattributed)
		event := marshalTaskEvent("attempt", "producer", 2, payloadCachePut, payload)
		if _, err := decodeTaskEvent(event); err == nil ||
			!strings.Contains(err.Error(), "whole-attempt abort") {
			t.Fatalf("expected whole-attempt abort error, got %v", err)
		}
	})

	t.Run("unavailable capability without abort", func(t *testing.T) {
		payload := appendVarintField(nil, 1, capabilityUnavailable)
		payload = appendStringField(payload, 2, "9.6.1")
		event := marshalTaskEvent("attempt", "producer", 3, payloadCapability, payload)
		if _, err := decodeTaskEvent(event); err == nil ||
			!strings.Contains(err.Error(), "requires a whole-attempt abort") {
			t.Fatalf("expected unavailable capability error, got %v", err)
		}
	})

	t.Run("exact capability with abort", func(t *testing.T) {
		payload := marshalCapability(
			capabilityExact,
			"9.6.1",
			"invalid exact declaration",
			abortUnattributedPut,
			"must not be present",
		)
		event := marshalTaskEvent("attempt", "producer", 3, payloadCapability, payload)
		if _, err := decodeTaskEvent(event); err == nil ||
			!strings.Contains(err.Error(), "must not carry an abort") {
			t.Fatalf("expected exact capability error, got %v", err)
		}
	})

	t.Run("unknown required enum", func(t *testing.T) {
		exact := appendStringField(nil, 1, "task-execution")
		exact = appendVarintField(exact, 2, 1)
		payload := appendStringField(nil, 1, "put-unknown-outcome")
		payload = appendStringField(payload, 2, "cache-key")
		payload = appendVarintField(payload, 3, 99)
		payload = appendMessageField(payload, 10, exact)
		event := marshalTaskEvent("attempt", "producer", 2, payloadCachePut, payload)
		if _, err := decodeTaskEvent(event); err == nil ||
			!strings.Contains(err.Error(), "unsupported put_outcome") {
			t.Fatalf("expected unknown enum error, got %v", err)
		}
	})
}

func TestFrameLimit(t *testing.T) {
	var prefix [binary.MaxVarintLen64]byte
	count := binary.PutUvarint(prefix[:], maxFrameBytes+1)
	if _, err := readDelimited(bytes.NewReader(prefix[:count])); err == nil ||
		!strings.Contains(err.Error(), "exceeds") {
		t.Fatalf("expected oversized frame error, got %v", err)
	}
}

func serveJavaExactSequence(listener *net.UnixListener) error {
	connection, err := listener.AcceptUnix()
	if err != nil {
		return fmt.Errorf("accept Java client: %w", err)
	}
	defer connection.Close()
	if err := connection.SetDeadline(time.Now().Add(10 * time.Second)); err != nil {
		return fmt.Errorf("set Java client deadline: %w", err)
	}

	expectedPayloads := []int{
		payloadProducerHello,
		payloadCachePut,
		payloadCapability,
	}
	for index, expectedPayload := range expectedPayloads {
		eventBytes, err := readDelimited(connection)
		if err != nil {
			return fmt.Errorf("read Java event %d: %w", index+1, err)
		}
		event, err := decodeTaskEvent(eventBytes)
		if err != nil {
			return fmt.Errorf("decode Java event %d: %w", index+1, err)
		}
		if event.attemptID != "attempt-java-exact" ||
			event.producerInstance != "java-producer" ||
			event.sequence != uint64(index+1) ||
			event.payloadField != expectedPayload ||
			event.abortImmediately {
			return fmt.Errorf("unexpected Java event %d: %+v", index+1, event)
		}
		if err := protocDecodes(eventBytes, event.schemaDescription...); err != nil {
			return fmt.Errorf("validate Java event %d against schema: %w", index+1, err)
		}

		ack := marshalTaskEventAck(
			event.attemptID,
			event.producerInstance,
			event.sequence,
			ackAccepted,
		)
		if err := writeDelimited(connection, ack); err != nil {
			return fmt.Errorf("write acknowledgement %d: %w", index+1, err)
		}
	}
	return nil
}

func decodeTaskEvent(data []byte) (taskEvent, error) {
	fields, err := parseFields(data)
	if err != nil {
		return taskEvent{}, err
	}
	attemptID, err := requiredString(fields, 1, "attempt_id")
	if err != nil {
		return taskEvent{}, err
	}
	producerInstance, err := requiredString(fields, 2, "producer_instance_id")
	if err != nil {
		return taskEvent{}, err
	}
	sequence, err := requiredVarint(fields, 3, "sequence_number")
	if err != nil || sequence == 0 {
		if err == nil {
			err = errors.New("sequence_number must be greater than zero")
		}
		return taskEvent{}, err
	}
	payloadField, payload, err := requiredOneof(fields, []int{
		payloadProducerHello,
		payloadCapability,
		payloadCachePut,
	}, "payload")
	if err != nil {
		return taskEvent{}, err
	}

	event := taskEvent{
		attemptID:        attemptID,
		producerInstance: producerInstance,
		sequence:         sequence,
		payloadField:     payloadField,
		payload:          payload,
	}
	switch payloadField {
	case payloadProducerHello:
		if err := validateProducerHello(payload); err != nil {
			return taskEvent{}, err
		}
		event.schemaDescription = []string{
			`attempt_id: "attempt-java-exact"`,
			"producer_hello {",
			"major: 1",
			"PRODUCER_KIND_GRADLE_PLUGIN",
		}
	case payloadCachePut:
		abort, exact, err := validateCachePut(payload)
		if err != nil {
			return taskEvent{}, err
		}
		event.abortImmediately = abort
		if exact {
			event.schemaDescription = []string{
				"cache_put_observed {",
				`put_operation_id: "put-java-2"`,
				`task_execution_id: "task-execution-java-7"`,
				"TASK_OUTCOME_SUCCEEDED",
			}
		}
	case payloadCapability:
		abort, exact, err := validateCapability(payload)
		if err != nil {
			return taskEvent{}, err
		}
		event.abortImmediately = abort
		if exact {
			event.schemaDescription = []string{
				"correlation_capability_declared {",
				"CORRELATION_CAPABILITY_EXACT",
				`gradle_version: "9.6.1"`,
			}
		}
	}
	return event, nil
}

func validateProducerHello(data []byte) error {
	fields, err := parseFields(data)
	if err != nil {
		return err
	}
	versionBytes, err := requiredMessage(fields, 1, "protocol_version")
	if err != nil {
		return err
	}
	versionFields, err := parseFields(versionBytes)
	if err != nil {
		return err
	}
	major, err := requiredVarint(versionFields, 1, "protocol_version.major")
	if err != nil {
		return err
	}
	minor, err := optionalVarint(versionFields, 2, "protocol_version.minor")
	if err != nil {
		return err
	}
	if major != 1 || minor != 0 {
		return fmt.Errorf("unsupported protocol version %d.%d", major, minor)
	}
	producer, err := requiredVarint(fields, 2, "producer")
	if err != nil || producer == 0 || producer > 2 {
		if err == nil {
			err = fmt.Errorf("unsupported producer %d", producer)
		}
		return err
	}
	_, err = requiredString(fields, 3, "implementation_version")
	return err
}

func validateCachePut(data []byte) (bool, bool, error) {
	fields, err := parseFields(data)
	if err != nil {
		return false, false, err
	}
	if _, err := requiredString(fields, 1, "put_operation_id"); err != nil {
		return false, false, err
	}
	if _, err := requiredString(fields, 2, "native_cache_key"); err != nil {
		return false, false, err
	}
	outcome, err := requiredVarint(fields, 3, "put_outcome")
	if err != nil || outcome == 0 || outcome > 4 {
		if err == nil {
			err = fmt.Errorf("unsupported put_outcome %d", outcome)
		}
		return false, false, err
	}
	attributionField, attribution, err := requiredOneof(
		fields,
		[]int{10, 11},
		"cache PUT attribution",
	)
	if err != nil {
		return false, false, err
	}
	attributionFields, err := parseFields(attribution)
	if err != nil {
		return false, false, err
	}
	if attributionField == 10 {
		if _, err := requiredString(
			attributionFields,
			1,
			"task_execution_id",
		); err != nil {
			return false, false, err
		}
		taskOutcome, err := requiredVarint(
			attributionFields,
			2,
			"task_outcome",
		)
		if err != nil || taskOutcome == 0 || taskOutcome > 3 {
			if err == nil {
				err = fmt.Errorf("unsupported task_outcome %d", taskOutcome)
			}
			return false, false, err
		}
		return false, true, nil
	}

	reason, err := requiredVarint(attributionFields, 1, "unattributed.reason")
	if err != nil || reason == 0 || reason > 4 {
		if err == nil {
			err = fmt.Errorf("unsupported unattributed reason %d", reason)
		}
		return false, false, err
	}
	abort, err := requiredMessage(
		attributionFields,
		2,
		"unattributed whole-attempt abort",
	)
	if err != nil {
		return false, false, err
	}
	if err := validateWholeAttemptAbort(abort, abortUnattributedPut); err != nil {
		return false, false, err
	}
	return true, false, nil
}

func validateCapability(data []byte) (bool, bool, error) {
	fields, err := parseFields(data)
	if err != nil {
		return false, false, err
	}
	capability, err := requiredVarint(fields, 1, "capability")
	if err != nil || capability == 0 {
		if err == nil {
			err = errors.New("capability must be specified")
		}
		return false, false, err
	}
	if _, err := requiredString(fields, 2, "gradle_version"); err != nil {
		return false, false, err
	}
	abortFields := matchingFields(fields, 10)
	if len(abortFields) > 1 {
		return false, false, errors.New("whole_attempt_abort appears more than once")
	}
	switch capability {
	case capabilityExact:
		if len(abortFields) != 0 {
			return false, false, errors.New("exact capability must not carry an abort")
		}
		return false, true, nil
	case capabilityUnavailable:
		if len(abortFields) != 1 {
			return false, false, errors.New("unavailable capability requires a whole-attempt abort")
		}
		if err := validateWholeAttemptAbort(abortFields[0].payload, 0); err != nil {
			return false, false, err
		}
		return true, false, nil
	default:
		return false, false, fmt.Errorf("unsupported correlation capability %d", capability)
	}
}

func validateWholeAttemptAbort(data []byte, expectedReason uint64) error {
	fields, err := parseFields(data)
	if err != nil {
		return err
	}
	reason, err := requiredVarint(fields, 1, "whole_attempt_abort.reason")
	if err != nil || reason == 0 || reason > 5 {
		if err == nil {
			err = fmt.Errorf("unsupported whole-attempt abort reason %d", reason)
		}
		return err
	}
	if expectedReason != 0 && reason != expectedReason {
		return fmt.Errorf(
			"whole-attempt abort reason is %d, want %d",
			reason,
			expectedReason,
		)
	}
	_, err = requiredString(fields, 2, "whole_attempt_abort.detail")
	return err
}

func decodeTaskEventAck(data []byte) (taskEventAck, error) {
	fields, err := parseFields(data)
	if err != nil {
		return taskEventAck{}, err
	}
	attemptID, err := requiredString(fields, 1, "ack.attempt_id")
	if err != nil {
		return taskEventAck{}, err
	}
	producer, err := requiredString(fields, 2, "ack.producer_instance_id")
	if err != nil {
		return taskEventAck{}, err
	}
	sequence, err := requiredVarint(fields, 3, "ack.sequence_number")
	if err != nil {
		return taskEventAck{}, err
	}
	status, err := requiredVarint(fields, 4, "ack.status")
	if err != nil || status == 0 || status > 3 {
		if err == nil {
			err = fmt.Errorf("unsupported ack.status %d", status)
		}
		return taskEventAck{}, err
	}
	return taskEventAck{
		attemptID:        attemptID,
		producerInstance: producer,
		sequence:         sequence,
		status:           status,
	}, nil
}

func marshalTaskEvent(
	attemptID string,
	producerInstance string,
	sequence uint64,
	payloadField int,
	payload []byte,
) []byte {
	message := appendStringField(nil, 1, attemptID)
	message = appendStringField(message, 2, producerInstance)
	message = appendVarintField(message, 3, sequence)
	return appendMessageField(message, payloadField, payload)
}

func marshalTaskEventAck(
	attemptID string,
	producerInstance string,
	sequence uint64,
	status uint64,
) []byte {
	message := appendStringField(nil, 1, attemptID)
	message = appendStringField(message, 2, producerInstance)
	message = appendVarintField(message, 3, sequence)
	return appendVarintField(message, 4, status)
}

func marshalProducerHello(
	major uint64,
	minor uint64,
	producer uint64,
	implementationVersion string,
) []byte {
	version := appendVarintField(nil, 1, major)
	if minor != 0 {
		version = appendVarintField(version, 2, minor)
	}
	message := appendMessageField(nil, 1, version)
	message = appendVarintField(message, 2, producer)
	return appendStringField(message, 3, implementationVersion)
}

func marshalUnattributedPut(
	putOperationID string,
	nativeCacheKey string,
	putOutcome uint64,
	unattributedReason uint64,
	abortReason uint64,
	abortDetail string,
) []byte {
	abort := marshalWholeAttemptAbort(abortReason, abortDetail)
	unattributed := appendVarintField(nil, 1, unattributedReason)
	unattributed = appendMessageField(unattributed, 2, abort)
	message := appendStringField(nil, 1, putOperationID)
	message = appendStringField(message, 2, nativeCacheKey)
	message = appendVarintField(message, 3, putOutcome)
	return appendMessageField(message, 11, unattributed)
}

func marshalCapability(
	capability uint64,
	gradleVersion string,
	detail string,
	abortReason uint64,
	abortDetail string,
) []byte {
	message := appendVarintField(nil, 1, capability)
	message = appendStringField(message, 2, gradleVersion)
	if detail != "" {
		message = appendStringField(message, 3, detail)
	}
	if abortReason != 0 {
		message = appendMessageField(
			message,
			10,
			marshalWholeAttemptAbort(abortReason, abortDetail),
		)
	}
	return message
}

func marshalWholeAttemptAbort(reason uint64, detail string) []byte {
	message := appendVarintField(nil, 1, reason)
	return appendStringField(message, 2, detail)
}

func appendStringField(data []byte, number int, value string) []byte {
	return appendMessageField(data, number, []byte(value))
}

func appendMessageField(data []byte, number int, value []byte) []byte {
	data = appendUvarint(data, uint64(number<<3|2))
	data = appendUvarint(data, uint64(len(value)))
	return append(data, value...)
}

func appendVarintField(data []byte, number int, value uint64) []byte {
	data = appendUvarint(data, uint64(number<<3))
	return appendUvarint(data, value)
}

func appendUvarint(data []byte, value uint64) []byte {
	var encoded [binary.MaxVarintLen64]byte
	count := binary.PutUvarint(encoded[:], value)
	return append(data, encoded[:count]...)
}

func parseFields(data []byte) ([]wireField, error) {
	fields := make([]wireField, 0)
	for len(data) > 0 {
		key, count := binary.Uvarint(data)
		if count <= 0 {
			return nil, errors.New("invalid Protobuf field key")
		}
		data = data[count:]
		number := int(key >> 3)
		wire := int(key & 7)
		if number == 0 {
			return nil, errors.New("Protobuf field number zero is invalid")
		}
		field := wireField{number: number, wire: wire}
		switch wire {
		case 0:
			value, valueCount := binary.Uvarint(data)
			if valueCount <= 0 {
				return nil, fmt.Errorf("invalid varint for field %d", number)
			}
			field.varint = value
			data = data[valueCount:]
		case 2:
			length, lengthCount := binary.Uvarint(data)
			if lengthCount <= 0 {
				return nil, fmt.Errorf("invalid length for field %d", number)
			}
			data = data[lengthCount:]
			if length > uint64(len(data)) {
				return nil, fmt.Errorf("truncated field %d", number)
			}
			field.payload = append([]byte(nil), data[:int(length)]...)
			data = data[int(length):]
		default:
			return nil, fmt.Errorf("unsupported wire type %d for field %d", wire, number)
		}
		fields = append(fields, field)
	}
	return fields, nil
}

func requiredString(fields []wireField, number int, name string) (string, error) {
	field, err := requiredField(fields, number, 2, name)
	if err != nil {
		return "", err
	}
	if len(field.payload) == 0 {
		return "", fmt.Errorf("%s must be non-empty", name)
	}
	return string(field.payload), nil
}

func requiredMessage(fields []wireField, number int, name string) ([]byte, error) {
	field, err := requiredField(fields, number, 2, name)
	if err != nil {
		return nil, err
	}
	if len(field.payload) == 0 {
		return nil, fmt.Errorf("%s must be non-empty", name)
	}
	return field.payload, nil
}

func requiredVarint(fields []wireField, number int, name string) (uint64, error) {
	field, err := requiredField(fields, number, 0, name)
	if err != nil {
		return 0, err
	}
	return field.varint, nil
}

func optionalVarint(fields []wireField, number int, name string) (uint64, error) {
	matches := matchingFields(fields, number)
	if len(matches) == 0 {
		return 0, nil
	}
	if len(matches) != 1 || matches[0].wire != 0 {
		return 0, fmt.Errorf("%s must appear at most once as a varint", name)
	}
	return matches[0].varint, nil
}

func requiredField(
	fields []wireField,
	number int,
	wire int,
	name string,
) (wireField, error) {
	matches := matchingFields(fields, number)
	if len(matches) != 1 {
		return wireField{}, fmt.Errorf("%s must appear exactly once", name)
	}
	if matches[0].wire != wire {
		return wireField{}, fmt.Errorf("%s has wire type %d, want %d", name, matches[0].wire, wire)
	}
	return matches[0], nil
}

func requiredOneof(
	fields []wireField,
	numbers []int,
	name string,
) (int, []byte, error) {
	var selected []wireField
	for _, number := range numbers {
		selected = append(selected, matchingFields(fields, number)...)
	}
	if len(selected) != 1 || selected[0].wire != 2 {
		return 0, nil, fmt.Errorf("%s requires exactly one attribution or payload arm", name)
	}
	if len(selected[0].payload) == 0 {
		return 0, nil, fmt.Errorf("%s arm must be non-empty", name)
	}
	return selected[0].number, selected[0].payload, nil
}

func matchingFields(fields []wireField, number int) []wireField {
	var matches []wireField
	for _, field := range fields {
		if field.number == number {
			matches = append(matches, field)
		}
	}
	return matches
}

func writeDelimited(writer io.Writer, message []byte) error {
	if len(message) > maxFrameBytes {
		return fmt.Errorf("frame length %d exceeds %d bytes", len(message), maxFrameBytes)
	}
	var prefix [binary.MaxVarintLen64]byte
	count := binary.PutUvarint(prefix[:], uint64(len(message)))
	if err := writeAll(writer, prefix[:count]); err != nil {
		return err
	}
	return writeAll(writer, message)
}

func writeAll(writer io.Writer, data []byte) error {
	for len(data) > 0 {
		count, err := writer.Write(data)
		if err != nil {
			return err
		}
		if count == 0 {
			return io.ErrShortWrite
		}
		data = data[count:]
	}
	return nil
}

func readDelimited(reader io.Reader) ([]byte, error) {
	length, err := readUvarint(reader)
	if err != nil {
		return nil, err
	}
	if length > maxFrameBytes {
		return nil, fmt.Errorf("frame length %d exceeds %d bytes", length, maxFrameBytes)
	}
	message := make([]byte, int(length))
	if _, err := io.ReadFull(reader, message); err != nil {
		return nil, err
	}
	return message, nil
}

func readUvarint(reader io.Reader) (uint64, error) {
	var value uint64
	for shift := uint(0); shift < 70; shift += 7 {
		var single [1]byte
		if _, err := io.ReadFull(reader, single[:]); err != nil {
			return 0, err
		}
		if single[0] < 0x80 {
			if shift == 63 && single[0] > 1 {
				return 0, errors.New("varint overflow")
			}
			return value | uint64(single[0])<<shift, nil
		}
		value |= uint64(single[0]&0x7f) << shift
	}
	return 0, errors.New("varint overflow")
}

type javaResult struct {
	output string
	err    error
}

func runJava(t *testing.T, arguments ...string) javaResult {
	t.Helper()
	command, output := javaCommand(t, arguments...)
	err := command.Run()
	return javaResult{output: output.String(), err: err}
}

func startJava(
	t *testing.T,
	arguments ...string,
) (*exec.Cmd, *bytes.Buffer) {
	t.Helper()
	command, output := javaCommand(t, arguments...)
	if err := command.Start(); err != nil {
		t.Fatalf("start Java participant: %v", err)
	}
	t.Cleanup(func() {
		if command.ProcessState == nil {
			_ = command.Process.Kill()
			_ = command.Wait()
		}
	})
	return command, output
}

func javaCommand(t *testing.T, arguments ...string) (*exec.Cmd, *bytes.Buffer) {
	t.Helper()
	java := requiredEnvironment(t, "BUILDOPT_PROTO_CONFORMANCE_JAVA")
	classDirectory := requiredEnvironment(t, "BUILDOPT_PROTO_CONFORMANCE_CLASSES")
	contextWithTimeout, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	t.Cleanup(cancel)
	commandArguments := []string{
		"-cp",
		classDirectory,
		"dev.buildopt.contracts.localevents.v1.TaskEventsRoundTrip",
	}
	commandArguments = append(commandArguments, arguments...)
	command := exec.CommandContext(contextWithTimeout, java, commandArguments...)
	output := &bytes.Buffer{}
	command.Stdout = output
	command.Stderr = output
	return command, output
}

func waitForFile(
	t *testing.T,
	path string,
	command *exec.Cmd,
	output *bytes.Buffer,
) {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		if _, err := os.Stat(path); err == nil {
			return
		}
		if command.ProcessState != nil && command.ProcessState.Exited() {
			t.Fatalf("Java server exited before readiness: %s", output.String())
		}
		time.Sleep(10 * time.Millisecond)
	}
	command.Process.Kill()
	command.Wait()
	t.Fatalf("Java server did not become ready: %s", output.String())
}

func assertProtocDecodes(t *testing.T, message []byte, wants ...string) {
	t.Helper()
	if err := protocDecodes(message, wants...); err != nil {
		t.Fatal(err)
	}
}

func protocDecodes(message []byte, wants ...string) error {
	protoc := os.Getenv("BUILDOPT_PROTO_CONFORMANCE_PROTOC")
	if protoc == "" {
		return errors.New("missing BUILDOPT_PROTO_CONFORMANCE_PROTOC")
	}
	protoRoot := os.Getenv("BUILDOPT_PROTO_CONFORMANCE_ROOT")
	if protoRoot == "" {
		return errors.New("missing BUILDOPT_PROTO_CONFORMANCE_ROOT")
	}
	protoFile := filepath.Join(protoRoot, "local-events", "v1", "task_events.proto")
	command := exec.Command(
		protoc,
		"--proto_path="+protoRoot,
		"--decode=buildopt.local_events.v1.TaskEvent",
		protoFile,
	)
	command.Stdin = bytes.NewReader(message)
	output, err := command.CombinedOutput()
	if err != nil {
		return fmt.Errorf("protoc could not decode conformance message: %w\n%s", err, output)
	}
	for _, want := range wants {
		if !bytes.Contains(output, []byte(want)) {
			return fmt.Errorf("protoc output is missing %q:\n%s", want, output)
		}
	}
	return nil
}

func requiredEnvironment(t *testing.T, name string) string {
	t.Helper()
	value := os.Getenv(name)
	if value == "" {
		t.Fatalf("missing %s", name)
	}
	return value
}

func requireCrossLanguageEnvironment(t *testing.T) {
	t.Helper()
	for _, name := range []string{
		"BUILDOPT_PROTO_CONFORMANCE_CLASSES",
		"BUILDOPT_PROTO_CONFORMANCE_JAVA",
		"BUILDOPT_PROTO_CONFORMANCE_PROTOC",
		"BUILDOPT_PROTO_CONFORMANCE_ROOT",
	} {
		if os.Getenv(name) == "" {
			t.Skip("run cross-language checks through ./dev/check-task-events-proto")
		}
	}
}
