package dev.buildopt.gradle;

import java.io.Serial;
import java.io.Serializable;
import java.net.URI;
import java.net.URISyntaxException;
import java.nio.file.InvalidPathException;
import java.nio.file.Path;
import java.util.Base64;
import java.util.regex.Pattern;
import org.gradle.api.Action;
import org.gradle.api.Plugin;
import org.gradle.api.Project;
import org.gradle.api.initialization.Settings;
import org.gradle.api.provider.ProviderFactory;
import org.gradle.caching.http.HttpBuildCache;

/**
 * Configures the native Gradle local cache from launcher-owned context.
 *
 * <p>The plugin never interprets Gradle's cache format. A complete read/write
 * context selects one scope-and-generation directory and native cleanup. An
 * absent or malformed context disables the managed build cache, while an
 * L2-writer context disables only the local cache so pending remote writes
 * cannot create a reusable local hit.
 */
public final class BuildOptManagedL1Plugin implements Plugin<Settings> {
    static final String DIRECTORY_ENVIRONMENT = "BUILDOPT_MANAGED_L1_DIRECTORY";
    static final String MODE_ENVIRONMENT = "BUILDOPT_MANAGED_L1_MODE";
    static final String GENERATION_ENVIRONMENT =
            "BUILDOPT_MANAGED_L1_SECURITY_GENERATION";
    static final String RETENTION_ENVIRONMENT =
            "BUILDOPT_MANAGED_L1_RETENTION_DAYS";
    static final String SHARED_MODE_ENVIRONMENT =
            "BUILDOPT_MANAGED_SHARED_MODE";
    static final String AUTHORITY_DIGEST_ENVIRONMENT =
            "BUILDOPT_MANAGED_AUTHORITY_DIGEST";
    static final String POLICY_DIGEST_ENVIRONMENT =
            "BUILDOPT_MANAGED_POLICY_DIGEST";
    static final String CONFIGURATION_DIGEST_ENVIRONMENT =
            "BUILDOPT_MANAGED_CONFIGURATION_POLICY_DIGEST";
    static final String AUTHORITY_CONTRACT_ENVIRONMENT =
            "BUILDOPT_MANAGED_AUTHORITY_CONTRACT";
    static final String GATEWAY_URL_ENVIRONMENT = "BUILDOPT_GATEWAY_URL";
    static final String GATEWAY_USERNAME_ENVIRONMENT =
            "BUILDOPT_GATEWAY_USERNAME";
    static final String GATEWAY_PASSWORD_ENVIRONMENT =
            "BUILDOPT_GATEWAY_PASSWORD";
    static final String GATEWAY_GENERATION_ENVIRONMENT =
            "BUILDOPT_GATEWAY_CONNECTION_GENERATION";

    static final String READ_WRITE_MODE = "READ_WRITE";
    static final String DISABLED_L2_WRITER_MODE = "DISABLED_L2_WRITER";
    static final String SHARED_READ_ONLY_MODE = "READ_ONLY";
    static final String SHARED_READ_WRITE_MODE = "READ_WRITE";
    static final String AUTHORITY_CONTRACT =
            "buildopt-local-cache-authority/v1";
    static final String CENTRAL_CONNECTION_CONTRACT =
            "buildopt-central-cache-connection/v1";
    static final int RETENTION_DAYS = 7;

    private static final Pattern SCOPE_DIGEST = Pattern.compile("[0-9a-f]{64}");
    private static final Pattern AUTHENTICATED_DIGEST =
            Pattern.compile("sha256:[0-9a-f]{64}");
    private static final Pattern UUID =
            Pattern.compile(
                    "[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}");

    @Override
    public void apply(Settings settings) {
        ManagedL1Decision decision = inspect(settings.getProviders());
        ManagedSharedDecision shared = inspectShared(settings.getProviders());
        if (decision.mode() == Mode.READ_WRITE) {
            settings.getCaches()
                    .buildCache(
                            cache ->
                                    cache.setRemoveUnusedEntriesAfterDays(
                                            RETENTION_DAYS));
        }
        settings.getGradle()
                .beforeProject(new ApplyTierOnePolicyAction());
        settings.getGradle()
                .settingsEvaluated(
                        new ConfigureManagedL1Action(decision, shared));
    }

