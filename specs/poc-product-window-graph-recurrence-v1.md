# Product-Window Graph Recurrence v1

`PRODUCT_WINDOW_GRAPH_RECURRENCE_V1` corrects the history-window mismatch exposed
by SJGC. It repeats the all-owner source classification over the exact retained
Spring, OpenTelemetry and Micronaut graphs, but uses the product's fixed 64
first-parent commits rather than the predecessor's exploratory 256 rows.

Every row is fresh. The TFGR and SJGC summaries may explain why this route
exists but may not supply a classification, group count or decision. The
classifier uses only changed paths plus SHA-256-bound graph and manifest facts;
repository and task names are labels only.

## Gate

- Exactly 64 chronological rows are reconstructed for each of 3 families.
- A group is eligible only with at least 5 exact owner/family commits and at
  least 1 omitted project.
- Eligible groups rank by compatible commits descending, omitted projects
  descending, structural identity ascending, then family label ascending.
- A passing result authorizes only a separately reviewed fresh graph
  confirmation for the selected exact commit. It does not authorize a
  candidate, Gradle timing, public patch or value claim.
- If no group passes, the route stops honestly.

Gradle, candidate execution, timing, public-source patching, production and
Test Optimization are outside this route.
