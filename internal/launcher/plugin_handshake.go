package launcher

import (
	"bufio"
	"crypto/rand"
	"crypto/subtle"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
	"sync"
	"syscall"
	"time"
	"unicode/utf8"
)

const (
	pluginAttemptIDEnvironment = "BUILDOPT_PLUGIN_ATTEMPT_ID"
	pluginSocketEnvironment    = "BUILDOPT_PLUGIN_EVENT_SOCKET"
	pluginTokenEnvironment     = "BUILDOPT_PLUGIN_EVENT_TOKEN"

	pluginHandshakeTimeout = 5 * time.Second
	pluginMaxFrameBytes    = 1 << 20
	pluginTokenBytes       = 32
	pluginAuthMagic        = "BOA1"

	pluginProducerHelloField = 10
	pluginProducerKindGradle = 1
	pluginAckAccepted        = 1
)

type pluginHandshakeResult struct {
	connected             bool
	implementationVersion string
	err                   error
}

type pluginHandshakeServer struct {
	attemptID string
	directory string
	listener  *net.UnixListener
	token     []byte
	tokenText string
	result    chan pluginHandshakeResult

	mutex      sync.Mutex
	connection net.Conn
	closing    bool
}

type pluginWireField struct {
	number  int
	wire    int
	varint  uint64
	payload []byte
}

type pluginHello struct {
	attemptID             string
	producerInstanceID    string
	sequenceNumber        uint64
	implementationVersion string
}

func startPluginHandshake() (*pluginHandshakeServer, error) {
	directory, err := os.MkdirTemp("", "buildopt-handshake-")
	if err != nil {
		return nil, fmt.Errorf("create private handshake directory: %w", err)
	}

	socketPath := filepath.Join(directory, "events.sock")
	listener, err := net.ListenUnix(
		"unix",
		&net.UnixAddr{Name: socketPath, Net: "unix"},
	)
	if err != nil {
		_ = os.RemoveAll(directory)
		return nil, fmt.Errorf("listen for plugin handshake: %w", err)
	}

	attemptID, err := newPluginAttemptID()
	if err != nil {
		_ = listener.Close()
		_ = os.RemoveAll(directory)
		return nil, err
	}
	token, tokenText, err := newLocalSecret(pluginTokenBytes)
	if err != nil {
		_ = listener.Close()
		_ = os.RemoveAll(directory)
		return nil, fmt.Errorf("generate plugin event credential: %w", err)
	}

	server := &pluginHandshakeServer{
		attemptID: attemptID,
		directory: directory,
		listener:  listener,
		token:     token,
		tokenText: tokenText,
		result:    make(chan pluginHandshakeResult, 1),
	}
	go server.serve()
	return server, nil
}

func (server *pluginHandshakeServer) childEnvironment(
	environment []string,
) []string {
	return replaceEnvironment(
		environment,
		map[string]string{
			pluginAttemptIDEnvironment: server.attemptID,
			pluginSocketEnvironment:    server.listener.Addr().String(),
			pluginTokenEnvironment:     server.tokenText,
		},
	)
}

func (server *pluginHandshakeServer) finish() pluginHandshakeResult {
	server.mutex.Lock()
	server.closing = true
	_ = server.listener.Close()
	if server.connection != nil {
		_ = server.connection.Close()
	}
	server.mutex.Unlock()

	result := <-server.result
	_ = os.RemoveAll(server.directory)
	return result
}

