import java.io.BufferedReader;
import java.io.ByteArrayInputStream;
import java.io.ByteArrayOutputStream;
import java.io.DataOutputStream;
import java.io.EOFException;
import java.io.InputStreamReader;
import java.lang.reflect.Modifier;
import java.nio.charset.StandardCharsets;
import java.nio.file.Files;
import java.nio.file.LinkOption;
import java.nio.file.Path;
import java.security.MessageDigest;
import java.util.ArrayList;
import java.util.Arrays;
import java.util.HexFormat;
import java.util.List;
import java.util.Map;
import java.util.Set;
import java.util.TreeMap;
import org.gradle.api.internal.cache.StringInterner;
import org.gradle.api.internal.tasks.compile.incremental.recomp.PreviousCompilationData;
import org.gradle.internal.serialize.Serializer;
import org.gradle.internal.serialize.kryo.KryoBackedDecoder;
import org.gradle.internal.serialize.kryo.KryoBackedEncoder;

/** Retained-output reader, run outside measured builds with frozen Gradle 9.7.1 jars. */
public final class MetadataProjection {
    static final int MAX_BYTES = 4 * 1024 * 1024;
    static final String PREFIX = "org.gradle.api.internal.tasks.compile.incremental.";
    static final String INT_SET = PREFIX + "deps.IntSetSerializer";
    static final String DEPENDENTS = PREFIX + "compilerapi.deps.DependentSetSerializer";
    static final Set<String> OBJECT_TYPES = Set.of(
            PREFIX + "recomp.PreviousCompilationData", PREFIX + "deps.ClassSetAnalysisData",
            PREFIX + "processing.AnnotationProcessingData", PREFIX + "compilerapi.CompilerApiData",
            PREFIX + "compilerapi.constants.ConstantToDependentsMapping",
            PREFIX + "compilerapi.deps.DependentsSet$DefaultDependentsSet",
            PREFIX + "compilerapi.deps.DependentsSet$EmptyDependentsSet",
            PREFIX + "compilerapi.deps.DependentsSet$DependencyToAll",
            PREFIX + "compilerapi.deps.GeneratedResource");
    record Decoded<T>(T value, String semantic, Map<String, Map<Integer, Integer>> cardinalities) {}

    static final class Decoder extends KryoBackedDecoder {
        final byte[] raw;
        final StackWalker walker = StackWalker.getInstance(StackWalker.Option.RETAIN_CLASS_REFERENCE);
        final Map<String, Map<Integer, Integer>> cardinalities = new TreeMap<>();
        int remainingInts;
        Decoder(byte[] bytes) { super(new ByteArrayInputStream(bytes)); raw = bytes; }

        void count(String kind, int value) {
            if (value < 0 || value > 1_000_000) throw new IllegalArgumentException("unbounded metadata collection");
            cardinalities.computeIfAbsent(kind, ignored -> new TreeMap<>()).merge(value, 1, Integer::sum);
        }
        @Override public int readInt() throws EOFException {
            int value = super.readInt();
            String caller = walker.getCallerClass().getName();
            if (caller.equals(INT_SET)) {
                if (remainingInts == 0) { count(caller, value); remainingInts = value; }
                else remainingInts--;
            } else if (caller.equals("org.gradle.internal.serialize.MapSerializer")
                    || caller.equals("org.gradle.internal.serialize.AbstractCollectionSerializer")) {
                count(caller, value);
            }
            return value;
        }
        @Override public int readSmallInt() throws EOFException {
            int value = super.readSmallInt();
            String caller = walker.getCallerClass().getName();
            if (caller.equals(PREFIX + "deps.ClassSetAnalysisData$Serializer") || caller.equals(DEPENDENTS)) {
                // DependentSetSerializer also reads nonnegative resource-location ordinals.
                // Keep those in the histogram: removing a duplicate cannot increase either
                // the number of reads or the sum of these nonnegative counts/ordinals.
                count(caller, value);
            }
            return value;
        }
        @Override public boolean readBoolean() throws EOFException {
            int position = Math.toIntExact(getReadPosition());
            boolean value = super.readBoolean();
            if (raw[position] != 0 && raw[position] != 1) throw new IllegalArgumentException("invalid boolean encoding");
            return value;
        }
        @Override public byte readByte() throws EOFException {
            byte value = super.readByte();
            if (walker.getCallerClass().getName().equals(DEPENDENTS) && value != 0 && value != 1)
                throw new IllegalArgumentException("invalid dependency discriminator");
            return value;
        }
    }

