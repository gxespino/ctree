# CTree

A NERDTree-style sidebar for managing Claude Code sessions in tmux.

CTree shows all your Claude Code sessions in a sidebar panel, with real-time status detection, git info, and a preview pane — so you can monitor multiple Claude agents without leaving your workflow.

## Features

- **Real-time status detection** — hooks into Claude Code lifecycle events (Working, Needs Input, Idle, Unread, Done)
- **Preview pane** — peek at any session's output without switching to it (`p` to toggle)
- **Git integration** — shows branch name and diff stats for each session
- **Global sidebar** — toggle opens/closes in all tmux windows simultaneously
- **Jump to unread** — quickly switch to the session that needs your attention (`tab`)
- **Bell notifications** — chime when a session finishes or needs input (`m` to mute)

## Status Indicators

| Status | Color | Meaning |
|--------|-------|---------|
| **Working...** | Yellow | Claude is actively processing |
| **Needs Input** | Orange | Claude is waiting for your input (permission, question) |
| **Unread** | Blue | Claude finished — you haven't looked yet |
| **Done** | Green | You've seen the output |
| **Idle** | Gray | At prompt, nothing happening |
| **Exited** | Dim | Session ended |

## Installation

### Prerequisites

- [Go 1.25+](https://go.dev/dl/)
- [tmux](https://github.com/tmux/tmux)
- [Claude Code](https://docs.anthropic.com/en/docs/claude-code)

### Build & install

```bash
git clone https://github.com/gxespino/ctree.git
cd ctree
make install
```

This installs `ctree`, `ctree-toggle`, and `ctree-auto-open` to `~/.local/bin`. Make sure `~/.local/bin` is in your `PATH`.

### Register Claude Code hooks

CTree needs hooks to detect session status in real-time. Run:

```bash
ctree setup
```

This writes hook entries into `~/.claude/settings.json` for the following Claude Code lifecycle events: `UserPromptSubmit`, `Stop`, `Notification`, `PermissionRequest`, `PostToolUse`, and `SessionEnd`. Running `ctree setup` again is safe — it replaces existing ctree hooks without duplicating them.

### tmux configuration

Add to `~/.tmux.conf`:

```tmux
# Toggle CTree sidebar with prefix + p
bind-key p run-shell "~/.local/bin/ctree-toggle"
```

Then reload: `tmux source-file ~/.tmux.conf`

`prefix + p` toggles the sidebar in all tmux windows simultaneously.

#### Auto-open in new windows (optional)

The sidebar auto-opens in new tmux windows while it's active — no extra config needed. This is handled automatically by `ctree-toggle` via a tmux `after-new-window` hook.

## Keybindings

| Key | Action |
|-----|--------|
| `j/k` | Navigate up/down |
| `enter` | Jump to selected session |
| `tab` | Jump to most recent unread/paused session |
| `p` | Toggle preview pane |
| `m` | Toggle bell notifications (mute/unmute) |
| `n` | Create new Claude workspace |
| `r` | Refresh |
| `/` | Filter sessions |
| `q` / `esc` | Quit |

## How It Works

CTree uses Claude Code's [hooks system](https://docs.anthropic.com/en/docs/claude-code/hooks) to detect session status in real-time:

- **UserPromptSubmit** → Working (user sent a prompt)
- **PostToolUse** → Working (tool completed, Claude continues)
- **PermissionRequest** → Needs Input (waiting for tool approval)
- **Notification** → Idle or Needs Input (depending on notification type)
- **Stop** → Idle (Claude finished responding)
- **SessionEnd** → Exited

Process liveness is verified via the process tree on each poll cycle (250ms).

## Configuration

CTree is zero-config by design. The few toggleable settings persist automatically:

| Setting | Toggle | Persisted at |
|---------|--------|-------------|
| Preview pane | `p` | `~/.config/ctree/preview` |
| Bell mute | `m` | `~/.config/ctree/bell-muted` |
| Sidebar width | `CTREE_SIDEBAR_WIDTH` env var | — |

All ctree instances sync toggle state from disk, so changes propagate across windows.

## License

[AGPL-3.0-or-later](LICENSE)