func (server *pluginHandshakeServer) serve() {
	connection, err := server.listener.AcceptUnix()
	if err != nil {
		server.mutex.Lock()
		closing := server.closing
		server.mutex.Unlock()
		if closing || errors.Is(err, net.ErrClosed) {
			server.result <- pluginHandshakeResult{}
			return
		}
		server.result <- pluginHandshakeResult{
			err: fmt.Errorf("accept plugin handshake: %w", err),
		}
		return
	}

	server.mutex.Lock()
	if server.closing {
		server.mutex.Unlock()
		_ = connection.Close()
		server.result <- pluginHandshakeResult{}
		return
	}
	server.connection = connection
	server.mutex.Unlock()

	defer connection.Close()
	if err := connection.SetDeadline(time.Now().Add(pluginHandshakeTimeout)); err != nil {
		server.result <- pluginHandshakeResult{
			connected: true,
			err:       fmt.Errorf("set plugin handshake deadline: %w", err),
		}
		return
	}
	if err := authenticatePluginConnection(connection, server.token); err != nil {
		server.result <- pluginHandshakeResult{
			connected: true,
			err:       fmt.Errorf("authenticate ProducerHello: %w", err),
		}
		return
	}

	eventBytes, err := readPluginDelimited(connection)
	if err != nil {
		server.result <- pluginHandshakeResult{
			connected: true,
			err:       fmt.Errorf("read ProducerHello: %w", err),
		}
		return
	}
	hello, err := decodePluginHello(eventBytes)
	if err != nil {
		server.result <- pluginHandshakeResult{
			connected: true,
			err:       fmt.Errorf("reject ProducerHello: %w", err),
		}
		return
	}
	if hello.attemptID != server.attemptID {
		server.result <- pluginHandshakeResult{
			connected: true,
			err: fmt.Errorf(
				"reject ProducerHello: attempt_id does not match this invocation",
			),
		}
		return
	}

	ack := marshalPluginAck(
		hello.attemptID,
		hello.producerInstanceID,
		hello.sequenceNumber,
		pluginAckAccepted,
	)
	if err := writePluginDelimited(connection, ack); err != nil {
		server.result <- pluginHandshakeResult{
			connected: true,
			err:       fmt.Errorf("acknowledge ProducerHello: %w", err),
		}
		return
	}

	server.result <- pluginHandshakeResult{
		connected:             true,
		implementationVersion: hello.implementationVersion,
	}
}

func authenticatePluginConnection(
	connection *net.UnixConn,
	expectedToken []byte,
) error {
	rawConnection, err := connection.SyscallConn()
	if err != nil {
		return errors.New("inspect plugin peer")
	}
	var peer *syscall.Ucred
	var credentialErr error
	if err := rawConnection.Control(func(fileDescriptor uintptr) {
		peer, credentialErr = syscall.GetsockoptUcred(
			int(fileDescriptor),
			syscall.SOL_SOCKET,
			syscall.SO_PEERCRED,
		)
	}); err != nil || credentialErr != nil || peer == nil {
		return errors.New("inspect plugin peer")
	}
	if peer.Uid != uint32(os.Geteuid()) {
		return errors.New("plugin peer user does not own the launcher")
	}

	preface := make([]byte, len(pluginAuthMagic)+pluginTokenBytes)
	if _, err := io.ReadFull(connection, preface); err != nil {
		return errors.New("read plugin authentication preface")
	}
	if subtle.ConstantTimeCompare(
		preface[:len(pluginAuthMagic)],
		[]byte(pluginAuthMagic),
	) != 1 ||
		subtle.ConstantTimeCompare(
			preface[len(pluginAuthMagic):],
			expectedToken,
		) != 1 {
		return errors.New("invalid plugin event credential")
	}
	return nil
}

