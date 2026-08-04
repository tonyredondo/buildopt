package dev.buildopt.pocvalue;

public final class ServiceA {
    private ServiceA() {}

    public static String value() {
        return "service-a:" + LibraryC.value();
    }
}
