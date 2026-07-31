package dev.buildopt.patcher;

import java.io.IOException;
import java.net.URI;
import java.security.GeneralSecurityException;
import java.security.PublicKey;
import java.security.Signature;
import java.time.Duration;
import java.time.Instant;
import java.time.format.DateTimeParseException;
import java.util.ArrayList;
import java.util.Base64;
import java.util.HashSet;
import java.util.LinkedHashMap;
import java.util.List;
import java.util.Locale;
import java.util.Map;
import java.util.Objects;
import java.util.Set;
import java.util.TreeMap;
import java.util.regex.Pattern;

import dev.buildopt.generated.BuildOptClientsV1;
import dev.buildopt.generated.BuildOptClientsV1.Endpoint;
import dev.buildopt.generated.BuildOptClientsV1.Response;

/**
 * Composes local patch validation with Test Optimization full relevant validation.
 */
public final class FullRelevantValidationGate {
    private static final int MAXIMUM_RESPONSE_BYTES = 2 * 1024 * 1024;
    private static final int MAXIMUM_POLLS = 128;
    private static final int MAXIMUM_POLL_DELAY_MS = 5000;
    private static final Pattern IDENTIFIER = Pattern.compile(
            "^[A-Za-z0-9][A-Za-z0-9._:/-]{0,255}$");
    private static final Pattern SHA256 = Pattern.compile("^sha256:[0-9a-f]{64}$");
    private static final Pattern DIGEST = Pattern.compile(
            "^(?:sha256|hmac-sha256):[0-9a-f]{64}$");
    private static final Pattern GIT_OBJECT = Pattern.compile("^[0-9a-f]{7,64}$");
    private static final Pattern MEDIA_TYPE = Pattern.compile(
            "^[a-z0-9][a-z0-9!#$&^_.+-]+/[a-z0-9][a-z0-9!#$&^_.+-]+$");

    private FullRelevantValidationGate() {
    }

    /** Whether policy requires Test Optimization for this patch. */
    public enum Applicability {
        REQUIRED,
        NOT_REQUIRED
    }

    /** Terminal promotion-gate state. */
    public enum Status {
        PASSED,
        FAILED,
        INCONCLUSIVE
    }

    /** Clock used to preserve one original deadline across submission and polling. */
    @FunctionalInterface
    public interface TimeSource {
        Instant now();
    }

    /** Bounded waiting boundary, replaceable by a deterministic test clock. */
    @FunctionalInterface
    public interface Waiter {
        void await(int milliseconds) throws InterruptedException;
    }

    /** Transport boundary matching the generated single-attempt client. */
    @FunctionalInterface
    public interface Transport {
        Response send(
                Endpoint endpoint,
                Map<String, String> pathValues,
                String requestId,
                String idempotencyKey,
                String ifMatch,
                Instant deadline,
                byte[] body) throws IOException, InterruptedException;
    }

    /**
     * Adapts the generated HTTPS client without changing its retry semantics.
     *
     * @param client configured generated client
     * @return transport accepted by this gate
     */
    public static Transport generatedTransport(BuildOptClientsV1.Client client) {
        Objects.requireNonNull(client, "client");
        return client::send;
    }


    /**
     * Resolves a policy-proven NOT_REQUIRED decision without a remote client.
     *
     * @param localValidation completed C4-005 result
     * @param request immutable NOT_REQUIRED request
     * @param now authoritative current time
     * @return PASSED only for a valid explicit NOT_REQUIRED request
     */
    public static Result evaluateNotRequired(
            PatchCandidateValidator.Result localValidation,
            Request request,
            Instant now) {
        return evaluate(
                localValidation,
                request,
                null,
                null,
                null,
                () -> now,
                ignored -> {
                });
    }

