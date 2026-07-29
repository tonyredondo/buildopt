package dev.buildopt.fixture;

public final class GoldenLane {
    private GoldenLane() {}

    public static String marker() {
        return "golden-lane-v1";
    }

    public static void main(String[] args) {
        System.out.println(marker());
    }
}
