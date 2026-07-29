package dev.buildopt.contracts.localevents.v1;

import java.io.ByteArrayInputStream;
import java.io.ByteArrayOutputStream;
import java.io.EOFException;
import java.io.IOException;
import java.io.InputStream;
import java.io.OutputStream;
import java.net.StandardProtocolFamily;
import java.net.UnixDomainSocketAddress;
import java.nio.channels.Channels;
import java.nio.channels.ServerSocketChannel;
import java.nio.channels.SocketChannel;
import java.nio.charset.StandardCharsets;
import java.nio.file.Files;
import java.nio.file.Path;
import java.util.ArrayList;
import java.util.Arrays;
import java.util.List;

/**
 * Standard-library conformance peer for F0-019. Product clients are generated
 * from task_events.proto later by F0-022; this fixture independently proves the
 * v1 wire bytes and Unix-socket framing in both language directions.
 */
public final class TaskEventsRoundTrip {
    private static final int MAX_FRAME_BYTES = 1 << 20;

    private static final int PAYLOAD_PRODUCER_HELLO = 10;
    private static final int PAYLOAD_CAPABILITY = 11;
    private static final int PAYLOAD_CACHE_PUT = 12;

    private static final long ACK_ACCEPTED = 1;
    private static final long ACK_ATTEMPT_ABORTED = 2;

    private static final long CAPABILITY_EXACT = 1;
    private static final long CAPABILITY_UNAVAILABLE = 2;

    private static final long ABORT_UNATTRIBUTED_PUT = 1;

    private TaskEventsRoundTrip() {
    }

    public static void main(String[] arguments) throws Exception {
        if (arguments.length == 1 && "self-test".equals(arguments[0])) {
            runSemanticSelfTest();
            return;
        }
        if (arguments.length == 2 && "client".equals(arguments[0])) {
            runExactClient(Path.of(arguments[1]));
            return;
        }
        if (arguments.length == 3 && "server".equals(arguments[0])) {
            runUnattributedServer(Path.of(arguments[1]), Path.of(arguments[2]));
            return;
        }
        throw new IllegalArgumentException(
                "usage: TaskEventsRoundTrip <self-test|client SOCKET|server SOCKET READY>");
    }

    private static void runExactClient(Path socketPath) throws IOException {
        UnixDomainSocketAddress address = UnixDomainSocketAddress.of(socketPath);
        try (SocketChannel channel = SocketChannel.open(StandardProtocolFamily.UNIX)) {
            channel.connect(address);
            InputStream input = Channels.newInputStream(channel);
            OutputStream output = Channels.newOutputStream(channel);

            List<byte[]> events = List.of(
                    marshalTaskEvent(
                            "attempt-java-exact",
                            "java-producer",
                            1,
                            PAYLOAD_PRODUCER_HELLO,
                            marshalProducerHello(1, 0, 1, "java-conformance-v1")),
                    marshalTaskEvent(
                            "attempt-java-exact",
                            "java-producer",
                            2,
                            PAYLOAD_CACHE_PUT,
                            marshalExactPut(
                                    "put-java-2",
                                    "c0111bcb4ba8ba492a6cb273f724a55b",
                                    1,
                                    "task-execution-java-7",
                                    1)),
                    marshalTaskEvent(
                            "attempt-java-exact",
                            "java-producer",
                            3,
                            PAYLOAD_CAPABILITY,
                            marshalCapability(
                                    CAPABILITY_EXACT,
                                    "9.6.1",
                                    "all observed PUTs have one completed task",
                                    0,
                                    "")));

            for (int index = 0; index < events.size(); index++) {
                writeDelimited(output, events.get(index));
                TaskEventAck acknowledgement = decodeTaskEventAck(readDelimited(input));
                long sequence = index + 1L;
                if (!"attempt-java-exact".equals(acknowledgement.attemptId)
                        || !"java-producer".equals(acknowledgement.producerInstance)
                        || acknowledgement.sequence != sequence
                        || acknowledgement.status != ACK_ACCEPTED) {
                    throw new IOException(
                            "unexpected acknowledgement " + sequence + ": " + acknowledgement);
                }
            }
        }
        System.out.println("java-client-exact-ok");
    }

