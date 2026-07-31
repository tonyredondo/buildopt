package dev.buildopt.patcher;

import java.io.ByteArrayInputStream;
import java.io.IOException;
import java.security.GeneralSecurityException;
import java.security.MessageDigest;
import java.util.EnumMap;
import java.util.HashSet;
import java.util.HexFormat;
import java.util.List;
import java.util.Map;
import java.util.Objects;
import java.util.Set;
import java.util.TreeMap;
import java.util.regex.Pattern;
import java.util.zip.ZipEntry;
import java.util.zip.ZipInputStream;

/**
 * Fail-closed candidate/control and artifact validator for Patch Autopilot.
 */
public final class PatchCandidateValidator {
    private static final int MAXIMUM_ARTIFACTS = 1024;
    private static final int MAXIMUM_ARTIFACT_BYTES = 64 * 1024 * 1024;
    private static final long MAXIMUM_TOTAL_BYTES = 256L * 1024L * 1024L;
    private static final long MAXIMUM_EXPANDED_ARCHIVE_BYTES = 256L * 1024L * 1024L;
    private static final Pattern IDENTIFIER = Pattern.compile(
            "^[A-Za-z0-9][A-Za-z0-9._:/-]{0,255}$");
    private static final Pattern SHA256 = Pattern.compile("^sha256:[0-9a-f]{64}$");
    private static final Pattern GIT_OBJECT = Pattern.compile(
            "^(?:[0-9a-f]{40}|[0-9a-f]{64})$");

    private PatchCandidateValidator() {
    }

    /** Validation arm. */
    public enum Arm {
        CANDIDATE,
        CONTROL
    }

    /** Required isolated execution phase for each arm. */
    public enum Phase {
        CLEAN,
        INCREMENTAL,
        RELOCATED
    }

    /** Explicit artifact comparison policy. */
    public enum ArtifactAdapter {
        EXACT_BYTES,
        ARCHIVE_CONTENTS_V1
    }

    /** Terminal qualification status. */
    public enum Status {
        PASSED,
        FAILED,
        INCONCLUSIVE
    }

    /**
     * Validates six isolated runs and their required artifacts.
     *
     * @param request complete immutable validation observation
     * @return a terminal fail-closed result; only {@code PASSED} permits promotion
     */
    public static Result validate(Request request) {
        if (request == null || !validRequest(request)) {
            return Result.inconclusive("INVALID_REQUEST");
        }

        Map<Arm, EnumMap<Phase, ArtifactSet>> observations =
                new EnumMap<>(Arm.class);
        Set<String> isolationKeys = new HashSet<>();
        Context expectedContext = null;

        for (Run run : request.runs()) {
            if (run == null || !validRun(run)) {
                return Result.inconclusive("INVALID_RUN");
            }
            if (!isolationKeys.add(run.isolationKey())) {
                return Result.inconclusive("ISOLATION_NOT_DISTINCT");
            }
            if (expectedContext == null) {
                expectedContext = run.context();
            } else if (!expectedContext.equals(run.context())) {
                return Result.inconclusive("CONTEXT_DRIFT");
            }
            EnumMap<Phase, ArtifactSet> armRuns =
                    observations.computeIfAbsent(
                            run.arm(),
                            ignored -> new EnumMap<>(Phase.class));
            if (armRuns.containsKey(run.phase())) {
                return Result.inconclusive("DUPLICATE_RUN");
            }
            if (run.exitCode() != 0) {
                return Result.failed("ARM_FAILED");
            }
            if (!expectedCacheState(run.phase()).equals(run.configurationCacheState())) {
                return Result.failed("CONFIGURATION_CACHE_FAILED");
            }
            ArtifactSet artifacts;
            try {
                artifacts = artifactSet(
                        run.artifacts(),
                        request.requiredArtifactPaths(),
                        request.artifactAdapter());
            } catch (IOException | GeneralSecurityException | IllegalArgumentException exception) {
                return Result.failed("INVALID_ARTIFACT");
            }
            armRuns.put(run.phase(), artifacts);
        }

        for (Arm arm : Arm.values()) {
            Map<Phase, ArtifactSet> armRuns = observations.get(arm);
            if (armRuns == null || armRuns.size() != Phase.values().length) {
                return Result.inconclusive("MISSING_RUN");
            }
            for (Phase phase : Phase.values()) {
                if (!armRuns.containsKey(phase)) {
                    return Result.inconclusive("MISSING_RUN");
                }
            }
        }

        ArtifactSet controlClean = observations.get(Arm.CONTROL).get(Phase.CLEAN);
        for (Arm arm : Arm.values()) {
            for (Phase phase : Phase.values()) {
                ArtifactSet artifacts = observations.get(arm).get(phase);
                if (!controlClean.logicalSetDigest().equals(
                        artifacts.logicalSetDigest())) {
                    return Result.failed("ARTIFACT_DIVERGENCE");
                }
            }
        }

        ArtifactSet candidateClean = observations.get(Arm.CANDIDATE).get(Phase.CLEAN);
        for (Phase phase : Phase.values()) {
            if (!candidateClean.rawSetDigest().equals(
                    observations.get(Arm.CANDIDATE).get(phase).rawSetDigest())) {
                return Result.failed("CANDIDATE_NOT_REPRODUCIBLE");
            }
        }

        return Result.passed(
                candidateClean.rawSetDigest(),
                controlClean.rawSetDigest(),
                controlClean.logicalSetDigest());
    }

