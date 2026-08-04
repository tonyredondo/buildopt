package dev.buildopt.breadth;

public final class DeveloperTool {
    private DeveloperTool() {}

    public static String response() {
        return "tool-" + PlatformCore.platform();
    }
}
