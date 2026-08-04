package dev.buildopt.breadth;

public final class ServiceWorker {
    private ServiceWorker() {}

    public static String response() {
        return "worker-" + SharedValue.value();
    }
}
