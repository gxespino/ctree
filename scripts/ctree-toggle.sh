#!/usr/bin/env bash
# ctree-toggle.sh - Toggle CTree sidebar in ALL tmux windows
#
# Usage: Add to ~/.tmux.conf:
#   bind-key p run-shell "/path/to/ctree-toggle"

set -euo pipefail

CTREE_PANE_TITLE="ctree-sidebar"
SIDEBAR_WIDTH="${CTREE_SIDEBAR_WIDTH:-40}"
SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
ctree_bin="${CTREE_BIN:-${SCRIPT_DIR}/ctree}"
auto_open="${SCRIPT_DIR}/ctree-auto-open"

# Collect all ctree-sidebar pane IDs across every window
sidebar_panes=()
while read -r pane_id pane_title; do
    if [ "$pane_title" = "$CTREE_PANE_TITLE" ]; then
        sidebar_panes+=("$pane_id")
    fi
done < <(tmux list-panes -a -F '#{pane_id} #{pane_title}' 2>/dev/null)

if [ ${#sidebar_panes[@]} -gt 0 ]; then
    # Sidebars exist — kill them all (toggle off)
    for pane_id in "${sidebar_panes[@]}"; do
        tmux kill-pane -t "$pane_id" 2>/dev/null || true
    done
    tmux set-hook -gu after-new-window

    # Kill any orphaned ctree processes that survived pane destruction
    pkill -f "^${ctree_bin}$" 2>/dev/null || true
else
    # No sidebars — open one in every window
    current_window=$(tmux display-message -p '#{window_id}')

    while read -r window_id; do
        tmux split-window -hb -t "$window_id" -l "$SIDEBAR_WIDTH" \
            "printf '\\033]2;${CTREE_PANE_TITLE}\\033\\\\'; exec ${ctree_bin}"
        # Focus back to the main (non-sidebar) pane in this window
        main_pane=$(tmux list-panes -t "$window_id" -F '#{pane_id} #{pane_title}' | grep -v "$CTREE_PANE_TITLE" | head -1 | cut -d' ' -f1)
        if [ -n "$main_pane" ]; then
            tmux select-pane -t "$main_pane"
        fi
    done < <(tmux list-windows -a -F '#{window_id}')

    # Restore focus to the original window and its non-sidebar pane
    tmux select-window -t "$current_window"
    focus_pane=$(tmux list-panes -F '#{pane_id} #{pane_title}' | grep -v "$CTREE_PANE_TITLE" | head -1 | cut -d' ' -f1)
    if [ -n "$focus_pane" ]; then
        tmux select-pane -t "$focus_pane"
    fi

    # Set hook so new windows also get a sidebar
    tmux set-hook -g after-new-window "run-shell '${auto_open}'"
fi
