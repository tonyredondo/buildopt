//go:build linux

package launcher

import (
	"bytes"
	"encoding/binary"
	"errors"
	"io"
	"net"
	"os"
	"strings"
	"testing"
	"time"
)

func TestPluginHandshakeRoundTrip(t *testing.T) {
	server, err := startPluginHandshake()
	if err != nil {
		t.Fatalf("start plugin handshake: %v", err)
	}
	directory := server.directory

	info, err := os.Stat(directory)
	if err != nil {
		t.Fatalf("stat private handshake directory: %v", err)
	}
	if permissions := info.Mode().Perm(); permissions != 0o700 {
		t.Fatalf("handshake directory permissions = %o, want 700", permissions)
	}

	environment := server.childEnvironment([]string{
		"PATH=/usr/bin",
		pluginAttemptIDEnvironment + "=parent-attempt",
		pluginSocketEnvironment + "=/tmp/parent.sock",
		pluginTokenEnvironment + "=parent-token",
	})
	if value := environmentValue(environment, pluginAttemptIDEnvironment); value != server.attemptID {
		t.Fatalf("child attempt ID = %q, want %q", value, server.attemptID)
	}
	socketPath := environmentValue(environment, pluginSocketEnvironment)
	if socketPath != server.listener.Addr().String() {
		t.Fatalf(
			"child socket = %q, want %q",
			socketPath,
			server.listener.Addr().String(),
		)
	}
	if value := environmentValue(environment, pluginTokenEnvironment); value != server.tokenText {
		t.Fatalf("child event token was not replaced with the invocation credential")
	}
	for _, key := range []string{
		pluginAttemptIDEnvironment,
		pluginSocketEnvironment,
		pluginTokenEnvironment,
	} {
		if count := environmentKeyCount(environment, key); count != 1 {
			t.Fatalf("child environment contains %d %s entries, want 1", count, key)
		}
	}

	connection, err := net.DialTimeout("unix", socketPath, time.Second)
	if err != nil {
		t.Fatalf("connect to plugin handshake: %v", err)
	}
	if err := connection.SetDeadline(time.Now().Add(time.Second)); err != nil {
		t.Fatalf("set client deadline: %v", err)
	}
	writeTestPluginAuthentication(t, connection, server.token)

	producerInstanceID := "gradle-producer-test"
	event := marshalTestPluginHello(
		server.attemptID,
		producerInstanceID,
		1,
		1,
		0,
		pluginProducerKindGradle,
		"plugin-test-version",
	)
	if err := writePluginDelimited(connection, event); err != nil {
		t.Fatalf("write ProducerHello: %v", err)
	}
	ackBytes, err := readPluginDelimited(connection)
	if err != nil {
		t.Fatalf("read ProducerHello acknowledgement: %v", err)
	}
	second, err := net.DialTimeout("unix", socketPath, time.Second)
	if err == nil {
		_ = second.Close()
		t.Fatal("single-producer handshake accepted a second connection")
	}
	if err := connection.Close(); err != nil {
		t.Fatalf("close handshake client: %v", err)
	}

	ackFields, err := parsePluginFields(ackBytes)
	if err != nil {
		t.Fatalf("parse acknowledgement: %v", err)
	}
	assertPluginString(t, ackFields, 1, server.attemptID)
	assertPluginString(t, ackFields, 2, producerInstanceID)
	assertPluginVarint(t, ackFields, 3, 1)
	assertPluginVarint(t, ackFields, 4, pluginAckAccepted)

	result := server.finish()
	if result.err != nil {
		t.Fatalf("finish plugin handshake: %v", result.err)
	}
	if !result.connected ||
		result.producerInstanceID != producerInstanceID ||
		result.implementationVersion != "plugin-test-version" {
		t.Fatalf("unexpected handshake result: %+v", result)
	}
	if _, err := os.Stat(directory); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("handshake directory remains after finish: %v", err)
	}
}

func TestPluginHandshakeSemanticRejections(t *testing.T) {
	testCases := []struct {
		name       string
		attemptID  string
		sequence   uint64
		major      uint64
		producer   uint64
		version    string
		wantDetail string
	}{
		{
			name:       "sequence starts after one",
			attemptID:  "attempt",
			sequence:   2,
			major:      1,
			producer:   pluginProducerKindGradle,
			version:    "version",
			wantDetail: "want 1",
		},
		{
			name:       "unsupported protocol",
			attemptID:  "attempt",
			sequence:   1,
			major:      2,
			producer:   pluginProducerKindGradle,
			version:    "version",
			wantDetail: "unsupported protocol version",
		},
		{
			name:       "wrong producer",
			attemptID:  "attempt",
			sequence:   1,
			major:      1,
			producer:   2,
			version:    "version",
			wantDetail: "want Gradle plugin",
		},
		{
			name:       "empty implementation version",
			attemptID:  "attempt",
			sequence:   1,
			major:      1,
			producer:   pluginProducerKindGradle,
			wantDetail: "implementation_version must be non-empty",
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			event := marshalTestPluginHello(
				testCase.attemptID,
				"producer",
				testCase.sequence,
				testCase.major,
				0,
				testCase.producer,
				testCase.version,
			)
			_, err := decodePluginHello(event)
			if err == nil || !strings.Contains(err.Error(), testCase.wantDetail) {
				t.Fatalf("decode error = %v, want %q", err, testCase.wantDetail)
			}
		})
	}
}