func decodePluginHello(data []byte) (pluginHello, error) {
	fields, err := parsePluginFields(data)
	if err != nil {
		return pluginHello{}, err
	}
	attemptID, err := requiredPluginString(fields, 1, "attempt_id")
	if err != nil {
		return pluginHello{}, err
	}
	producerInstanceID, err := requiredPluginString(
		fields,
		2,
		"producer_instance_id",
	)
	if err != nil {
		return pluginHello{}, err
	}
	sequenceNumber, err := requiredPluginVarint(fields, 3, "sequence_number")
	if err != nil {
		return pluginHello{}, err
	}
	if sequenceNumber != 1 {
		return pluginHello{}, fmt.Errorf(
			"ProducerHello sequence_number is %d, want 1",
			sequenceNumber,
		)
	}

	payloadFields := matchingPluginFields(fields, pluginProducerHelloField)
	for _, number := range []int{11, 12} {
		payloadFields = append(
			payloadFields,
			matchingPluginFields(fields, number)...,
		)
	}
	if len(payloadFields) != 1 ||
		payloadFields[0].number != pluginProducerHelloField ||
		payloadFields[0].wire != 2 ||
		len(payloadFields[0].payload) == 0 {
		return pluginHello{}, errors.New(
			"first task event must contain exactly one ProducerHello payload",
		)
	}

	helloFields, err := parsePluginFields(payloadFields[0].payload)
	if err != nil {
		return pluginHello{}, fmt.Errorf("decode ProducerHello: %w", err)
	}
	versionBytes, err := requiredPluginMessage(
		helloFields,
		1,
		"protocol_version",
	)
	if err != nil {
		return pluginHello{}, err
	}
	versionFields, err := parsePluginFields(versionBytes)
	if err != nil {
		return pluginHello{}, fmt.Errorf("decode protocol_version: %w", err)
	}
	major, err := requiredPluginVarint(
		versionFields,
		1,
		"protocol_version.major",
	)
	if err != nil {
		return pluginHello{}, err
	}
	minor, err := optionalPluginVarint(
		versionFields,
		2,
		"protocol_version.minor",
	)
	if err != nil {
		return pluginHello{}, err
	}
	if major != 1 || minor != 0 {
		return pluginHello{}, fmt.Errorf(
			"unsupported protocol version %d.%d",
			major,
			minor,
		)
	}
	producer, err := requiredPluginVarint(helloFields, 2, "producer")
	if err != nil {
		return pluginHello{}, err
	}
	if producer != pluginProducerKindGradle {
		return pluginHello{}, fmt.Errorf(
			"ProducerHello producer is %d, want Gradle plugin",
			producer,
		)
	}
	implementationVersion, err := requiredPluginString(
		helloFields,
		3,
		"implementation_version",
	)
	if err != nil {
		return pluginHello{}, err
	}

	return pluginHello{
		attemptID:             attemptID,
		producerInstanceID:    producerInstanceID,
		sequenceNumber:        sequenceNumber,
		implementationVersion: implementationVersion,
	}, nil
}

func parsePluginFields(data []byte) ([]pluginWireField, error) {
	fields := make([]pluginWireField, 0, 4)
	for len(data) > 0 {
		key, count := binary.Uvarint(data)
		if count == 0 {
			return nil, io.ErrUnexpectedEOF
		}
		if count < 0 {
			return nil, errors.New("Protobuf field key overflows uint64")
		}
		data = data[count:]
		number := int(key >> 3)
		wire := int(key & 7)
		if number <= 0 {
			return nil, errors.New("invalid Protobuf field number")
		}

		field := pluginWireField{number: number, wire: wire}
		switch wire {
		case 0:
			value, valueCount := binary.Uvarint(data)
			if valueCount == 0 {
				return nil, io.ErrUnexpectedEOF
			}
			if valueCount < 0 {
				return nil, fmt.Errorf("field %d varint overflows uint64", number)
			}
			field.varint = value
			data = data[valueCount:]
		case 1:
			if len(data) < 8 {
				return nil, io.ErrUnexpectedEOF
			}
			data = data[8:]
		case 2:
			length, lengthCount := binary.Uvarint(data)
			if lengthCount == 0 {
				return nil, io.ErrUnexpectedEOF
			}
			if lengthCount < 0 || length > uint64(len(data)-lengthCount) {
				return nil, fmt.Errorf("invalid length for field %d", number)
			}
			data = data[lengthCount:]
			field.payload = data[:int(length)]
			data = data[int(length):]
		case 5:
			if len(data) < 4 {
				return nil, io.ErrUnexpectedEOF
			}
			data = data[4:]
		default:
			return nil, fmt.Errorf(
				"unsupported wire type %d for field %d",
				wire,
				number,
			)
		}
		fields = append(fields, field)
	}
	return fields, nil
}

func requiredPluginString(
	fields []pluginWireField,
	number int,
	name string,
) (string, error) {
	field, err := requiredPluginField(fields, number, 2, name)
	if err != nil {
		return "", err
	}
	if len(field.payload) == 0 {
		return "", fmt.Errorf("%s must be non-empty", name)
	}
	if !utf8.Valid(field.payload) {
		return "", fmt.Errorf("%s must be valid UTF-8", name)
	}
	return string(field.payload), nil
}

