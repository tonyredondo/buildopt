package dev.buildopt.patcher;

import static java.nio.file.LinkOption.NOFOLLOW_LINKS;

import java.io.IOException;
import java.io.InputStream;
import java.math.BigDecimal;
import java.nio.ByteBuffer;
import java.nio.charset.CharacterCodingException;
import java.nio.charset.CodingErrorAction;
import java.nio.charset.StandardCharsets;
import java.nio.file.Files;
import java.nio.file.Path;
import java.util.ArrayList;
import java.util.Comparator;
import java.util.LinkedHashMap;
import java.util.List;
import java.util.Locale;
import java.util.Map;

/**
 * Dependency-free strict JSON parser and JCS encoder.
 *
 * <p>The parser rejects duplicate keys, malformed UTF-8, trailing values,
 * non-finite numbers, and unpaired surrogates. Bundle validation further
 * restricts every numeric field to its declared integer range.</p>
 */
final class StrictJson {
    private StrictJson() {
    }

    static Object parse(Path path, long maximumBytes) throws IOException {
        if (maximumBytes < 1 || maximumBytes >= Integer.MAX_VALUE) {
            throw new IllegalArgumentException(
                    "JSON byte limit is outside the supported range");
        }
        byte[] content;
        try (InputStream input = Files.newInputStream(path, NOFOLLOW_LINKS)) {
            content = input.readNBytes((int) maximumBytes + 1);
        }
        if (content.length < 1 || content.length > maximumBytes) {
            throw new IllegalArgumentException(
                    "JSON size is outside 1.." + maximumBytes);
        }
        return parse(content);
    }

    static Object parse(byte[] input) {
        String json;
        try {
            json = StandardCharsets.UTF_8.newDecoder()
                    .onMalformedInput(CodingErrorAction.REPORT)
                    .onUnmappableCharacter(CodingErrorAction.REPORT)
                    .decode(ByteBuffer.wrap(input))
                    .toString();
        } catch (CharacterCodingException exception) {
            throw new IllegalArgumentException("invalid-json: invalid UTF-8", exception);
        }
        return new Parser(json).parse();
    }

    static byte[] canonicalBytes(Object value) {
        return canonical(value).getBytes(StandardCharsets.UTF_8);
    }

    static String canonical(Object value) {
        StringBuilder output = new StringBuilder();
        appendCanonical(output, value);
        return output.toString();
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
        } else if (value instanceof Number numberValue) {
            output.append(canonicalNumber(numberValue.toString()));
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
                if (!(key instanceof String stringKey)) {
                    throw new IllegalArgumentException("JSON object key is not a string");
                }
                keys.add(stringKey);
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
                    "unsupported JSON value " + value.getClass().getName());
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

    record JsonNumber(String source) {
    }

    private static final class Parser {
        private final String source;
        private int position;

        private Parser(String source) {
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
