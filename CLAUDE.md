# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Commands

```bash
go build -o statusline .       # Build the binary
go build -o statusline . && echo '{"model":{"display_name":"Claude Sonnet 4.6"},"context_window":{"used_percentage":42,"remaining_percentage":58,"context_window_size":200000,"current_usage":{"input_tokens":1200,"output_tokens":400,"cache_read_input_tokens":300,"cache_creation_input_tokens":100},"total_input_tokens":5000,"total_output_tokens":1500},"cost":{"total_lines_added":10,"total_lines_removed":3},"session_id":"test","cwd":"."}' | ./statusline   # Build and test with sample input
go vet ./...                   # Static analysis (no linter config present)
```

## Architecture

Single-file Go binary (`main.go`, ~450 lines, no external dependencies) that reads a JSON blob from stdin and writes an ANSI-colored Powerline-style statusline to stdout.

### Data flow

1. **Input**: Claude Code pipes a `StatusInput` JSON struct to stdin containing model name, context window stats, token counts, cost/line-change data, rate limits, cwd, and session ID.
2. **Git**: `getGitInfo()` queries git for branch/dirty/ahead/behind with a 5-second file cache at `/tmp/claude-statusline-git-<sessionID>`.
3. **Rendering**: Segments are composed as `[]Segment{BG Color, Text string}` and passed to `renderLeft()` / `renderRight()` which emit Powerline separators (`\ue0b0` / `\ue0b2`) between them.
4. **Layout**: `termWidth()` checks `$COLUMNS`, then tmux, then tput. If left+right segments fit on one line they're joined with a spacer; otherwise they wrap to two lines.
5. **Bridge files**: On each render, two JSON files are written to `/tmp/`:
   - `claude-ctx-<sessionID>.json` — context remaining % for a PostToolUse hook that warns when context is low
   - `claude-monitor-<sessionID>.json` — full token/rate-limit metrics for an external monitor

### Segment composition (right-to-left reading order in source)

Right segments (always shown): `ctxSeg` (percent + window size) → `barSeg` (10-block bar) → `tokensSeg` (input/output/cache counts) → `modelSeg` (model name)

Left segments (conditional): git branch+dirty+ahead/behind → lines added/removed → COMPACT warning (when context ≥ 85%)

### Colors

Catppuccin Mocha palette defined as `Color{R, G, B}` with `.FG()` / `.BG()` helpers that emit true-color ANSI escape codes. Bar color thresholds: green < 70%, yellow < 90%, red ≥ 90%.

### Integration with Claude Code

The binary is invoked by Claude Code's statusline hook. The `.claude/settings.local.json` restricts allowed bash commands to `git:*` only (used by the git info queries).