func requiredPluginMessage(
	fields []pluginWireField,
	number int,
	name string,
) ([]byte, error) {
	field, err := requiredPluginField(fields, number, 2, name)
	if err != nil {
		return nil, err
	}
	if len(field.payload) == 0 {
		return nil, fmt.Errorf("%s must be non-empty", name)
	}
	return field.payload, nil
}

func requiredPluginVarint(
	fields []pluginWireField,
	number int,
	name string,
) (uint64, error) {
	field, err := requiredPluginField(fields, number, 0, name)
	if err != nil {
		return 0, err
	}
	return field.varint, nil
}

func optionalPluginVarint(
	fields []pluginWireField,
	number int,
	name string,
) (uint64, error) {
	matches := matchingPluginFields(fields, number)
	if len(matches) == 0 {
		return 0, nil
	}
	if len(matches) != 1 || matches[0].wire != 0 {
		return 0, fmt.Errorf("%s must appear at most once as a varint", name)
	}
	return matches[0].varint, nil
}

func requiredPluginField(
	fields []pluginWireField,
	number int,
	wire int,
	name string,
) (pluginWireField, error) {
	matches := matchingPluginFields(fields, number)
	if len(matches) != 1 {
		return pluginWireField{}, fmt.Errorf("%s must appear exactly once", name)
	}
	if matches[0].wire != wire {
		return pluginWireField{}, fmt.Errorf(
			"%s has wire type %d, want %d",
			name,
			matches[0].wire,
			wire,
		)
	}
	return matches[0], nil
}

func matchingPluginFields(
	fields []pluginWireField,
	number int,
) []pluginWireField {
	matches := make([]pluginWireField, 0, 1)
	for _, field := range fields {
		if field.number == number {
			matches = append(matches, field)
		}
	}
	return matches
}

func marshalPluginAck(
	attemptID string,
	producerInstanceID string,
	sequenceNumber uint64,
	status uint64,
) []byte {
	message := appendPluginStringField(nil, 1, attemptID)
	message = appendPluginStringField(message, 2, producerInstanceID)
	message = appendPluginVarintField(message, 3, sequenceNumber)
	return appendPluginVarintField(message, 4, status)
}

func appendPluginStringField(data []byte, number int, value string) []byte {
	return appendPluginMessageField(data, number, []byte(value))
}

func appendPluginMessageField(data []byte, number int, value []byte) []byte {
	data = binary.AppendUvarint(data, uint64(number<<3|2))
	data = binary.AppendUvarint(data, uint64(len(value)))
	return append(data, value...)
}

func appendPluginVarintField(data []byte, number int, value uint64) []byte {
	data = binary.AppendUvarint(data, uint64(number<<3))
	return binary.AppendUvarint(data, value)
}

func readPluginDelimited(reader io.Reader) ([]byte, error) {
	buffered := bufio.NewReader(reader)
	length, err := binary.ReadUvarint(buffered)
	if err != nil {
		return nil, err
	}
	if length > pluginMaxFrameBytes {
		return nil, fmt.Errorf(
			"frame length %d exceeds %d bytes",
			length,
			pluginMaxFrameBytes,
		)
	}
	message := make([]byte, int(length))
	if _, err := io.ReadFull(buffered, message); err != nil {
		return nil, err
	}
	return message, nil
}

func writePluginDelimited(writer io.Writer, message []byte) error {
	if len(message) > pluginMaxFrameBytes {
		return fmt.Errorf(
			"frame length %d exceeds %d bytes",
			len(message),
			pluginMaxFrameBytes,
		)
	}
	prefix := binary.AppendUvarint(nil, uint64(len(message)))
	if _, err := writer.Write(prefix); err != nil {
		return err
	}
	_, err := writer.Write(message)
	return err
}

func newPluginAttemptID() (string, error) {
	var identifier [16]byte
	if _, err := rand.Read(identifier[:]); err != nil {
		return "", fmt.Errorf("generate plugin attempt ID: %w", err)
	}
	identifier[6] = identifier[6]&0x0f | 0x40
	identifier[8] = identifier[8]&0x3f | 0x80
	return fmt.Sprintf(
		"%x-%x-%x-%x-%x",
		identifier[0:4],
		identifier[4:6],
		identifier[6:8],
		identifier[8:10],
		identifier[10:16],
	), nil
}