    private static boolean validRequest(Request request) {
        if (request.requiredArtifactPaths() == null
                || request.runs() == null
                || !supportedRecipe(request)
                || request.requiredArtifactPaths().isEmpty()
                || request.requiredArtifactPaths().size() > MAXIMUM_ARTIFACTS
                || request.runs().isEmpty()
                || request.runs().size() > Arm.values().length * Phase.values().length) {
            return false;
        }
        Set<String> paths = new HashSet<>();
        for (String path : request.requiredArtifactPaths()) {
            if (!safeRelativePath(path) || !paths.add(path)) {
                return false;
            }
        }
        return true;
    }

    private static boolean supportedRecipe(Request request) {
        return (ArchiveReproducibilityRecipe.RECIPE_ID.equals(request.recipeId())
                        && ArchiveReproducibilityRecipe.RECIPE_VERSION.equals(
                                request.recipeVersion())
                        && request.artifactAdapter()
                                == ArtifactAdapter.ARCHIVE_CONTENTS_V1)
                || ("CUSTOM_TASK_CONTRACT_JAVA_V1".equals(request.recipeId())
                        && "1.0".equals(request.recipeVersion())
                        && request.artifactAdapter() == ArtifactAdapter.EXACT_BYTES);
    }

    private static boolean validRun(Run run) {
        return run.arm() != null
                && run.phase() != null
                && run.context() != null
                && validContext(run.context())
                && run.isolationKey() != null
                && IDENTIFIER.matcher(run.isolationKey()).matches()
                && run.configurationCacheState() != null
                && run.exitCode() >= 0
                && run.exitCode() <= 255
                && run.artifacts() != null;
    }

    private static boolean validContext(Context context) {
        return identifier(context.repositoryId())
                && identifier(context.actionId())
                && context.revision() != null
                && GIT_OBJECT.matcher(context.revision()).matches()
                && digest(context.sourceStateDigest())
                && digest(context.workUnitsFingerprint())
                && digest(context.requiredDeliverablesManifestDigest())
                && digest(context.policyDigest())
                && identifier(context.toolchainId())
                && identifier(context.runnerClass());
    }

    private static String expectedCacheState(Phase phase) {
        return phase == Phase.INCREMENTAL ? "REUSED" : "STORED";
    }

    private static ArtifactSet artifactSet(
            List<Artifact> artifacts,
            List<String> requiredPaths,
            ArtifactAdapter adapter)
            throws IOException, GeneralSecurityException {
        if (artifacts.size() != requiredPaths.size()
                || artifacts.isEmpty()
                || artifacts.size() > MAXIMUM_ARTIFACTS) {
            throw new IllegalArgumentException("artifact set does not match manifest");
        }

        Set<String> required = Set.copyOf(requiredPaths);
        Map<String, String> raw = new TreeMap<>();
        Map<String, String> logical = new TreeMap<>();
        long totalBytes = 0;
        for (Artifact artifact : artifacts) {
            if (artifact == null
                    || !required.contains(artifact.relativePath())
                    || !safeRelativePath(artifact.relativePath())
                    || raw.containsKey(artifact.relativePath())) {
                throw new IllegalArgumentException("unexpected artifact");
            }
            byte[] content = artifact.content();
            if (content.length == 0 || content.length > MAXIMUM_ARTIFACT_BYTES) {
                throw new IllegalArgumentException("artifact size is outside bounds");
            }
            totalBytes += content.length;
            if (totalBytes > MAXIMUM_TOTAL_BYTES) {
                throw new IllegalArgumentException("artifact set is outside bounds");
            }
            String rawDigest = PatchBundleVerifier.digestBytes(content);
            raw.put(artifact.relativePath(), rawDigest + ":" + content.length);
            logical.put(
                    artifact.relativePath(),
                    logicalDigest(artifact.relativePath(), content, adapter));
        }
        if (!raw.keySet().equals(required)) {
            throw new IllegalArgumentException("required artifact is missing");
        }
        return new ArtifactSet(digestMap(raw), digestMap(logical));
    }