    /**
     * Runs the production gate with the system UTC clock and bounded sleeps.
     *
     * @param localValidation completed C4-005 result
     * @param request immutable Test Optimization request
     * @param expectedKeyId pinned Test Optimization signing key ID
     * @param trustedKey pinned Ed25519 public key
     * @param client configured generated HTTPS client
     * @return terminal gate result
     */
    public static Result evaluate(
            PatchCandidateValidator.Result localValidation,
            Request request,
            String expectedKeyId,
            PublicKey trustedKey,
            BuildOptClientsV1.Client client) {
        return evaluate(
                localValidation,
                request,
                expectedKeyId,
                trustedKey,
                generatedTransport(client),
                Instant::now,
                Thread::sleep);
    }

    static Result evaluate(
            PatchCandidateValidator.Result localValidation,
            Request request,
            String expectedKeyId,
            PublicKey trustedKey,
            Transport transport,
            TimeSource timeSource,
            Waiter waiter) {
        if (localValidation == null
                || localValidation.status() != PatchCandidateValidator.Status.PASSED) {
            return Result.failed("LOCAL_VALIDATION_NOT_PASSED");
        }
        if (request == null || timeSource == null || waiter == null) {
            return Result.inconclusive("INVALID_GATE_INPUT");
        }

        Instant now = timeSource.now();
        if (!validRequest(request, now)) {
            return Result.inconclusive("INVALID_REQUEST");
        }
        if (request.applicability() == Applicability.NOT_REQUIRED) {
            if (!request.artifacts().isEmpty()) {
                return Result.inconclusive("NOT_REQUIRED_HAS_ARTIFACTS");
            }
            return Result.passed("NOT_REQUIRED", null, null, null);
        }
        if (transport == null
                || !validIdentifier(expectedKeyId)
                || trustedKey == null
                || !"EdDSA".equals(trustedKey.getAlgorithm())) {
            return Result.inconclusive("INVALID_GATE_INPUT");
        }

        String artifactSetDigest;
        try {
            artifactSetDigest = artifactSetDigest(request.artifacts());
        } catch (GeneralSecurityException | IllegalArgumentException exception) {
            return Result.inconclusive("INVALID_ARTIFACT_SET");
        }
        if (!artifactSetDigest.equals(localValidation.candidateArtifactSetDigest())) {
            return Result.failed("LOCAL_ARTIFACT_SET_MISMATCH");
        }
        String expectedIdempotencyKey =
                request.actionId() + ":" + artifactSetDigest;
        if (!expectedIdempotencyKey.equals(request.idempotencyKey())
                || expectedIdempotencyKey.length() > 512) {
            return Result.inconclusive("INVALID_IDEMPOTENCY_KEY");
        }

        byte[] body;
        try {
            body = requestBody(request);
        } catch (IllegalArgumentException exception) {
            return Result.inconclusive("INVALID_REQUEST");
        }

        try {
            Response response = sendWithOneExactRetry(
                    transport,
                    BuildOptClientsV1.SUBMIT_TEST_BUILD_VALIDATION,
                    Map.of(),
                    request,
                    request.idempotencyKey(),
                    body,
                    timeSource);
            if (response == null) {
                return Result.inconclusive("TRANSPORT_FAILURE");
            }
            String operationId = null;
            for (int poll = 0; response.statusCode() == 202; poll++) {
                if (poll >= MAXIMUM_POLLS) {
                    return Result.inconclusive("POLL_LIMIT");
                }
                if (!matchingRequestHeader(response, request.requestId())) {
                    return Result.inconclusive("RESPONSE_ID_MISMATCH");
                }
                Receipt receipt = receipt(response.body(), request, operationId, timeSource.now());
                if (receipt == null) {
                    return Result.inconclusive("INVALID_RECEIPT");
                }
                operationId = receipt.operationId();
                int delay = pollDelay(request.requestId(), poll, receipt.pollAfterMs());
                Instant beforeWait = timeSource.now();
                if (!beforeWait.plusMillis(delay).isBefore(request.deadline())) {
                    return Result.inconclusive("DEADLINE_EXCEEDED");
                }
                waiter.await(delay);
                if (!timeSource.now().isBefore(request.deadline())) {
                    return Result.inconclusive("DEADLINE_EXCEEDED");
                }
                response = sendWithOneExactRetry(
                        transport,
                        BuildOptClientsV1.GET_TEST_BUILD_VALIDATION,
                        Map.of("operationId", operationId),
                        request,
                        "",
                        new byte[0],
                        timeSource);
                if (response == null) {
                    return Result.inconclusive("TRANSPORT_FAILURE");
                }
            }
            if (response.statusCode() != 200
                    || !matchingRequestHeader(response, request.requestId())) {
                return Result.inconclusive("TRANSPORT_REJECTED");
            }
            return verifyResult(
                    response.body(),
                    request,
                    artifactSetDigest,
                    expectedKeyId,
                    trustedKey,
                    timeSource.now());
        } catch (IOException exception) {
            return Result.inconclusive("TRANSPORT_FAILURE");
        } catch (InterruptedException exception) {
            Thread.currentThread().interrupt();
            return Result.inconclusive("INTERRUPTED");
        }
    }

