package dev.buildopt.patcher;

import java.io.ByteArrayOutputStream;
import java.io.IOException;
import java.nio.charset.StandardCharsets;
import java.security.KeyPair;
import java.security.KeyPairGenerator;
import java.security.Signature;
import java.time.Instant;
import java.util.ArrayList;
import java.util.Arrays;
import java.util.Base64;
import java.util.LinkedHashMap;
import java.util.List;
import java.util.Map;
import java.util.zip.ZipEntry;
import java.util.zip.ZipOutputStream;

import dev.buildopt.generated.BuildOptClientsV1;
import dev.buildopt.generated.BuildOptClientsV1.Endpoint;
import dev.buildopt.generated.BuildOptClientsV1.Response;
import dev.buildopt.patcher.FullRelevantValidationGate.Applicability;
import dev.buildopt.patcher.FullRelevantValidationGate.ArtifactRef;
import dev.buildopt.patcher.FullRelevantValidationGate.Request;
import dev.buildopt.patcher.FullRelevantValidationGate.Result;
import dev.buildopt.patcher.FullRelevantValidationGate.Status;
import dev.buildopt.patcher.PatchCandidateValidator.Arm;
import dev.buildopt.patcher.PatchCandidateValidator.Artifact;
import dev.buildopt.patcher.PatchCandidateValidator.ArtifactAdapter;
import dev.buildopt.patcher.PatchCandidateValidator.Context;
import dev.buildopt.patcher.PatchCandidateValidator.Phase;
import dev.buildopt.patcher.PatchCandidateValidator.Run;

/** Focused C4-006 conformance cases for the production integration gate. */
final class FullRelevantValidationSpike {
    private static final String KEY_ID = "testopt-signing-2026-q3";
    private static final String REQUEST_ID = "request-full-validation";
    private static final String ACTION_ID = "archive-validation";
    private static final String TENANT = "tenant-7";
    private static final String REPOSITORY = "repo-42";
    private static final String TRUST_DOMAIN = "trusted-ci";
    private static final String REVISION = "a".repeat(40);
    private static final String SOURCE_STATE = digest('1');
    private static final String POLICY = digest('4');
    private static final String APPLICABILITY = digest('8');
    private static final Instant NOW = Instant.parse("2026-07-31T12:00:00Z");
    private static final List<String> REQUIRED =
            List.of("build/distributions/app.zip", "build/reports/manifest.txt");

    private FullRelevantValidationSpike() {
    }

