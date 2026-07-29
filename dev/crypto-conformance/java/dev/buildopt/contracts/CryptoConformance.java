package dev.buildopt.contracts;

import java.io.IOException;
import java.math.BigDecimal;
import java.nio.ByteBuffer;
import java.nio.charset.CharacterCodingException;
import java.nio.charset.CodingErrorAction;
import java.nio.charset.StandardCharsets;
import java.nio.file.Files;
import java.nio.file.Path;
import java.security.KeyFactory;
import java.security.MessageDigest;
import java.security.PublicKey;
import java.security.Signature;
import java.security.spec.X509EncodedKeySpec;
import java.time.Instant;
import java.time.format.DateTimeParseException;
import java.util.ArrayList;
import java.util.Base64;
import java.util.Comparator;
import java.util.HexFormat;
import java.util.LinkedHashMap;
import java.util.List;
import java.util.Locale;
import java.util.Map;
import java.util.regex.Pattern;

/**
 * Dependency-free Java 17 consumer for the shared F0-020 cryptographic vectors.
 */
public final class CryptoConformance {
    private static final byte[] ED25519_X509_PREFIX =
            HexFormat.of().parseHex("302a300506032b6570032100");
    private static final Pattern UTC_TIMESTAMP = Pattern.compile(
            "^\\d{4}-\\d{2}-\\d{2}T\\d{2}:\\d{2}:[0-5]\\d(?:\\.\\d{1,9})?Z$");

    private CryptoConformance() {
    }

    /**
     * Runs all language-neutral vectors under the supplied repository root.
     *
     * @param arguments exactly one repository-root path
     * @throws Exception when a vector or cryptographic primitive fails
     */
    public static void main(String[] arguments) throws Exception {
        if (arguments.length != 1) {
            throw new IllegalArgumentException("usage: CryptoConformance <repository-root>");
        }
        Path vectors = Path.of(arguments[0], "contracts", "test-vectors");
        int canonicalCount = checkCanonicalVectors(
                vectors.resolve("canonical-json").resolve("vectors.tsv"));
        int timestampCount = checkTimestampVectors(
                vectors.resolve("canonical-json").resolve("timestamps.tsv"));
        int signatureCount = checkSignatureVectors(
                vectors.resolve("signatures").resolve("ed25519.tsv"));
        System.out.printf(
                Locale.ROOT,
                "Java crypto conformance OK: %d JCS, %d timestamp, %d Ed25519 vectors%n",
                canonicalCount,
                timestampCount,
                signatureCount);
    }

    private static int checkCanonicalVectors(Path path) throws Exception {
        List<String[]> rows = readTsv(path, 5);
        for (String[] row : rows) {
            byte[] input = Base64.getDecoder().decode(row[2]);
            try {
                byte[] canonical = canonicalize(input);
                if ("INVALID".equals(row[1])) {
                    throw new AssertionError(
                            row[0] + " unexpectedly canonicalized to "
                                    + new String(canonical, StandardCharsets.UTF_8));
                }
                byte[] expected = Base64.getDecoder().decode(row[3]);
                if (!MessageDigest.isEqual(canonical, expected)) {
                    throw new AssertionError(row[0] + " canonical bytes differ");
                }
                String digest = "sha256:" + HexFormat.of().formatHex(
                        MessageDigest.getInstance("SHA-256").digest(canonical));
                if (!digest.equals(row[4])) {
                    throw new AssertionError(
                            row[0] + " digest " + digest + " != " + row[4]);
                }
            } catch (IllegalArgumentException exception) {
                if (!"INVALID".equals(row[1])) {
                    throw exception;
                }
                if (!exception.getMessage().contains(row[3])) {
                    throw new AssertionError(
                            row[0] + " error " + exception.getMessage()
                                    + " does not contain " + row[3],
                            exception);
                }
            }
        }
        return rows.size();
    }

    private static int checkTimestampVectors(Path path) throws IOException {
        List<String[]> rows = readTsv(path, 3);
        for (String[] row : rows) {
            boolean valid = validUtcTimestamp(row[2]);
            boolean expected = "VALID".equals(row[1]);
            if (valid != expected) {
                throw new AssertionError(
                        row[0] + " timestamp validity " + valid + " != " + expected);
            }
        }
        return rows.size();
    }

