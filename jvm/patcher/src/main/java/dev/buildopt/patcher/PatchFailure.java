package dev.buildopt.patcher;

/**
 * A fail-closed PatchBundle result with the state the caller must preserve.
 */
public final class PatchFailure extends Exception {
    private static final long serialVersionUID = 1L;

    /** Outcomes shared with the executable SPK-004 acceptance matrix. */
    public enum Status {
        PROPOSED,
        REJECTED,
        UNCHANGED
    }

    private final Status status;

    public PatchFailure(Status status, String message) {
        super(message);
        this.status = status;
    }

    public PatchFailure(Status status, String message, Throwable cause) {
        super(message, cause);
        this.status = status;
    }

    public Status status() {
        return status;
    }
}
