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
buildopt_run="${buildopt_repo_root}/dev/run"
buildopt_arch=$("${buildopt_run}" --toolchain go -- go env GOARCH)
case "${buildopt_arch}" in
    amd64|arm64) ;;
    *) printf 'unsupported Linux architecture: %s\n' "${buildopt_arch}" >&2; exit 1 ;;
esac

mkdir -p -- "${buildopt_output}"
buildopt_work=$(mktemp -d)
trap 'rm -rf -- "${buildopt_work}"' EXIT
buildopt_base="buildopt-${buildopt_version}-linux-${buildopt_arch}"
buildopt_root="${buildopt_work}/${buildopt_base}"
mkdir -p -- "${buildopt_root}/bin" "${buildopt_root}/lib"

cd "${buildopt_repo_root}"
for buildopt_command in buildopt buildopt-impact buildopt-server buildopt-edge; do
    CGO_ENABLED=0 GOOS=linux GOARCH="${buildopt_arch}" \
        "${buildopt_run}" --toolchain go -- go build -mod=readonly -buildvcs=false -trimpath \
        -o "${buildopt_root}/bin/${buildopt_command}" "./cmd/${buildopt_command}"
done
"${buildopt_run}" -- ./gradlew --no-daemon --offline "-PbuildoptVersion=${buildopt_version}" \
    :jvm:gradle-plugin:jar :jvm:jvm-agent:jar >&2
cp -- "jvm/gradle-plugin/build/libs/buildopt-gradle-plugin-${buildopt_version}.jar" \
    "${buildopt_root}/lib/"
cp -- "jvm/jvm-agent/build/libs/buildopt-jvm-agent-${buildopt_version}.jar" \
    "${buildopt_root}/lib/"
cp -- .github/actions/buildopt.init.gradle "${buildopt_root}/lib/"
cp -- packaging/linux/install.sh packaging/linux/uninstall.sh "${buildopt_root}/"
chmod 0755 "${buildopt_root}"/bin/* "${buildopt_root}"/*.sh
chmod 0644 "${buildopt_root}"/lib/*
(
    cd "${buildopt_root}"
    sha256sum bin/* lib/* > SHA256SUMS
)
buildopt_archive="${buildopt_output}/${buildopt_base}.tar.gz"
tar -C "${buildopt_work}" -czf "${buildopt_archive}" "${buildopt_base}"
(
    cd "${buildopt_output}"
    sha256sum "${buildopt_base}.tar.gz" > "${buildopt_base}.tar.gz.sha256"
)
printf '%s\n' "${buildopt_archive}"
