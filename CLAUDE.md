# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Commands

```bash
go build -o statusline .       # Build the binary
go build -o statusline . && echo '<json>' | ./statusline   # Build and render (see smoke test input below)
go vet ./...                   # Static analysis
```

Smoke test input (pipe to `./statusline`):
```json
{"model":{"display_name":"Claude Sonnet 4.6"},"context_window":{"used_percentage":42,"remaining_percentage":58,"context_window_size":200000,"current_usage":{"input_tokens":1200,"output_tokens":400,"cache_read_input_tokens":300,"cache_creation_input_tokens":100},"total_input_tokens":5000,"total_output_tokens":1500},"cost":{"total_lines_added":10,"total_lines_removed":3},"session_id":"test","cwd":"."}
```

Theme selection via env var (default: arrow):
```bash
STATUSLINE_THEME=round   ./statusline   # rounded separators
STATUSLINE_THEME=slant   ./statusline   # slanted separators
STATUSLINE_THEME=flame   ./statusline   # flame separators
STATUSLINE_THEME=plain   ./statusline   # plain │ dividers
```

## Architecture

Single-package Go binary (no external dependencies, Go 1.22+) that reads a JSON blob from stdin and writes an ANSI-colored Powerline-style statusline to stdout.

### File map

| File | Responsibility |
|---|---|
| `main.go` | `main()` only — decodes input, resolves theme, builds segments, renders, writes bridge files |
| `input.go` | `StatusInput`, `Model`, `ContextWindow`, `CurrentUsage`, `Cost` JSON structs |
| `color.go` | `Color` type with `.FG()`/`.BG()`, `Palette` struct, `CatppuccinMocha` var, `reset` const |
| `theme.go` | `Theme` struct, predefined themes (`ThemeArrow`…`ThemePlain`), `activeTheme()` |
| `segment.go` | `Segment` struct, `renderLeft`, `renderRight`, `segmentsWidth` |
| `git.go` | `GitInfo`, `getGitInfo` (5-second file cache), `writeGitCache` |
| `layout.go` | `fmtTokens`, `visibleWidth`, `termWidth` |
| `bridge.go` | `writeMonitorBridge`, `writeCtxBridge`, `updateCacheAccum` |
| `ratelimit.go` | 5h/7d sampling + projection: `updateRateHistory` (account-global history in `$TMPDIR/claude-ratelimit-history.json`), `slopePerSec`, `projectedAtReset`, `windowColor`, `burnArrow` (5h); `workingDayBudget`, `budgetColor` (7d) |

### Data flow

1. **Input**: Claude Code pipes a `StatusInput` JSON struct to stdin.
2. **Theme**: `activeTheme()` reads `STATUSLINE_THEME` and returns a `Theme` with separator chars and a `Palette`.
3. **Segments**: `main()` builds `[]Segment` for left side (git branch, lines changed, COMPACT warning) and right side (context bar, tokens, model name). Color references use `p := theme.Colors`.
4. **Rendering**: `renderLeft`/`renderRight` in `segment.go` emit Powerline separators from `theme.SepRight`/`theme.SepLeft`.
5. **Layout**: `termWidth()` checks `$COLUMNS` → tmux → tput → 80. Single-line layout if segments fit; two-line otherwise.
6. **Bridge files** written to `/tmp/` each render:
   - `claude-ctx-<sessionID>.json` — context remaining % for a PostToolUse hook
   - `claude-monitor-<sessionID>.json` — full token/rate-limit metrics

### Theming

`Theme` holds `SepRight`, `SepLeft` (Nerd Font glyphs), and a `Palette` (15 named color slots). Adding a new theme: add a `var ThemeXxx = Theme{...}` in `theme.go` and a `case` in `activeTheme()`. Colors use Catppuccin Mocha by default; swap `Colors` in a theme var to use a different palette.

### Context bar color thresholds

- Green: < 70% used
- Yellow: 70–89% used
- Red: ≥ 90% used
- COMPACT warning shown on left side at ≥ 85%

### Rate-limit windows (5h / 7d)

Rendered automatically when Claude Code passes `rate_limits` (`resets_at > 0`); hidden otherwise.

**5h** (`5h used%/budget% ↗`) — the `used%` is colored by *projected* usage at reset, extrapolated
from the recent burn-rate slope (`ratelimit.go`, persisted history): green < 90%, Peach 90–100%,
red ≥ 100% (will exhaust before reset). The `budget%` is the even-pace allowance: the fraction of
the 5-hour window elapsed (spending 100% evenly across 5h lands you exactly there now), so
`used < budget` means you're under the even burn line. The **arrow** is colored independently by the burn *rate* itself
(steepness rises with rate): `→` green steady (≤ ~1.8%/h, idle), `↗` Peach acceptable (up to 20%/h,
the full-window pace), `↑` red too-high (> 20%/h, faster than the window lasts).

**7d** (`7d used%/budget%`) — colored against a working-day pace budget. A 5-day work-week
allows 20%/working-day, so the cumulative budget you're "allowed" by today is 20% × (Mon–Fri days
from the window's start through today, capped at 100%). Color is `used / budget`: green < 90% of
budget, Peach 90–100%, red ≥ 100% (at/over the plan). No trend arrow.

### Env flags

- `STATUSLINE_THEME` — separator/palette theme (see above)
- `STATUSLINE_CACHE_EFFICIENCY=1` — enable the `󰆼 %` cache-hit and `↗ Nx` waste meters
- `STATUSLINE_TOKEN_COUNTERS=1` — re-enable the `↑in/total ↓out/total` token counters (off by
  default; the `~cache` figure always shows)

### Git integration

`getGitInfo()` queries branch, dirty state, ahead/behind via `git` subprocesses. Results cached in `/tmp/claude-statusline-git-<sessionID>` for 5 seconds to avoid hammering git on every render.