    private static Response sendWithOneExactRetry(
            Transport transport,
            Endpoint endpoint,
            Map<String, String> pathValues,
            Request request,
            String idempotencyKey,
            byte[] body,
            TimeSource timeSource)
            throws IOException, InterruptedException {
        try {
            return transport.send(
                    endpoint,
                    pathValues,
                    request.requestId(),
                    idempotencyKey,
                    "",
                    request.deadline(),
                    body.clone());
        } catch (IOException firstFailure) {
            if (!timeSource.now().isBefore(request.deadline())) {
                throw firstFailure;
            }
            return transport.send(
                    endpoint,
                    pathValues,
                    request.requestId(),
                    idempotencyKey,
                    "",
                    request.deadline(),
                    body.clone());
        }
    }

    private static boolean validRequest(Request request, Instant now) {
        if (request.applicability() == null
                || request.idempotencyKey() == null
                || !validDigest(request.applicabilityEvidenceDigest())
                || !validIdentifier(request.requestId())
                || !validIdentifier(request.tenant())
                || !validIdentifier(request.repository())
                || !validIdentifier(request.trustDomain())
                || !validIdentifier(request.actionId())
                || request.revision() == null
                || !GIT_OBJECT.matcher(request.revision()).matches()
                || !validAnyDigest(request.sourceStateDigest())
                || !validDigest(request.policyDigest())
                || request.deadline() == null
                || !request.deadline().isAfter(now)
                || Duration.between(now, request.deadline()).compareTo(Duration.ofHours(24)) > 0
                || request.artifacts() == null) {
            return false;
        }
        if (request.applicability() == Applicability.REQUIRED) {
            return !request.artifacts().isEmpty() && request.artifacts().size() <= 1024;
        }
        return request.idempotencyKey().isEmpty();
    }

    private static byte[] requestBody(Request request) {
        Map<String, Object> repository = new LinkedHashMap<>();
        repository.put("tenant", request.tenant());
        repository.put("repository", request.repository());
        repository.put("trustDomain", request.trustDomain());

        Map<String, Object> context = new LinkedHashMap<>();
        context.put("requestId", request.requestId());
        context.put("repository", repository);
        context.put("revision", request.revision());
        context.put("sourceStateDigest", request.sourceStateDigest());
        context.put("deadline", request.deadline().toString());

        List<Object> artifacts = new ArrayList<>();
        Set<String> identifiers = new HashSet<>();
        for (ArtifactRef artifact : request.artifacts()) {
            if (!validArtifact(artifact) || !identifiers.add(artifact.artifactId())) {
                throw new IllegalArgumentException("invalid or duplicate artifact");
            }
            Map<String, Object> retrieval = new LinkedHashMap<>();
            retrieval.put("kind", artifact.retrievalKind());
            retrieval.put("locator", artifact.locator());

            Map<String, Object> item = new LinkedHashMap<>();
            item.put("artifactId", artifact.artifactId());
            item.put("digest", artifact.digest());
            item.put("sizeBytes", artifact.sizeBytes());
            item.put("mediaType", artifact.mediaType());
            item.put("retrieval", retrieval);
            artifacts.add(item);
        }

        Map<String, Object> root = new LinkedHashMap<>();
        root.put("contractVersion", "test-optimization/v1");
        root.put("context", context);
        root.put("actionId", request.actionId());
        root.put("validationMode", "FULL_RELEVANT_VALIDATION");
        root.put("policyDigest", request.policyDigest());
        root.put("candidateArtifacts", artifacts);
        return StrictJson.canonicalBytes(root);
    }

