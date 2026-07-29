package dev.buildopt.gradle;

import java.io.ByteArrayInputStream;
import java.io.ByteArrayOutputStream;
import java.io.EOFException;
import java.io.IOException;
import java.io.InputStream;
import java.io.OutputStream;
import java.net.StandardProtocolFamily;
import java.net.UnixDomainSocketAddress;
import java.nio.channels.Channels;
import java.nio.channels.SocketChannel;
import java.nio.charset.StandardCharsets;
import java.util.ArrayList;
import java.util.Arrays;
import java.util.List;
import java.util.UUID;
import org.gradle.api.logging.Logger;
import org.gradle.api.logging.Logging;
import org.gradle.api.services.BuildService;
import org.gradle.api.services.BuildServiceParameters;
import org.gradle.tooling.events.FinishEvent;
import org.gradle.tooling.events.OperationCompletionListener;

/**
 * Per-invocation authenticated producer handshake with the BuildOpt launcher.
 *
 * <p>The service intentionally sends only the {@code ProducerHello} frame
 * defined by {@code task_events.proto} after verifying the local gateway and
 * event credential. A missing or rejected receiver disables BuildOpt for the
 * invocation without failing the baseline Gradle build.
 */
public abstract class BuildOptHandshakeService
        implements
                BuildService<BuildServiceParameters.None>,
                OperationCompletionListener,
                AutoCloseable {
    private static final Logger LOGGER = Logging.getLogger(BuildOptHandshakeService.class);
    private static final int MAX_FRAME_BYTES = 1 << 20;
    private static final long ACK_ACCEPTED = 1;

    /** Creates the lazily realized service and attempts one handshake. */
    public BuildOptHandshakeService() {
        try {
            BuildOptRendezvousContext context =
                    BuildOptRendezvousContext.fromEnvironment();
            if (context == null) {
                return;
            }
            context.verifyGateway();
            exchangeHello(
                    context,
                    UUID.randomUUID().toString(),
                    implementationVersion());
        } catch (IOException | RuntimeException exception) {
            LOGGER.warn(
                    "BuildOpt authenticated rendezvous unavailable: {}",
                    exception.getMessage());
        }
    }

    @Override
    public void close() {
        // WS-004 owns only readiness plus one handshake; there is no retained channel.
    }

    @Override
    public void onFinish(FinishEvent event) {
        // Registration forces one service realization even when every task is up-to-date.
    }

    private static void exchangeHello(
            BuildOptRendezvousContext context,
            String producerInstanceId,
            String implementationVersion)
            throws IOException {
        UnixDomainSocketAddress address =
                UnixDomainSocketAddress.of(context.socketPath());
        try (SocketChannel channel = SocketChannel.open(StandardProtocolFamily.UNIX)) {
            channel.connect(address);
            OutputStream output = Channels.newOutputStream(channel);
            InputStream input = Channels.newInputStream(channel);

            context.writeEventAuthentication(output);
            byte[] hello = marshalProducerHello(1, 0, 1, implementationVersion);
            byte[] event =
                    marshalTaskEvent(
                            context.attemptId(),
                            producerInstanceId,
                            1,
                            10,
                            hello);
            writeDelimited(output, event);

            TaskEventAck ack = decodeTaskEventAck(readDelimited(input));
            if (!context.attemptId().equals(ack.attemptId)
                    || !producerInstanceId.equals(ack.producerInstanceId)
                    || ack.sequenceNumber != 1
                    || ack.status != ACK_ACCEPTED) {
                throw new IOException("launcher returned a mismatched or rejected acknowledgement");
            }
        }
    }

    private static String implementationVersion() {
        String version =
                BuildOptProjectPlugin.class.getPackage().getImplementationVersion();
        return isBlank(version) ? "development" : version;
    }

    private static boolean isBlank(String value) {
        return value == null || value.isBlank();
    }

    private static byte[] marshalTaskEvent(
            String attemptId,
            String producerInstanceId,
            long sequenceNumber,
            int payloadField,
            byte[] payload) {
        byte[] message = appendStringField(null, 1, attemptId);
        message = appendStringField(message, 2, producerInstanceId);
        message = appendVarintField(message, 3, sequenceNumber);
        return appendMessageField(message, payloadField, payload);
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

    private static byte[] appendStringField(
            byte[] data,
            int number,
            String value) {
        return appendMessageField(
                data,
                number,
                value.getBytes(StandardCharsets.UTF_8));
    }

    private static byte[] appendMessageField(
            byte[] data,
            int number,
            byte[] value) {
        byte[] result = appendUnsignedVarint(data, ((long) number << 3) | 2);
        result = appendUnsignedVarint(result, value.length);
        return appendBytes(result, value);
    }

    private static byte[] appendVarintField(
            byte[] data,
            int number,
            long value) {
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

    private static TaskEventAck decodeTaskEventAck(byte[] data) throws IOException {
        List<Field> fields = parseFields(data);
        String attemptId = requiredString(fields, 1, "ack.attempt_id");
        String producerInstanceId =
                requiredString(fields, 2, "ack.producer_instance_id");
        long sequenceNumber =
                requiredVarint(fields, 3, "ack.sequence_number");
        long status = requiredVarint(fields, 4, "ack.status");
        if (status == 0 || status > 3) {
            throw new IOException("unsupported ack.status " + status);
        }
        return new TaskEventAck(
                attemptId,
                producerInstanceId,
                sequenceNumber,
                status);
    }

    private static List<Field> parseFields(byte[] data) throws IOException {
        List<Field> fields = new ArrayList<>();
        Cursor cursor = new Cursor();
        while (cursor.offset < data.length) {
            long key = readUnsignedVarint(data, cursor);
            int number = (int) (key >>> 3);
            int wire = (int) (key & 7);
            if (number <= 0) {
                throw new IOException("invalid Protobuf field number");
            }
            if (wire == 0) {
                fields.add(
                        Field.varint(
                                number,
                                readUnsignedVarint(data, cursor)));
            } else if (wire == 2) {
                long length = readUnsignedVarint(data, cursor);
                if (length < 0 || length > data.length - cursor.offset) {
                    throw new IOException("truncated field " + number);
                }
                int end = cursor.offset + (int) length;
                fields.add(
                        Field.payload(
                                number,
                                Arrays.copyOfRange(data, cursor.offset, end)));
                cursor.offset = end;
            } else {
                throw new IOException(
                        "unsupported wire type "
                                + wire
                                + " for field "
                                + number);
            }
        }
        return fields;
    }

    private static long readUnsignedVarint(
            byte[] data,
            Cursor cursor)
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
            String name)
            throws IOException {
        Field field = requiredField(fields, number, 2, name);
        if (field.payload.length == 0) {
            throw new IOException(name + " must be non-empty");
        }
        return new String(field.payload, StandardCharsets.UTF_8);
    }

    private static long requiredVarint(
            List<Field> fields,
            int number,
            String name)
            throws IOException {
        return requiredField(fields, number, 0, name).varint;
    }

    private static Field requiredField(
            List<Field> fields,
            int number,
            int wire,
            String name)
            throws IOException {
        List<Field> matches = new ArrayList<>();
        for (Field field : fields) {
            if (field.number == number) {
                matches.add(field);
            }
        }
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

    private static void writeDelimited(
            OutputStream output,
            byte[] message)
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

    private static long readUnsignedVarint(InputStream input)
            throws IOException {
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

    private static final class TaskEventAck {
        private final String attemptId;
        private final String producerInstanceId;
        private final long sequenceNumber;
        private final long status;

        private TaskEventAck(
                String attemptId,
                String producerInstanceId,
                long sequenceNumber,
                long status) {
            this.attemptId = attemptId;
            this.producerInstanceId = producerInstanceId;
            this.sequenceNumber = sequenceNumber;
            this.status = status;
        }
    }
}
