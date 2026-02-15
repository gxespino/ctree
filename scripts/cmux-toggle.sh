#!/usr/bin/env bash
# cmux-toggle.sh - Toggle cmux sidebar pane in tmux
#
# Usage: Add to ~/.tmux.conf:
#   bind-key p run-shell "/path/to/cmux-toggle"

set -euo pipefail

CMUX_PANE_TITLE="cmux-sidebar"
SIDEBAR_WIDTH="${CMUX_SIDEBAR_WIDTH:-40}"

# Find existing cmux pane in current window by checking pane title
find_cmux_pane() {
    tmux list-panes -F '#{pane_id} #{pane_title}' 2>/dev/null | \
        while read -r pane_id pane_title; do
            if [ "$pane_title" = "$CMUX_PANE_TITLE" ]; then
                echo "$pane_id"
                return 0
            fi
        done
    return 1
}

cmux_pane=$(find_cmux_pane || true)

if [ -n "$cmux_pane" ]; then
    # Sidebar exists - kill it (toggle off)
    tmux kill-pane -t "$cmux_pane"
else
    # Create sidebar pane on the left
    # -b: place before (left of) current pane
    # -h: horizontal split
    # -l: width in columns
    SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
    cmux_bin="${CMUX_BIN:-${SCRIPT_DIR}/cmux}"
    tmux split-window -hb -l "$SIDEBAR_WIDTH" \
        "printf '\\033]2;${CMUX_PANE_TITLE}\\033\\\\'; exec ${cmux_bin}"
fi
