# Split & Theming Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Split the monolithic `main.go` into focused files and introduce a `Theme` struct that lets users switch Powerline separator styles via the `STATUSLINE_THEME` env var.

**Architecture:** Each logical concern moves to its own file within the same `main` package. A `Theme` struct holds separator characters and a `Palette` (named color slots). `renderLeft`/`renderRight` accept a `Theme` instead of using hardcoded constants. The active theme is resolved once in `main()` and threaded through.

**Tech Stack:** Go 1.26, stdlib only, single `main` package.

---

## File Map

| File | Responsibility |
|---|---|
| `input.go` | `StatusInput`, `Model`, `ContextWindow`, `CurrentUsage`, `Cost` structs |
| `color.go` | `Color` type + `.FG()`/`.BG()`, `Palette` struct, `CatppuccinMocha` var, `reset` const |
| `theme.go` | `Theme` struct, predefined themes (`ThemeArrow`, `ThemeRound`, `ThemeSlant`, `ThemeFlame`, `ThemePlain`), `activeTheme()` |
| `segment.go` | `Segment` struct, `renderLeft(segs, theme, b)`, `renderRight(segs, theme, b)`, `segmentsWidth` |
| `git.go` | `GitInfo`, `gitCachePath`, `getGitInfo`, `writeGitCache` |
| `layout.go` | `ansiRE`, `visibleWidth`, `fmtTokens`, `termWidth` |
| `bridge.go` | `writeMonitorBridge` |
| `main.go` | `main()` only — decodes input, resolves theme, builds segments, renders, writes bridge files |

---

### Task 1: Extract input structs → `input.go`

**Files:**
- Create: `input.go`
- Modify: `main.go` (remove the structs)

- [ ] **Step 1: Create `input.go`**

```go
package main

type StatusInput struct {
	Model         Model         `json:"model"`
	ContextWindow ContextWindow `json:"context_window"`
	Cost          Cost          `json:"cost"`
	Exceeds200k   bool          `json:"exceeds_200k_tokens"`
	Cwd           string        `json:"cwd"`
	SessionID     string        `json:"session_id"`
	RateLimits    struct {
		FiveHour struct {
			UsedPercentage float64 `json:"used_percentage"`
			ResetsAt       float64 `json:"resets_at"`
		} `json:"five_hour"`
		SevenDay struct {
			UsedPercentage float64 `json:"used_percentage"`
			ResetsAt       float64 `json:"resets_at"`
		} `json:"seven_day"`
	} `json:"rate_limits"`
}

type Model struct {
	DisplayName string `json:"display_name"`
}

type ContextWindow struct {
	UsedPercentage      float64      `json:"used_percentage"`
	RemainingPercentage float64      `json:"remaining_percentage"`
	ContextWindowSize   int          `json:"context_window_size"`
	CurrentUsage        CurrentUsage `json:"current_usage"`
	TotalInputTokens    int          `json:"total_input_tokens"`
	TotalOutputTokens   int          `json:"total_output_tokens"`
}

type CurrentUsage struct {
	InputTokens              int `json:"input_tokens"`
	OutputTokens             int `json:"output_tokens"`
	CacheReadInputTokens     int `json:"cache_read_input_tokens"`
	CacheCreationInputTokens int `json:"cache_creation_input_tokens"`
}

type Cost struct {
	TotalLinesAdded   int `json:"total_lines_added"`
	TotalLinesRemoved int `json:"total_lines_removed"`
}
```

- [ ] **Step 2: Remove those structs from `main.go`** (lines 18–60, the block under `// ── JSON input ──`)

- [ ] **Step 3: Verify build**

```bash
go build ./...
```
Expected: no errors.

- [ ] **Step 4: Commit**

```bash
git add input.go main.go
git commit -m "refactor: extract input structs to input.go"
```

---

### Task 2: Extract colors → `color.go`

**Files:**
- Create: `color.go`
- Modify: `main.go` (remove Color type, palette vars, reset const, sepR/sepL consts)

