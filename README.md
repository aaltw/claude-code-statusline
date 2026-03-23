# claude-code-statusline

A [Claude Code](https://claude.ai/code) statusline in [Catppuccin Mocha](https://github.com/catppuccin/catppuccin) with powerline arrows.

## What it shows

**Left side — Workspace**
```
 feature/auth ✦ ↑3   +42 -7
```
- Git branch + dirty indicator `✦`, commits ahead `↑` / behind `↓` upstream (cached 5s)
- Lines added/removed in the session

**Right side — AI metrics**
```
 42%/1M ▓▓▓▓▒▒▒▒▒▒▒ ↑15.2k/120.5k ↓1.8k/22.0k ~8.7k  Opus 4.6
```
- Context usage percentage + window size (200k or 1M)
- Progress bar: color-coded green / yellow / red at 70% / 90%
- Token breakdown: input `↑` (current/total), output `↓` (current/total), cached `~`
- Model name (strips "Claude " prefix)
- `⚡ COMPACT` warning when context hits 85%+

Adapts to a two-line layout when the terminal is too narrow.

## Versions

### Go (recommended)

Compiled binary with faster startup and no runtime dependencies beyond `git`.

#### Install from release

Download the binary for your platform from [Releases](https://github.com/aaltw/claude-code-statusline/releases):

```bash
# macOS Apple Silicon
curl -L -o ~/.claude/statusline-bin \
  https://github.com/aaltw/claude-code-statusline/releases/latest/download/statusline-bin-darwin-arm64
chmod +x ~/.claude/statusline-bin
```

#### Build from source

```bash
git clone https://github.com/aaltw/claude-code-statusline.git
cd claude-code-statusline
make install
```

Or manually:

```bash
cd go && go build -ldflags "-s -w" -o ~/.claude/statusline-bin .
```

#### Configure

Add to `~/.claude/settings.json`:

```json
{
  "statusLine": {
    "type": "command",
    "command": "~/.claude/statusline-bin"
  }
}
```

### Bash (lightweight)

Shell script with no build step. Requires `jq`.

```bash
curl -o ~/.claude/statusline.sh \
  https://raw.githubusercontent.com/aaltw/claude-code-statusline/main/statusline.sh
chmod +x ~/.claude/statusline.sh
```

Add to `~/.claude/settings.json`:

```json
{
  "statusLine": {
    "type": "command",
    "command": "~/.claude/statusline.sh"
  }
}
```

## Tmux window icons

The `hooks/claude-tmux-status.sh` script sets tmux window name icons to show Claude Code's state:

| Icon | State |
|------|-------|
| `○`  | Idle / waiting for input |
| `◐`  | Working (tool use / notification) |

It preserves and restores the original window name and `automatic-rename` setting.

### Install

```bash
curl -o ~/.claude/hooks/claude-tmux-status.sh \
  https://raw.githubusercontent.com/aaltw/claude-code-statusline/main/hooks/claude-tmux-status.sh
chmod +x ~/.claude/hooks/claude-tmux-status.sh
```

Add hooks to `~/.claude/settings.json`:

```json
{
  "hooks": {
    "Stop": [
      {
        "type": "command",
        "command": "~/.claude/hooks/claude-tmux-status.sh stop"
      }
    ],
    "Notification": [
      {
        "type": "command",
        "command": "~/.claude/hooks/claude-tmux-status.sh notification"
      }
    ],
    "PreToolUse": [
      {
        "type": "command",
        "command": "~/.claude/hooks/claude-tmux-status.sh pretooluse"
      }
    ]
  }
}
```

## Requirements

- [Nerd Fonts](https://www.nerdfonts.com/) in your terminal (powerline arrows + `󰧑` icon)
- `git` (optional, for the branch segment)
- **Go version**: Go 1.21+ (build only)
- **Bash version**: `jq`

## Theme

Colors use [Catppuccin Mocha](https://github.com/catppuccin/catppuccin). The palette is defined at the top of each implementation, making it easy to swap to another flavour.

| Segment       | Color   | Hex       |
|---------------|---------|-----------|
| Model         | Blue    | `#89b4fa` |
| Input tokens  | Green   | `#a6e3a1` |
| Output tokens | Yellow  | `#f9e2af` |
| Cached tokens | Teal    | `#94e2d5` |
| Bar (low)     | Green   | `#a6e3a1` |
| Bar (medium)  | Yellow  | `#f9e2af` |
| Bar (high)    | Red     | `#f38ba8` |
| Compact warn  | Red     | `#f38ba8` |
| Git branch    | Pink    | `#f5c2e7` |
| Lines changed | Green/Red | `#a6e3a1` / `#f38ba8` |