    private static Receipt receipt(
            byte[] body,
            Request request,
            String existingOperationId,
            Instant now) {
        if (body == null || body.length == 0 || body.length > MAXIMUM_RESPONSE_BYTES) {
            return null;
        }
        try {
            Map<String, Object> root = object(StrictJson.parse(body));
            if (!root.keySet().equals(Set.of(
                    "contractVersion",
                    "requestId",
                    "actionId",
                    "operationId",
                    "status",
                    "pollAfterMs",
                    "expiresAt"))
                    || !"test-optimization/v1".equals(root.get("contractVersion"))
                    || !request.requestId().equals(root.get("requestId"))
                    || !request.actionId().equals(root.get("actionId"))
                    || !"PENDING".equals(root.get("status"))) {
                return null;
            }
            String operationId = string(root, "operationId");
            long pollAfter = integer(root, "pollAfterMs", 0, MAXIMUM_POLL_DELAY_MS);
            Instant expiresAt = timestamp(root, "expiresAt");
            if (!validIdentifier(operationId)
                    || (existingOperationId != null
                            && !existingOperationId.equals(operationId))
                    || !expiresAt.isAfter(now)
                    || expiresAt.isAfter(request.deadline())) {
                return null;
            }
            return new Receipt(operationId, (int) pollAfter);
        } catch (IllegalArgumentException exception) {
            return null;
        }
    }

    private static int pollDelay(String requestId, int poll, int requestedMs) {
        int jitter = Math.floorMod(Objects.hash(requestId, poll), 251);
        return Math.min(MAXIMUM_POLL_DELAY_MS, requestedMs + jitter);
    }