    private static void runUnattributedServer(Path socketPath, Path readyPath)
            throws IOException {
        Files.deleteIfExists(socketPath);
        UnixDomainSocketAddress address = UnixDomainSocketAddress.of(socketPath);
        try (ServerSocketChannel server = ServerSocketChannel.open(StandardProtocolFamily.UNIX)) {
            server.bind(address);
            Files.writeString(readyPath, "ready\n", StandardCharsets.UTF_8);
            try (SocketChannel channel = server.accept()) {
                InputStream input = Channels.newInputStream(channel);
                OutputStream output = Channels.newOutputStream(channel);

                int[] expectedPayloads = {
                    PAYLOAD_PRODUCER_HELLO,
                    PAYLOAD_CACHE_PUT,
                    PAYLOAD_CAPABILITY
                };
                long[] expectedStatuses = {
                    ACK_ACCEPTED,
                    ACK_ATTEMPT_ABORTED,
                    ACK_ATTEMPT_ABORTED
                };
                for (int index = 0; index < expectedPayloads.length; index++) {
                    TaskEvent event = decodeTaskEvent(readDelimited(input));
                    long sequence = index + 1L;
                    if (!"attempt-go-unattributed".equals(event.attemptId)
                            || !"go-producer".equals(event.producerInstance)
                            || event.sequence != sequence
                            || event.payloadField != expectedPayloads[index]
                            || event.abortImmediately != (sequence > 1)) {
                        throw new IOException("unexpected Go event " + sequence + ": " + event);
                    }
                    writeDelimited(
                            output,
                            marshalTaskEventAck(
                                    event.attemptId,
                                    event.producerInstance,
                                    event.sequence,
                                    expectedStatuses[index]));
                }
            }
        } finally {
            Files.deleteIfExists(socketPath);
        }
        System.out.println("java-server-unattributed-ok");
    }

    private static void runSemanticSelfTest() throws IOException {
        expectFailure(
                "missing attribution",
                () -> {
                    byte[] cachePut = appendStringField(null, 1, "put-no-attribution");
                    cachePut = appendStringField(cachePut, 2, "cache-key");
                    cachePut = appendVarintField(cachePut, 3, 1);
                    validateCachePut(cachePut);
                });

        expectFailure(
                "missing whole-attempt abort",
                () -> {
                    byte[] unattributed = appendVarintField(null, 1, 1);
                    byte[] cachePut = appendStringField(null, 1, "put-no-abort");
                    cachePut = appendStringField(cachePut, 2, "cache-key");
                    cachePut = appendVarintField(cachePut, 3, 1);
                    cachePut = appendMessageField(cachePut, 11, unattributed);
                    validateCachePut(cachePut);
                });

        expectFailure(
                "unavailable capability without abort",
                () -> {
                    byte[] capability = appendVarintField(null, 1, CAPABILITY_UNAVAILABLE);
                    capability = appendStringField(capability, 2, "9.6.1");
                    validateCapability(capability);
                });

        expectFailure(
                "exact capability with abort",
                () -> validateCapability(
                        marshalCapability(
                                CAPABILITY_EXACT,
                                "9.6.1",
                                "invalid exact declaration",
                                ABORT_UNATTRIBUTED_PUT,
                                "must not be present")));

        expectFailure(
                "unknown required enum",
                () -> validateCachePut(
                        marshalExactPut(
                                "put-unknown-outcome",
                                "cache-key",
                                99,
                                "task-execution",
                                1)));

        byte[] oversizedPrefix = appendUnsignedVarint(null, MAX_FRAME_BYTES + 1L);
        expectFailure(
                "oversized frame",
                () -> readDelimited(new ByteArrayInputStream(oversizedPrefix)));

        System.out.println("java-semantic-self-test-ok");
    }