    static void assertConformance() throws Exception {
        Fixture fixture = fixture();
        byte[] passed = signedResult(
                fixture,
                fixture.artifacts(),
                "PASSED",
                NOW.minusSeconds(1),
                NOW.plusSeconds(600),
                fixture.keyPair());

        FakeClock delayedClock = new FakeClock(NOW);
        FakeTransport delayed = new FakeTransport(List.of(
                response(202, receipt(fixture.request(), "operation-1", 100)),
                response(202, receipt(fixture.request(), "operation-1", 100)),
                response(200, passed)));
        Result delayedResult = evaluate(fixture, delayed, delayedClock);
        require(delayedResult.status() == Status.PASSED
                        && "FULL_RELEVANT_VALIDATION".equals(delayedResult.reason())
                        && fixture.localValidation().candidateArtifactSetDigest()
                                .equals(delayedResult.artifactSetDigest())
                        && delayed.calls.size() == 3
                        && delayed.calls.get(0).endpoint()
                                == BuildOptClientsV1.SUBMIT_TEST_BUILD_VALIDATION
                        && delayed.calls.get(1).endpoint()
                                == BuildOptClientsV1.GET_TEST_BUILD_VALIDATION
                        && delayed.calls.get(2).endpoint()
                                == BuildOptClientsV1.GET_TEST_BUILD_VALIDATION
                        && delayedClock.waitCount == 2,
                "delayed signed validation");

        FakeClock retryClock = new FakeClock(NOW);
        FakeTransport exactRetry = new FakeTransport(List.of(
                new IOException("transient"),
                response(200, passed)));
        requireResult(
                evaluate(fixture, exactRetry, retryClock),
                Status.PASSED,
                "FULL_RELEVANT_VALIDATION");
        require(exactRetry.calls.size() == 2
                        && exactRetry.calls.get(0).endpoint()
                                == BuildOptClientsV1.SUBMIT_TEST_BUILD_VALIDATION
                        && exactRetry.calls.get(1).endpoint()
                                == BuildOptClientsV1.SUBMIT_TEST_BUILD_VALIDATION
                        && exactRetry.calls.get(0).idempotencyKey()
                                .equals(exactRetry.calls.get(1).idempotencyKey())
                        && exactRetry.calls.get(0).deadline()
                                .equals(exactRetry.calls.get(1).deadline())
                        && Arrays.equals(
                                exactRetry.calls.get(0).body(),
                                exactRetry.calls.get(1).body()),
                "exact submit retry");

        byte[] failed = signedResult(
                fixture,
                fixture.artifacts(),
                "FAILED",
                NOW.minusSeconds(1),
                NOW.plusSeconds(600),
                fixture.keyPair());
        requireResult(
                evaluate(
                        fixture,
                        new FakeTransport(List.of(response(200, failed))),
                        new FakeClock(NOW)),
                Status.FAILED,
                "TEST_VALIDATION_FAILED");

        byte[] inconclusive = signedResult(
                fixture,
                fixture.artifacts(),
                "INCONCLUSIVE",
                NOW.minusSeconds(1),
                NOW.plusSeconds(600),
                fixture.keyPair());
        requireResult(
                evaluate(
                        fixture,
                        new FakeTransport(List.of(response(200, inconclusive))),
                        new FakeClock(NOW)),
                Status.INCONCLUSIVE,
                "TEST_VALIDATION_INCONCLUSIVE");

        KeyPair untrusted = KeyPairGenerator.getInstance("Ed25519").generateKeyPair();
        byte[] wrongSignature = signedResult(
                fixture,
                fixture.artifacts(),
                "PASSED",
                NOW.minusSeconds(1),
                NOW.plusSeconds(600),
                untrusted);
        requireResult(
                evaluate(
                        fixture,
                        new FakeTransport(List.of(response(200, wrongSignature))),
                        new FakeClock(NOW)),
                Status.INCONCLUSIVE,
                "UNTRUSTED_RESULT");

        List<ArtifactRef> rebound = new ArrayList<>(fixture.artifacts());
        ArtifactRef original = rebound.get(0);
        rebound.set(0, new ArtifactRef(
                original.artifactId(),
                digest('d'),
                original.sizeBytes(),
                original.mediaType(),
                original.retrievalKind(),
                original.locator()));
        byte[] artifactMismatch = signedResult(
                fixture,
                rebound,
                "PASSED",
                NOW.minusSeconds(1),
                NOW.plusSeconds(600),
                fixture.keyPair());
        requireResult(
                evaluate(
                        fixture,
                        new FakeTransport(List.of(response(200, artifactMismatch))),
                        new FakeClock(NOW)),
                Status.FAILED,
                "RESULT_ARTIFACT_MISMATCH");

        byte[] expired = signedResult(
                fixture,
                fixture.artifacts(),
                "PASSED",
                NOW.minusSeconds(600),
                NOW.minusSeconds(1),
                fixture.keyPair());
        requireResult(
                evaluate(
                        fixture,
                        new FakeTransport(List.of(response(200, expired))),
                        new FakeClock(NOW)),
                Status.INCONCLUSIVE,
                "RESULT_EXPIRED");

        FakeTransport mismatchedHeader = new FakeTransport(List.of(
                new Response(
                        200,
                        Map.of("X-BuildOpt-Request-ID", List.of("different-request")),
                        passed)));
        requireResult(
                evaluate(fixture, mismatchedHeader, new FakeClock(NOW)),
                Status.INCONCLUSIVE,
                "TRANSPORT_REJECTED");

        FakeTransport operationDrift = new FakeTransport(List.of(
                response(202, receipt(fixture.request(), "operation-1", 100)),
                response(202, receipt(fixture.request(), "operation-2", 100))));
        requireResult(
                evaluate(fixture, operationDrift, new FakeClock(NOW)),
                Status.INCONCLUSIVE,
                "INVALID_RECEIPT");

        Request shortDeadline = copyRequest(
                fixture.request(),
                fixture.request().artifacts(),
                NOW.plusMillis(50));
        Fixture shortFixture = new Fixture(
                fixture.localValidation(),
                fixture.artifacts(),
                shortDeadline,
                fixture.keyPair());
        FakeTransport deadline = new FakeTransport(List.of(
                response(202, receipt(shortDeadline, "operation-1", 100))));
        requireResult(
                evaluate(shortFixture, deadline, new FakeClock(NOW)),
                Status.INCONCLUSIVE,
                "DEADLINE_EXCEEDED");

        List<ArtifactRef> callerPath = new ArrayList<>(fixture.artifacts());
        ArtifactRef first = callerPath.get(0);
        callerPath.set(0, new ArtifactRef(
                first.artifactId(),
                first.digest(),
                first.sizeBytes(),
                first.mediaType(),
                "CUSTOMER_CHANNEL",
                "/tmp/caller-path"));
        Request invalidArtifactRequest = copyRequest(
                fixture.request(),
                callerPath,
                fixture.request().deadline());
        Fixture invalidArtifact = new Fixture(
                fixture.localValidation(),
                callerPath,
                invalidArtifactRequest,
                fixture.keyPair());
        FakeTransport mustNotSend = new FakeTransport(List.of(response(200, passed)));
        requireResult(
                evaluate(invalidArtifact, mustNotSend, new FakeClock(NOW)),
                Status.INCONCLUSIVE,
                "INVALID_ARTIFACT_SET");
        require(mustNotSend.calls.isEmpty(), "invalid artifact was not submitted");

        PatchCandidateValidator.Result localFailure =
                PatchCandidateValidator.validate(null);
        FakeTransport localMustNotSend = new FakeTransport(List.of(response(200, passed)));
        Result localBlocked = FullRelevantValidationGate.evaluate(
                localFailure,
                fixture.request(),
                KEY_ID,
                fixture.keyPair().getPublic(),
                localMustNotSend,
                new FakeClock(NOW),
                ignored -> {
                });
        requireResult(localBlocked, Status.FAILED, "LOCAL_VALIDATION_NOT_PASSED");
        require(localMustNotSend.calls.isEmpty(), "failed local validation was not submitted");

        Request notRequired = new Request(
                Applicability.NOT_REQUIRED,
                APPLICABILITY,
                REQUEST_ID,
                "",
                TENANT,
                REPOSITORY,
                TRUST_DOMAIN,
                REVISION,
                SOURCE_STATE,
                ACTION_ID,
                POLICY,
                NOW.plusSeconds(600),
                List.of());
        requireResult(
                FullRelevantValidationGate.evaluateNotRequired(
                        fixture.localValidation(),
                        notRequired,
                        NOW),
                Status.PASSED,
                "NOT_REQUIRED");
    }

