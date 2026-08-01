#!/usr/bin/env bash
set -euo pipefail
: "${BUILDOPT_CANCEL_PID_FILE:?}"
sleep 300 &
buildopt_child=$!
printf '%s\n' "${buildopt_child}" >"${BUILDOPT_CANCEL_PID_FILE}"
wait "${buildopt_child}"