- [ ] **Step 1: Create `color.go`**

```go
package main

import "fmt"

// Color holds an RGB triple and emits true-color ANSI escape codes.
type Color struct{ R, G, B int }

func (c Color) FG() string { return fmt.Sprintf("\033[38;2;%d;%d;%dm", c.R, c.G, c.B) }
func (c Color) BG() string { return fmt.Sprintf("\033[48;2;%d;%d;%dm", c.R, c.G, c.B) }

const reset = "\033[0m"

// Palette maps semantic color roles to RGB values.
// A Theme carries one Palette, allowing different color schemes.
type Palette struct {
	Base      Color
	Surface0  Color
	Surface1  Color
	Blue      Color
	Green     Color
	Green80   Color
	GreenDim  Color
	Yellow    Color
	Yellow80  Color
	YellowDim Color
	Red       Color
	Red80     Color
	RedDim    Color
	Pink      Color
	Teal      Color
}

// CatppuccinMocha is the default palette.
var CatppuccinMocha = Palette{
	Base:      Color{30, 30, 46},
	Surface0:  Color{49, 50, 68},
	Surface1:  Color{69, 71, 90},
	Blue:      Color{137, 180, 250},
	Green:     Color{166, 227, 161},
	Green80:   Color{133, 182, 129},
	GreenDim:  Color{83, 113, 80},
	Yellow:    Color{249, 226, 175},
	Yellow80:  Color{199, 181, 140},
	YellowDim: Color{124, 113, 87},
	Red:       Color{243, 139, 168},
	Red80:     Color{194, 111, 134},
	RedDim:    Color{121, 69, 84},
	Pink:      Color{245, 194, 231},
	Teal:      Color{148, 226, 213},
}
```

- [ ] **Step 2: Remove from `main.go`:**
  - `type Color struct{ R, G, B int }` and its methods
  - `var (base = Color{...} ... teal = Color{...})` block
  - `const (sepR = ... sepL = ... reset = ...)`

- [ ] **Step 3: Verify build**

```bash
go build ./...
```

- [ ] **Step 4: Commit**

```bash
git add color.go main.go
git commit -m "refactor: extract Color type and Catppuccin palette to color.go"
```

---

### Task 3: Introduce theming → `theme.go`

**Files:**
- Create: `theme.go`
- Modify: `main.go` (remove `sepR`/`sepL` usage — handled in next task via theme)

- [ ] **Step 1: Create `theme.go`**

```go
package main

import (
	"os"
	"strings"
)

// Theme controls separator glyphs and the color palette.
type Theme struct {
	SepRight string // separator between left-side segments (points right)
	SepLeft  string // separator between right-side segments (points left)
	Colors   Palette
}

// Predefined themes. Add new ones here; wire them in activeTheme().
var (
	ThemeArrow = Theme{SepRight: "\ue0b0", SepLeft: "\ue0b2", Colors: CatppuccinMocha}
	ThemeRound = Theme{SepRight: "\ue0b4", SepLeft: "\ue0b6", Colors: CatppuccinMocha}
	ThemeSlant = Theme{SepRight: "\ue0b8", SepLeft: "\ue0ba", Colors: CatppuccinMocha}
	ThemeFlame = Theme{SepRight: "\ue0bc", SepLeft: "\ue0be", Colors: CatppuccinMocha}
	ThemePlain = Theme{SepRight: " │", SepLeft: "│ ", Colors: CatppuccinMocha}
)

// activeTheme reads STATUSLINE_THEME and returns the matching theme.
// Defaults to ThemeArrow when unset or unrecognized.
func activeTheme() Theme {
	switch strings.ToLower(strings.TrimSpace(os.Getenv("STATUSLINE_THEME"))) {
	case "round":
		return ThemeRound
	case "slant":
		return ThemeSlant
	case "flame":
		return ThemeFlame
	case "plain":
		return ThemePlain
	default:
		return ThemeArrow
	}
}
```

