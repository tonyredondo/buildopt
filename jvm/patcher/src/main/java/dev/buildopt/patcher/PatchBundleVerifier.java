package dev.buildopt.patcher;

import static java.nio.file.LinkOption.NOFOLLOW_LINKS;

import java.io.IOException;
import java.io.InputStream;
import java.nio.ByteBuffer;
import java.nio.charset.CharacterCodingException;
import java.nio.charset.CodingErrorAction;
import java.nio.charset.StandardCharsets;
import java.nio.file.Files;
import java.nio.file.Path;
import java.nio.file.attribute.BasicFileAttributes;
import java.security.GeneralSecurityException;
import java.security.MessageDigest;
import java.security.PublicKey;
import java.security.Signature;
import java.time.Instant;
import java.time.format.DateTimeParseException;
import java.util.ArrayList;
import java.util.Base64;
import java.util.Comparator;
import java.util.HexFormat;
import java.util.LinkedHashMap;
import java.util.HashMap;
import java.util.HashSet;
import java.util.List;
import java.util.Map;
import java.util.Set;
import java.util.regex.Pattern;

/**
 * Strict, dependency-free verifier for the closed PatchBundle v1 envelope.
 */
public final class PatchBundleVerifier {
    private static final long MAXIMUM_MANIFEST_BYTES = 2L * 1024L * 1024L;
    private static final long MAXIMUM_BLOB_BYTES = 1024L * 1024L;
    private static final Pattern IDENTIFIER = Pattern.compile(
            "^[A-Za-z0-9][A-Za-z0-9._:/-]{0,255}$");
    private static final Pattern BRANCH_IDENTIFIER = Pattern.compile(
            "^[A-Za-z0-9][A-Za-z0-9._-]{0,127}$");
    private static final Pattern SHA256 = Pattern.compile("^sha256:[0-9a-f]{64}$");
    private static final Pattern GIT_OBJECT = Pattern.compile(
            "^(?:[0-9a-f]{40}|[0-9a-f]{64})$");
    private static final Pattern UTC_TIMESTAMP = Pattern.compile(
            "^\\d{4}-\\d{2}-\\d{2}T\\d{2}:\\d{2}:[0-5]\\d(?:\\.\\d{1,9})?Z$");

    private PatchBundleVerifier() {
    }

    /**
     * Verifies a bundle, its configured authority, and every exact blob.
     *
     * @param manifestPath signed bundle JSON
     * @param bundleRoot root beneath which replacement blobs are stored
     * @param expectedRepositoryId repository identity supplied out of band
     * @param expectedActionId action identity supplied out of band
     * @param expectedKeyId configured trust-root identity
     * @param trustedKey configured Ed25519 public key
     * @param now authoritative verification time
     * @return immutable verified materialization input
     * @throws PatchFailure when any check fails closed
     */
    public static VerifiedPatchBundle verify(
            Path manifestPath,
            Path bundleRoot,
            String expectedRepositoryId,
            String expectedActionId,
            String expectedKeyId,
            PublicKey trustedKey,
            Instant now) throws PatchFailure {
        try {
            return verifyInternal(
                    manifestPath,
                    bundleRoot,
                    expectedRepositoryId,
                    expectedActionId,
                    expectedKeyId,
                    trustedKey,
                    now);
        } catch (PatchFailure failure) {
            throw failure;
        } catch (IOException | GeneralSecurityException | IllegalArgumentException exception) {
            throw new PatchFailure(
                    PatchFailure.Status.REJECTED,
                    "bundle verification failed: " + exception.getMessage(),
                    exception);
        }
    }