    private static Result evaluate(
            Fixture fixture,
            FakeTransport transport,
            FakeClock clock) {
        return FullRelevantValidationGate.evaluate(
                fixture.localValidation(),
                fixture.request(),
                KEY_ID,
                fixture.keyPair().getPublic(),
                transport,
                clock,
                clock);
    }

    private static Fixture fixture() throws Exception {
        byte[] archive = archive();
        byte[] report = "artifact manifest v1\n".getBytes(StandardCharsets.UTF_8);
        List<Artifact> candidateArtifacts = List.of(
                new Artifact(REQUIRED.get(0), archive),
                new Artifact(REQUIRED.get(1), report));

        Context context = new Context(
                TENANT + "/" + REPOSITORY,
                ACTION_ID,
                REVISION,
                SOURCE_STATE,
                digest('2'),
                digest('3'),
                POLICY,
                "temurin-21-gradle-9.6.1",
                "linux-amd64-4cpu-16gib");
        List<Run> runs = new ArrayList<>();
        for (Arm arm : Arm.values()) {
            for (Phase phase : Phase.values()) {
                runs.add(new Run(
                        arm,
                        phase,
                        context,
                        arm.name().toLowerCase(java.util.Locale.ROOT)
                                + "-"
                                + phase.name().toLowerCase(java.util.Locale.ROOT),
                        phase == Phase.INCREMENTAL ? "REUSED" : "STORED",
                        0,
                        candidateArtifacts));
            }
        }
        PatchCandidateValidator.Result local = PatchCandidateValidator.validate(
                new PatchCandidateValidator.Request(
                        ArchiveReproducibilityRecipe.RECIPE_ID,
                        ArchiveReproducibilityRecipe.RECIPE_VERSION,
                        ArtifactAdapter.ARCHIVE_CONTENTS_V1,
                        REQUIRED,
                        runs));
        require(local.status() == PatchCandidateValidator.Status.PASSED,
                "local candidate validation fixture");

        List<ArtifactRef> refs = List.of(
                new ArtifactRef(
                        REQUIRED.get(0),
                        PatchBundleVerifier.digestBytes(archive),
                        archive.length,
                        "application/zip",
                        "CUSTOMER_CHANNEL",
                        "candidate-archive"),
                new ArtifactRef(
                        REQUIRED.get(1),
                        PatchBundleVerifier.digestBytes(report),
                        report.length,
                        "text/plain",
                        "CUSTOMER_CHANNEL",
                        "candidate-manifest"));
        Request request = new Request(
                Applicability.REQUIRED,
                APPLICABILITY,
                REQUEST_ID,
                ACTION_ID + ":" + local.candidateArtifactSetDigest(),
                TENANT,
                REPOSITORY,
                TRUST_DOMAIN,
                REVISION,
                SOURCE_STATE,
                ACTION_ID,
                POLICY,
                NOW.plusSeconds(1800),
                refs);
        KeyPair keyPair = KeyPairGenerator.getInstance("Ed25519").generateKeyPair();
        return new Fixture(local, refs, request, keyPair);
    }