- [ ] **Step 2: Verify build**

```bash
go build ./...
```

- [ ] **Step 3: Commit**

```bash
git add theme.go
git commit -m "feat: add Theme struct and predefined separator styles"
```

---

### Task 4: Extract segments → `segment.go`, wire theme into renderers

**Files:**
- Create: `segment.go`
- Modify: `main.go` (remove Segment type and render functions, update call sites to pass `theme`)

- [ ] **Step 1: Create `segment.go`**

```go
package main

import "strings"

// Segment is one visual block in the statusline.
type Segment struct {
	BG       Color
	Text     string  // may contain ANSI codes; visibleWidth strips them for measurement
	NoSep    bool    // skip the separator before this segment (right-side only)
	SepColor *Color  // override separator fg; defaults to segment BG
}

func renderLeft(segs []Segment, theme Theme, b *strings.Builder) {
	for i, s := range segs {
		b.WriteString(s.BG.BG())
		b.WriteString(s.Text)
		if i+1 < len(segs) {
			b.WriteString(reset)
			b.WriteString(s.BG.FG())
			b.WriteString(segs[i+1].BG.BG())
			b.WriteString(theme.SepRight)
		} else {
			b.WriteString(reset)
			b.WriteString(s.BG.FG())
			b.WriteString(theme.SepRight)
			b.WriteString(reset)
		}
	}
}

func renderRight(segs []Segment, theme Theme, b *strings.Builder) {
	for i, s := range segs {
		if i == 0 {
			b.WriteString(reset)
			b.WriteString(s.BG.FG())
			b.WriteString(theme.SepLeft)
		} else if s.NoSep {
			// intentionally no separator
		} else {
			sepFG := s.BG
			if s.SepColor != nil {
				sepFG = *s.SepColor
			}
			b.WriteString(sepFG.FG())
			b.WriteString(segs[i-1].BG.BG())
			b.WriteString(theme.SepLeft)
		}
		b.WriteString(s.BG.BG())
		b.WriteString(s.Text)
	}
	b.WriteString(reset)
}

func segmentsWidth(segs []Segment) int {
	w := 0
	for _, s := range segs {
		w += visibleWidth(s.Text)
	}
	w += len(segs) // one separator per segment
	return w
}
```

- [ ] **Step 2: In `main.go`, remove:**
  - `type Segment struct { ... }` block
  - `func renderLeft(...)` — the old signature without `theme`
  - `func renderRight(...)` — the old signature without `theme`
  - `func segmentsWidth(...)` 

- [ ] **Step 3: In `main.go` `main()`, add `theme := activeTheme()` near the top (after input decode), replace `p.Surface1` etc. with `theme.Colors` fields (see Task 8 for the full rewrite), and update the two call sites:**

```go
renderLeft(leftSegs, theme, &out)
renderRight(rightSegs, theme, &out)
```

- [ ] **Step 4: Verify build**

```bash
go build ./...
```

- [ ] **Step 5: Commit**

```bash
git add segment.go main.go
git commit -m "refactor: extract Segment and renderers to segment.go, thread Theme through"
```

---

### Task 5: Extract git logic → `git.go`

**Files:**
- Create: `git.go`
- Modify: `main.go` (remove git functions and GitInfo type)

- [ ] **Step 1: Create `git.go`**