    private static VerifiedBundle verifyInternal(
            Path manifestPath,
            Path bundleRoot,
            String expectedRepositoryId,
            String expectedActionId,
            String expectedKeyId,
            PublicKey trustedKey,
            Instant now)
            throws IOException, GeneralSecurityException, PatchFailure {
        if (trustedKey == null || !"EdDSA".equals(trustedKey.getAlgorithm())) {
            reject("configured trust root is not Ed25519");
        }
        Path canonicalRoot = bundleRoot.toRealPath(NOFOLLOW_LINKS);
        if (!Files.isDirectory(canonicalRoot, NOFOLLOW_LINKS)) {
            reject("bundle root is not a directory");
        }
        Path canonicalManifest = manifestPath.toRealPath(NOFOLLOW_LINKS);
        if (!canonicalManifest.startsWith(canonicalRoot)
                || !Files.isRegularFile(canonicalManifest, NOFOLLOW_LINKS)) {
            reject("manifest is not a regular child of the bundle root");
        }

        Map<String, Object> root = object(
                StrictJson.parse(canonicalManifest, MAXIMUM_MANIFEST_BYTES),
                "bundle");
        exactFields(root, "bundle", Set.of(
                "schemaVersion",
                "recordType",
                "contractVersion",
                "repositoryId",
                "actionId",
                "bundleDigest",
                "baseRevision",
                "baseTree",
                "sourceStateDigest",
                "recipe",
                "createdAt",
                "expiresAt",
                "operations",
                "blobs",
                "validation",
                "delivery",
                "signature"));
        constant(root, "schemaVersion", "1.0");
        constant(root, "recordType", "PATCH_BUNDLE");
        constant(root, "contractVersion", "buildopt-patch-bundle/v1");

        String repositoryId = identifier(root, "repositoryId");
        String actionId = identifier(root, "actionId");
        if (!BRANCH_IDENTIFIER.matcher(actionId).matches()) {
            reject("actionId cannot form a new buildopt branch");
        }
        String declaredBundleDigest = sha256Digest(root, "bundleDigest");
        String baseRevision = gitObject(root, "baseRevision");
        String baseTree = gitObject(root, "baseTree");
        String sourceStateDigest = sha256Digest(root, "sourceStateDigest");

        Recipe recipe = parseRecipe(object(root.get("recipe"), "recipe"));
        Instant createdAt = timestamp(root, "createdAt");
        Instant expiresAt = timestamp(root, "expiresAt");
        if (!createdAt.isBefore(expiresAt)) {
            reject("bundle has an empty validity window");
        }

        parseValidation(object(root.get("validation"), "validation"), createdAt, expiresAt);
        parseDelivery(object(root.get("delivery"), "delivery"));

        List<Object> rawBlobs = array(root.get("blobs"), "blobs");
        List<Object> rawOperations = array(root.get("operations"), "operations");
        List<Blob> blobDeclarations = parseBlobDeclarations(
                rawBlobs,
                canonicalRoot);
        parseOperations(rawOperations, blobDeclarations);

        String calculatedBundleDigest = calculateBundleDigest(root);
        if (!declaredBundleDigest.equals(calculatedBundleDigest)) {
            reject("bundleDigest does not match canonical manifest and blob inventory");
        }

        Map<String, Object> signature = object(root.get("signature"), "signature");
        exactFields(signature, "signature", Set.of(
                "algorithm",
                "canonicalization",
                "keyId",
                "signedBundleDigest",
                "value"));
        constant(signature, "algorithm", "Ed25519");
        constant(signature, "canonicalization", "JCS");
        String keyId = identifier(signature, "keyId");
        if (!keyId.equals(expectedKeyId)) {
            reject("signature keyId is not the configured trust root");
        }
        String signedDigest = sha256Digest(signature, "signedBundleDigest");
        if (!signedDigest.equals(calculatedBundleDigest)) {
            reject("signature is bound to another bundle digest");
        }
        String encodedSignature = string(signature, "value");
        if (!encodedSignature.matches("^[A-Za-z0-9_-]{86}$")) {
            reject("signature value is not unpadded base64url Ed25519");
        }
        byte[] signatureBytes;
        try {
            signatureBytes = Base64.getUrlDecoder().decode(encodedSignature);
        } catch (IllegalArgumentException exception) {
            throw new PatchFailure(
                    PatchFailure.Status.REJECTED,
                    "signature value is invalid",
                    exception);
        }
        if (signatureBytes.length != 64
                || !verifySignature(
                        trustedKey,
                        signaturePayload(
                                calculatedBundleDigest,
                                "buildopt-patch-bundle/v1",
                                keyId),
                        signatureBytes)) {
            reject("Ed25519 signature verification failed");
        }
        if (!repositoryId.equals(expectedRepositoryId)
                || !actionId.equals(expectedActionId)) {
            reject("repositoryId or actionId does not match caller authority");
        }
        if (now.isBefore(createdAt) || !now.isBefore(expiresAt)) {
            reject("bundle is not currently valid");
        }

        List<Blob> blobs = verifyBlobContents(blobDeclarations, canonicalRoot);
        List<Operation> operations = parseOperations(rawOperations, blobs);

        return new VerifiedBundle(
                repositoryId,
                actionId,
                calculatedBundleDigest,
                baseRevision,
                baseTree,
                sourceStateDigest,
                recipe,
                createdAt,
                expiresAt,
                List.copyOf(operations),
                List.copyOf(blobs));
    }

