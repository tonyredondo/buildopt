#!/usr/bin/env bash
set -euo pipefail

buildopt_version=latest
buildopt_prefix=${HOME}/.local
while (($# > 0)); do
    case "$1" in
        --version) buildopt_version=${2:?missing version}; shift 2 ;;
        --prefix) buildopt_prefix=${2:?missing prefix}; shift 2 ;;
        *) printf 'usage: %s [--version <semver>] [--prefix <absolute-directory>]\n' "$0" >&2; exit 64 ;;
    esac
done
if [[ "${buildopt_prefix}" != /* ]]; then
    printf 'installation prefix must be absolute\n' >&2
    exit 64
fi

case "$(uname -s)" in
    Linux) buildopt_os=linux ;;
    Darwin) buildopt_os=darwin ;;
    *) printf 'use install.ps1 on Windows; this installer supports Linux and macOS\n' >&2; exit 1 ;;
esac
case "$(uname -m)" in
    x86_64|amd64) buildopt_arch=amd64 ;;
    arm64|aarch64) buildopt_arch=arm64 ;;
    *) printf 'unsupported architecture: %s\n' "$(uname -m)" >&2; exit 1 ;;
esac
if [[ "${buildopt_os}" == linux && "${buildopt_arch}" != amd64 ]]; then
    printf 'published Linux packages currently support AMD64 only\n' >&2
    exit 1
fi
if [[ "${buildopt_version}" == latest ]]; then
    buildopt_version=$(curl --fail --silent --show-error --location \
        --proto '=https' --proto-redir '=https' --tlsv1.2 \
        https://github.com/tonyredondo/buildopt/releases/latest/download/buildopt-version.txt)
fi
if [[ ! "${buildopt_version}" =~ ^[0-9]+[.][0-9]+[.][0-9]+([+-][0-9A-Za-z.-]+)?$ ]]; then
    printf 'invalid BuildOpt version: %s\n' "${buildopt_version}" >&2
    exit 1
fi

buildopt_base="buildopt-${buildopt_version}-${buildopt_os}-${buildopt_arch}"
buildopt_release="https://github.com/tonyredondo/buildopt/releases/download/v${buildopt_version}"
buildopt_work=$(mktemp -d)
trap 'rm -rf -- "${buildopt_work}"' EXIT
curl --fail --silent --show-error --location --proto '=https' --proto-redir '=https' \
    --tlsv1.2 --output "${buildopt_work}/${buildopt_base}.tar.gz" \
    "${buildopt_release}/${buildopt_base}.tar.gz"
curl --fail --silent --show-error --location --proto '=https' --proto-redir '=https' \
    --tlsv1.2 --output "${buildopt_work}/${buildopt_base}.tar.gz.sha256" \
    "${buildopt_release}/${buildopt_base}.tar.gz.sha256"
(
    cd "${buildopt_work}"
    if command -v sha256sum >/dev/null 2>&1; then
        sha256sum -c "${buildopt_base}.tar.gz.sha256"
    else
        shasum -a 256 -c "${buildopt_base}.tar.gz.sha256"
    fi
)
if tar -tzf "${buildopt_work}/${buildopt_base}.tar.gz" | \
    awk -v root="${buildopt_base}/" '$0 !~ "^" root || $0 ~ /(^|\/)\.\.\// {exit 1}'; then
    :
else
    printf 'release archive contains an unsafe path\n' >&2
    exit 1
fi
if ! tar -tvzf "${buildopt_work}/${buildopt_base}.tar.gz" | \
    awk 'substr($1,1,1) != "-" && substr($1,1,1) != "d" {exit 1}'; then
    printf 'release archive contains an unsafe entry type\n' >&2
    exit 1
fi
tar -xzf "${buildopt_work}/${buildopt_base}.tar.gz" -C "${buildopt_work}"
"${buildopt_work}/${buildopt_base}/install.sh" --prefix "${buildopt_prefix}"