    private static Result verifyResult(
            byte[] body,
            Request request,
            String expectedArtifactSetDigest,
            String expectedKeyId,
            PublicKey trustedKey,
            Instant now) {
        if (body == null || body.length == 0 || body.length > MAXIMUM_RESPONSE_BYTES) {
            return Result.inconclusive("INVALID_RESULT");
        }
        try {
            Map<String, Object> root = object(StrictJson.parse(body));
            String status = string(root, "status");
            Set<String> fields = new HashSet<>(Set.of(
                    "schemaVersion",
                    "recordType",
                    "contractVersion",
                    "resultId",
                    "requestId",
                    "actionId",
                    "repository",
                    "revision",
                    "sourceStateDigest",
                    "validationMode",
                    "status",
                    "validatedArtifactRefs",
                    "artifactSetDigest",
                    "policyDigest",
                    "evidenceRef",
                    "completedAt",
                    "expiresAt",
                    "signature"));
            if ("FAILED".equals(status)) {
                fields.add("failure");
            } else if ("INCONCLUSIVE".equals(status)) {
                fields.add("inconclusive");
            } else if (!"PASSED".equals(status)) {
                return Result.inconclusive("INVALID_RESULT");
            }
            if (!root.keySet().equals(fields)
                    || !"1.0".equals(root.get("schemaVersion"))
                    || !"TEST_VALIDATION_RESULT".equals(root.get("recordType"))
                    || !"test-optimization/v1".equals(root.get("contractVersion"))
                    || !"FULL_RELEVANT_VALIDATION".equals(root.get("validationMode"))
                    || !request.requestId().equals(root.get("requestId"))
                    || !request.actionId().equals(root.get("actionId"))
                    || !request.revision().equals(root.get("revision"))
                    || !request.sourceStateDigest().equals(root.get("sourceStateDigest"))
                    || !request.policyDigest().equals(root.get("policyDigest"))) {
                return Result.inconclusive("RESULT_CONTEXT_MISMATCH");
            }

            String resultId = string(root, "resultId");
            String evidenceRef = string(root, "evidenceRef");
            if (!validIdentifier(resultId) || !validIdentifier(evidenceRef)) {
                return Result.inconclusive("INVALID_RESULT");
            }
            Map<String, Object> repository = object(root.get("repository"));
            if (!repository.keySet().equals(Set.of("tenant", "repository", "trustDomain"))
                    || !request.tenant().equals(repository.get("tenant"))
                    || !request.repository().equals(repository.get("repository"))
                    || !request.trustDomain().equals(repository.get("trustDomain"))) {
                return Result.inconclusive("RESULT_CONTEXT_MISMATCH");
            }

            Instant completedAt = timestamp(root, "completedAt");
            Instant expiresAt = timestamp(root, "expiresAt");
            if (completedAt.isAfter(now)
                    || completedAt.isAfter(request.deadline())
                    || !expiresAt.isAfter(now)
                    || !completedAt.isBefore(expiresAt)) {
                return Result.inconclusive("RESULT_EXPIRED");
            }
            if (!verifySignature(root, expectedKeyId, trustedKey)) {
                return Result.inconclusive("UNTRUSTED_RESULT");
            }

            String artifactSetDigest = string(root, "artifactSetDigest");
            if (!expectedArtifactSetDigest.equals(artifactSetDigest)
                    || !matchingArtifacts(
                            array(root.get("validatedArtifactRefs")),
                            request.artifacts())) {
                return Result.failed("RESULT_ARTIFACT_MISMATCH");
            }

            if ("FAILED".equals(status)) {
                if (!validFailure(object(root.get("failure")))) {
                    return Result.inconclusive("INVALID_RESULT");
                }
                return Result.failed(
                        "TEST_VALIDATION_FAILED",
                        resultId,
                        artifactSetDigest,
                        evidenceRef);
            }
            if ("INCONCLUSIVE".equals(status)) {
                if (!validInconclusive(object(root.get("inconclusive")))) {
                    return Result.inconclusive("INVALID_RESULT");
                }
                return Result.inconclusive(
                        "TEST_VALIDATION_INCONCLUSIVE",
                        resultId,
                        artifactSetDigest,
                        evidenceRef);
            }
            return Result.passed("FULL_RELEVANT_VALIDATION", resultId, artifactSetDigest, evidenceRef);
        } catch (IllegalArgumentException | GeneralSecurityException exception) {
            return Result.inconclusive("INVALID_RESULT");
        }
    }

    private static boolean matchingArtifacts(
            List<Object> rawArtifacts,
            List<ArtifactRef> expectedArtifacts) {
        if (rawArtifacts.size() != expectedArtifacts.size()) {
            return false;
        }
        Map<String, ArtifactRef> expected = new TreeMap<>();
        for (ArtifactRef artifact : expectedArtifacts) {
            if (expected.put(artifact.artifactId(), artifact) != null) {
                return false;
            }
        }
        Set<String> seen = new HashSet<>();
        for (Object raw : rawArtifacts) {
            Map<String, Object> artifact = object(raw);
            if (!artifact.keySet().equals(
                    Set.of("artifactId", "digest", "sizeBytes", "mediaType"))) {
                return false;
            }
            String artifactId = string(artifact, "artifactId");
            ArtifactRef wanted = expected.get(artifactId);
            if (wanted == null
                    || !seen.add(artifactId)
                    || !wanted.digest().equals(artifact.get("digest"))
                    || wanted.sizeBytes()
                            != integer(artifact, "sizeBytes", 0, 107374182400L)
                    || !wanted.mediaType().equals(artifact.get("mediaType"))) {
                return false;
            }
        }
        return seen.size() == expected.size();
    }