    static String calculateBundleDigest(Map<String, Object> root)
            throws GeneralSecurityException, PatchFailure {
        Map<String, Object> manifest = new LinkedHashMap<>(root);
        manifest.remove("bundleDigest");
        manifest.remove("signature");
        Object rawBlobs = manifest.remove("blobs");
        List<Object> inventory = new ArrayList<>();
        for (Object item : array(rawBlobs, "blobs")) {
            Map<String, Object> blob = object(item, "blob");
            Map<String, Object> entry = new LinkedHashMap<>();
            entry.put("blobRef", string(blob, "blobRef"));
            entry.put("blobSha256", sha256Digest(blob, "blobSha256"));
            entry.put("sizeBytes", integer(blob, "sizeBytes", 1, MAXIMUM_BLOB_BYTES));
            inventory.add(entry);
        }
        inventory.sort(Comparator.comparing(
                value -> (String) ((Map<?, ?>) value).get("blobRef")));
        Map<String, Object> canonicalInput = new LinkedHashMap<>();
        canonicalInput.put("manifest", manifest);
        canonicalInput.put("blobs", inventory);
        return digestBytes(StrictJson.canonicalBytes(canonicalInput));
    }

    static byte[] signaturePayload(
            String bundleDigest,
            String contractVersion,
            String keyId) {
        Map<String, Object> payload = new LinkedHashMap<>();
        payload.put("bundleDigest", bundleDigest);
        payload.put("contractVersion", contractVersion);
        payload.put("keyId", keyId);
        return StrictJson.canonicalBytes(payload);
    }

    static String digestBytes(byte[] content) throws GeneralSecurityException {
        return "sha256:" + HexFormat.of().formatHex(
                MessageDigest.getInstance("SHA-256").digest(content));
    }

    private static Recipe parseRecipe(Map<String, Object> recipe) throws PatchFailure {
        String id = string(recipe, "id");
        constant(recipe, "version", "1.0");
        if ("ARCHIVE_REPRODUCIBILITY_KOTLIN_DSL_V1".equals(id)) {
            exactFields(recipe, "recipe", Set.of("id", "version"));
            return new Recipe(id, "1.0", "");
        }
        if ("CUSTOM_TASK_CONTRACT_JAVA_V1".equals(id)) {
            exactFields(recipe, "recipe", Set.of("id", "version", "reviewedAdapter"));
            Map<String, Object> adapter = object(
                    recipe.get("reviewedAdapter"),
                    "reviewedAdapter");
            exactFields(adapter, "reviewedAdapter", Set.of(
                    "adapterId",
                    "adapterDigest",
                    "evidenceRef"));
            String adapterId = identifier(adapter, "adapterId");
            sha256Digest(adapter, "adapterDigest");
            identifier(adapter, "evidenceRef");
            return new Recipe(id, "1.0", adapterId);
        }
        reject("unsupported recipe id");
        throw new AssertionError("unreachable");
    }