    private static ManagedL1Decision inspect(ProviderFactory providers) {
        String mode =
                providers.environmentVariable(MODE_ENVIRONMENT).getOrElse("");
        String directory =
                providers.environmentVariable(DIRECTORY_ENVIRONMENT).getOrElse("");
        String generation =
                providers.environmentVariable(GENERATION_ENVIRONMENT).getOrElse("");
        String retention =
                providers.environmentVariable(RETENTION_ENVIRONMENT).getOrElse("");

        Long parsedGeneration = parseGeneration(generation);
        if (mode.equals(READ_WRITE_MODE)
                && parsedGeneration != null
                && retention.equals(Integer.toString(RETENTION_DAYS))
                && isManagedDirectory(directory, parsedGeneration)) {
            return ManagedL1Decision.readWrite(directory);
        }
        if (mode.equals(DISABLED_L2_WRITER_MODE)
                && directory.isEmpty()
                && parsedGeneration != null
                && retention.equals(Integer.toString(RETENTION_DAYS))) {
            return ManagedL1Decision.disabledL2Writer();
        }
        return ManagedL1Decision.invalid();
    }

    private static Long parseGeneration(String value) {
        try {
            long parsed = Long.parseLong(value);
            if (parsed < 0 || !Long.toString(parsed).equals(value)) {
                return null;
            }
            return parsed;
        } catch (NumberFormatException ignored) {
            return null;
        }
    }

    private static ManagedSharedDecision inspectShared(
            ProviderFactory providers) {
        String mode =
                providers.environmentVariable(SHARED_MODE_ENVIRONMENT)
                        .getOrElse("");
        String authorityDigest =
                providers.environmentVariable(AUTHORITY_DIGEST_ENVIRONMENT)
                        .getOrElse("");
        String policyDigest =
                providers.environmentVariable(POLICY_DIGEST_ENVIRONMENT)
                        .getOrElse("");
        String configurationDigest =
                providers.environmentVariable(CONFIGURATION_DIGEST_ENVIRONMENT)
                        .getOrElse("");
        String authorityContract =
                providers.environmentVariable(AUTHORITY_CONTRACT_ENVIRONMENT)
                        .getOrElse("");

        boolean absent =
                mode.isEmpty()
                        && authorityDigest.isEmpty()
                        && policyDigest.isEmpty()
                        && configurationDigest.isEmpty()
                        && authorityContract.isEmpty();
        if (absent) {
            return ManagedSharedDecision.disabled();
        }
        String gatewayUrl =
                providers.environmentVariable(GATEWAY_URL_ENVIRONMENT)
                        .getOrElse("");
        String gatewayUsername =
                providers.environmentVariable(GATEWAY_USERNAME_ENVIRONMENT)
                        .getOrElse("");
        String gatewayPassword =
                providers.environmentVariable(GATEWAY_PASSWORD_ENVIRONMENT)
                        .getOrElse("");
        String gatewayGeneration =
                providers.environmentVariable(GATEWAY_GENERATION_ENVIRONMENT)
                        .getOrElse("");
        boolean push;
        if (mode.equals(SHARED_READ_ONLY_MODE)) {
            push = false;
        } else if (mode.equals(SHARED_READ_WRITE_MODE)) {
            push = true;
        } else {
            return ManagedSharedDecision.invalid();
        }
        URI gateway = parseLoopbackGateway(gatewayUrl);
        if (gateway == null
                || !gatewayUsername.equals("buildopt")
                || !validGatewayPassword(gatewayPassword)
                || !UUID.matcher(gatewayGeneration).matches()
                || !AUTHENTICATED_DIGEST.matcher(authorityDigest).matches()
                || !AUTHENTICATED_DIGEST.matcher(policyDigest).matches()
                || !AUTHENTICATED_DIGEST.matcher(configurationDigest).matches()
                || (!authorityContract.equals(AUTHORITY_CONTRACT)
                        && !authorityContract.equals(CENTRAL_CONNECTION_CONTRACT))) {
            return ManagedSharedDecision.invalid();
        }
        return ManagedSharedDecision.enabled(
                gateway.resolve("/cache/").toString(),
                gatewayUsername,
                gatewayPassword,
                push);
    }

    private static URI parseLoopbackGateway(String value) {
        try {
            URI gateway = new URI(value);
            String path = gateway.getRawPath();
            if (!gateway.getScheme().equals("http")
                    || !gateway.getHost().equals("127.0.0.1")
                    || gateway.getPort() < 1
                    || gateway.getPort() > 65535
                    || gateway.getUserInfo() != null
                    || gateway.getRawQuery() != null
                    || gateway.getRawFragment() != null
                    || !(path == null
                            || path.isEmpty()
                            || path.equals("/"))) {
                return null;
            }
            String canonical = "http://127.0.0.1:" + gateway.getPort();
            if (!value.equals(canonical) && !value.equals(canonical + "/")) {
                return null;
            }
            return new URI(canonical + "/");
        } catch (NullPointerException | URISyntaxException ignored) {
            return null;
        }
    }

    private static boolean validGatewayPassword(String value) {
        try {
            byte[] decoded = Base64.getUrlDecoder().decode(value);
            return decoded.length == 32
                    && Base64.getUrlEncoder()
                            .withoutPadding()
                            .encodeToString(decoded)
                            .equals(value);
        } catch (IllegalArgumentException ignored) {
            return false;
        }
    }