    private static boolean verifySignature(
            Map<String, Object> root,
            String expectedKeyId,
            PublicKey trustedKey)
            throws GeneralSecurityException {
        Map<String, Object> signature = object(root.get("signature"));
        if (!signature.keySet().equals(Set.of(
                "algorithm",
                "canonicalization",
                "keyId",
                "signedPayloadDigest",
                "value"))
                || !"Ed25519".equals(signature.get("algorithm"))
                || !"JCS".equals(signature.get("canonicalization"))
                || !expectedKeyId.equals(signature.get("keyId"))) {
            return false;
        }
        Map<String, Object> payload = new LinkedHashMap<>(root);
        payload.remove("signature");
        String calculatedDigest =
                PatchBundleVerifier.digestBytes(StrictJson.canonicalBytes(payload));
        String signedDigest = string(signature, "signedPayloadDigest");
        if (!calculatedDigest.equals(signedDigest)) {
            return false;
        }
        byte[] value;
        try {
            value = Base64.getUrlDecoder().decode(string(signature, "value") + "==");
        } catch (IllegalArgumentException exception) {
            return false;
        }
        if (value.length != 64) {
            return false;
        }
        Signature verifier = Signature.getInstance("Ed25519");
        verifier.initVerify(trustedKey);
        verifier.update(signatureInput(calculatedDigest, expectedKeyId));
        return verifier.verify(value);
    }

    static byte[] signatureInput(String payloadDigest, String keyId) {
        return ("test-optimization/v1\n"
                        + "TEST_VALIDATION_RESULT\n"
                        + payloadDigest
                        + "\n"
                        + keyId)
                .getBytes(java.nio.charset.StandardCharsets.UTF_8);
    }

    private static boolean validFailure(Map<String, Object> failure) {
        return failure.keySet().equals(Set.of("reason", "summary"))
                && Set.of(
                        "TEST_FAILURE",
                        "ARTIFACT_DIVERGENCE",
                        "CORRECTNESS_GUARDRAIL")
                        .contains(failure.get("reason"))
                && failure.get("summary") instanceof String summary
                && !summary.isEmpty()
                && summary.length() <= 1024;
    }

    private static boolean validInconclusive(Map<String, Object> inconclusive) {
        return inconclusive.keySet().equals(Set.of("reason"))
                && Set.of(
                        "TIMEOUT",
                        "INCOMPATIBLE_VERSION",
                        "MISSING_ARTIFACT",
                        "CORRUPT_ARTIFACT",
                        "INFRA_FAILURE",
                        "INCOMPLETE_RESULT")
                        .contains(inconclusive.get("reason"));
    }

    private static boolean matchingRequestHeader(Response response, String requestId) {
        if (response == null || response.headers() == null) {
            return false;
        }
        for (Map.Entry<String, List<String>> header : response.headers().entrySet()) {
            if ("x-buildopt-request-id".equals(
                            header.getKey().toLowerCase(Locale.ROOT))
                    && header.getValue().size() == 1
                    && requestId.equals(header.getValue().get(0))) {
                return true;
            }
        }
        return false;
    }

    private static String artifactSetDigest(List<ArtifactRef> artifacts)
            throws GeneralSecurityException {
        Map<String, String> values = new TreeMap<>();
        for (ArtifactRef artifact : artifacts) {
            if (!validArtifact(artifact)
                    || values.put(
                            artifact.artifactId(),
                            artifact.digest() + ":" + artifact.sizeBytes()) != null) {
                throw new IllegalArgumentException("invalid or duplicate artifact");
            }
        }
        StringBuilder canonical = new StringBuilder();
        for (Map.Entry<String, String> entry : values.entrySet()) {
            appendField(canonical, entry.getKey());
            appendField(canonical, entry.getValue());
        }
        return PatchBundleVerifier.digestBytes(
                canonical.toString().getBytes(java.nio.charset.StandardCharsets.UTF_8));
    }

