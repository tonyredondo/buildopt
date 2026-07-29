# Compatibility vectors

`n-n-minus-1.tsv` is the language-neutral `F0-022` negotiation corpus consumed
by the generated Go and Java clients.

Within major version 1, adjacent minor versions N/N-1 are accepted in either
direction. A gap larger than one minor or another major fails closed for the
optimization and preserves the Gradle baseline. Signed commands reject unknown
fields even when versions are compatible; exportable records accept and
preserve additive unknown fields.

Validate generation drift, compile both clients, execute this same corpus in
Go and Java 17, and exercise the Go HTTPS transport with:

```bash
./dev/check-generated-clients
```
