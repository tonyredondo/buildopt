package dev.buildopt.agent;

import java.lang.instrument.Instrumentation;
import java.util.Objects;

/**
 * Neutral JVM instrumentation entry point used to verify the Java 17 build and packaging contract.
 *
 * <p>Instrumentation, buffering, and failure behavior remain gated by {@code SPK-002} and
 * {@code C1-001}.
 */
public final class BuildOptAgent {
    private BuildOptAgent() {}

    /**
     * Accepts the JVM-provided instrumentation handle without installing transformers.
     *
     * @param agentArguments optional arguments supplied by the JVM
     * @param instrumentation JVM instrumentation service
     */
    public static void premain(String agentArguments, Instrumentation instrumentation) {
        Objects.requireNonNull(instrumentation, "instrumentation");
    }
}