    private static void parseValidation(
            Map<String, Object> validation,
            Instant createdAt,
            Instant bundleExpiresAt) throws PatchFailure {
        exactFields(validation, "validation", Set.of(
                "mode",
                "status",
                "requestId",
                "resultId",
                "artifactSetDigest",
                "completedAt",
                "expiresAt"));
        constant(validation, "mode", "FULL_RELEVANT_VALIDATION");
        constant(validation, "status", "PASSED");
        identifier(validation, "requestId");
        identifier(validation, "resultId");
        sha256Digest(validation, "artifactSetDigest");
        Instant completedAt = timestamp(validation, "completedAt");
        Instant expiresAt = timestamp(validation, "expiresAt");
        if (createdAt.isBefore(completedAt) || expiresAt.isBefore(bundleExpiresAt)) {
            reject("validation does not cover the bundle lifetime");
        }
    }

    private static void parseDelivery(Map<String, Object> delivery)
            throws PatchFailure {
        exactFields(delivery, "delivery", Set.of(
                "branchPrefix",
                "draftPullRequest",
                "autoMerge",
                "forcePush",
                "modifyExistingBranch"));
        constant(delivery, "branchPrefix", "buildopt/");
        booleanConstant(delivery, "draftPullRequest", true);
        booleanConstant(delivery, "autoMerge", false);
        booleanConstant(delivery, "forcePush", false);
        booleanConstant(delivery, "modifyExistingBranch", false);
    }

    private static List<Blob> parseBlobDeclarations(
            List<Object> rawBlobs,
            Path root) throws PatchFailure {
        if (rawBlobs.isEmpty() || rawBlobs.size() > 256) {
            reject("blobs count is outside 1..256");
        }
        List<Blob> blobs = new ArrayList<>();
        Set<String> refs = new HashSet<>();
        String priorRef = null;
        for (Object rawBlob : rawBlobs) {
            Map<String, Object> blob = object(rawBlob, "blob");
            exactFields(blob, "blob", Set.of(
                    "blobRef",
                    "blobSha256",
                    "sizeBytes",
                    "mediaType",
                    "encoding"));
            String ref = safePath(string(blob, "blobRef"), "blobRef");
            if (!refs.add(ref) || (priorRef != null && priorRef.compareTo(ref) >= 0)) {
                reject("blob references must be unique and sorted");
            }
            priorRef = ref;
            String declaredDigest = sha256Digest(blob, "blobSha256");
            long declaredSize = integer(blob, "sizeBytes", 1, MAXIMUM_BLOB_BYTES);
            constant(blob, "mediaType", "text/plain");
            constant(blob, "encoding", "UTF-8");
            Path path = root.resolve(ref).normalize();
            if (!path.startsWith(root)) {
                reject("blob reference escapes its root: " + ref);
            }
            blobs.add(new Blob(
                    ref,
                    declaredDigest,
                    declaredSize,
                    path,
                    new byte[0]));
        }
        return blobs;
    }

    private static List<Blob> verifyBlobContents(
            List<Blob> declarations,
            Path root)
            throws IOException, GeneralSecurityException, PatchFailure {
        List<Blob> verified = new ArrayList<>();
        for (Blob declaration : declarations) {
            Path path = resolveLinkSafe(root, declaration.reference(), "blob");
            byte[] content;
            try (InputStream input = Files.newInputStream(path, NOFOLLOW_LINKS)) {
                content = input.readNBytes(Math.toIntExact(declaration.size()) + 1);
            }
            if (content.length != declaration.size()) {
                reject("blob size does not match exact bytes: "
                        + declaration.reference());
            }
            try {
                StandardCharsets.UTF_8.newDecoder()
                        .onMalformedInput(CodingErrorAction.REPORT)
                        .onUnmappableCharacter(CodingErrorAction.REPORT)
                        .decode(ByteBuffer.wrap(content));
            } catch (CharacterCodingException exception) {
                throw new PatchFailure(
                        PatchFailure.Status.REJECTED,
                        "blob is not valid UTF-8: " + declaration.reference(),
                        exception);
            }
            if (!declaration.digest().equals(digestBytes(content))) {
                reject("blob digest does not match exact bytes: "
                        + declaration.reference());
            }
            verified.add(new Blob(
                    declaration.reference(),
                    declaration.digest(),
                    declaration.size(),
                    path,
                    content));
        }
        return verified;
    }