    private static TaskEvent decodeTaskEvent(byte[] data) throws IOException {
        List<Field> fields = parseFields(data);
        String attemptId = requiredString(fields, 1, "attempt_id");
        String producerInstance = requiredString(fields, 2, "producer_instance_id");
        long sequence = requiredVarint(fields, 3, "sequence_number");
        if (sequence == 0) {
            throw new IOException("sequence_number must be greater than zero");
        }
        Field payload = requiredOneof(
                fields,
                new int[] {
                    PAYLOAD_PRODUCER_HELLO,
                    PAYLOAD_CAPABILITY,
                    PAYLOAD_CACHE_PUT
                },
                "payload");

        boolean abortImmediately;
        if (payload.number == PAYLOAD_PRODUCER_HELLO) {
            validateProducerHello(payload.payload);
            abortImmediately = false;
        } else if (payload.number == PAYLOAD_CACHE_PUT) {
            abortImmediately = validateCachePut(payload.payload);
        } else {
            abortImmediately = validateCapability(payload.payload);
        }
        return new TaskEvent(
                attemptId,
                producerInstance,
                sequence,
                payload.number,
                abortImmediately);
    }

    private static void validateProducerHello(byte[] data) throws IOException {
        List<Field> fields = parseFields(data);
        List<Field> versionFields = parseFields(
                requiredMessage(fields, 1, "protocol_version"));
        long major = requiredVarint(versionFields, 1, "protocol_version.major");
        long minor = optionalVarint(versionFields, 2, "protocol_version.minor");
        if (major != 1 || minor != 0) {
            throw new IOException("unsupported protocol version " + major + "." + minor);
        }
        long producer = requiredVarint(fields, 2, "producer");
        if (producer == 0 || producer > 2) {
            throw new IOException("unsupported producer " + producer);
        }
        requiredString(fields, 3, "implementation_version");
    }

    private static boolean validateCachePut(byte[] data) throws IOException {
        List<Field> fields = parseFields(data);
        requiredString(fields, 1, "put_operation_id");
        requiredString(fields, 2, "native_cache_key");
        long putOutcome = requiredVarint(fields, 3, "put_outcome");
        if (putOutcome == 0 || putOutcome > 4) {
            throw new IOException("unsupported put_outcome " + putOutcome);
        }
        Field attribution = requiredOneof(
                fields,
                new int[] {10, 11},
                "cache PUT attribution");
        List<Field> attributionFields = parseFields(attribution.payload);
        if (attribution.number == 10) {
            requiredString(attributionFields, 1, "task_execution_id");
            long taskOutcome = requiredVarint(
                    attributionFields,
                    2,
                    "task_outcome");
            if (taskOutcome == 0 || taskOutcome > 3) {
                throw new IOException("unsupported task_outcome " + taskOutcome);
            }
            return false;
        }

        long unattributedReason = requiredVarint(
                attributionFields,
                1,
                "unattributed.reason");
        if (unattributedReason == 0 || unattributedReason > 4) {
            throw new IOException(
                    "unsupported unattributed reason " + unattributedReason);
        }
        byte[] abort = requiredMessage(
                attributionFields,
                2,
                "unattributed whole-attempt abort");
        validateWholeAttemptAbort(abort, ABORT_UNATTRIBUTED_PUT);
        return true;
    }

    private static boolean validateCapability(byte[] data) throws IOException {
        List<Field> fields = parseFields(data);
        long capability = requiredVarint(fields, 1, "capability");
        if (capability == 0) {
            throw new IOException("capability must be specified");
        }
        requiredString(fields, 2, "gradle_version");
        List<Field> abortFields = matchingFields(fields, 10);
        if (abortFields.size() > 1) {
            throw new IOException("whole_attempt_abort appears more than once");
        }
        if (capability == CAPABILITY_EXACT) {
            if (!abortFields.isEmpty()) {
                throw new IOException("exact capability must not carry an abort");
            }
            return false;
        }
        if (capability == CAPABILITY_UNAVAILABLE) {
            if (abortFields.size() != 1) {
                throw new IOException(
                        "unavailable capability requires a whole-attempt abort");
            }
            validateWholeAttemptAbort(abortFields.get(0).payload, 0);
            return true;
        }
        throw new IOException("unsupported correlation capability " + capability);
    }

