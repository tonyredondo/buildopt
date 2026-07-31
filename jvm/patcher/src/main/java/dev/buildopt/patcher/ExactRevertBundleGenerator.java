package dev.buildopt.patcher;

import static java.nio.file.LinkOption.NOFOLLOW_LINKS;
import static java.nio.file.StandardCopyOption.ATOMIC_MOVE;
import static java.nio.file.StandardCopyOption.REPLACE_EXISTING;

import java.io.IOException;
import java.nio.ByteBuffer;
import java.nio.charset.CharacterCodingException;
import java.nio.charset.CodingErrorAction;
import java.nio.charset.StandardCharsets;
import java.nio.file.Files;
import java.nio.file.Path;
import java.security.GeneralSecurityException;
import java.security.PrivateKey;
import java.time.Instant;
import java.util.ArrayList;
import java.util.Comparator;
import java.util.LinkedHashMap;
import java.util.List;
import java.util.Map;
import java.util.regex.Pattern;

import dev.buildopt.patcher.PatchBundleSigner.SignedPatchBundle;
import dev.buildopt.patcher.PatchBundleVerifier.Operation;
import dev.buildopt.patcher.PatchBundleVerifier.VerifiedBundle;
import dev.buildopt.patcher.PatchBundleVerifier.VerifiedPatchBundle;

/** Creates an exact, signed inverse PatchBundle for a merged MODIFY-only patch. */
public final class ExactRevertBundleGenerator {
    private static final Pattern ACTION_ID = Pattern.compile(
            "^[A-Za-z0-9][A-Za-z0-9._-]{0,127}$");
    private static final Pattern IDENTIFIER = Pattern.compile(
            "^[A-Za-z0-9][A-Za-z0-9._:/-]{0,255}$");
    private static final Pattern SHA256 = Pattern.compile("^sha256:[0-9a-f]{64}$");
    private static final Pattern GIT_OBJECT = Pattern.compile("^(?:[0-9a-f]{40}|[0-9a-f]{64})$");
    private static final int MAXIMUM_GIT_OUTPUT = 1024 * 1024;

    private ExactRevertBundleGenerator() {
    }

    /** Current validation authority attached to the inverse bundle. */
    public record Validation(
            String requestId,
            String resultId,
            String artifactSetDigest,
            Instant completedAt,
            Instant expiresAt) {
    }

    /** Generated signed manifest and immutable replacement blobs. */
    public static final class GeneratedRevert {
        private final String actionId;
        private final String bundleDigest;
        private final byte[] manifest;
        private final Map<String, byte[]> blobs;

        private GeneratedRevert(
                String actionId,
                String bundleDigest,
                byte[] manifest,
                Map<String, byte[]> blobs) {
            this.actionId = actionId;
            this.bundleDigest = bundleDigest;
            this.manifest = manifest.clone();
            Map<String, byte[]> copy = new LinkedHashMap<>();
            blobs.forEach((path, content) -> copy.put(path, content.clone()));
            this.blobs = Map.copyOf(copy);
        }

        public String actionId() {
            return actionId;
        }

        public String bundleDigest() {
            return bundleDigest;
        }

        public byte[] manifest() {
            return manifest.clone();
        }

        public Map<String, byte[]> blobs() {
            Map<String, byte[]> copy = new LinkedHashMap<>();
            blobs.forEach((path, content) -> copy.put(path, content.clone()));
            return Map.copyOf(copy);
        }

        /** Writes the generated bundle into a new or empty private directory. */
        public Path write(Path bundleRoot) throws PatchFailure {
            try {
                Files.createDirectories(bundleRoot);
                if (!Files.isDirectory(bundleRoot, NOFOLLOW_LINKS)
                        || Files.isSymbolicLink(bundleRoot)) {
                    throw new IOException("bundle root is not a regular directory");
                }
                try (var entries = Files.list(bundleRoot)) {
                    if (entries.findAny().isPresent()) {
                        throw new IOException("bundle root is not empty");
                    }
                }
                for (Map.Entry<String, byte[]> blob : blobs.entrySet()) {
                    Path target = bundleRoot.resolve(blob.getKey()).normalize();
                    if (!target.startsWith(bundleRoot)) {
                        throw new IOException("blob escapes bundle root");
                    }
                    Files.createDirectories(target.getParent());
                    atomicWrite(target, blob.getValue());
                }
                Path manifestPath = bundleRoot.resolve("manifest.json");
                atomicWrite(manifestPath, manifest);
                return manifestPath;
            } catch (IOException exception) {
                throw new PatchFailure(
                        PatchFailure.Status.UNCHANGED,
                        "cannot write exact revert bundle",
                        exception);
            }
        }
    }