    private static int checkSignatureVectors(Path path) throws Exception {
        List<String[]> rows = readTsv(path, 5);
        for (String[] row : rows) {
            boolean valid;
            try {
                byte[] rawKey = HexFormat.of().parseHex(row[2]);
                byte[] payload = Base64.getDecoder().decode(row[3]);
                byte[] signatureBytes = HexFormat.of().parseHex(row[4]);
                valid = rawKey.length == 32
                        && signatureBytes.length == 64
                        && verifyEd25519(rawKey, payload, signatureBytes);
            } catch (IllegalArgumentException exception) {
                valid = false;
            }
            boolean expected = "VALID".equals(row[1]);
            if (valid != expected) {
                throw new AssertionError(
                        row[0] + " signature validity " + valid + " != " + expected);
            }
        }
        return rows.size();
    }

    private static boolean verifyEd25519(
            byte[] rawKey,
            byte[] payload,
            byte[] signatureBytes) throws Exception {
        ByteBuffer encoded = ByteBuffer.allocate(ED25519_X509_PREFIX.length + rawKey.length);
        encoded.put(ED25519_X509_PREFIX);
        encoded.put(rawKey);
        PublicKey publicKey = KeyFactory.getInstance("Ed25519").generatePublic(
                new X509EncodedKeySpec(encoded.array()));
        Signature verifier = Signature.getInstance("Ed25519");
        verifier.initVerify(publicKey);
        verifier.update(payload);
        return verifier.verify(signatureBytes);
    }

    private static boolean validUtcTimestamp(String value) {
        if (!UTC_TIMESTAMP.matcher(value).matches()) {
            return false;
        }
        try {
            Instant.parse(value);
            return true;
        } catch (DateTimeParseException exception) {
            return false;
        }
    }

    private static byte[] canonicalize(byte[] input) {
        String json;
        try {
            json = StandardCharsets.UTF_8.newDecoder()
                    .onMalformedInput(CodingErrorAction.REPORT)
                    .onUnmappableCharacter(CodingErrorAction.REPORT)
                    .decode(ByteBuffer.wrap(input))
                    .toString();
        } catch (CharacterCodingException exception) {
            throw new IllegalArgumentException("invalid-utf8", exception);
        }
        Object value = new JsonParser(json).parse();
        StringBuilder output = new StringBuilder(json.length());
        appendCanonical(output, value);
        return output.toString().getBytes(StandardCharsets.UTF_8);
    }

    private static void appendCanonical(StringBuilder output, Object value) {
        if (value == null) {
            output.append("null");
        } else if (value instanceof Boolean booleanValue) {
            output.append(booleanValue);
        } else if (value instanceof String stringValue) {
            appendString(output, stringValue);
        } else if (value instanceof JsonNumber numberValue) {
            output.append(canonicalNumber(numberValue.source()));
        } else if (value instanceof List<?> arrayValue) {
            output.append('[');
            for (int index = 0; index < arrayValue.size(); index++) {
                if (index > 0) {
                    output.append(',');
                }
                appendCanonical(output, arrayValue.get(index));
            }
            output.append(']');
        } else if (value instanceof Map<?, ?> objectValue) {
            List<String> keys = new ArrayList<>();
            for (Object key : objectValue.keySet()) {
                keys.add((String) key);
            }
            keys.sort(Comparator.naturalOrder());
            output.append('{');
            for (int index = 0; index < keys.size(); index++) {
                if (index > 0) {
                    output.append(',');
                }
                String key = keys.get(index);
                appendString(output, key);
                output.append(':');
                appendCanonical(output, objectValue.get(key));
            }
            output.append('}');
        } else {
            throw new IllegalArgumentException(
                    "invalid-json: unsupported value " + value.getClass().getName());
        }
    }

