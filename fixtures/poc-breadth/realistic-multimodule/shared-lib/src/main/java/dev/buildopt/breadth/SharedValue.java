package dev.buildopt.breadth;

public final class SharedValue {
    private SharedValue() {}

    public static String value() {
        return "shared-" + PlatformCore.platform();
    }
}
