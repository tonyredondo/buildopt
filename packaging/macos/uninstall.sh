#!/usr/bin/env bash
set -euo pipefail

buildopt_prefix=${HOME}/.local
if (($# == 2)) && [[ "$1" == "--prefix" ]]; then
    buildopt_prefix=$2
elif (($# != 0)); then
    printf 'usage: %s [--prefix <directory>]\n' "$0" >&2
    exit 64
fi
buildopt_receipt="${buildopt_prefix}/share/buildopt/receipt"
if [[ ! -f "${buildopt_receipt}" ]] || [[ "$(sed -n '1p' "${buildopt_receipt}")" != 'buildopt.install/v1' ]]; then
    printf 'BuildOpt installation receipt is missing or invalid\n' >&2
    exit 1
fi
while IFS= read -r buildopt_relative; do
    [[ "${buildopt_relative}" == 'buildopt.install/v1' ]] && continue
    case "${buildopt_relative}" in
        bin/buildopt|bin/buildopt-impact) rm -f -- "${buildopt_prefix}/${buildopt_relative}" ;;
        *) printf 'unsafe BuildOpt receipt entry: %s\n' "${buildopt_relative}" >&2; exit 1 ;;
    esac
done <"${buildopt_receipt}"
rm -f -- "${buildopt_receipt}"
rmdir "${buildopt_prefix}/share/buildopt" 2>/dev/null || true
printf 'BuildOpt removed from %s\n' "${buildopt_prefix}"
