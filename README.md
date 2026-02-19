# claude-code-statusline

A two-line [Claude Code](https://claude.ai/code) statusline in [Catppuccin Mocha](https://github.com/catppuccin/catppuccin) with powerline arrows.

## What it shows

**Line 1 — AI metrics**
```
󰧑 Sonnet 4.6  ↑15.2k ↓1.8k ~8.7k  ████░░░░░░ 42%/200k
```
- Model name (strips the "Claude " prefix)
- Token breakdown: input `↑`, output `↓`, cached `~`
- Context bar: 10-char block progress, color-coded green → yellow → red at 70% / 90%
- Context size: shows your actual window limit (200k or 1M)
- `⚡ COMPACT` warning when context hits 85%+

**Line 2 — Workspace**
```
 feature/auth ✦ ↑3                              ~/projects/myapp
```
- Git branch + dirty indicator `✦`, commits ahead `↑` / behind `↓` upstream (cached 5s)
- Current directory right-aligned, `~`-prefixed, truncated to `…/parent/dir` if long

## Requirements

- [Nerd Fonts](https://www.nerdfonts.com/) in your terminal (for the powerline arrows and `󰧑` icon)
- `jq`
- `git` (optional, for the branch segment)

## Install

```bash
curl -o ~/.claude/statusline.sh \
  https://raw.githubusercontent.com/aaltw/claude-code-statusline/main/statusline.sh
chmod +x ~/.claude/statusline.sh
```

Then add to `~/.claude/settings.json`:

```json
{
  "statusLine": {
    "type": "command",
    "command": "~/.claude/statusline.sh"
  }
}
```

## Theme

Colors are hardcoded to [Catppuccin Mocha](https://github.com/catppuccin/catppuccin). The palette is defined at the top of the script as `R;G;B` values, so swapping to another flavour or theme is straightforward.

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
| CWD           | Mauve   | `#cba6f7` |