    private static Request copyRequest(
            Request request,
            List<ArtifactRef> artifacts,
            Instant deadline) {
        return new Request(
                request.applicability(),
                request.applicabilityEvidenceDigest(),
                request.requestId(),
                request.idempotencyKey(),
                request.tenant(),
                request.repository(),
                request.trustDomain(),
                request.revision(),
                request.sourceStateDigest(),
                request.actionId(),
                request.policyDigest(),
                deadline,
                artifacts);
    }

    private static byte[] signedResult(
            Fixture fixture,
            List<ArtifactRef> artifacts,
            String status,
            Instant completedAt,
            Instant expiresAt,
            KeyPair signingKey) throws Exception {
        Map<String, Object> repository = new LinkedHashMap<>();
        repository.put("tenant", TENANT);
        repository.put("repository", REPOSITORY);
        repository.put("trustDomain", TRUST_DOMAIN);

        List<Object> validated = new ArrayList<>();
        for (ArtifactRef artifact : artifacts) {
            Map<String, Object> item = new LinkedHashMap<>();
            item.put("artifactId", artifact.artifactId());
            item.put("digest", artifact.digest());
            item.put("sizeBytes", artifact.sizeBytes());
            item.put("mediaType", artifact.mediaType());
            validated.add(item);
        }

        Map<String, Object> root = new LinkedHashMap<>();
        root.put("schemaVersion", "1.0");
        root.put("recordType", "TEST_VALIDATION_RESULT");
        root.put("contractVersion", "test-optimization/v1");
        root.put("resultId", "test-result-1");
        root.put("requestId", REQUEST_ID);
        root.put("actionId", ACTION_ID);
        root.put("repository", repository);
        root.put("revision", REVISION);
        root.put("sourceStateDigest", SOURCE_STATE);
        root.put("validationMode", "FULL_RELEVANT_VALIDATION");
        root.put("status", status);
        root.put("validatedArtifactRefs", validated);
        root.put(
                "artifactSetDigest",
                fixture.localValidation().candidateArtifactSetDigest());
        root.put("policyDigest", POLICY);
        root.put("evidenceRef", "testopt-evidence-1");
        root.put("completedAt", completedAt.toString());
        root.put("expiresAt", expiresAt.toString());
        if ("FAILED".equals(status)) {
            root.put("failure", Map.of(
                    "reason", "TEST_FAILURE",
                    "summary", "relevant test failed"));
        } else if ("INCONCLUSIVE".equals(status)) {
            root.put("inconclusive", Map.of("reason", "INFRA_FAILURE"));
        }

        String payloadDigest =
                PatchBundleVerifier.digestBytes(StrictJson.canonicalBytes(root));
        Signature signer = Signature.getInstance("Ed25519");
        signer.initSign(signingKey.getPrivate());
        signer.update(FullRelevantValidationGate.signatureInput(payloadDigest, KEY_ID));

        Map<String, Object> signature = new LinkedHashMap<>();
        signature.put("algorithm", "Ed25519");
        signature.put("canonicalization", "JCS");
        signature.put("keyId", KEY_ID);
        signature.put("signedPayloadDigest", payloadDigest);
        signature.put(
                "value",
                Base64.getUrlEncoder().withoutPadding().encodeToString(signer.sign()));
        root.put("signature", signature);
        return StrictJson.canonicalBytes(root);
    }