```go
package main

import (
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

type GitInfo struct {
	Branch string
	Dirty  string
	Ahead  int
	Behind int
}

func gitCachePath(sessionID string) string {
	if sessionID != "" {
		return "/tmp/claude-statusline-git-" + sessionID
	}
	return "/tmp/claude-statusline-git-cache"
}

func getGitInfo(cwd, sessionID string) GitInfo {
	cachePath := gitCachePath(sessionID)
	if info, err := os.Stat(cachePath); err == nil {
		if time.Since(info.ModTime()) < 5*time.Second {
			if data, err := os.ReadFile(cachePath); err == nil {
				lines := strings.Split(string(data), "\n")
				if len(lines) >= 4 {
					ahead, _ := strconv.Atoi(lines[2])
					behind, _ := strconv.Atoi(lines[3])
					return GitInfo{Branch: lines[0], Dirty: lines[1], Ahead: ahead, Behind: behind}
				}
			}
		}
	}

	gi := GitInfo{}
	if cwd == "" {
		cwd, _ = os.Getwd()
	}

	if out, err := exec.Command("git", "-C", cwd, "--no-optional-locks", "rev-parse", "--abbrev-ref", "HEAD").Output(); err == nil {
		gi.Branch = strings.TrimSpace(string(out))
	}
	if gi.Branch == "" {
		writeGitCache(gi, cachePath)
		return gi
	}

	err1 := exec.Command("git", "-C", cwd, "--no-optional-locks", "diff", "--quiet").Run()
	err2 := exec.Command("git", "-C", cwd, "--no-optional-locks", "diff", "--cached", "--quiet").Run()
	if err1 != nil || err2 != nil {
		gi.Dirty = " ✦"
	}

	if out, err := exec.Command("git", "-C", cwd, "--no-optional-locks", "rev-list", "--count", "@{u}..HEAD").Output(); err == nil {
		gi.Ahead, _ = strconv.Atoi(strings.TrimSpace(string(out)))
	}
	if out, err := exec.Command("git", "-C", cwd, "--no-optional-locks", "rev-list", "--count", "HEAD..@{u}").Output(); err == nil {
		gi.Behind, _ = strconv.Atoi(strings.TrimSpace(string(out)))
	}

	writeGitCache(gi, cachePath)
	return gi
}

func writeGitCache(gi GitInfo, cachePath string) {
	data := fmt.Sprintf("%s\n%s\n%d\n%d\n", gi.Branch, gi.Dirty, gi.Ahead, gi.Behind)
	_ = os.WriteFile(cachePath, []byte(data), 0644)
}
```

- [ ] **Step 2: Remove `type GitInfo`, `gitCachePath`, `getGitInfo`, `writeGitCache` from `main.go`**

- [ ] **Step 3: Verify build**

```bash
go build ./...
```

- [ ] **Step 4: Commit**

```bash
git add git.go main.go
git commit -m "refactor: extract git logic to git.go"
```

---

### Task 6: Extract layout helpers → `layout.go`

**Files:**
- Create: `layout.go`
- Modify: `main.go` (remove `fmtTokens`, `ansiRE`, `visibleWidth`, `termWidth`)

- [ ] **Step 1: Create `layout.go`**

