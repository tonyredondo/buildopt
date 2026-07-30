package dev.buildopt.gradle;

import java.io.IOException;
import java.io.OutputStream;
import java.net.HttpURLConnection;
import java.net.Proxy;
import java.net.URI;
import java.net.URISyntaxException;
import java.net.URLConnection;
import java.nio.charset.StandardCharsets;
import java.nio.file.Path;
import java.util.Base64;

/**
 * Invocation-scoped credentials for the neutral local BuildOpt rendezvous.
 *
 * <p>The gateway credential authorizes only the loopback gateway, including
 * readiness and an independently authenticated managed-cache route. It is
 * never the Shared credential. The event token authenticates the Unix-socket
 * preface and is never serialized in the Protobuf payload.
 */
final class BuildOptRendezvousContext {
    private static final String ATTEMPT_ID_ENVIRONMENT = "BUILDOPT_PLUGIN_ATTEMPT_ID";
    private static final String SOCKET_ENVIRONMENT = "BUILDOPT_PLUGIN_EVENT_SOCKET";
    private static final String EVENT_TOKEN_ENVIRONMENT = "BUILDOPT_PLUGIN_EVENT_TOKEN";
    private static final String GATEWAY_URL_ENVIRONMENT = "BUILDOPT_GATEWAY_URL";
    private static final String GATEWAY_USERNAME_ENVIRONMENT = "BUILDOPT_GATEWAY_USERNAME";
    private static final String GATEWAY_PASSWORD_ENVIRONMENT = "BUILDOPT_GATEWAY_PASSWORD";
    private static final String GATEWAY_GENERATION_ENVIRONMENT =
            "BUILDOPT_GATEWAY_CONNECTION_GENERATION";

    private static final String READY_PATH = "/_buildopt/ready";
    private static final String GENERATION_HEADER =
            "BuildOpt-Gateway-Connection-Generation";
    private static final byte[] EVENT_AUTH_MAGIC =
            "BOA1".getBytes(StandardCharsets.US_ASCII);
    private static final int EVENT_TOKEN_BYTES = 32;
    private static final int TIMEOUT_MILLIS = 5_000;

    private final String attemptId;
    private final Path socketPath;
    private final byte[] eventToken;
    private final URI gatewayUri;
    private final String gatewayUsername;
    private final String gatewayPassword;
    private final String gatewayGeneration;

    private BuildOptRendezvousContext(
            String attemptId,
            Path socketPath,
            byte[] eventToken,
            URI gatewayUri,
            String gatewayUsername,
            String gatewayPassword,
            String gatewayGeneration) {
        this.attemptId = attemptId;
        this.socketPath = socketPath;
        this.eventToken = eventToken.clone();
        this.gatewayUri = gatewayUri;
        this.gatewayUsername = gatewayUsername;
        this.gatewayPassword = gatewayPassword;
        this.gatewayGeneration = gatewayGeneration;
    }

    static BuildOptRendezvousContext fromEnvironment() throws IOException {
        String attemptId = System.getenv(ATTEMPT_ID_ENVIRONMENT);
        String socket = System.getenv(SOCKET_ENVIRONMENT);
        String eventToken = System.getenv(EVENT_TOKEN_ENVIRONMENT);
        String gatewayUrl = System.getenv(GATEWAY_URL_ENVIRONMENT);
        String gatewayUsername = System.getenv(GATEWAY_USERNAME_ENVIRONMENT);
        String gatewayPassword = System.getenv(GATEWAY_PASSWORD_ENVIRONMENT);
        String gatewayGeneration = System.getenv(GATEWAY_GENERATION_ENVIRONMENT);

        if (attemptId == null
                && socket == null
                && eventToken == null
                && gatewayUrl == null
                && gatewayUsername == null
                && gatewayPassword == null
                && gatewayGeneration == null) {
            return null;
        }
        if (isBlank(attemptId)
                || isBlank(socket)
                || isBlank(eventToken)
                || isBlank(gatewayUrl)
                || isBlank(gatewayUsername)
                || isBlank(gatewayPassword)
                || isBlank(gatewayGeneration)) {
            throw new IOException("incomplete launcher rendezvous context");
        }
        if (gatewayUsername.indexOf(':') >= 0) {
            throw new IOException("invalid local gateway username");
        }

        byte[] decodedEventToken;
        try {
            decodedEventToken = Base64.getUrlDecoder().decode(eventToken);
        } catch (IllegalArgumentException exception) {
            throw new IOException("invalid plugin event credential", exception);
        }
        if (decodedEventToken.length != EVENT_TOKEN_BYTES) {
            throw new IOException("invalid plugin event credential");
        }

        URI parsedGateway = parseGatewayUri(gatewayUrl);
        Path parsedSocket;
        try {
            parsedSocket = Path.of(socket);
        } catch (RuntimeException exception) {
            throw new IOException("invalid plugin event socket", exception);
        }
        return new BuildOptRendezvousContext(
                attemptId,
                parsedSocket,
                decodedEventToken,
                parsedGateway,
                gatewayUsername,
                gatewayPassword,
                gatewayGeneration);
    }

    String attemptId() {
        return attemptId;
    }

    Path socketPath() {
        return socketPath;
    }

    void verifyGateway() throws IOException {
        URLConnection connection =
                gatewayUri.resolve(READY_PATH).toURL().openConnection(Proxy.NO_PROXY);
        if (!(connection instanceof HttpURLConnection http)) {
            throw new IOException("local gateway is not HTTP");
        }

        String basicCredential =
                Base64.getEncoder()
                        .encodeToString(
                                (gatewayUsername + ":" + gatewayPassword)
                                        .getBytes(StandardCharsets.UTF_8));
        http.setConnectTimeout(TIMEOUT_MILLIS);
        http.setReadTimeout(TIMEOUT_MILLIS);
        http.setInstanceFollowRedirects(false);
        http.setRequestMethod("GET");
        http.setRequestProperty("Authorization", "Basic " + basicCredential);
        try {
            int status = http.getResponseCode();
            if (status != HttpURLConnection.HTTP_NO_CONTENT) {
                throw new IOException(
                        "local gateway readiness returned HTTP " + status);
            }
            String generation = http.getHeaderField(GENERATION_HEADER);
            if (!gatewayGeneration.equals(generation)) {
                throw new IOException(
                        "local gateway returned a mismatched connection generation");
            }
        } finally {
            http.disconnect();
        }
    }

    void writeEventAuthentication(OutputStream output) throws IOException {
        output.write(EVENT_AUTH_MAGIC);
        output.write(eventToken);
    }

    private static URI parseGatewayUri(String gatewayUrl) throws IOException {
        URI gateway;
        try {
            gateway = new URI(gatewayUrl);
        } catch (URISyntaxException exception) {
            throw new IOException("invalid local gateway URL", exception);
        }
        String path = gateway.getRawPath();
        if (!"http".equals(gateway.getScheme())
                || !"127.0.0.1".equals(gateway.getHost())
                || gateway.getPort() <= 0
                || gateway.getUserInfo() != null
                || gateway.getRawQuery() != null
                || gateway.getRawFragment() != null
                || !(path == null || path.isEmpty() || "/".equals(path))) {
            throw new IOException("local gateway URL is not a validated loopback endpoint");
        }
        return gateway;
    }

    private static boolean isBlank(String value) {
        return value == null || value.isBlank();
    }
}
