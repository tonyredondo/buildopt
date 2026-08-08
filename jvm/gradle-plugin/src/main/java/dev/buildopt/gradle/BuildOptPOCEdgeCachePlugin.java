package dev.buildopt.gradle;

import java.net.URI;
import java.net.URISyntaxException;
import org.gradle.api.Plugin;
import org.gradle.api.initialization.Settings;
import org.gradle.caching.http.HttpBuildCache;

/**
 * Configures the read-only loopback Edge cache selected by a repository-owned
 * POC profile.
 *
 * <p>This intentionally narrow adapter accepts no remote host, credential, or
 * write mode. An absent or malformed endpoint leaves Gradle's native cache
 * configuration unchanged. Runtime authority and production cache policy stay
 * outside this proof-of-concept surface.
 */
public final class BuildOptPOCEdgeCachePlugin implements Plugin<Settings> {
    static final String URL_ENVIRONMENT = "BUILDOPT_POC_EDGE_CACHE_URL";

    @Override
    public void apply(Settings settings) {
        String value =
                settings.getProviders()
                        .environmentVariable(URL_ENVIRONMENT)
                        .getOrElse("");
        URI endpoint = parseLoopbackOrigin(value);
        if (endpoint == null) {
            return;
        }
        settings.getBuildCache()
                .local(
                        cache -> {
                            cache.setEnabled(false);
                            cache.setPush(false);
                        });
        settings.getBuildCache()
                .remote(
                        HttpBuildCache.class,
                        cache -> {
                            cache.setUrl(endpoint.resolve("/cache/"));
                            cache.setEnabled(true);
                            cache.setPush(false);
                            cache.setAllowInsecureProtocol(true);
                            cache.setUseExpectContinue(true);
                        });
    }

    private static URI parseLoopbackOrigin(String value) {
        try {
            URI endpoint = new URI(value);
            String path = endpoint.getRawPath();
            if (!endpoint.getScheme().equals("http")
                    || !endpoint.getHost().equals("127.0.0.1")
                    || endpoint.getPort() < 1
                    || endpoint.getPort() > 65535
                    || endpoint.getUserInfo() != null
                    || endpoint.getRawQuery() != null
                    || endpoint.getRawFragment() != null
                    || !(path == null || path.isEmpty() || path.equals("/"))) {
                return null;
            }
            String canonical = "http://127.0.0.1:" + endpoint.getPort();
            if (!value.equals(canonical) && !value.equals(canonical + "/")) {
                return null;
            }
            return new URI(canonical + "/");
        } catch (NullPointerException | URISyntaxException ignored) {
            return null;
        }
    }
}
