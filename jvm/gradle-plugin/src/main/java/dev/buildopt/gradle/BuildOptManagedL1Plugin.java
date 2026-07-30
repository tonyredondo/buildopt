package dev.buildopt.gradle;

import java.io.Serial;
import java.io.Serializable;
import java.nio.file.InvalidPathException;
import java.nio.file.Path;
import java.util.regex.Pattern;
import org.gradle.api.Action;
import org.gradle.api.Plugin;
import org.gradle.api.Project;
import org.gradle.api.initialization.Settings;
import org.gradle.api.provider.ProviderFactory;

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

    static final String READ_WRITE_MODE = "READ_WRITE";
    static final String DISABLED_L2_WRITER_MODE = "DISABLED_L2_WRITER";
    static final int RETENTION_DAYS = 7;

    private static final Pattern SCOPE_DIGEST = Pattern.compile("[0-9a-f]{64}");

    @Override
    public void apply(Settings settings) {
        ManagedL1Decision decision = inspect(settings.getProviders());
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
                .settingsEvaluated(new ConfigureManagedL1Action(decision));
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

    private static final class ConfigureManagedL1Action
            implements Action<Settings>, Serializable {
        @Serial private static final long serialVersionUID = 1L;

        private final ManagedL1Decision decision;

        ConfigureManagedL1Action(ManagedL1Decision decision) {
            this.decision = decision;
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