    private static void appendString(StringBuilder output, String value) {
        output.append('"');
        for (int index = 0; index < value.length();) {
            int codePoint = value.codePointAt(index);
            index += Character.charCount(codePoint);
            switch (codePoint) {
                case '"' -> output.append("\\\"");
                case '\\' -> output.append("\\\\");
                case '\b' -> output.append("\\b");
                case '\t' -> output.append("\\t");
                case '\n' -> output.append("\\n");
                case '\f' -> output.append("\\f");
                case '\r' -> output.append("\\r");
                default -> {
                    if (codePoint < 0x20) {
                        output.append(String.format(Locale.ROOT, "\\u%04x", codePoint));
                    } else {
                        output.appendCodePoint(codePoint);
                    }
                }
            }
        }
        output.append('"');
    }

    private static String canonicalNumber(String source) {
        final double value;
        try {
            value = Double.parseDouble(source);
        } catch (NumberFormatException exception) {
            throw new IllegalArgumentException("invalid-json: number " + source, exception);
        }
        if (!Double.isFinite(value)) {
            throw new IllegalArgumentException("invalid-json: non-finite number " + source);
        }
        if (value == 0.0d) {
            return "0";
        }
        BigDecimal decimal = BigDecimal.valueOf(value).stripTrailingZeros();
        double absolute = Math.abs(value);
        if (absolute >= 1.0e-6d && absolute < 1.0e21d) {
            return decimal.toPlainString();
        }
        return decimal.toString().toLowerCase(Locale.ROOT);
    }

    private static List<String[]> readTsv(Path path, int fieldCount) throws IOException {
        List<String[]> rows = new ArrayList<>();
        for (String line : Files.readAllLines(path, StandardCharsets.UTF_8)) {
            if (line.isEmpty() || line.startsWith("#")) {
                continue;
            }
            String[] fields = line.split("\\t", -1);
            if (fields.length != fieldCount) {
                throw new IllegalArgumentException(
                        path + " row has " + fields.length + " fields: " + line);
            }
            if (!"VALID".equals(fields[1]) && !"INVALID".equals(fields[1])) {
                throw new IllegalArgumentException(
                        path + " has unknown outcome " + fields[1]);
            }
            rows.add(fields);
        }
        if (rows.isEmpty()) {
            throw new IllegalArgumentException(path + " has no vectors");
        }
        return rows;
    }

    private record JsonNumber(String source) {
    }

    private static final class JsonParser {
        private final String source;
        private int position;

        private JsonParser(String source) {
            this.source = source;
        }

        private Object parse() {
            skipWhitespace();
            Object value = parseValue();
            skipWhitespace();
            if (position != source.length()) {
                fail("trailing value");
            }
            return value;
        }

        private Object parseValue() {
            if (position >= source.length()) {
                fail("unexpected end");
            }
            return switch (source.charAt(position)) {
                case '{' -> parseObject();
                case '[' -> parseArray();
                case '"' -> parseString();
                case 't' -> parseLiteral("true", Boolean.TRUE);
                case 'f' -> parseLiteral("false", Boolean.FALSE);
                case 'n' -> parseLiteral("null", null);
                default -> parseNumber();
            };
        }

        private Map<String, Object> parseObject() {
            expect('{');
            skipWhitespace();
            Map<String, Object> object = new LinkedHashMap<>();
            if (consume('}')) {
                return object;
            }
            while (true) {
                if (position >= source.length() || source.charAt(position) != '"') {
                    fail("object key");
                }
                String key = parseString();
                if (object.containsKey(key)) {
                    throw new IllegalArgumentException("duplicate-key: " + key);
                }
                skipWhitespace();
                expect(':');
                skipWhitespace();
                object.put(key, parseValue());
                skipWhitespace();
                if (consume('}')) {
                    return object;
                }
                expect(',');
                skipWhitespace();
            }
        }

        private List<Object> parseArray() {
            expect('[');
            skipWhitespace();
            List<Object> array = new ArrayList<>();
            if (consume(']')) {
                return array;
            }
            while (true) {
                array.add(parseValue());
                skipWhitespace();
                if (consume(']')) {
                    return array;
                }
                expect(',');
                skipWhitespace();
            }
        }

