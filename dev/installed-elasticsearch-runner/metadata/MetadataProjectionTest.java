import java.io.ByteArrayOutputStream;
import java.util.Arrays;
import java.util.LinkedHashMap;
import java.util.LinkedHashSet;
import java.util.List;
import java.util.Map;
import java.util.Set;
import org.gradle.internal.serialize.BaseSerializerFactory;
import org.gradle.internal.serialize.MapSerializer;
import org.gradle.internal.serialize.SetSerializer;
import org.gradle.internal.serialize.kryo.KryoBackedEncoder;
import org.gradle.api.internal.tasks.compile.incremental.deps.IntSetSerializer;
import org.gradle.api.internal.tasks.compile.incremental.compilerapi.deps.DependentSetSerializer;
import org.gradle.api.internal.cache.StringInterner;
import org.gradle.internal.serialize.HierarchicalNameSerializer;
import org.gradle.api.internal.tasks.compile.incremental.recomp.PreviousCompilationData;
import java.nio.file.Files;
import java.nio.file.Path;

/** Standalone qualification: no Gradle invocation and no capture is modified. */
public final class MetadataProjectionTest {
    interface Action { void run() throws Exception; }
    static int checks;
    static void require(boolean condition) {
        if (!condition) throw new AssertionError("metadata qualification failed");
        checks++;
    }
    static void rejects(Action action) throws Exception {
        try { action.run(); } catch (Exception expected) { checks++; return; }
        throw new AssertionError("invalid metadata accepted");
    }
    static byte[] strings(String... entries) throws Exception {
        ByteArrayOutputStream bytes = new ByteArrayOutputStream();
        try (KryoBackedEncoder encoder = new KryoBackedEncoder(bytes)) {
            encoder.writeInt(entries.length);
            for (String entry : entries) encoder.writeString(entry);
        }
        return bytes.toByteArray();
    }
    public static void main(String[] args) throws Exception {
        var strings = BaseSerializerFactory.STRING_SERIALIZER;
        var sets = new SetSerializer<String>(strings);
        var maps = new MapSerializer<String, String>(strings, strings);
        byte[] set = strings("a", "b");
        require(MetadataProjection.checked(sets, set).semantic().equals(
                MetadataProjection.checked(sets, strings("b", "a")).semantic()));
        rejects(() -> MetadataProjection.checked(sets, strings("a", "a")));
        rejects(() -> MetadataProjection.checked(sets, Arrays.copyOf(set, set.length - 1)));
        rejects(() -> MetadataProjection.checked(sets, Arrays.copyOf(set, set.length + 1)));
        byte[] huge = set.clone(); Arrays.fill(huge, 0, 4, (byte)0x7f);
        rejects(() -> MetadataProjection.checked(sets, huge));
        byte[] negative = set.clone(); Arrays.fill(negative, 0, 4, (byte)0xff);
        rejects(() -> MetadataProjection.checked(sets, negative));
        ByteArrayOutputStream bytes = new ByteArrayOutputStream();
        try (KryoBackedEncoder encoder = new KryoBackedEncoder(bytes)) {
            encoder.writeInt(2);
            encoder.writeString("key"); encoder.writeString("a");
            encoder.writeString("key"); encoder.writeString("b");
        }
        rejects(() -> MetadataProjection.checked(maps, bytes.toByteArray()));
        Map<String, String> a = new LinkedHashMap<>(); a.put("a", "1"); a.put("b", "2");
        Map<String, String> b = new LinkedHashMap<>(); b.put("b", "2"); b.put("a", "1");
        require(MetadataProjection.checked(maps, MetadataProjection.encode(maps, a)).semantic().equals(
                MetadataProjection.checked(maps, MetadataProjection.encode(maps, b)).semantic()));
        b.put("a", "3");
        require(!MetadataProjection.semantic(a).equals(MetadataProjection.semantic(b)));
        require(MetadataProjection.semantic(new LinkedHashSet<>(List.of("a", "b"))).equals(
                MetadataProjection.semantic(new LinkedHashSet<>(List.of("b", "a")))));
        require(!MetadataProjection.semantic(List.of("a", "b")).equals(MetadataProjection.semantic(List.of("b", "a"))));
        require(!MetadataProjection.semantic(List.of("a", "b")).equals(MetadataProjection.semantic(Set.of("a", "b"))));
        require(!MetadataProjection.semantic(null).equals(MetadataProjection.semantic("null")));
        require(!MetadataProjection.semantic(Map.of("a=b", "c")).equals(MetadataProjection.semantic(Map.of("a", "b=c"))));
        rejects(() -> MetadataProjection.semantic(new Object()));
        require(!MetadataProjection.semantic("\uD800").equals(MetadataProjection.semantic("\uD801")));
        ByteArrayOutputStream ints = new ByteArrayOutputStream();
        try (KryoBackedEncoder encoder = new KryoBackedEncoder(ints)) {
            encoder.writeInt(2); encoder.writeInt(7); encoder.writeInt(7);
        }
        rejects(() -> MetadataProjection.checked(IntSetSerializer.INSTANCE, ints.toByteArray()));
        byte[] distinct = ints.toByteArray(); distinct[11] = 8;
        var constants = MetadataProjection.checked(IntSetSerializer.INSTANCE, distinct);
        byte[] changed = distinct.clone(); changed[11] = 9;
        require(!constants.semantic().equals(MetadataProjection.checked(IntSetSerializer.INSTANCE, changed).semantic()));
        ByteArrayOutputStream dependencies = new ByteArrayOutputStream();
        try (KryoBackedEncoder encoder = new KryoBackedEncoder(dependencies)) {
            var names = new HierarchicalNameSerializer(new StringInterner());
            encoder.writeByte((byte)1); encoder.writeSmallInt(2);
            names.write(encoder, "a.Class"); names.write(encoder, "a.Class");
            encoder.writeSmallInt(0); encoder.writeSmallInt(0);
        }
        var dependents = new DependentSetSerializer(() -> new HierarchicalNameSerializer(new StringInterner()));
        rejects(() -> MetadataProjection.checked(dependents, dependencies.toByteArray()));
        byte[] discriminator = dependencies.toByteArray(); discriminator[0] = 2;
        rejects(() -> MetadataProjection.checked(dependents, discriminator));
        if (args.length != 1) throw new IllegalArgumentException("one retained real metadata file required");
        byte[] original = Files.readAllBytes(Path.of(args[0]));
        var previous = new PreviousCompilationData.Serializer(new StringInterner());
        var parsed = MetadataProjection.checked(previous, original);
        rejects(() -> MetadataProjection.checked(previous, Arrays.copyOf(original, original.length - 1)));
        rejects(() -> MetadataProjection.checked(previous, Arrays.copyOf(original, original.length + 1)));
        var snapshotField = PreviousCompilationData.class.getDeclaredField("outputSnapshot"); snapshotField.setAccessible(true);
        Object snapshot = snapshotField.get(parsed.value());
        var hashesField = snapshot.getClass().getDeclaredField("classHashes"); hashesField.setAccessible(true);
        Map<?, ?> hashes = (Map<?, ?>) hashesField.get(snapshot);
        require(!hashes.isEmpty());
        Map<Object, Object> alteredHashes = new LinkedHashMap<>(hashes);
        Object firstKey = alteredHashes.keySet().iterator().next();
        alteredHashes.put(firstKey, org.gradle.internal.hash.HashCode.fromBytes(new byte[16]));
        hashesField.set(snapshot, alteredHashes);
        var altered = MetadataProjection.checked(previous, MetadataProjection.encode(previous, parsed.value()));
        require(!parsed.semantic().equals(altered.semantic()));
        System.out.println("PASS standalone metadata checks=" + checks);
    }
}