    private static List<Operation> parseOperations(
            List<Object> rawOperations,
            List<Blob> blobs) throws PatchFailure {
        if (rawOperations.isEmpty() || rawOperations.size() > 256) {
            reject("operations count is outside 1..256");
        }
        Map<String, Blob> blobsByRef = new HashMap<>();
        for (Blob blob : blobs) {
            blobsByRef.put(blob.reference(), blob);
        }
        Set<String> paths = new HashSet<>();
        Set<String> usedBlobs = new HashSet<>();
        List<Operation> operations = new ArrayList<>();
        for (int index = 0; index < rawOperations.size(); index++) {
            Map<String, Object> operation = object(
                    rawOperations.get(index),
                    "operation");
            String type = string(operation, "type");
            Set<String> expectedFields;
            if ("MODIFY".equals(type)) {
                expectedFields = Set.of(
                        "order",
                        "type",
                        "path",
                        "expectedMode",
                        "preimageDigest",
                        "postimageDigest",
                        "replacementBlob");
            } else if ("ADD".equals(type)) {
                expectedFields = Set.of(
                        "order",
                        "type",
                        "path",
                        "expectedMode",
                        "postimageDigest",
                        "replacementBlob");
            } else {
                reject("operation type is not ADD or MODIFY");
                throw new AssertionError("unreachable");
            }
            exactFields(operation, "operation", expectedFields);
            long order = integer(operation, "order", 1, 256);
            if (order != index + 1L) {
                reject("operation order is not contiguous");
            }
            String path = safePath(string(operation, "path"), "operation path");
            if (!paths.add(path)) {
                reject("duplicate operation path: " + path);
            }
            constant(operation, "expectedMode", "100644");
            String preimage = "MODIFY".equals(type)
                    ? sha256Digest(operation, "preimageDigest")
                    : "";
            String postimage = sha256Digest(operation, "postimageDigest");
            String blobRef = safePath(
                    string(operation, "replacementBlob"),
                    "replacementBlob");
            if (!usedBlobs.add(blobRef)) {
                reject("replacement blob is reused: " + blobRef);
            }
            Blob blob = blobsByRef.get(blobRef);
            if (blob == null || !postimage.equals(blob.digest())) {
                reject("operation postimage does not match its replacement blob");
            }
            operations.add(new Operation(
                    (int) order,
                    type,
                    path,
                    preimage,
                    postimage,
                    blob));
        }
        if (usedBlobs.size() != blobs.size()) {
            reject("bundle contains an unreferenced blob");
        }
        return operations;
    }

    private static Path resolveLinkSafe(Path root, String relative, String label)
            throws IOException, PatchFailure {
        Path current = root;
        String[] segments = relative.split("/", -1);
        for (String segment : segments) {
            current = current.resolve(segment);
            BasicFileAttributes attributes = Files.readAttributes(
                    current,
                    BasicFileAttributes.class,
                    NOFOLLOW_LINKS);
            if (attributes.isSymbolicLink()) {
                reject(label + " path follows a symlink: " + relative);
            }
        }
        Path normalized = current.normalize();
        if (!normalized.startsWith(root)
                || !Files.isRegularFile(normalized, NOFOLLOW_LINKS)) {
            reject(label + " is not a regular child path: " + relative);
        }
        return normalized;
    }

