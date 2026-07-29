package dev.buildopt.fixtures.correlation;

import com.sun.net.httpserver.HttpExchange;
import com.sun.net.httpserver.HttpServer;
import java.io.IOException;
import java.net.InetAddress;
import java.net.InetSocketAddress;
import java.nio.charset.StandardCharsets;
import java.nio.file.Files;
import java.nio.file.Path;
import java.nio.file.StandardCopyOption;
import java.nio.file.StandardOpenOption;
import java.util.Locale;
import java.util.concurrent.CountDownLatch;
import java.util.concurrent.Executors;
import java.util.regex.Pattern;

/** Minimal loopback HTTP build cache that records every completed PUT. */
public final class CorrelationCacheServer {
    private static final Pattern CACHE_KEY = Pattern.compile("[0-9a-f]{32,64}");
    private static final Object LOG_LOCK = new Object();

    private CorrelationCacheServer() {}

    public static void main(String[] arguments) throws Exception {
        if (arguments.length != 3) {
            throw new IllegalArgumentException(
                    "usage: CorrelationCacheServer <cache-dir> <ready-file> <put-log>");
        }

        Path cacheDirectory = Path.of(arguments[0]).toAbsolutePath().normalize();
        Path readyFile = Path.of(arguments[1]).toAbsolutePath().normalize();
        Path putLog = Path.of(arguments[2]).toAbsolutePath().normalize();
        Files.createDirectories(cacheDirectory);
        Files.createDirectories(readyFile.getParent());
        Files.createDirectories(putLog.getParent());
        Files.writeString(
                putLog,
                "",
                StandardCharsets.UTF_8,
                StandardOpenOption.CREATE,
                StandardOpenOption.TRUNCATE_EXISTING);

        HttpServer server = HttpServer.create(
                new InetSocketAddress(InetAddress.getByName("127.0.0.1"), 0),
                0);
        server.createContext(
                "/cache/",
                exchange -> handle(exchange, cacheDirectory, putLog));
        server.setExecutor(Executors.newCachedThreadPool());
        server.start();

        CountDownLatch stopped = new CountDownLatch(1);
        Runtime.getRuntime().addShutdownHook(new Thread(() -> {
            server.stop(0);
            stopped.countDown();
        }, "correlation-cache-shutdown"));

        String url = "http://127.0.0.1:"
                + server.getAddress().getPort()
                + "/cache/";
        Files.writeString(
                readyFile,
                url + "\n",
                StandardCharsets.UTF_8,
                StandardOpenOption.CREATE_NEW);
        stopped.await();
    }

    private static void handle(
            HttpExchange exchange,
            Path cacheDirectory,
            Path putLog) throws IOException {
        try (exchange) {
            String key = exchange.getRequestURI()
                    .getPath()
                    .substring("/cache/".length())
                    .toLowerCase(Locale.ROOT);
            if (!CACHE_KEY.matcher(key).matches()) {
                exchange.sendResponseHeaders(400, -1);
                return;
            }

            Path entry = cacheDirectory.resolve(key);
            switch (exchange.getRequestMethod()) {
                case "GET" -> load(exchange, entry);
                case "PUT" -> store(exchange, entry, key, putLog);
                default -> exchange.sendResponseHeaders(405, -1);
            }
        } catch (RuntimeException | IOException exception) {
            exception.printStackTrace(System.err);
            throw exception;
        }
    }

    private static void load(HttpExchange exchange, Path entry) throws IOException {
        if (!Files.isRegularFile(entry)) {
            exchange.sendResponseHeaders(404, -1);
            return;
        }
        long size = Files.size(entry);
        exchange.sendResponseHeaders(200, size);
        Files.copy(entry, exchange.getResponseBody());
    }

    private static void store(
            HttpExchange exchange,
            Path entry,
            String key,
            Path putLog) throws IOException {
        Path temporary = Files.createTempFile(
                entry.getParent(),
                key + ".",
                ".pending");
        boolean moved = false;
        try {
            Files.copy(
                    exchange.getRequestBody(),
                    temporary,
                    StandardCopyOption.REPLACE_EXISTING);
            Files.move(
                    temporary,
                    entry,
                    StandardCopyOption.ATOMIC_MOVE,
                    StandardCopyOption.REPLACE_EXISTING);
            moved = true;
            synchronized (LOG_LOCK) {
                Files.writeString(
                        putLog,
                        key + "\n",
                        StandardCharsets.UTF_8,
                        StandardOpenOption.CREATE,
                        StandardOpenOption.APPEND);
            }
            exchange.sendResponseHeaders(200, -1);
        } finally {
            if (!moved) {
                Files.deleteIfExists(temporary);
            }
        }
    }
}
