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
if [[ ! "${buildopt_prefix}" = /* ]]; then
    printf 'installation prefix must be absolute\n' >&2
    exit 1
fi

(cd "${buildopt_package_root}" && shasum -a 256 -c SHA256SUMS >/dev/null)
mkdir -p -- "${buildopt_prefix}/bin" "${buildopt_prefix}/share/buildopt"
for buildopt_binary in buildopt buildopt-impact buildopt-server buildopt-edge; do
    buildopt_temporary="${buildopt_prefix}/bin/.${buildopt_binary}.new.$$"
    install -m 0755 "${buildopt_package_root}/bin/${buildopt_binary}" "${buildopt_temporary}"
    mv -f -- "${buildopt_temporary}" "${buildopt_prefix}/bin/${buildopt_binary}"
done
printf '%s\n' \
    'buildopt.install/v1' \
	'bin/buildopt' \
	'bin/buildopt-impact' \
	'bin/buildopt-server' \
	'bin/buildopt-edge' \
    >"${buildopt_prefix}/share/buildopt/receipt"
printf 'BuildOpt installed in %s/bin\n' "${buildopt_prefix}"
