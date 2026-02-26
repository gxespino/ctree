#!/usr/bin/env bash
# cmux-toggle.sh - Toggle cmux sidebar in ALL tmux windows
#
# Usage: Add to ~/.tmux.conf:
#   bind-key p run-shell "/path/to/cmux-toggle"

set -euo pipefail

CMUX_PANE_TITLE="cmux-sidebar"
SIDEBAR_WIDTH="${CMUX_SIDEBAR_WIDTH:-40}"
SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
cmux_bin="${CMUX_BIN:-${SCRIPT_DIR}/cmux}"
auto_open="${SCRIPT_DIR}/cmux-auto-open"

# Collect all cmux-sidebar pane IDs across every window
sidebar_panes=()
while read -r pane_id pane_title; do
    if [ "$pane_title" = "$CMUX_PANE_TITLE" ]; then
        sidebar_panes+=("$pane_id")
    fi
done < <(tmux list-panes -a -F '#{pane_id} #{pane_title}' 2>/dev/null)

if [ ${#sidebar_panes[@]} -gt 0 ]; then
    # Sidebars exist — kill them all (toggle off)
    for pane_id in "${sidebar_panes[@]}"; do
        tmux kill-pane -t "$pane_id" 2>/dev/null || true
    done
    tmux set-hook -gu after-new-window

    # Kill any orphaned cmux processes that survived pane destruction
    pkill -f "^${cmux_bin}$" 2>/dev/null || true
else
    # No sidebars — open one in every window
    current_window=$(tmux display-message -p '#{window_id}')

    while read -r window_id; do
        # Select the target window, create the sidebar, then move back
        tmux split-window -hb -t "$window_id" -l "$SIDEBAR_WIDTH" \
            "printf '\\033]2;${CMUX_PANE_TITLE}\\033\\\\'; exec ${cmux_bin}"
    done < <(tmux list-windows -a -F '#{window_id}')

    # Restore focus to the original window and its non-sidebar pane
    tmux select-window -t "$current_window"
    focus_pane=$(tmux list-panes -F '#{pane_id} #{pane_title}' | grep -v "$CMUX_PANE_TITLE" | head -1 | cut -d' ' -f1)
    if [ -n "$focus_pane" ]; then
        tmux select-pane -t "$focus_pane"
    fi

    # Set hook so new windows also get a sidebar
    tmux set-hook -g after-new-window "run-shell '${auto_open}'"
fi