    private static void validateWholeAttemptAbort(byte[] data, long expectedReason)
            throws IOException {
        List<Field> fields = parseFields(data);
        long reason = requiredVarint(fields, 1, "whole_attempt_abort.reason");
        if (reason == 0 || reason > 5) {
            throw new IOException(
                    "unsupported whole-attempt abort reason " + reason);
        }
        if (expectedReason != 0 && reason != expectedReason) {
            throw new IOException(
                    "whole-attempt abort reason is "
                            + reason
                            + ", want "
                            + expectedReason);
        }
        requiredString(fields, 2, "whole_attempt_abort.detail");
    }

    private static TaskEventAck decodeTaskEventAck(byte[] data) throws IOException {
        List<Field> fields = parseFields(data);
        String attemptId = requiredString(fields, 1, "ack.attempt_id");
        String producerInstance = requiredString(
                fields,
                2,
                "ack.producer_instance_id");
        long sequence = requiredVarint(fields, 3, "ack.sequence_number");
        long status = requiredVarint(fields, 4, "ack.status");
        if (status == 0 || status > 3) {
            throw new IOException("unsupported ack.status " + status);
        }
        return new TaskEventAck(attemptId, producerInstance, sequence, status);
    }

    private static byte[] marshalTaskEvent(
            String attemptId,
            String producerInstance,
            long sequence,
            int payloadField,
            byte[] payload) {
        byte[] message = appendStringField(null, 1, attemptId);
        message = appendStringField(message, 2, producerInstance);
        message = appendVarintField(message, 3, sequence);
        return appendMessageField(message, payloadField, payload);
    }

    private static byte[] marshalTaskEventAck(
            String attemptId,
            String producerInstance,
            long sequence,
            long status) {
        byte[] message = appendStringField(null, 1, attemptId);
        message = appendStringField(message, 2, producerInstance);
        message = appendVarintField(message, 3, sequence);
        return appendVarintField(message, 4, status);
    }

    private static byte[] marshalProducerHello(
            long major,
            long minor,
            long producer,
            String implementationVersion) {
        byte[] version = appendVarintField(null, 1, major);
        if (minor != 0) {
            version = appendVarintField(version, 2, minor);
        }
        byte[] message = appendMessageField(null, 1, version);
        message = appendVarintField(message, 2, producer);
        return appendStringField(message, 3, implementationVersion);
    }

    private static byte[] marshalExactPut(
            String putOperationId,
            String nativeCacheKey,
            long putOutcome,
            String taskExecutionId,
            long taskOutcome) {
        byte[] exact = appendStringField(null, 1, taskExecutionId);
        exact = appendVarintField(exact, 2, taskOutcome);
        byte[] message = appendStringField(null, 1, putOperationId);
        message = appendStringField(message, 2, nativeCacheKey);
        message = appendVarintField(message, 3, putOutcome);
        return appendMessageField(message, 10, exact);
    }

    private static byte[] marshalCapability(
            long capability,
            String gradleVersion,
            String detail,
            long abortReason,
            String abortDetail) {
        byte[] message = appendVarintField(null, 1, capability);
        message = appendStringField(message, 2, gradleVersion);
        if (!detail.isEmpty()) {
            message = appendStringField(message, 3, detail);
        }
        if (abortReason != 0) {
            message = appendMessageField(
                    message,
                    10,
                    marshalWholeAttemptAbort(abortReason, abortDetail));
        }
        return message;
    }

    private static byte[] marshalWholeAttemptAbort(long reason, String detail) {
        byte[] message = appendVarintField(null, 1, reason);
        return appendStringField(message, 2, detail);
    }

    private static byte[] appendStringField(byte[] data, int number, String value) {
        return appendMessageField(
                data,
                number,
                value.getBytes(StandardCharsets.UTF_8));
    }

    private static byte[] appendMessageField(byte[] data, int number, byte[] value) {
        byte[] result = appendUnsignedVarint(data, ((long) number << 3) | 2);
        result = appendUnsignedVarint(result, value.length);
        return appendBytes(result, value);
    }

    private static byte[] appendVarintField(byte[] data, int number, long value) {
        byte[] result = appendUnsignedVarint(data, (long) number << 3);
        return appendUnsignedVarint(result, value);
    }