    private static String logicalDigest(
            String path,
            byte[] content,
            ArtifactAdapter adapter)
            throws IOException, GeneralSecurityException {
        if (adapter == ArtifactAdapter.EXACT_BYTES || !isArchive(path)) {
            return PatchBundleVerifier.digestBytes(content);
        }
        return archiveContentsDigest(content);
    }

    private static String archiveContentsDigest(byte[] content)
            throws IOException, GeneralSecurityException {
        Map<String, String> entries = new TreeMap<>();
        long expandedBytes = 0;
        byte[] buffer = new byte[8192];
        try (ZipInputStream archive =
                new ZipInputStream(new ByteArrayInputStream(content))) {
            for (ZipEntry entry = archive.getNextEntry();
                    entry != null;
                    entry = archive.getNextEntry()) {
                String name = entry.getName();
                if (!safeArchiveEntry(name) || entries.containsKey(name)) {
                    throw new IllegalArgumentException("unsafe or duplicate archive entry");
                }
                if (entry.isDirectory()) {
                    entries.put(name, "DIRECTORY");
                } else {
                    MessageDigest digest = MessageDigest.getInstance("SHA-256");
                    long size = 0;
                    for (int count = archive.read(buffer);
                            count >= 0;
                            count = archive.read(buffer)) {
                        if (count == 0) {
                            continue;
                        }
                        size += count;
                        expandedBytes += count;
                        if (size > MAXIMUM_ARTIFACT_BYTES
                                || expandedBytes > MAXIMUM_EXPANDED_ARCHIVE_BYTES) {
                            throw new IllegalArgumentException(
                                    "expanded archive is outside bounds");
                        }
                        digest.update(buffer, 0, count);
                    }
                    entries.put(
                            name,
                            "sha256:" + HexFormat.of().formatHex(digest.digest())
                                    + ":" + size);
                }
                archive.closeEntry();
            }
        }
        if (entries.isEmpty()) {
            throw new IllegalArgumentException("archive has no entries");
        }
        return digestMap(entries);
    }

    private static String digestMap(Map<String, String> values)
            throws GeneralSecurityException {
        StringBuilder canonical = new StringBuilder();
        for (Map.Entry<String, String> entry : values.entrySet()) {
            appendField(canonical, entry.getKey());
            appendField(canonical, entry.getValue());
        }
        return PatchBundleVerifier.digestBytes(
                canonical.toString().getBytes(java.nio.charset.StandardCharsets.UTF_8));
    }

    private static void appendField(StringBuilder target, String value) {
        target.append(value.length()).append(':').append(value);
    }

    private static boolean isArchive(String path) {
        String lower = path.toLowerCase(java.util.Locale.ROOT);
        return lower.endsWith(".zip")
                || lower.endsWith(".jar")
                || lower.endsWith(".war")
                || lower.endsWith(".ear")
                || lower.endsWith(".aar");
    }

    private static boolean safeRelativePath(String path) {
        if (path == null
                || path.isEmpty()
                || path.length() > 1024
                || path.startsWith("/")
                || path.endsWith("/")
                || path.indexOf('\\') >= 0
                || path.indexOf('\u0000') >= 0) {
            return false;
        }
        return safeSegments(path, false);
    }

    private static boolean safeArchiveEntry(String path) {
        if (path == null
                || path.isEmpty()
                || path.length() > 4096
                || path.startsWith("/")
                || path.indexOf('\\') >= 0
                || path.indexOf('\u0000') >= 0) {
            return false;
        }
        return safeSegments(path, path.endsWith("/"));
    }

    private static boolean safeSegments(String path, boolean directory) {
        String candidate = directory ? path.substring(0, path.length() - 1) : path;
        if (candidate.isEmpty()) {
            return false;
        }
        for (String segment : candidate.split("/", -1)) {
            if (segment.isEmpty() || ".".equals(segment) || "..".equals(segment)) {
                return false;
            }
        }
        return true;
    }

    private static boolean identifier(String value) {
        return value != null && IDENTIFIER.matcher(value).matches();
    }

    private static boolean digest(String value) {
        return value != null && SHA256.matcher(value).matches();
    }