func TestPluginHandshakeRejectsWrongInvocation(t *testing.T) {
	server, err := startPluginHandshake()
	if err != nil {
		t.Fatalf("start plugin handshake: %v", err)
	}
	socketPath := server.listener.Addr().String()

	connection, err := net.DialTimeout("unix", socketPath, time.Second)
	if err != nil {
		t.Fatalf("connect to plugin handshake: %v", err)
	}
	if err := connection.SetDeadline(time.Now().Add(time.Second)); err != nil {
		t.Fatalf("set client deadline: %v", err)
	}
	writeTestPluginAuthentication(t, connection, server.token)
	event := marshalTestPluginHello(
		"wrong-attempt",
		"producer",
		1,
		1,
		0,
		pluginProducerKindGradle,
		"version",
	)
	if err := writePluginDelimited(connection, event); err != nil {
		t.Fatalf("write mismatched ProducerHello: %v", err)
	}
	if _, err := readPluginDelimited(connection); err == nil {
		t.Fatal("mismatched ProducerHello unexpectedly received an acknowledgement")
	}
	_ = connection.Close()

	result := server.finish()
	if !result.connected ||
		result.err == nil ||
		!strings.Contains(result.err.Error(), "does not match this invocation") {
		t.Fatalf("unexpected mismatched-invocation result: %+v", result)
	}
}

func TestPluginHandshakeRejectsWrongCredential(t *testing.T) {
	server, err := startPluginHandshake()
	if err != nil {
		t.Fatalf("start plugin handshake: %v", err)
	}

	connection, err := net.DialTimeout(
		"unix",
		server.listener.Addr().String(),
		time.Second,
	)
	if err != nil {
		t.Fatalf("connect to plugin handshake: %v", err)
	}
	wrongToken := bytes.Repeat([]byte{0x5a}, pluginTokenBytes)
	if err := connection.SetDeadline(time.Now().Add(time.Second)); err != nil {
		t.Fatalf("set invalid-credential deadline: %v", err)
	}
	writeTestPluginAuthentication(t, connection, wrongToken)
	if _, err := connection.Read(make([]byte, 1)); err == nil {
		t.Fatal("invalid plugin credential did not close the connection")
	}
	_ = connection.Close()

	result := server.finish()
	if !result.connected ||
		result.err == nil ||
		!strings.Contains(result.err.Error(), "invalid plugin event credential") {
		t.Fatalf("unexpected invalid-credential result: %+v", result)
	}
}

func TestPluginHandshakeFrameLimit(t *testing.T) {
	prefix := binary.AppendUvarint(nil, pluginMaxFrameBytes+1)
	if _, err := readPluginDelimited(bytes.NewReader(prefix)); err == nil ||
		!strings.Contains(err.Error(), "exceeds") {
		t.Fatalf("oversized read error = %v", err)
	}
	if err := writePluginDelimited(
		&bytes.Buffer{},
		make([]byte, pluginMaxFrameBytes+1),
	); err == nil || !strings.Contains(err.Error(), "exceeds") {
		t.Fatalf("oversized write error = %v", err)
	}
}

func TestPluginHandshakeRejectsTruncatedLength(t *testing.T) {
	_, err := parsePluginFields([]byte{0x0a, 0x80})
	if !errors.Is(err, io.ErrUnexpectedEOF) {
		t.Fatalf("truncated length error = %v, want unexpected EOF", err)
	}
}

func TestPluginHandshakeFinishesWithoutProducer(t *testing.T) {
	server, err := startPluginHandshake()
	if err != nil {
		t.Fatalf("start plugin handshake: %v", err)
	}
	directory := server.directory
	result := server.finish()
	if result.connected || result.err != nil {
		t.Fatalf("unexpected no-producer result: %+v", result)
	}
	if _, err := os.Stat(directory); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("handshake directory remains after no-producer finish: %v", err)
	}
}

func writeTestPluginAuthentication(
	t *testing.T,
	connection net.Conn,
	token []byte,
) {
	t.Helper()
	preface := append([]byte(pluginAuthMagic), token...)
	if _, err := connection.Write(preface); err != nil {
		t.Fatalf("write plugin authentication: %v", err)
	}
}

func marshalTestPluginHello(
	attemptID string,
	producerInstanceID string,
	sequenceNumber uint64,
	major uint64,
	minor uint64,
	producer uint64,
	implementationVersion string,
) []byte {
	version := appendPluginVarintField(nil, 1, major)
	if minor != 0 {
		version = appendPluginVarintField(version, 2, minor)
	}
	hello := appendPluginMessageField(nil, 1, version)
	hello = appendPluginVarintField(hello, 2, producer)
	hello = appendPluginStringField(hello, 3, implementationVersion)

	event := appendPluginStringField(nil, 1, attemptID)
	event = appendPluginStringField(event, 2, producerInstanceID)
	event = appendPluginVarintField(event, 3, sequenceNumber)
	return appendPluginMessageField(event, pluginProducerHelloField, hello)
}

func assertPluginString(
	t *testing.T,
	fields []pluginWireField,
	number int,
	want string,
) {
	t.Helper()
	value, err := requiredPluginString(fields, number, "test string")
	if err != nil {
		t.Fatalf("read field %d: %v", number, err)
	}
	if value != want {
		t.Fatalf("field %d = %q, want %q", number, value, want)
	}
}

func assertPluginVarint(
	t *testing.T,
	fields []pluginWireField,
	number int,
	want uint64,
) {
	t.Helper()
	value, err := requiredPluginVarint(fields, number, "test varint")
	if err != nil {
		t.Fatalf("read field %d: %v", number, err)
	}
	if value != want {
		t.Fatalf("field %d = %d, want %d", number, value, want)
	}
}