    private static byte[] appendUnsignedVarint(byte[] data, long value) {
        ByteArrayOutputStream output = new ByteArrayOutputStream();
        if (data != null) {
            output.writeBytes(data);
        }
        long remaining = value;
        while ((remaining & ~0x7fL) != 0) {
            output.write((int) (remaining & 0x7fL) | 0x80);
            remaining >>>= 7;
        }
        output.write((int) remaining);
        return output.toByteArray();
    }

    private static byte[] appendBytes(byte[] first, byte[] second) {
        byte[] joined = Arrays.copyOf(first, first.length + second.length);
        System.arraycopy(second, 0, joined, first.length, second.length);
        return joined;
    }

    private static List<Field> parseFields(byte[] data) throws IOException {
        List<Field> fields = new ArrayList<>();
        Cursor cursor = new Cursor();
        while (cursor.offset < data.length) {
            long key = readUnsignedVarint(data, cursor);
            int number = (int) (key >>> 3);
            int wire = (int) (key & 7);
            if (key < 0 || number <= 0) {
                throw new IOException("invalid Protobuf field number");
            }
            if (wire == 0) {
                fields.add(Field.varint(number, readUnsignedVarint(data, cursor)));
            } else if (wire == 2) {
                long length = readUnsignedVarint(data, cursor);
                if (length < 0 || length > data.length - cursor.offset) {
                    throw new IOException("truncated field " + number);
                }
                int end = cursor.offset + (int) length;
                byte[] payload = Arrays.copyOfRange(data, cursor.offset, end);
                cursor.offset = end;
                fields.add(Field.payload(number, payload));
            } else {
                throw new IOException(
                        "unsupported wire type " + wire + " for field " + number);
            }
        }
        return fields;
    }

    private static long readUnsignedVarint(byte[] data, Cursor cursor)
            throws IOException {
        long value = 0;
        for (int shift = 0; shift < 70; shift += 7) {
            if (cursor.offset >= data.length) {
                throw new EOFException("truncated varint");
            }
            int current = data[cursor.offset++] & 0xff;
            if (current < 0x80) {
                if (shift == 63 && current > 1) {
                    throw new IOException("varint overflow");
                }
                return value | (long) current << shift;
            }
            value |= (long) (current & 0x7f) << shift;
        }
        throw new IOException("varint overflow");
    }

    private static String requiredString(
            List<Field> fields,
            int number,
            String name) throws IOException {
        Field field = requiredField(fields, number, 2, name);
        if (field.payload.length == 0) {
            throw new IOException(name + " must be non-empty");
        }
        return new String(field.payload, StandardCharsets.UTF_8);
    }

    private static byte[] requiredMessage(
            List<Field> fields,
            int number,
            String name) throws IOException {
        Field field = requiredField(fields, number, 2, name);
        if (field.payload.length == 0) {
            throw new IOException(name + " must be non-empty");
        }
        return field.payload;
    }

    private static long requiredVarint(
            List<Field> fields,
            int number,
            String name) throws IOException {
        return requiredField(fields, number, 0, name).varint;
    }

    private static long optionalVarint(
            List<Field> fields,
            int number,
            String name) throws IOException {
        List<Field> matches = matchingFields(fields, number);
        if (matches.isEmpty()) {
            return 0;
        }
        if (matches.size() != 1 || matches.get(0).wire != 0) {
            throw new IOException(name + " must appear at most once as a varint");
        }
        return matches.get(0).varint;
    }

    private static Field requiredField(
            List<Field> fields,
            int number,
            int wire,
            String name) throws IOException {
        List<Field> matches = matchingFields(fields, number);
        if (matches.size() != 1) {
            throw new IOException(name + " must appear exactly once");
        }
        if (matches.get(0).wire != wire) {
            throw new IOException(
                    name
                            + " has wire type "
                            + matches.get(0).wire
                            + ", want "
                            + wire);
        }
        return matches.get(0);
    }

    private static Field requiredOneof(
            List<Field> fields,
            int[] numbers,
            String name) throws IOException {
        List<Field> selected = new ArrayList<>();
        for (int number : numbers) {
            selected.addAll(matchingFields(fields, number));
        }
        if (selected.size() != 1 || selected.get(0).wire != 2) {
            throw new IOException(
                    name + " requires exactly one attribution or payload arm");
        }
        if (selected.get(0).payload.length == 0) {
            throw new IOException(name + " arm must be non-empty");
        }
        return selected.get(0);
    }