```go
package main

import (
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"unicode/utf8"
)

func fmtTokens(n int) string {
	if n >= 1000 {
		return fmt.Sprintf("%5s", fmt.Sprintf("%.1fk", float64(n)/1000))
	}
	return fmt.Sprintf("%5d", n)
}

var ansiRE = regexp.MustCompile(`\033\[[0-9;]*m`)

func visibleWidth(s string) int {
	stripped := ansiRE.ReplaceAllString(s, "")
	return utf8.RuneCountInString(stripped)
}

// termWidth returns the terminal column count.
// Checks $COLUMNS first, then tmux, then tput, then falls back to 80.
func termWidth() int {
	if cols := os.Getenv("COLUMNS"); cols != "" {
		if n, err := strconv.Atoi(cols); err == nil && n > 0 {
			return n
		}
	}
	if out, err := exec.Command("tmux", "display-message", "-p", "#{pane_width}").Output(); err == nil {
		if n, err := strconv.Atoi(strings.TrimSpace(string(out))); err == nil && n > 0 {
			return n
		}
	}
	if out, err := exec.Command("tput", "cols").Output(); err == nil {
		if n, err := strconv.Atoi(strings.TrimSpace(string(out))); err == nil && n > 0 {
			return n
		}
	}
	return 80
}
```

- [ ] **Step 2: Remove `fmtTokens`, `ansiRE`, `visibleWidth`, `termWidth` from `main.go`**

- [ ] **Step 3: Verify build**

```bash
go build ./...
```

- [ ] **Step 4: Commit**

```bash
git add layout.go main.go
git commit -m "refactor: extract layout helpers to layout.go"
```

---

### Task 7: Extract bridge writers → `bridge.go`

**Files:**
- Create: `bridge.go`
- Modify: `main.go` (remove `writeMonitorBridge`; keep the inline ctx bridge write in `main()` for now — it will move in Task 8)

- [ ] **Step 1: Create `bridge.go`**

```go
package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"
)

func writeMonitorBridge(input StatusInput) {
	if input.SessionID == "" {
		return
	}
	bridgePath := filepath.Join(os.TempDir(), "claude-monitor-"+input.SessionID+".json")
	bridgeData, _ := json.Marshal(map[string]any{
		"session_id": input.SessionID,
		"timestamp":  time.Now().Unix(),
		"rate_limits": map[string]any{
			"five_hour": map[string]any{
				"used_percentage": input.RateLimits.FiveHour.UsedPercentage,
				"resets_at":       input.RateLimits.FiveHour.ResetsAt,
			},
			"seven_day": map[string]any{
				"used_percentage": input.RateLimits.SevenDay.UsedPercentage,
				"resets_at":       input.RateLimits.SevenDay.ResetsAt,
			},
		},
		"tokens": map[string]any{
			"input":          input.ContextWindow.CurrentUsage.InputTokens,
			"output":         input.ContextWindow.CurrentUsage.OutputTokens,
			"cache_read":     input.ContextWindow.CurrentUsage.CacheReadInputTokens,
			"cache_creation": input.ContextWindow.CurrentUsage.CacheCreationInputTokens,
			"total_input":    input.ContextWindow.TotalInputTokens,
			"total_output":   input.ContextWindow.TotalOutputTokens,
		},
		"model": input.Model.DisplayName,
		"cwd":   input.Cwd,
	})
	_ = os.WriteFile(bridgePath, bridgeData, 0644)
}

func writeCtxBridge(input StatusInput, usedPct int) {
	if input.SessionID == "" {
		return
	}
	bridgePath := filepath.Join(os.TempDir(), "claude-ctx-"+input.SessionID+".json")
	bridgeData, _ := json.Marshal(map[string]any{
		"session_id":           input.SessionID,
		"remaining_percentage": input.ContextWindow.RemainingPercentage,
		"used_pct":             usedPct,
		"timestamp":            time.Now().Unix(),
	})
	_ = os.WriteFile(bridgePath, bridgeData, 0644)
}
```

- [ ] **Step 2: Remove `writeMonitorBridge` from `main.go`**

- [ ] **Step 3: Verify build**

```bash
go build ./...
```

- [ ] **Step 4: Commit**

```bash
git add bridge.go main.go
git commit -m "refactor: extract bridge writers to bridge.go"
```

---

### Task 8: Slim down `main.go` — use palette via theme, call `writeCtxBridge`

After Tasks 1–7 the only remaining code in `main.go` that still references the old global color vars is the segment-building logic inside `main()`. Replace all bare color vars (`base`, `surface0`, `green`, etc.) with `p.<Field>` where `p := theme.Colors`.

**Files:**
- Modify: `main.go`

- [ ] **Step 1: Replace `main.go` with the final slim version**

```go
package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

func main() {
	var input StatusInput
	if err := json.NewDecoder(os.Stdin).Decode(&input); err != nil {
		fmt.Fprintln(os.Stderr, "statusline: invalid JSON input")
		os.Exit(1)
	}

	theme := activeTheme()
	p := theme.Colors

	// ── Model segment ────────────────────────────────────────────────────────
	shortModel := "Claude"
	if input.Model.DisplayName != "" {
		shortModel = strings.TrimPrefix(input.Model.DisplayName, "Claude ")
	}
	modelSeg := Segment{
		BG:   p.Surface1,
		Text: p.Blue.FG() + " 󰧑 " + shortModel + " ",
	}

	// ── Tokens segment ───────────────────────────────────────────────────────
	cacheT := input.ContextWindow.CurrentUsage.CacheReadInputTokens +
		input.ContextWindow.CurrentUsage.CacheCreationInputTokens
	tokensSeg := Segment{
		BG: p.Surface0,
		Text: p.Green.FG() + " ↑" + fmtTokens(input.ContextWindow.CurrentUsage.InputTokens) +
			"/" + fmtTokens(input.ContextWindow.TotalInputTokens) + " " +
			p.Yellow.FG() + "↓" + fmtTokens(input.ContextWindow.CurrentUsage.OutputTokens) +
			"/" + fmtTokens(input.ContextWindow.TotalOutputTokens) + " " +
			p.Teal.FG() + "~" + fmtTokens(cacheT) + " ",
	}

	// ── Context bar ──────────────────────────────────────────────────────────
	pct := int(input.ContextWindow.UsedPercentage)

	var ctxLabel string
	if input.ContextWindow.ContextWindowSize >= 1_000_000 {
		ctxLabel = fmt.Sprintf("%dM", input.ContextWindow.ContextWindowSize/1_000_000)
	} else if input.ContextWindow.ContextWindowSize > 0 {
		ctxLabel = fmt.Sprintf("%dk", input.ContextWindow.ContextWindowSize/1000)
	}

	var barColor, barFill, barDim Color
	switch {
	case pct < 70:
		barColor, barFill, barDim = p.Green, p.Green80, p.GreenDim
	case pct < 90:
		barColor, barFill, barDim = p.Yellow, p.Yellow80, p.YellowDim
	default:
		barColor, barFill, barDim = p.Red, p.Red80, p.RedDim
	}

	sizeSuffix := ""
	if ctxLabel != "" {
		sizeSuffix = "/" + ctxLabel
	}

	var ctxSeg, barSeg Segment
	if pct > 0 {
		filled := min(pct*10/100, 10)
		var barStr strings.Builder
		for range filled {
			barStr.WriteString(barFill.FG() + barDim.BG() + "█")
		}
		for range 10 - filled {
			barStr.WriteString(barDim.FG() + barDim.BG() + "▒")
		}
		ctxSeg = Segment{BG: p.Surface1, Text: barColor.FG() + fmt.Sprintf(" %d%%%s ", pct, sizeSuffix)}
		barSeg = Segment{BG: barDim, Text: barStr.String() + barDim.FG() + "▒", SepColor: &barFill}
	} else {
		var barStr strings.Builder
		for range 10 {
			barStr.WriteString(p.GreenDim.FG() + p.GreenDim.BG() + "▒")
		}
		gd := p.GreenDim
		ctxSeg = Segment{BG: p.Surface1, Text: p.Green.FG() + fmt.Sprintf(" --%% %s ", sizeSuffix)}
		barSeg = Segment{BG: p.GreenDim, Text: barStr.String() + p.GreenDim.FG() + "▒", SepColor: &gd}
	}

	// ── Left segments (git, lines changed, compact warning) ──────────────────
	gi := getGitInfo(input.Cwd, input.SessionID)
	var leftSegs []Segment

	if gi.Branch != "" {
		label := gi.Branch + gi.Dirty
		if gi.Ahead > 0 {
			label += fmt.Sprintf(" ↑%d", gi.Ahead)
		}
		if gi.Behind > 0 {
			label += fmt.Sprintf(" ↓%d", gi.Behind)
		}
		leftSegs = append(leftSegs, Segment{BG: p.Surface1, Text: p.Pink.FG() + "  " + label + " "})
	}

	if input.Cost.TotalLinesAdded > 0 || input.Cost.TotalLinesRemoved > 0 {
		leftSegs = append(leftSegs, Segment{
			BG: p.Surface0,
			Text: p.Green.FG() + fmt.Sprintf("  +%d ", input.Cost.TotalLinesAdded) +
				p.Red.FG() + fmt.Sprintf("-%d ", input.Cost.TotalLinesRemoved),
		})
	}

	if pct >= 85 {
		leftSegs = append(leftSegs, Segment{BG: p.Red, Text: p.Base.FG() + " ⚡ COMPACT "})
	}

	// ── Layout ───────────────────────────────────────────────────────────────
	rightSegs := []Segment{ctxSeg, barSeg, tokensSeg, modelSeg}
	tw := termWidth() - 6
	leftW := segmentsWidth(leftSegs)
	rightW := segmentsWidth(rightSegs)

	var out strings.Builder
	out.WriteString(reset)

	if leftW+rightW+2 <= tw {
		if len(leftSegs) > 0 {
			renderLeft(leftSegs, theme, &out)
		}
		gap := max(tw-leftW-rightW, 1)
		out.WriteString(strings.Repeat(" ", gap))
		renderRight(rightSegs, theme, &out)
	} else {
		if len(leftSegs) > 0 {
			renderLeft(leftSegs, theme, &out)
		}
		out.WriteString("\n")
		gap := max(tw-rightW, 0)
		out.WriteString(strings.Repeat(" ", gap))
		renderRight(rightSegs, theme, &out)
	}
	out.WriteString("\n")
	os.Stdout.WriteString(out.String())

	// ── Bridge files ─────────────────────────────────────────────────────────
	writeCtxBridge(input, pct)
	writeMonitorBridge(input)
}
```

Note: uses `min`/`max` builtins (Go 1.21+) and range-over-int (Go 1.22+) — both available in Go 1.26.

- [ ] **Step 2: Verify build and render**

```bash
go build -o statusline . && echo '{"model":{"display_name":"Claude Sonnet 4.6"},"context_window":{"used_percentage":42,"remaining_percentage":58,"context_window_size":200000,"current_usage":{"input_tokens":1200,"output_tokens":400,"cache_read_input_tokens":300,"cache_creation_input_tokens":100},"total_input_tokens":5000,"total_output_tokens":1500},"cost":{"total_lines_added":10,"total_lines_removed":3},"session_id":"test","cwd":"."}' | ./statusline
```

Expected: colored statusline renders without error.

- [ ] **Step 3: Smoke-test theme switching**

```bash
STATUSLINE_THEME=round echo '{"model":{"display_name":"Claude Sonnet 4.6"},"context_window":{"used_percentage":42,"remaining_percentage":58,"context_window_size":200000,"current_usage":{"input_tokens":1200,"output_tokens":400,"cache_read_input_tokens":300,"cache_creation_input_tokens":100},"total_input_tokens":5000,"total_output_tokens":1500},"cost":{"total_lines_added":10,"total_lines_removed":3},"session_id":"test","cwd":"."}' | ./statusline
```

Expected: renders — separator glyphs differ from default.

- [ ] **Step 4: Commit**

```bash
git add main.go
git commit -m "refactor: slim main() to segment assembly and layout only, use theme palette"
```

---

### Task 9: Update CLAUDE.md to reflect new structure

**Files:**
- Modify: `CLAUDE.md`

- [ ] **Step 1: Update the Architecture section** to list the new files and their roles (replace the single-file description).

- [ ] **Step 2: Add `STATUSLINE_THEME` env var docs** to the Commands section:

```
STATUSLINE_THEME=round ./statusline   # round separators
STATUSLINE_THEME=slant ./statusline   # slanted separators
STATUSLINE_THEME=flame ./statusline   # flame separators
STATUSLINE_THEME=plain ./statusline   # plain │ dividers
# default (unset): arrow separators
```

- [ ] **Step 3: Commit**

```bash
git add CLAUDE.md
git commit -m "docs: update CLAUDE.md for split file structure and theming"
```
