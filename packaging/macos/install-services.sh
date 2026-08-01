#!/usr/bin/env bash
set -euo pipefail

buildopt_prefix=${HOME}/.local
buildopt_agents=${HOME}/Library/LaunchAgents
buildopt_server_config=
buildopt_edge_config=
buildopt_load=0
while (($# > 0)); do
    case "$1" in
        --prefix) buildopt_prefix=$2; shift 2 ;;
        --launch-agents-dir) buildopt_agents=$2; shift 2 ;;
        --server-config) buildopt_server_config=$2; shift 2 ;;
        --edge-config) buildopt_edge_config=$2; shift 2 ;;
        --load) buildopt_load=1; shift ;;
        *) printf 'usage: %s [--prefix ABSOLUTE] [--launch-agents-dir ABSOLUTE] [--server-config ABSOLUTE] [--edge-config ABSOLUTE] [--load]\n' "$0" >&2; exit 64 ;;
    esac
done
if [[ "${buildopt_prefix}" != /* || "${buildopt_agents}" != /* ]] ||
    [[ -z "${buildopt_server_config}" && -z "${buildopt_edge_config}" ]]; then
    printf 'service paths must be absolute and at least one config is required\n' >&2
    exit 64
fi

buildopt_xml_escape() {
    sed -e 's/&/\&amp;/g' -e 's/</\&lt;/g' -e 's/>/\&gt;/g' -e 's/"/\&quot;/g' <<<"$1"
}

buildopt_install_agent() {
    local buildopt_label=$1
    local buildopt_binary=$2
    local buildopt_flag=$3
    local buildopt_config=$4
    [[ "${buildopt_config}" == /* && -f "${buildopt_config}" && -x "${buildopt_binary}" ]] || {
        printf 'service binary or config is unavailable for %s\n' "${buildopt_label}" >&2
        exit 1
    }
    local buildopt_target="${buildopt_agents}/${buildopt_label}.plist"
    local buildopt_temporary="${buildopt_target}.new.$$"
    {
        printf '%s\n' '<?xml version="1.0" encoding="UTF-8"?>' '<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">' '<plist version="1.0"><dict>'
        printf '<key>Label</key><string>%s</string>\n' "$(buildopt_xml_escape "${buildopt_label}")"
        printf '<key>ProgramArguments</key><array><string>%s</string><string>serve</string><string>%s</string><string>%s</string></array>\n' \
            "$(buildopt_xml_escape "${buildopt_binary}")" "$(buildopt_xml_escape "${buildopt_flag}")" "$(buildopt_xml_escape "${buildopt_config}")"
        printf '%s\n' '<key>RunAtLoad</key><true/><key>KeepAlive</key><true/><key>ProcessType</key><string>Background</string><key>Umask</key><integer>63</integer></dict></plist>'
    } >"${buildopt_temporary}"
    plutil -lint "${buildopt_temporary}" >/dev/null
    chmod 0600 "${buildopt_temporary}"
    mv -f "${buildopt_temporary}" "${buildopt_target}"
    if ((buildopt_load)); then
        launchctl bootout "gui/$(id -u)" "${buildopt_target}" 2>/dev/null || true
        launchctl bootstrap "gui/$(id -u)" "${buildopt_target}"
    fi
    printf '%s\n' "${buildopt_target}"
}

mkdir -p "${buildopt_agents}"
if [[ -n "${buildopt_server_config}" ]]; then
    buildopt_install_agent com.tonyredondo.buildopt.server "${buildopt_prefix}/bin/buildopt-server" --self-hosted-config "${buildopt_server_config}"
fi
if [[ -n "${buildopt_edge_config}" ]]; then
    "${buildopt_prefix}/bin/buildopt-edge" validate --config "${buildopt_edge_config}" >/dev/null
    buildopt_install_agent com.tonyredondo.buildopt.edge "${buildopt_prefix}/bin/buildopt-edge" --config "${buildopt_edge_config}"
fi
