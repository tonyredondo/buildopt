# Automatic qualified-profile portfolio

## Decision

`buildopt optimize` stores every independently qualified structural candidate as a repository-scoped profile family. The portfolio is learning state for the proof of concept; it does not select or activate a profile.

## Why a portfolio

One repository can benefit from different structural reductions for a dependency change, a resource-only change, a leaf change, or a change spanning multiple source owners. A single last-known profile would overwrite useful evidence. The portfolio retains independently qualified families and replaces only an older exact binding for the same logical family.

## Structural classification

The classifier uses only normalized changed paths, Gradle project ownership, and typed project dependencies:

| Family | Structural facts |
|---|---|
| `DEPENDENCY_SOURCE` | One source owner and at least one transitive dependent project |
| `RESOURCE` | Every changed path is under a resources source set |
| `LEAF_SOURCE` | One source owner with no dependent project |
| `MIXED_SOURCE` | Multiple source owners or a source/resource mixture |

Repository-name rules, known-project switches, and target-specific adapters are forbidden. The logical family digest binds the stable repository identity, original and candidate entrypoints, project owners, and required outputs. It never binds the local checkout path. It intentionally excludes the target revision so later commits with the same structural shape replace the family instead of creating unbounded history.

## Atomic storage and validation

Each family owns private copies of the manifest, graph, generated manifest, qualification evidence, and qualified profile below `.buildopt/optimize/v1/portfolio/profiles/<family-sha256>/`. Every file is bound by SHA-256. BuildOpt writes those artifacts first and publishes `portfolio.json` last as the commit pointer.

An exact checkpoint is reused without calibration or rewrites. Drift or tampering invalidates the portfolio. BuildOpt may rebuild the current family from still-valid calibration evidence; otherwise optimized native Gradle remains authoritative. The portfolio is bounded to 64 families.

## POC boundary

This block proves that generic learning can persist multiple qualified shapes safely. It does not prove that an unseen change matches a stored family, and it does not execute the smaller graph automatically. Those responsibilities belong to automatic replay. Production authorization is always false and Test Optimization remains out of scope.
