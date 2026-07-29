# Data lifecycle fixtures

`raw-input.json` contains only synthetic sensitive strings. The four
`expected-*.json` files are the exact managed outputs for a deterministic test
HMAC key and never contain those raw values or source content.

`valid-events.jsonl` demonstrates at-least-once delivery with an exact
duplicate and a missing sequence. `conflicting-duplicate.jsonl` reuses an
event ID with changed content and must be rejected.

Run `./dev/check-data-lifecycle`.
