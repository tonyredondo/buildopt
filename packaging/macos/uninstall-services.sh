#!/usr/bin/env bash
set -euo pipefail

buildopt_agents=${HOME}/Library/LaunchAgents
if (($# == 2)) && [[ "$1" == --launch-agents-dir ]]; then
    buildopt_agents=$2
elif (($# != 0)); then
    printf 'usage: %s [--launch-agents-dir ABSOLUTE]\n' "$0" >&2
    exit 64
fi
[[ "${buildopt_agents}" == /* ]] || { printf 'launch agents path must be absolute\n' >&2; exit 64; }
for buildopt_label in com.tonyredondo.buildopt.server com.tonyredondo.buildopt.edge; do
    buildopt_plist="${buildopt_agents}/${buildopt_label}.plist"
    if [[ -f "${buildopt_plist}" ]]; then
        launchctl bootout "gui/$(id -u)" "${buildopt_plist}" 2>/dev/null || true
        rm -f -- "${buildopt_plist}"
    fi
done
