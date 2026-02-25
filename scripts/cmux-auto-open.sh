#!/usr/bin/env bash
# cmux-auto-open.sh - Called by tmux after-new-window hook.
# Opens a cmux sidebar in the new window if one exists in any other window.

set -euo pipefail

CMUX_PANE_TITLE="cmux-sidebar"
SIDEBAR_WIDTH="${CMUX_SIDEBAR_WIDTH:-40}"

# Check if the current (new) window already has a sidebar
current_has=$(tmux list-panes -F '#{pane_title}' 2>/dev/null | grep -c "^${CMUX_PANE_TITLE}$" || true)
if [ "$current_has" -gt 0 ]; then
    exit 0
fi

# Check if any other window has a sidebar
any_has=$(tmux list-panes -a -F '#{pane_title}' 2>/dev/null | grep -c "^${CMUX_PANE_TITLE}$" || true)
if [ "$any_has" -eq 0 ]; then
    exit 0
fi

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
cmux_bin="${CMUX_BIN:-${SCRIPT_DIR}/cmux}"
tmux split-window -hb -l "$SIDEBAR_WIDTH" \
    "printf '\\033]2;${CMUX_PANE_TITLE}\\033\\\\'; exec ${cmux_bin}"
