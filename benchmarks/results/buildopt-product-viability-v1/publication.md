# Research evidence published with the quarter review

Published from the research snapshot reviewed on 10 September 2026. The
[main report](../../../docs/findings/buildopt-next-quarter-investment-review-2026-09-10.md)
and [evidence appendix](../../../docs/findings/buildopt-next-quarter-evidence-2026-09-10.md)
link the results used for the decision.

Git contains the reports, plans, implementation, candidate patches and linked
result summaries. It does not contain the complete local capture collection:
raw build outputs, caches, process recordings and temporary test directories
remain on the research workstation. References to those files in historical
JSON manifests describe the original captures; they are not a claim that every
file in a manifest is available in this checkout. No experiment was rerun for
publication, and no measured result or acceptance criterion was changed.

Three links in the historical evidence READMEs now identify local artifacts
instead of pointing to files that are absent from Git:

| Local artifact, relative to this directory | Bytes | SHA-256 |
| --- | ---: | --- |
| `bv005/raw-fixtures.tar.zst` | 154,655,964 | `3a66d333775c243f35990e9fe1713b4ae47fde61319e3ed2b957d1ba32d3b973` |
| `bv006-daemon-diagnostic/raw/owner/profile-I.jfr` | 3,349,500 | `fe9505e6768325cbada31df7637d75d23224fbec2d05c3af52ae2b2ab10637b0` |
| `bv006-daemon-diagnostic/raw/owner/profile-N.jfr` | 3,629,896 | `96eccdec83f40f16ba7697a43efede54290ef47f783cc815df89bf63437df4c4` |

Historical seals refer to the original captured files. The two README edits
only describe publication availability. Their original SHA-256 values were:

- `bv005/README.md`: `796ddf2c387c938421914a405353d08d12b9fd5a8b8452265686ceac294ef777`.
- `bv006-daemon-diagnostic/README.md`: `817ae7fd1025f93071f0231bfaf644005ac436e6bab2de95f82453f23a8cef04`.
