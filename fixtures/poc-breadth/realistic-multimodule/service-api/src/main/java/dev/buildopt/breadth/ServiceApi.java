package dev.buildopt.breadth;

public final class ServiceApi {
    private ServiceApi() {}

    public static String response() {
        return "api-" + SharedValue.value();
    }
}