    static String safePath(String value, String label) throws PatchFailure {
        if (value.isEmpty()
                || value.length() > 1024
                || value.indexOf('\0') >= 0
                || value.startsWith("/")
                || value.endsWith("/")
                || value.contains("//")
                || value.indexOf('\\') >= 0) {
            reject(label + " is not a canonical relative path");
        }
        for (String segment : value.split("/", -1)) {
            if (segment.isEmpty()
                    || ".".equals(segment)
                    || "..".equals(segment)
                    || ".git".equals(segment)) {
                reject(label + " contains a forbidden segment");
            }
        }
        return value;
    }

    private static boolean verifySignature(
            PublicKey key,
            byte[] payload,
            byte[] signatureBytes) throws GeneralSecurityException {
        Signature verifier = Signature.getInstance("Ed25519");
        verifier.initVerify(key);
        verifier.update(payload);
        return verifier.verify(signatureBytes);
    }

    private static Map<String, Object> object(Object value, String label)
            throws PatchFailure {
        if (!(value instanceof Map<?, ?>)) {
            reject(label + " is not an object");
        }
        Map<?, ?> raw = (Map<?, ?>) value;
        Map<String, Object> result = new LinkedHashMap<>();
        for (Map.Entry<?, ?> entry : raw.entrySet()) {
            if (!(entry.getKey() instanceof String key)) {
                reject(label + " has a non-string key");
                throw new AssertionError("unreachable");
            }
            result.put(key, entry.getValue());
        }
        return result;
    }

    private static List<Object> array(Object value, String label)
            throws PatchFailure {
        if (!(value instanceof List<?>)) {
            reject(label + " is not an array");
        }
        List<?> raw = (List<?>) value;
        return new ArrayList<>(raw);
    }

    private static void exactFields(
            Map<String, Object> object,
            String label,
            Set<String> expected) throws PatchFailure {
        if (!object.keySet().equals(expected)) {
            reject(label + " fields do not match the closed schema");
        }
    }

    private static String string(Map<String, Object> object, String field)
            throws PatchFailure {
        Object value = object.get(field);
        if (!(value instanceof String)) {
            reject(field + " is not a string");
        }
        return (String) value;
    }

    private static String identifier(Map<String, Object> object, String field)
            throws PatchFailure {
        String value = string(object, field);
        if (!IDENTIFIER.matcher(value).matches()) {
            reject(field + " is not a bounded identifier");
        }
        return value;
    }

    private static String sha256Digest(Map<String, Object> object, String field)
            throws PatchFailure {
        String value = string(object, field);
        if (!SHA256.matcher(value).matches()) {
            reject(field + " is not a SHA-256 digest");
        }
        return value;
    }

    private static String gitObject(Map<String, Object> object, String field)
            throws PatchFailure {
        String value = string(object, field);
        if (!GIT_OBJECT.matcher(value).matches()) {
            reject(field + " is not a full Git object id");
        }
        return value;
    }

    private static long integer(
            Map<String, Object> object,
            String field,
            long minimum,
            long maximum) throws PatchFailure {
        Object value = object.get(field);
        if (!(value instanceof StrictJson.JsonNumber)) {
            reject(field + " is not a non-negative JSON integer");
        }
        StrictJson.JsonNumber number = (StrictJson.JsonNumber) value;
        if (!number.source().matches("^(?:0|[1-9][0-9]*)$")) {
            reject(field + " is not a non-negative JSON integer");
        }
        long parsed;
        try {
            parsed = Long.parseLong(number.source());
        } catch (NumberFormatException exception) {
            throw new PatchFailure(
                    PatchFailure.Status.REJECTED,
                    field + " is outside the integer range",
                    exception);
        }
        if (parsed < minimum || parsed > maximum) {
            reject(field + " is outside " + minimum + ".." + maximum);
        }
        return parsed;
    }

