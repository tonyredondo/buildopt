# GitLab CI synthetic proof v1

The owner-controlled fixture parses the component YAML, installs the existing
checksum-pinned synthetic Release Bundle through the GitLab setup path, proves
idempotent reuse, preserves exact child argv and failure exit status, and
emits normalized success, failure, cancellation, and unavailable artifacts.

A cross-project merge-request case is marked as a fork without consuming a
credential. The HTTPS download uses the deterministic fixture transport and
performs no remote mutation.

Run `./dev/check-gitlab-ci`.
