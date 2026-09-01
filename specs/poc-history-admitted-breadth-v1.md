# History-Admitted Breadth v1

Status: `HAB-001..004` closed as `STOP_PATH_ONLY_HISTORY_ADMISSION`.

This route transfers the history-admitted Kafka result to a second repository
without repository-name behavior. The generic selection derives the already
observed owner prefix, accepts a public first-parent commit only when every
changed path remains under that owner and excludes build logic. Spring target
`75705bcb` is the first preregistered observation after the previous four-match
row that satisfies those source facts.

Current BuildOpt runs three fresh ordinary requests from empty state. It must
naturally produce native/native/candidate and the required-output SHA/count
must remain identical across all three. Maximum is three Gradle starts. Timing,
state intervention, public source writes and historical result reuse are
forbidden.

The fresh Spring invocation resolved the expected owner, change family and
candidate, but the exact structural graph counted only one compatible
historical match. BuildOpt correctly retained native after one successful
request. Prefix-only source screening is therefore insufficient; no candidate
or timing ran and product failures were zero.