    static <T> Decoded<T> decode(Serializer<T> serializer, byte[] bytes) throws Exception {
        if (bytes.length == 0 || bytes.length > MAX_BYTES) throw new IllegalArgumentException("bounded metadata required");
        try (Decoder decoder = new Decoder(bytes)) {
            T data = serializer.read(decoder);
            if (decoder.remainingInts != 0 || decoder.getReadPosition() != bytes.length)
                throw new IllegalArgumentException("incomplete or trailing metadata");
            return new Decoded<>(data, semantic(data), decoder.cardinalities);
        }
    }
    static <T> byte[] encode(Serializer<T> serializer, T value) throws Exception {
        ByteArrayOutputStream bytes = new ByteArrayOutputStream();
        try (KryoBackedEncoder encoder = new KryoBackedEncoder(bytes)) { serializer.write(encoder, value); }
        return bytes.toByteArray();
    }
    static <T> Decoded<T> checked(Serializer<T> serializer, byte[] bytes) throws Exception {
        Decoded<T> first = decode(serializer, bytes);
        Decoded<T> roundTrip = decode(serializer, encode(serializer, first.value()));
        // Gradle's Map/Set readers silently collapse duplicates. Every serialized
        // collection count must survive decoding and re-encoding. Reordering nested
        // collections may permute counts, hence an exact histogram per serializer.
        // ImmutableMap readers already reject duplicate keys themselves.
        if (!first.cardinalities().equals(roundTrip.cardinalities()) || !first.semantic().equals(roundTrip.semantic()))
            throw new IllegalArgumentException("duplicate members or lossy metadata decoding");
        return first;
    }
    static String sha(byte[] bytes) throws Exception {
        return HexFormat.of().formatHex(MessageDigest.getInstance("SHA-256").digest(bytes));
    }
    static String semantic(Object value) throws Exception { return sha(node(value, 0)); }
    static void string(DataOutputStream out, String value) throws Exception {
        // Preserve every UTF-16 code unit, including malformed surrogate input;
        // UTF-8 replacement encoding could collapse distinct decoded strings.
        out.writeInt(value.length());
        for (int i = 0; i < value.length(); i++) out.writeChar(value.charAt(i));
    }
    static byte[] node(Object value, int depth) throws Exception {
        if (depth > 30) throw new IllegalArgumentException("metadata nesting limit");
        ByteArrayOutputStream bytes = new ByteArrayOutputStream();
        try (DataOutputStream out = new DataOutputStream(bytes)) {
            if (value == null) { out.writeByte(0); }
            else if (value instanceof String s) { out.writeByte(1); string(out, s); }
            else if (value instanceof Integer || value instanceof Long || value instanceof Boolean) {
                out.writeByte(2); string(out, value.getClass().getName()); string(out, value.toString());
            } else if (value instanceof org.gradle.internal.hash.HashCode hash) {
                out.writeByte(3); string(out, hash.toString());
            } else if (value instanceof Enum<?> e && e.getDeclaringClass().getName().equals(PREFIX + "compilerapi.deps.GeneratedResource$Location")) {
                out.writeByte(4); string(out, e.getDeclaringClass().getName()); string(out, e.name());
            } else if (value instanceof Map<?, ?> map) {
                out.writeByte(5); out.writeInt(map.size());
                List<byte[]> entries = new ArrayList<>();
                for (var entry : map.entrySet()) {
                    byte[] pair = new byte[64];
                    System.arraycopy(node(entry.getKey(), depth + 1), 0, pair, 0, 32);
                    System.arraycopy(node(entry.getValue(), depth + 1), 0, pair, 32, 32);
                    entries.add(pair);
                }
                entries.sort(Arrays::compareUnsigned); for (byte[] entry : entries) out.write(entry);
            } else if (value instanceof Set<?> || value instanceof List<?>) {
                boolean ordered = value instanceof List<?>;
                var collection = (java.util.Collection<?>) value;
                out.writeByte(ordered ? 7 : 6); out.writeInt(collection.size());
                List<byte[]> entries = new ArrayList<>();
                for (Object entry : collection) entries.add(node(entry, depth + 1));
                if (!ordered) entries.sort(Arrays::compareUnsigned);
                for (byte[] entry : entries) out.write(entry);
            } else {
                if (!OBJECT_TYPES.contains(value.getClass().getName())) throw new IllegalArgumentException("unsupported decoded type: " + value.getClass().getName());
                out.writeByte(8); string(out, value.getClass().getName());
                Map<String, Object> fields = new TreeMap<>();
                for (Class<?> type = value.getClass(); type != Object.class; type = type.getSuperclass()) {
                    for (var field : type.getDeclaredFields()) {
                        if (Modifier.isStatic(field.getModifiers())) continue;
                        field.setAccessible(true); fields.put(type.getName() + "." + field.getName(), field.get(value));
                    }
                }
                out.writeInt(fields.size());
                for (var field : fields.entrySet()) { string(out, field.getKey()); out.write(node(field.getValue(), depth + 1)); }
            }
        }
        return MessageDigest.getInstance("SHA-256").digest(bytes.toByteArray());
    }
    public static void main(String[] args) throws Exception {
        if (args.length != 0) throw new IllegalArgumentException("metadata paths must arrive on stdin");
        try (BufferedReader input = new BufferedReader(new InputStreamReader(System.in, StandardCharsets.UTF_8))) {
            int count = 0;
            for (String name; (name = input.readLine()) != null;) {
                if (++count > 454 || name.length() > 8192) throw new IllegalArgumentException("metadata batch limit");
                Path path = Path.of(name);
                if (!path.isAbsolute() || !path.normalize().equals(path) || !Files.isRegularFile(path, LinkOption.NOFOLLOW_LINKS)
                        || Files.size(path) > MAX_BYTES) throw new IllegalArgumentException("invalid metadata input");
                byte[] bytes = Files.readAllBytes(path);
                var data = checked(new PreviousCompilationData.Serializer(new StringInterner()), bytes);
                System.out.println(sha(bytes) + "\t" + data.semantic());
            }
            if (count == 0) throw new IllegalArgumentException("empty metadata batch");
        }
    }
}
