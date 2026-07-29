package dev.buildopt.contracts;

import dev.buildopt.generated.BuildOptClientsV1;
import java.net.URI;
import java.net.http.HttpClient;
import java.nio.charset.StandardCharsets;
import java.nio.file.Files;
import java.nio.file.Path;
import java.util.HashSet;
import java.util.List;
import java.util.Locale;
import java.util.Set;

/**
 * Java 17 consumer for generated endpoint and N/N-1 compatibility contracts.
 */
public final class ClientConformance {
    private ClientConformance() {
    }

    /**
     * Validates the generated Java client against the shared fixture.
     *
     * @param arguments exactly one repository-root path
     * @throws Exception when the fixture cannot be read or a contract fails
     */
    public static void main(String[] arguments) throws Exception {
        if (arguments.length != 1) {
            throw new IllegalArgumentException("usage: ClientConformance <repository-root>");
        }
        checkEndpoints();
        int vectorCount = checkCompatibilityVectors(Path.of(arguments[0]));
        checkUnsafeTransport();
        checkDefensiveResponseBody();
        System.out.printf(
                Locale.ROOT,
                "Java generated client OK: %d endpoints, %d compatibility vectors%n",
                BuildOptClientsV1.ALL_ENDPOINTS.size(),
                vectorCount);
    }

    private static void checkEndpoints() {
        if (BuildOptClientsV1.ALL_ENDPOINTS.size() != 13) {
            throw new AssertionError(
                    "endpoint count " + BuildOptClientsV1.ALL_ENDPOINTS.size());
        }
        Set<String> operationIds = new HashSet<>();
        int mutations = 0;
        for (BuildOptClientsV1.Endpoint endpoint : BuildOptClientsV1.ALL_ENDPOINTS) {
            if (!operationIds.add(endpoint.operationId())) {
                throw new AssertionError("duplicate operation " + endpoint.operationId());
            }
            if (endpoint.mutation()) {
                mutations++;
            }
        }
        if (mutations != 8) {
            throw new AssertionError("mutation count " + mutations);
        }
    }

    private static int checkCompatibilityVectors(Path repositoryRoot) throws Exception {
        Path path = repositoryRoot
                .resolve("contracts")
                .resolve("test-vectors")
                .resolve("compatibility")
                .resolve("n-n-minus-1.tsv");
        int count = 0;
        for (String line : Files.readAllLines(path, StandardCharsets.UTF_8)) {
            if (line.isEmpty() || line.startsWith("#")) {
                continue;
            }
            String[] fields = line.split("\\t", -1);
            if (fields.length != 8) {
                throw new AssertionError("malformed compatibility row " + line);
            }
            BuildOptClientsV1.CompatibilityDecision actual =
                    BuildOptClientsV1.negotiate(
                            Integer.parseInt(fields[1]),
                            Integer.parseInt(fields[2]),
                            Integer.parseInt(fields[3]),
                            Integer.parseInt(fields[4]),
                            BuildOptClientsV1.Shape.valueOf(fields[5]),
                            Boolean.parseBoolean(fields[6]));
            if (!actual.name().equals(fields[7])) {
                throw new AssertionError(
                        fields[0] + " decision " + actual + " != " + fields[7]);
            }
            count++;
        }
        if (count != 9) {
            throw new AssertionError("compatibility vector count " + count);
        }
        return count;
    }

    private static void checkUnsafeTransport() {
        try {
            new BuildOptClientsV1.Client(
                    URI.create("http://127.0.0.1"),
                    "test-token",
                    HttpClient.newHttpClient());
            throw new AssertionError("HTTP control-plane origin was accepted");
        } catch (IllegalArgumentException expected) {
            if (!expected.getMessage().contains("HTTPS")) {
                throw expected;
            }
        }
    }

    private static void checkDefensiveResponseBody() {
        byte[] source = {1, 2, 3};
        BuildOptClientsV1.Response response = new BuildOptClientsV1.Response(
                200,
                java.util.Map.of("content-type", List.of("application/json")),
                source);
        source[0] = 9;
        byte[] firstRead = response.body();
        firstRead[1] = 9;
        byte[] secondRead = response.body();
        if (secondRead[0] != 1 || secondRead[1] != 2) {
            throw new AssertionError("response body is not defensive");
        }
    }
}