    /**
     * Generates a signed inverse with an action ID derived from the original bundle digest.
     * Every original postimage must still exist exactly at the merged revision. ADD operations and later edits fail closed.
     */
    public static GeneratedRevert generate(
            VerifiedPatchBundle original,
            Path repository,
            String mergedRevision,
            Validation validation,
            Instant createdAt,
            Instant expiresAt,
            String keyId,
            PrivateKey signingKey) throws PatchFailure {
        if (!(original instanceof VerifiedBundle bundle)
                || repository == null
                || mergedRevision == null
                || !GIT_OBJECT.matcher(mergedRevision).matches()

                || validation == null
                || createdAt == null
                || expiresAt == null
                || !createdAt.isBefore(expiresAt)
                || !validValidation(validation, createdAt, expiresAt)) {
            throw new PatchFailure(PatchFailure.Status.PROPOSED, "invalid revert input");
        }
        String revertActionId = "revert-" + bundle.bundleDigest().substring("sha256:".length());
        if (!ACTION_ID.matcher(revertActionId).matches()) {
            throw new PatchFailure(PatchFailure.Status.PROPOSED, "invalid derived revert identity");
        }
        if (!"ARCHIVE_REPRODUCIBILITY_KOTLIN_DSL_V1".equals(bundle.recipe().id())) {
            throw new PatchFailure(
                    PatchFailure.Status.PROPOSED,
                    "exact inverse is unavailable for this recipe");
        }

        Path root;
        try {
            root = repository.toRealPath(NOFOLLOW_LINKS);
        } catch (IOException exception) {
            throw new PatchFailure(PatchFailure.Status.PROPOSED, "cannot resolve repository", exception);
        }
        String exactMerged = gitText(root, "rev-parse", "--verify", mergedRevision + "^{commit}");
        if (!mergedRevision.equals(exactMerged)) {
            throw new PatchFailure(PatchFailure.Status.PROPOSED, "merged revision is not exact");
        }
        String baseTree = gitText(root, "rev-parse", mergedRevision + "^{tree}");
        String sourceState = PatchBundleApplier.sourceStateDigest(root, mergedRevision);

        List<Object> operations = new ArrayList<>();
        List<BlobValue> blobValues = new ArrayList<>();
        Map<String, byte[]> blobs = new LinkedHashMap<>();
        for (int index = 0; index < bundle.operations().size(); index++) {
            Operation operation = bundle.operations().get(index);
            if (!"MODIFY".equals(operation.type())) {
                throw new PatchFailure(
                        PatchFailure.Status.PROPOSED,
                        "exact inverse cannot remove an ADD operation");
            }
            byte[] mergedBytes = gitBytes(
                    root,
                    "show",
                    mergedRevision + ":" + operation.path());
            if (!operation.postimageDigest().equals(digest(mergedBytes))) {
                throw new PatchFailure(
                        PatchFailure.Status.PROPOSED,
                        "merged path no longer matches the patch postimage");
            }
            byte[] originalBytes = gitBytes(
                    root,
                    "show",
                    bundle.baseRevision() + ":" + operation.path());
            if (!operation.preimageDigest().equals(digest(originalBytes))) {
                throw new PatchFailure(
                        PatchFailure.Status.PROPOSED,
                        "original path no longer matches the signed preimage");
            }
            if (originalBytes.length == 0) {
                throw new PatchFailure(
                        PatchFailure.Status.PROPOSED,
                        "exact inverse cannot encode an empty preimage");
            }
            requireUtf8(originalBytes);
            String blobRef = String.format(java.util.Locale.ROOT, "blobs/revert-%03d.txt", index + 1);
            String blobDigest = digest(originalBytes);
            blobs.put(blobRef, originalBytes.clone());
            blobValues.add(new BlobValue(blobRef, blobDigest, originalBytes.length));

            Map<String, Object> inverse = new LinkedHashMap<>();
            inverse.put("order", index + 1);
            inverse.put("type", "MODIFY");
            inverse.put("path", operation.path());
            inverse.put("expectedMode", "100644");
            inverse.put("preimageDigest", operation.postimageDigest());
            inverse.put("postimageDigest", operation.preimageDigest());
            inverse.put("replacementBlob", blobRef);
            operations.add(inverse);
        }
        blobValues.sort(Comparator.comparing(BlobValue::reference));
        List<Object> declarations = new ArrayList<>();
        for (BlobValue blob : blobValues) {
            Map<String, Object> declaration = new LinkedHashMap<>();
            declaration.put("blobRef", blob.reference());
            declaration.put("blobSha256", blob.digest());
            declaration.put("sizeBytes", blob.size());
            declaration.put("mediaType", "text/plain");
            declaration.put("encoding", "UTF-8");
            declarations.add(declaration);
        }

        Map<String, Object> recipe = new LinkedHashMap<>();
        recipe.put("id", bundle.recipe().id());
        recipe.put("version", bundle.recipe().version());
        Map<String, Object> validationValue = new LinkedHashMap<>();
        validationValue.put("mode", "FULL_RELEVANT_VALIDATION");
        validationValue.put("status", "PASSED");
        validationValue.put("requestId", validation.requestId());
        validationValue.put("resultId", validation.resultId());
        validationValue.put("artifactSetDigest", validation.artifactSetDigest());
        validationValue.put("completedAt", validation.completedAt().toString());
        validationValue.put("expiresAt", validation.expiresAt().toString());
        Map<String, Object> delivery = new LinkedHashMap<>();
        delivery.put("branchPrefix", "buildopt/");
        delivery.put("draftPullRequest", true);
        delivery.put("autoMerge", false);
        delivery.put("forcePush", false);
        delivery.put("modifyExistingBranch", false);

        Map<String, Object> rootValue = new LinkedHashMap<>();
        rootValue.put("schemaVersion", "1.0");
        rootValue.put("recordType", "PATCH_BUNDLE");
        rootValue.put("contractVersion", "buildopt-patch-bundle/v1");
        rootValue.put("repositoryId", bundle.repositoryId());
        rootValue.put("actionId", revertActionId);
        rootValue.put("baseRevision", mergedRevision);
        rootValue.put("baseTree", baseTree);
        rootValue.put("sourceStateDigest", sourceState);
        rootValue.put("recipe", recipe);
        rootValue.put("createdAt", createdAt.toString());
        rootValue.put("expiresAt", expiresAt.toString());
        rootValue.put("operations", operations);
        rootValue.put("blobs", declarations);
        rootValue.put("validation", validationValue);
        rootValue.put("delivery", delivery);
        SignedPatchBundle signed = PatchBundleSigner.sign(
                StrictJson.canonicalBytes(rootValue),
                keyId,
                signingKey);
        return new GeneratedRevert(
                revertActionId,
                signed.bundleDigest(),
                signed.canonicalManifest(),
                blobs);
    }