    /** Immutable context which must be identical across all six runs. */
    public record Context(
            String repositoryId,
            String actionId,
            String revision,
            String sourceStateDigest,
            String workUnitsFingerprint,
            String requiredDeliverablesManifestDigest,
            String policyDigest,
            String toolchainId,
            String runnerClass) {
    }

    /** Immutable defensive artifact observation. */
    public static final class Artifact {
        private final String relativePath;
        private final byte[] content;

        public Artifact(String relativePath, byte[] content) {
            this.relativePath = Objects.requireNonNull(relativePath, "relativePath");
            this.content = Objects.requireNonNull(content, "content").clone();
        }

        public String relativePath() {
            return relativePath;
        }

        public byte[] content() {
            return content.clone();
        }
    }

    /** One isolated arm/phase observation. */
    public static final class Run {
        private final Arm arm;
        private final Phase phase;
        private final Context context;
        private final String isolationKey;
        private final String configurationCacheState;
        private final int exitCode;
        private final List<Artifact> artifacts;

        public Run(
                Arm arm,
                Phase phase,
                Context context,
                String isolationKey,
                String configurationCacheState,
                int exitCode,
                List<Artifact> artifacts) {
            this.arm = arm;
            this.phase = phase;
            this.context = context;
            this.isolationKey = isolationKey;
            this.configurationCacheState = configurationCacheState;
            this.exitCode = exitCode;
            this.artifacts = artifacts == null ? null : List.copyOf(artifacts);
        }

        public Arm arm() {
            return arm;
        }

        public Phase phase() {
            return phase;
        }

        public Context context() {
            return context;
        }

        public String isolationKey() {
            return isolationKey;
        }

        public String configurationCacheState() {
            return configurationCacheState;
        }

        public int exitCode() {
            return exitCode;
        }

        public List<Artifact> artifacts() {
            return artifacts;
        }
    }

    /** Complete validation request for one recipe generation. */
    public static final class Request {
        private final String recipeId;
        private final String recipeVersion;
        private final ArtifactAdapter artifactAdapter;
        private final List<String> requiredArtifactPaths;
        private final List<Run> runs;

        public Request(
                String recipeId,
                String recipeVersion,
                ArtifactAdapter artifactAdapter,
                List<String> requiredArtifactPaths,
                List<Run> runs) {
            this.recipeId = recipeId;
            this.recipeVersion = recipeVersion;
            this.artifactAdapter = artifactAdapter;
            this.requiredArtifactPaths =
                    requiredArtifactPaths == null ? null : List.copyOf(requiredArtifactPaths);
            this.runs = runs == null ? null : List.copyOf(runs);
        }

        public String recipeId() {
            return recipeId;
        }

        public String recipeVersion() {
            return recipeVersion;
        }

        public ArtifactAdapter artifactAdapter() {
            return artifactAdapter;
        }

        public List<String> requiredArtifactPaths() {
            return requiredArtifactPaths;
        }

        public List<Run> runs() {
            return runs;
        }
    }

    /** Immutable terminal validation result. */
    public static final class Result {
        private final Status status;
        private final String reason;
        private final String candidateArtifactSetDigest;
        private final String controlArtifactSetDigest;
        private final String logicalArtifactSetDigest;

        private Result(
                Status status,
                String reason,
                String candidateArtifactSetDigest,
                String controlArtifactSetDigest,
                String logicalArtifactSetDigest) {
            this.status = status;
            this.reason = reason;
            this.candidateArtifactSetDigest = candidateArtifactSetDigest;
            this.controlArtifactSetDigest = controlArtifactSetDigest;
            this.logicalArtifactSetDigest = logicalArtifactSetDigest;
        }

        private static Result passed(
                String candidateDigest,
                String controlDigest,
                String logicalDigest) {
            return new Result(
                    Status.PASSED,
                    "PASSED",
                    candidateDigest,
                    controlDigest,
                    logicalDigest);
        }

        private static Result failed(String reason) {
            return new Result(Status.FAILED, reason, null, null, null);
        }

        private static Result inconclusive(String reason) {
            return new Result(Status.INCONCLUSIVE, reason, null, null, null);
        }

        public Status status() {
            return status;
        }

        public String reason() {
            return reason;
        }

        public String candidateArtifactSetDigest() {
            return candidateArtifactSetDigest;
        }

        public String controlArtifactSetDigest() {
            return controlArtifactSetDigest;
        }

        public String logicalArtifactSetDigest() {
            return logicalArtifactSetDigest;
        }
    }

    private record ArtifactSet(String rawSetDigest, String logicalSetDigest) {
    }
}