    private static boolean isManagedDirectory(String value, long generation) {
        try {
            Path directory = Path.of(value);
            if (!directory.isAbsolute()
                    || !directory.normalize().equals(directory)
                    || directory.getNameCount() < 6
                    || !directory.getFileName().toString().equals("cache")) {
                return false;
            }
            Path generationDirectory = directory.getParent();
            Path scopeDirectory = generationDirectory.getParent();
            Path scopesDirectory = scopeDirectory.getParent();
            Path l1Directory = scopesDirectory.getParent();
            return generationDirectory
                            .getFileName()
                            .toString()
                            .equals("generation-" + generation)
                    && SCOPE_DIGEST.matcher(
                                    scopeDirectory.getFileName().toString())
                            .matches()
                    && scopesDirectory.getFileName().toString().equals("scopes")
                    && l1Directory.getFileName().toString().equals("l1");
        } catch (InvalidPathException | NullPointerException ignored) {
            return false;
        }
    }

    private enum Mode {
        READ_WRITE,
        DISABLED_L2_WRITER,
        INVALID
    }

    private record ManagedL1Decision(Mode mode, String directory)
            implements Serializable {
        @Serial private static final long serialVersionUID = 1L;

        static ManagedL1Decision readWrite(String directory) {
            return new ManagedL1Decision(Mode.READ_WRITE, directory);
        }

        static ManagedL1Decision disabledL2Writer() {
            return new ManagedL1Decision(Mode.DISABLED_L2_WRITER, "");
        }

        static ManagedL1Decision invalid() {
            return new ManagedL1Decision(Mode.INVALID, "");
        }
    }

    private enum SharedMode {
        ENABLED,
        DISABLED,
        INVALID
    }

    private record ManagedSharedDecision(
            SharedMode mode,
            String url,
            String username,
            String password,
            boolean push)
            implements Serializable {
        @Serial private static final long serialVersionUID = 1L;

        static ManagedSharedDecision enabled(
                String url,
                String username,
                String password,
                boolean push) {
            return new ManagedSharedDecision(
                    SharedMode.ENABLED,
                    url,
                    username,
                    password,
                    push);
        }

        static ManagedSharedDecision disabled() {
            return new ManagedSharedDecision(
                    SharedMode.DISABLED, "", "", "", false);
        }

        static ManagedSharedDecision invalid() {
            return new ManagedSharedDecision(
                    SharedMode.INVALID, "", "", "", false);
        }
    }

    private static final class ConfigureManagedL1Action
            implements Action<Settings>, Serializable {
        @Serial private static final long serialVersionUID = 1L;

        private final ManagedL1Decision decision;
        private final ManagedSharedDecision shared;

        ConfigureManagedL1Action(
                ManagedL1Decision decision,
                ManagedSharedDecision shared) {
            this.decision = decision;
            this.shared = shared;
        }

        @Override
        public void execute(Settings settings) {
            switch (decision.mode()) {
                case READ_WRITE -> {
                    settings.getBuildCache()
                            .local(
                                    cache -> {
                                        cache.setDirectory(decision.directory());
                                        cache.setEnabled(true);
                                        cache.setPush(true);
                                    });
                }
                case DISABLED_L2_WRITER ->
                        settings.getBuildCache()
                                .local(
                                        cache -> {
                                            cache.setEnabled(false);
                                            cache.setPush(false);
                                        });
                case INVALID -> {
                    settings.getGradle()
                            .getStartParameter()
                            .setBuildCacheEnabled(false);
                    settings.getBuildCache()
                            .local(
                                    cache -> {
                                        cache.setEnabled(false);
                                        cache.setPush(false);
                                    });
                }
            }
            if (shared.mode() == SharedMode.ENABLED) {
                settings.getBuildCache()
                        .remote(
                                HttpBuildCache.class,
                                cache -> {
                                    cache.setUrl(shared.url());
                                    cache.setEnabled(true);
                                    cache.setPush(shared.push());
                                    cache.setAllowInsecureProtocol(true);
                                    cache.setUseExpectContinue(true);
                                    cache.credentials(
                                            credentials -> {
                                                credentials.setUsername(
                                                        shared.username());
                                                credentials.setPassword(
                                                        shared.password());
                                            });
                                });
            } else {
                settings.getBuildCache()
                        .remote(
                                HttpBuildCache.class,
                                cache -> {
                                    cache.setEnabled(false);
                                    cache.setPush(false);
                                });
            }
        }
    }

    private static final class ApplyTierOnePolicyAction
            implements Action<Project>, Serializable {
        @Serial private static final long serialVersionUID = 1L;

        @Override
        public void execute(Project project) {
            project.getPluginManager()
                    .apply(BuildOptTierOnePolicyPlugin.class);
        }
    }
}