    private static boolean validValidation(
            Validation validation,
            Instant createdAt,
            Instant expiresAt) {
        return validIdentifier(validation.requestId())
                && validIdentifier(validation.resultId())
                && validation.artifactSetDigest() != null
                && SHA256.matcher(validation.artifactSetDigest()).matches()
                && validation.completedAt() != null
                && validation.expiresAt() != null
                && !createdAt.isBefore(validation.completedAt())
                && !validation.expiresAt().isBefore(expiresAt);
    }

    private static void requireUtf8(byte[] content) throws PatchFailure {
        try {
            StandardCharsets.UTF_8.newDecoder()
                    .onMalformedInput(CodingErrorAction.REPORT)
                    .onUnmappableCharacter(CodingErrorAction.REPORT)
                    .decode(ByteBuffer.wrap(content));
        } catch (CharacterCodingException exception) {
            throw new PatchFailure(PatchFailure.Status.PROPOSED, "revert preimage is not UTF-8", exception);
        }
    }

    private static String digest(byte[] content) throws PatchFailure {
        try {
            return PatchBundleVerifier.digestBytes(content);
        } catch (GeneralSecurityException exception) {
            throw new PatchFailure(PatchFailure.Status.UNCHANGED, "cannot digest revert content", exception);
        }
    }

    private static String gitText(Path repository, String... arguments) throws PatchFailure {
        return new String(gitBytes(repository, arguments), StandardCharsets.UTF_8).trim();
    }

    private static byte[] gitBytes(Path repository, String... arguments) throws PatchFailure {
        List<String> command = new ArrayList<>();
        command.add("git");
        command.add("-C");
        command.add(repository.toString());
        command.addAll(List.of(arguments));
        ProcessBuilder builder = new ProcessBuilder(command);
        builder.redirectErrorStream(true);
        try {
            Process process = builder.start();
            process.getOutputStream().close();
            byte[] bytes = process.getInputStream().readNBytes(MAXIMUM_GIT_OUTPUT + 1);
            if (bytes.length > MAXIMUM_GIT_OUTPUT) {
                process.destroyForcibly();
                process.waitFor();
                throw new PatchFailure(
                        PatchFailure.Status.PROPOSED,
                        "bounded Git read exceeded its limit");
            }
            int exitCode = process.waitFor();
            if (exitCode != 0) {
                throw new PatchFailure(
                        PatchFailure.Status.PROPOSED,
                        "bounded Git read failed");
            }
            return bytes;
        } catch (IOException exception) {
            throw new PatchFailure(PatchFailure.Status.PROPOSED, "cannot execute Git", exception);
        } catch (InterruptedException exception) {
            Thread.currentThread().interrupt();
            throw new PatchFailure(PatchFailure.Status.UNCHANGED, "Git read interrupted", exception);
        }
    }

    private static void atomicWrite(Path target, byte[] content) throws IOException {
        Path temporary = Files.createTempFile(target.getParent(), ".buildopt-revert-", ".tmp");
        try {
            Files.write(temporary, content);
            Files.move(temporary, target, ATOMIC_MOVE, REPLACE_EXISTING);
        } finally {
            Files.deleteIfExists(temporary);
        }
    }

    private static boolean validIdentifier(String value) {
        return value != null && IDENTIFIER.matcher(value).matches();
    }

    private record BlobValue(String reference, String digest, int size) {
    }
}
