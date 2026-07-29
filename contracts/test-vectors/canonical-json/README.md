# Canonical JSON vectors

`vectors.tsv` is the language-neutral `F0-020` JCS corpus. Each row carries
base64-encoded input bytes, exact canonical UTF-8 bytes or a stable rejection,
and the lowercase SHA-256 digest of valid canonical bytes. It covers member
ordering by UTF-16 code units, escaping and preserved Unicode, IEEE-754 number
rendering, duplicate-key rejection, and invalid UTF-8.

`timestamps.tsv` accepts only UTC RFC 3339 instants ending in uppercase `Z`;
offsets, lowercase suffixes, missing seconds, and leap seconds are rejected.

Both corpora are consumed without language-specific copies by:

```bash
./dev/check-contract-crypto
```

The checker runs independent Go and Java 17 implementations over the exact
same rows. TSV is a fixture envelope only; decoded payload bytes are the
normative JCS inputs and outputs.
