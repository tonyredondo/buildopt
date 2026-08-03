#!/usr/bin/env bash
set -euo pipefail

buildopt_package_root=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)
buildopt_prefix=${HOME}/.local
if (($# == 2)) && [[ "$1" == "--prefix" ]]; then
    buildopt_prefix=$2
elif (($# != 0)); then
    printf 'usage: %s [--prefix <directory>]\n' "$0" >&2
    exit 64
fi
if [[ "${buildopt_prefix}" != /* ]]; then
    printf 'installation prefix must be absolute\n' >&2
    exit 1
fi

(cd "${buildopt_package_root}" && sha256sum -c SHA256SUMS >/dev/null)
mkdir -p -- "${buildopt_prefix}/bin" "${buildopt_prefix}/share/buildopt"
for buildopt_binary in buildopt buildopt-impact buildopt-server buildopt-edge; do
    install -m 0755 "${buildopt_package_root}/bin/${buildopt_binary}" \
        "${buildopt_prefix}/bin/.${buildopt_binary}.new.$$"
    mv -f -- "${buildopt_prefix}/bin/.${buildopt_binary}.new.$$" \
        "${buildopt_prefix}/bin/${buildopt_binary}"
done
install -m 0644 "${buildopt_package_root}/lib/buildopt.init.gradle" \
    "${buildopt_prefix}/share/buildopt/buildopt.init.gradle"
install -m 0644 "${buildopt_package_root}"/lib/buildopt-gradle-plugin-*.jar \
    "${buildopt_prefix}/share/buildopt/buildopt-gradle-plugin.jar"
install -m 0644 "${buildopt_package_root}"/lib/buildopt-jvm-agent-*.jar \
    "${buildopt_prefix}/share/buildopt/buildopt-jvm-agent.jar"
printf '%s\n' \
    'buildopt.install/v1' \
    'bin/buildopt' \
    'bin/buildopt-impact' \
    'bin/buildopt-server' \
    'bin/buildopt-edge' \
    'share/buildopt/buildopt.init.gradle' \
    'share/buildopt/buildopt-gradle-plugin.jar' \
    'share/buildopt/buildopt-jvm-agent.jar' \
    >"${buildopt_prefix}/share/buildopt/receipt"
printf 'BuildOpt installed in %s/bin\n' "${buildopt_prefix}"
if [[ ":${PATH}:" != *":${buildopt_prefix}/bin:"* ]]; then
    printf 'Add %s/bin to PATH, then run: buildopt doctor\n' "${buildopt_prefix}"
fi