    private static List<Field> matchingFields(List<Field> fields, int number) {
        List<Field> matches = new ArrayList<>();
        for (Field field : fields) {
            if (field.number == number) {
                matches.add(field);
            }
        }
        return matches;
    }

    private static void writeDelimited(OutputStream output, byte[] message)
            throws IOException {
        if (message.length > MAX_FRAME_BYTES) {
            throw new IOException(
                    "frame length "
                            + message.length
                            + " exceeds "
                            + MAX_FRAME_BYTES
                            + " bytes");
        }
        output.write(appendUnsignedVarint(null, message.length));
        output.write(message);
        output.flush();
    }

    private static byte[] readDelimited(InputStream input) throws IOException {
        long length = readUnsignedVarint(input);
        if (length < 0 || length > MAX_FRAME_BYTES) {
            throw new IOException(
                    "frame length "
                            + length
                            + " exceeds "
                            + MAX_FRAME_BYTES
                            + " bytes");
        }
        byte[] message = input.readNBytes((int) length);
        if (message.length != length) {
            throw new EOFException("truncated delimited message");
        }
        return message;
    }

    private static long readUnsignedVarint(InputStream input) throws IOException {
        long value = 0;
        for (int shift = 0; shift < 70; shift += 7) {
            int current = input.read();
            if (current < 0) {
                throw new EOFException("truncated varint");
            }
            if (current < 0x80) {
                if (shift == 63 && current > 1) {
                    throw new IOException("varint overflow");
                }
                return value | (long) current << shift;
            }
            value |= (long) (current & 0x7f) << shift;
        }
        throw new IOException("varint overflow");
    }

    private static void expectFailure(String name, CheckedRunnable runnable)
            throws IOException {
        try {
            runnable.run();
        } catch (IOException expected) {
            return;
        }
        throw new IOException("expected failure: " + name);
    }

    @FunctionalInterface
    private interface CheckedRunnable {
        void run() throws IOException;
    }

    private static final class Cursor {
        private int offset;
    }

    private static final class Field {
        private final int number;
        private final int wire;
        private final long varint;
        private final byte[] payload;

        private Field(int number, int wire, long varint, byte[] payload) {
            this.number = number;
            this.wire = wire;
            this.varint = varint;
            this.payload = payload;
        }

        private static Field varint(int number, long value) {
            return new Field(number, 0, value, new byte[0]);
        }

        private static Field payload(int number, byte[] value) {
            return new Field(number, 2, 0, value);
        }
    }

    private static final class TaskEvent {
        private final String attemptId;
        private final String producerInstance;
        private final long sequence;
        private final int payloadField;
        private final boolean abortImmediately;

        private TaskEvent(
                String attemptId,
                String producerInstance,
                long sequence,
                int payloadField,
                boolean abortImmediately) {
            this.attemptId = attemptId;
            this.producerInstance = producerInstance;
            this.sequence = sequence;
            this.payloadField = payloadField;
            this.abortImmediately = abortImmediately;
        }

        @Override
        public String toString() {
            return "TaskEvent{"
                    + "attemptId='"
                    + attemptId
                    + '\''
                    + ", producerInstance='"
                    + producerInstance
                    + '\''
                    + ", sequence="
                    + sequence
                    + ", payloadField="
                    + payloadField
                    + ", abortImmediately="
                    + abortImmediately
                    + '}';
        }
    }

    private static final class TaskEventAck {
        private final String attemptId;
        private final String producerInstance;
        private final long sequence;
        private final long status;

        private TaskEventAck(
                String attemptId,
                String producerInstance,
                long sequence,
                long status) {
            this.attemptId = attemptId;
            this.producerInstance = producerInstance;
            this.sequence = sequence;
            this.status = status;
        }

        @Override
        public String toString() {
            return "TaskEventAck{"
                    + "attemptId='"
                    + attemptId
                    + '\''
                    + ", producerInstance='"
                    + producerInstance
                    + '\''
                    + ", sequence="
                    + sequence
                    + ", status="
                    + status
                    + '}';
        }
    }
}