    private static Instant timestamp(Map<String, Object> object, String field)
            throws PatchFailure {
        String value = string(object, field);
        if (!UTC_TIMESTAMP.matcher(value).matches()) {
            reject(field + " is not a UTC RFC 3339 timestamp");
        }
        try {
            return Instant.parse(value);
        } catch (DateTimeParseException exception) {
            throw new PatchFailure(
                    PatchFailure.Status.REJECTED,
                    field + " is not a valid timestamp",
                    exception);
        }
    }

    private static void constant(
            Map<String, Object> object,
            String field,
            String expected) throws PatchFailure {
        if (!expected.equals(string(object, field))) {
            reject(field + " must equal " + expected);
        }
    }

    private static void booleanConstant(
            Map<String, Object> object,
            String field,
            boolean expected) throws PatchFailure {
        Object value = object.get(field);
        if (!(value instanceof Boolean booleanValue) || booleanValue != expected) {
            reject(field + " must equal " + expected);
        }
    }

    private static void reject(String message) throws PatchFailure {
        throw new PatchFailure(PatchFailure.Status.REJECTED, message);
    }

    /**
     * Opaque verified input accepted by {@link PatchBundleApplier}.
     *
     * <p>The only implementation has a private constructor, so callers cannot
     * bypass {@link #verify(Path, Path, String, String, String, PublicKey,
     * Instant)} by constructing trusted state.</p>
     */
    public sealed interface VerifiedPatchBundle permits VerifiedBundle {
        String repositoryId();

        String actionId();

        String bundleDigest();

        String baseRevision();

        String baseTree();

        String sourceStateDigest();

        Instant createdAt();
    }

    static final class VerifiedBundle implements VerifiedPatchBundle {
        private final String repositoryId;
        private final String actionId;
        private final String bundleDigest;
        private final String baseRevision;
        private final String baseTree;
        private final String sourceStateDigest;
        private final Recipe recipe;
        private final Instant createdAt;
        private final Instant expiresAt;
        private final List<Operation> operations;
        private final List<Blob> blobs;

        private VerifiedBundle(
                String repositoryId,
                String actionId,
                String bundleDigest,
                String baseRevision,
                String baseTree,
                String sourceStateDigest,
                Recipe recipe,
                Instant createdAt,
                Instant expiresAt,
                List<Operation> operations,
                List<Blob> blobs) {
            this.repositoryId = repositoryId;
            this.actionId = actionId;
            this.bundleDigest = bundleDigest;
            this.baseRevision = baseRevision;
            this.baseTree = baseTree;
            this.sourceStateDigest = sourceStateDigest;
            this.recipe = recipe;
            this.createdAt = createdAt;
            this.expiresAt = expiresAt;
            this.operations = List.copyOf(operations);
            this.blobs = List.copyOf(blobs);
        }

        @Override
        public String repositoryId() {
            return repositoryId;
        }

        @Override
        public String actionId() {
            return actionId;
        }

        @Override
        public String bundleDigest() {
            return bundleDigest;
        }

        @Override
        public String baseRevision() {
            return baseRevision;
        }

        @Override
        public String baseTree() {
            return baseTree;
        }

        @Override
        public String sourceStateDigest() {
            return sourceStateDigest;
        }

        @Override
        public Instant createdAt() {
            return createdAt;
        }

        Recipe recipe() {
            return recipe;
        }

        Instant expiresAt() {
            return expiresAt;
        }

        List<Operation> operations() {
            return operations;
        }

        List<Blob> blobs() {
            return blobs;
        }
    }

    /** One of the two bounded recipe/version combinations. */
    public record Recipe(String id, String version, String reviewedAdapterId) {
    }

    /** An ordered exact replacement operation. */
    public record Operation(
            int order,
            String type,
            String path,
            String preimageDigest,
            String postimageDigest,
            Blob blob) {
    }

    /** Verified exact UTF-8 replacement bytes. */
    public record Blob(
            String reference,
            String digest,
            long size,
            Path path,
            byte[] content) {
        public Blob {
            content = content.clone();
        }

        @Override
        public byte[] content() {
            return content.clone();
        }
    }
}
