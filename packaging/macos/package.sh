#!/usr/bin/env bash
set -euo pipefail

if (($# != 4)) || [[ "$1" != "--version" ]] || [[ "$3" != "--output" ]]; then
    printf 'usage: %s --version <semver> --output <directory>\n' "$0" >&2
    exit 64
fi

buildopt_version=$2
buildopt_output=$4
if [[ ! "${buildopt_version}" =~ ^[0-9]+[.][0-9]+[.][0-9]+([+-][0-9A-Za-z.-]+)?$ ]]; then
    printf 'invalid BuildOpt version: %s\n' "${buildopt_version}" >&2
    exit 64
fi

buildopt_script_dir=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)
buildopt_repo_root=$(cd -- "${buildopt_script_dir}/../.." && pwd)
buildopt_arch=$(go env GOARCH)
case "${buildopt_arch}" in
    amd64|arm64) ;;
    *) printf 'unsupported macOS architecture: %s\n' "${buildopt_arch}" >&2; exit 1 ;;
esac

mkdir -p -- "${buildopt_output}"
buildopt_work=$(mktemp -d)
trap 'rm -rf -- "${buildopt_work}"' EXIT
buildopt_base="buildopt-${buildopt_version}-darwin-${buildopt_arch}"
buildopt_root="${buildopt_work}/${buildopt_base}"
mkdir -p -- "${buildopt_root}/bin"

cd "${buildopt_repo_root}"
CGO_ENABLED=0 GOOS=darwin GOARCH="${buildopt_arch}" \
    go build -mod=readonly -buildvcs=false -trimpath \
    -o "${buildopt_root}/bin/buildopt" ./cmd/buildopt
CGO_ENABLED=0 GOOS=darwin GOARCH="${buildopt_arch}" \
    go build -mod=readonly -buildvcs=false -trimpath \
    -o "${buildopt_root}/bin/buildopt-impact" ./cmd/buildopt-impact
cp -- packaging/macos/install.sh packaging/macos/uninstall.sh "${buildopt_root}/"
chmod 0755 "${buildopt_root}/bin/buildopt" "${buildopt_root}/bin/buildopt-impact" "${buildopt_root}/install.sh" "${buildopt_root}/uninstall.sh"
(
    cd "${buildopt_root}"
    shasum -a 256 bin/buildopt bin/buildopt-impact > SHA256SUMS
)
tar -C "${buildopt_work}" -czf "${buildopt_output}/${buildopt_base}.tar.gz" "${buildopt_base}"
printf '%s\n' "${buildopt_output}/${buildopt_base}.tar.gz"
