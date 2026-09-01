# Spring JMS Graph-Aware Confirmation v1

Status: `SJGC-001` complete; `SJGC-002` is current.

This successor reevaluates the exact public Spring JMS row at commit
`75705bcbb1da866d3ed44bf7526437a7fdf1136b` with the current graph-aware
history classifier. It is selected because the fresh 256-row audit now finds
six exact `:spring-jms` / `DEPENDENCY_SOURCE` matches with 24/27 projects
omitted, while the prior optimized-native observation at the same revision
proved a complete output partition, 14,423 exact outputs and captured
materialization. The earlier product version counted only one match.

The prior observation and source audit select the subject only; neither can
supply a fresh result. One optimized-native `testClasses` request may confirm
complete output closure, at least five graph-aware matches, a partial graph and
exact outputs. Candidate execution and timing are forbidden. Any mismatch
stops the route; a pass can authorize only a separate correctness contract.