        private String parseString() {
            expect('"');
            StringBuilder value = new StringBuilder();
            while (position < source.length()) {
                char character = source.charAt(position++);
                if (character == '"') {
                    return value.toString();
                }
                if (character < 0x20) {
                    fail("unescaped control");
                }
                if (character != '\\') {
                    if (Character.isSurrogate(character)) {
                        if (!Character.isHighSurrogate(character)
                                || position >= source.length()
                                || !Character.isLowSurrogate(source.charAt(position))) {
                            throw new IllegalArgumentException("unpaired-surrogate");
                        }
                        value.append(character);
                        value.append(source.charAt(position++));
                    } else {
                        value.append(character);
                    }
                    continue;
                }
                if (position >= source.length()) {
                    fail("unfinished escape");
                }
                char escape = source.charAt(position++);
                switch (escape) {
                    case '"', '\\', '/' -> value.append(escape);
                    case 'b' -> value.append('\b');
                    case 'f' -> value.append('\f');
                    case 'n' -> value.append('\n');
                    case 'r' -> value.append('\r');
                    case 't' -> value.append('\t');
                    case 'u' -> appendUnicodeEscape(value);
                    default -> fail("invalid escape");
                }
            }
            fail("unterminated string");
            return "";
        }

        private void appendUnicodeEscape(StringBuilder output) {
            char first = readHexCodeUnit();
            if (Character.isHighSurrogate(first)) {
                if (position + 2 > source.length()
                        || source.charAt(position) != '\\'
                        || source.charAt(position + 1) != 'u') {
                    throw new IllegalArgumentException("unpaired-surrogate");
                }
                position += 2;
                char second = readHexCodeUnit();
                if (!Character.isLowSurrogate(second)) {
                    throw new IllegalArgumentException("unpaired-surrogate");
                }
                output.append(first).append(second);
            } else if (Character.isLowSurrogate(first)) {
                throw new IllegalArgumentException("unpaired-surrogate");
            } else {
                output.append(first);
            }
        }

        private char readHexCodeUnit() {
            if (position + 4 > source.length()) {
                fail("short unicode escape");
            }
            int value = 0;
            for (int offset = 0; offset < 4; offset++) {
                int digit = Character.digit(source.charAt(position++), 16);
                if (digit < 0) {
                    fail("invalid unicode escape");
                }
                value = value * 16 + digit;
            }
            return (char) value;
        }

        private Object parseLiteral(String literal, Object value) {
            if (!source.startsWith(literal, position)) {
                fail("invalid literal");
            }
            position += literal.length();
            return value;
        }

        private JsonNumber parseNumber() {
            int start = position;
            consume('-');
            if (consume('0')) {
                if (position < source.length()
                        && Character.isDigit(source.charAt(position))) {
                    fail("leading zero");
                }
            } else {
                requireDigits();
            }
            if (consume('.')) {
                requireDigits();
            }
            if (consume('e') || consume('E')) {
                if (!consume('+')) {
                    consume('-');
                }
                requireDigits();
            }
            if (start == position) {
                fail("expected value");
            }
            String number = source.substring(start, position);
            double parsed;
            try {
                parsed = Double.parseDouble(number);
            } catch (NumberFormatException exception) {
                throw new IllegalArgumentException("invalid-json: number", exception);
            }
            if (!Double.isFinite(parsed)) {
                fail("non-finite number");
            }
            return new JsonNumber(number);
        }

        private void requireDigits() {
            int start = position;
            while (position < source.length()
                    && Character.isDigit(source.charAt(position))) {
                position++;
            }
            if (start == position) {
                fail("expected digit");
            }
        }

        private void skipWhitespace() {
            while (position < source.length()) {
                char character = source.charAt(position);
                if (character != ' '
                        && character != '\t'
                        && character != '\n'
                        && character != '\r') {
                    return;
                }
                position++;
            }
        }

        private boolean consume(char character) {
            if (position < source.length() && source.charAt(position) == character) {
                position++;
                return true;
            }
            return false;
        }

        private void expect(char character) {
            if (!consume(character)) {
                fail("expected " + character);
            }
        }

        private void fail(String message) {
            throw new IllegalArgumentException(
                    "invalid-json: " + message + " at " + position);
        }
    }
}