    private static boolean validArtifact(ArtifactRef artifact) {
        if (artifact == null
                || !validIdentifier(artifact.artifactId())
                || !validDigest(artifact.digest())
                || artifact.sizeBytes() < 1
                || artifact.sizeBytes() > 107374182400L
                || artifact.mediaType() == null
                || !MEDIA_TYPE.matcher(artifact.mediaType()).matches()
                || artifact.locator() == null
                || artifact.locator().isEmpty()
                || artifact.locator().length() > 2048) {
            return false;
        }
        if ("CUSTOMER_CHANNEL".equals(artifact.retrievalKind())) {
            return validIdentifier(artifact.locator());
        }
        if (!"EPHEMERAL_HTTPS".equals(artifact.retrievalKind())) {
            return false;
        }
        try {
            URI uri = URI.create(artifact.locator());
            return "https".equalsIgnoreCase(uri.getScheme())
                    && uri.getHost() != null
                    && uri.getUserInfo() == null
                    && uri.getFragment() == null;
        } catch (IllegalArgumentException exception) {
            return false;
        }
    }

    private static Map<String, Object> object(Object value) {
        if (!(value instanceof Map<?, ?> raw)) {
            throw new IllegalArgumentException("expected object");
        }
        Map<String, Object> result = new LinkedHashMap<>();
        for (Map.Entry<?, ?> entry : raw.entrySet()) {
            if (!(entry.getKey() instanceof String key)) {
                throw new IllegalArgumentException("object key is not a string");
            }
            result.put(key, entry.getValue());
        }
        return result;
    }

    private static List<Object> array(Object value) {
        if (!(value instanceof List<?> raw)) {
            throw new IllegalArgumentException("expected array");
        }
        return new ArrayList<>(raw);
    }

    private static String string(Map<String, Object> object, String field) {
        Object value = object.get(field);
        if (!(value instanceof String string)) {
            throw new IllegalArgumentException(field + " is not a string");
        }
        return string;
    }

    private static long integer(
            Map<String, Object> object,
            String field,
            long minimum,
            long maximum) {
        Object value = object.get(field);
        String source;
        if (value instanceof StrictJson.JsonNumber number) {
            source = number.source();
        } else if (value instanceof Number number) {
            source = number.toString();
        } else {
            throw new IllegalArgumentException(field + " is not an integer");
        }
        if (!source.matches("-?(?:0|[1-9][0-9]*)")) {
            throw new IllegalArgumentException(field + " is not an integer");
        }
        long parsed = Long.parseLong(source);
        if (parsed < minimum || parsed > maximum) {
            throw new IllegalArgumentException(field + " is outside bounds");
        }
        return parsed;
    }

    private static Instant timestamp(Map<String, Object> object, String field) {
        try {
            return Instant.parse(string(object, field));
        } catch (DateTimeParseException exception) {
            throw new IllegalArgumentException(field + " is not a UTC timestamp", exception);
        }
    }

    private static void appendField(StringBuilder target, String value) {
        target.append(value.length()).append(':').append(value);
    }

    private static boolean validIdentifier(String value) {
        return value != null && IDENTIFIER.matcher(value).matches();
    }

    private static boolean validDigest(String value) {
        return value != null && SHA256.matcher(value).matches();
    }

    private static boolean validAnyDigest(String value) {
        return value != null && DIGEST.matcher(value).matches();
    }

    /** Immutable content-addressed candidate artifact reference. */
    public record ArtifactRef(
            String artifactId,
            String digest,
            long sizeBytes,
            String mediaType,
            String retrievalKind,
            String locator) {
    }