    private static byte[] receipt(
            Request request,
            String operationId,
            int pollAfterMs) {
        Map<String, Object> receipt = new LinkedHashMap<>();
        receipt.put("contractVersion", "test-optimization/v1");
        receipt.put("requestId", request.requestId());
        receipt.put("actionId", request.actionId());
        receipt.put("operationId", operationId);
        receipt.put("status", "PENDING");
        receipt.put("pollAfterMs", pollAfterMs);
        receipt.put("expiresAt", request.deadline().toString());
        return StrictJson.canonicalBytes(receipt);
    }

    private static Response response(int status, byte[] body) {
        return new Response(
                status,
                Map.of("X-BuildOpt-Request-ID", List.of(REQUEST_ID)),
                body);
    }

    private static byte[] archive() throws Exception {
        ByteArrayOutputStream output = new ByteArrayOutputStream();
        try (ZipOutputStream zip = new ZipOutputStream(output)) {
            ZipEntry entry = new ZipEntry("data/value.txt");
            entry.setTime(0L);
            zip.putNextEntry(entry);
            zip.write("payload\n".getBytes(StandardCharsets.UTF_8));
            zip.closeEntry();
        }
        return output.toByteArray();
    }

    private static String digest(char value) {
        return "sha256:" + String.valueOf(value).repeat(64);
    }

    private static void requireResult(Result result, Status status, String reason) {
        require(result.status() == status && reason.equals(result.reason()),
                "expected " + status + "/" + reason
                        + ", got " + result.status() + "/" + result.reason());
    }

    private static void require(boolean condition, String message) {
        if (!condition) {
            throw new AssertionError(message);
        }
    }

    private record Fixture(
            PatchCandidateValidator.Result localValidation,
            List<ArtifactRef> artifacts,
            Request request,
            KeyPair keyPair) {
    }

    private record Call(
            Endpoint endpoint,
            String requestId,
            String idempotencyKey,
            Instant deadline,
            byte[] body) {
        private Call {
            body = body.clone();
        }

        @Override
        public byte[] body() {
            return body.clone();
        }
    }

    private static final class FakeTransport
            implements FullRelevantValidationGate.Transport {
        private final List<Object> actions;
        private final List<Call> calls = new ArrayList<>();
        private int nextAction;

        private FakeTransport(List<Object> actions) {
            this.actions = List.copyOf(actions);
        }

        @Override
        public Response send(
                Endpoint endpoint,
                Map<String, String> pathValues,
                String requestId,
                String idempotencyKey,
                String ifMatch,
                Instant deadline,
                byte[] body) throws IOException {
            calls.add(new Call(endpoint, requestId, idempotencyKey, deadline, body));
            if (nextAction >= actions.size()) {
                throw new IOException("unexpected transport call");
            }
            Object action = actions.get(nextAction++);
            if (action instanceof IOException failure) {
                throw failure;
            }
            return (Response) action;
        }
    }

    private static final class FakeClock
            implements FullRelevantValidationGate.TimeSource,
                    FullRelevantValidationGate.Waiter {
        private Instant now;
        private int waitCount;

        private FakeClock(Instant now) {
            this.now = now;
        }

        @Override
        public Instant now() {
            return now;
        }

        @Override
        public void await(int milliseconds) {
            waitCount++;
            now = now.plusMillis(milliseconds);
        }
    }
}
