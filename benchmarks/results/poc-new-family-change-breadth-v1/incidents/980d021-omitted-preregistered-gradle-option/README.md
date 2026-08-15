# Rejected Ktor change-breadth collection

This directory preserves the complete first diagnostic collection from
BuildOpt `980d0215f09dd49c3dc867cd7c1fb6898b19a019`. It is not terminal evidence
and none of its 48 timed pairs may be reused.

The frozen specification included
`-Pktor.develocity.skipBuildScans=true`, but the generic matrix runner still
used its older common option list. Both native and BuildOpt arms were
internally comparable and the three selective cells produced favorable,
correct diagnostic results, but the execution did not match every
preregistered input.

The correction makes the generic runner consume `method.gradleOptions`
exactly when a specification supplies them while retaining legacy defaults
for older matrices. All terminal captures restart from zero on the correction
revision. No repository, revision, change, output, option, threshold, order,
fallback or POC boundary was changed after observing this incident.