    /** Immutable full relevant validation request. */
    public static final class Request {
        private final Applicability applicability;
        private final String applicabilityEvidenceDigest;
        private final String requestId;
        private final String idempotencyKey;
        private final String tenant;
        private final String repository;
        private final String trustDomain;
        private final String revision;
        private final String sourceStateDigest;
        private final String actionId;
        private final String policyDigest;
        private final Instant deadline;
        private final List<ArtifactRef> artifacts;

        public Request(
                Applicability applicability,
                String applicabilityEvidenceDigest,
                String requestId,
                String idempotencyKey,
                String tenant,
                String repository,
                String trustDomain,
                String revision,
                String sourceStateDigest,
                String actionId,
                String policyDigest,
                Instant deadline,
                List<ArtifactRef> artifacts) {
            this.applicability = applicability;
            this.applicabilityEvidenceDigest = applicabilityEvidenceDigest;
            this.requestId = requestId;
            this.idempotencyKey = idempotencyKey;
            this.tenant = tenant;
            this.repository = repository;
            this.trustDomain = trustDomain;
            this.revision = revision;
            this.sourceStateDigest = sourceStateDigest;
            this.actionId = actionId;
            this.policyDigest = policyDigest;
            this.deadline = deadline;
            this.artifacts = artifacts == null ? null : List.copyOf(artifacts);
        }

        public Applicability applicability() {
            return applicability;
        }

        public String applicabilityEvidenceDigest() {
            return applicabilityEvidenceDigest;
        }

        public String requestId() {
            return requestId;
        }

        public String idempotencyKey() {
            return idempotencyKey;
        }

        public String tenant() {
            return tenant;
        }

        public String repository() {
            return repository;
        }

        public String trustDomain() {
            return trustDomain;
        }

        public String revision() {
            return revision;
        }

        public String sourceStateDigest() {
            return sourceStateDigest;
        }

        public String actionId() {
            return actionId;
        }

        public String policyDigest() {
            return policyDigest;
        }

        public Instant deadline() {
            return deadline;
        }

        public List<ArtifactRef> artifacts() {
            return artifacts;
        }
    }

    /** Immutable terminal gate result. */
    public static final class Result {
        private final Status status;
        private final String reason;
        private final String resultId;
        private final String artifactSetDigest;
        private final String evidenceRef;

        private Result(
                Status status,
                String reason,
                String resultId,
                String artifactSetDigest,
                String evidenceRef) {
            this.status = status;
            this.reason = reason;
            this.resultId = resultId;
            this.artifactSetDigest = artifactSetDigest;
            this.evidenceRef = evidenceRef;
        }

        private static Result passed(
                String reason,
                String resultId,
                String artifactSetDigest,
                String evidenceRef) {
            return new Result(Status.PASSED, reason, resultId, artifactSetDigest, evidenceRef);
        }

        private static Result failed(String reason) {
            return new Result(Status.FAILED, reason, null, null, null);
        }

        private static Result failed(
                String reason,
                String resultId,
                String artifactSetDigest,
                String evidenceRef) {
            return new Result(Status.FAILED, reason, resultId, artifactSetDigest, evidenceRef);
        }

        private static Result inconclusive(String reason) {
            return new Result(Status.INCONCLUSIVE, reason, null, null, null);
        }

        private static Result inconclusive(
                String reason,
                String resultId,
                String artifactSetDigest,
                String evidenceRef) {
            return new Result(
                    Status.INCONCLUSIVE,
                    reason,
                    resultId,
                    artifactSetDigest,
                    evidenceRef);
        }

        public Status status() {
            return status;
        }

        public String reason() {
            return reason;
        }

        public String resultId() {
            return resultId;
        }

        public String artifactSetDigest() {
            return artifactSetDigest;
        }

        public String evidenceRef() {
            return evidenceRef;
        }
    }

    private record Receipt(String operationId, int pollAfterMs) {
    }
}
