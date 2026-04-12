package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"
)

// ── Colors ──────────────────────────────────────────────────────────────────
// Color type, palette, and reset constant are defined in color.go

var (
	base      = CatppuccinMocha.Base
	surface0  = CatppuccinMocha.Surface0
	surface1  = CatppuccinMocha.Surface1
	blue      = CatppuccinMocha.Blue
	green     = CatppuccinMocha.Green
	green80   = CatppuccinMocha.Green80
	greenDim  = CatppuccinMocha.GreenDim
	yellow    = CatppuccinMocha.Yellow
	yellow80  = CatppuccinMocha.Yellow80
	yellowDim = CatppuccinMocha.YellowDim
	red       = CatppuccinMocha.Red
	red80     = CatppuccinMocha.Red80
	redDim    = CatppuccinMocha.RedDim
	pink      = CatppuccinMocha.Pink
	teal      = CatppuccinMocha.Teal
)


// ── Helpers ─────────────────────────────────────────────────────────────────

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


func writeMonitorBridge(input StatusInput) {
	if input.SessionID == "" {
		return
	}
	bridgePath := filepath.Join(os.TempDir(), "claude-monitor-"+input.SessionID+".json")
	cacheT := input.ContextWindow.CurrentUsage.CacheReadInputTokens + input.ContextWindow.CurrentUsage.CacheCreationInputTokens
	_ = cacheT
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

// ── Main ────────────────────────────────────────────────────────────────────

func main() {
	var input StatusInput
	if err := json.NewDecoder(os.Stdin).Decode(&input); err != nil {
		fmt.Fprintln(os.Stderr, "statusline: invalid JSON input")
		os.Exit(1)
	}

	theme := activeTheme()

	// ── Model segment ───────────────────────────────────────────────────────
	shortModel := "Claude"
	if input.Model.DisplayName != "" {
		shortModel = strings.TrimPrefix(input.Model.DisplayName, "Claude ")
	}
	modelSeg := Segment{
		BG:   surface1,
		Text: blue.FG() + " 󰧑 " + shortModel + " ",
	}

	// ── Tokens segment ──────────────────────────────────────────────────────
	cacheT := input.ContextWindow.CurrentUsage.CacheReadInputTokens + input.ContextWindow.CurrentUsage.CacheCreationInputTokens
	tokensSeg := Segment{
		BG: surface0,
		Text: green.FG() + " ↑" + fmtTokens(input.ContextWindow.CurrentUsage.InputTokens) + "/" + fmtTokens(input.ContextWindow.TotalInputTokens) + " " +
			yellow.FG() + "↓" + fmtTokens(input.ContextWindow.CurrentUsage.OutputTokens) + "/" + fmtTokens(input.ContextWindow.TotalOutputTokens) + " " +
			teal.FG() + "~" + fmtTokens(cacheT) + " ",
	}

	// ── Context bar ─────────────────────────────────────────────────────────
	pct := int(input.ContextWindow.UsedPercentage)

	var ctxLabel string
	if input.ContextWindow.ContextWindowSize >= 1000000 {
		ctxLabel = fmt.Sprintf("%dM", input.ContextWindow.ContextWindowSize/1000000)
	} else if input.ContextWindow.ContextWindowSize > 0 {
		ctxLabel = fmt.Sprintf("%dk", input.ContextWindow.ContextWindowSize/1000)
	}

	var barColor, barFill, barDim Color
	if pct < 70 {
		barColor, barFill, barDim = green, green80, greenDim
	} else if pct < 90 {
		barColor, barFill, barDim = yellow, yellow80, yellowDim
	} else {
		barColor, barFill, barDim = red, red80, redDim
	}

	sizeSuffix := ""
	if ctxLabel != "" {
		sizeSuffix = "/" + ctxLabel
	}

	var ctxSeg, barSeg Segment
	if pct > 0 {
		filled := pct * 10 / 100
		if filled > 10 {
			filled = 10
		}
		var barStr strings.Builder
		for i := 0; i < filled; i++ {
			barStr.WriteString(barFill.FG() + barDim.BG() + "█")
		}
		for i := 0; i < 10-filled; i++ {
			barStr.WriteString(barDim.FG() + barDim.BG() + "▒")
		}
		ctxSeg = Segment{BG: surface1, Text: barColor.FG() + fmt.Sprintf(" %d%%%s ", pct, sizeSuffix)}
		barSeg = Segment{BG: barDim, Text: barStr.String() + barDim.FG() + "▒", SepColor: &barFill}
	} else {
		var barStr strings.Builder
		for i := 0; i < 10; i++ {
			barStr.WriteString(greenDim.FG() + greenDim.BG() + "▒")
		}
		ctxSeg = Segment{BG: surface1, Text: green.FG() + fmt.Sprintf(" --%% %s ", sizeSuffix)}
		gd := greenDim
		barSeg = Segment{BG: greenDim, Text: barStr.String() + greenDim.FG() + "▒", SepColor: &gd}
	}

	// ── Git segment ─────────────────────────────────────────────────────────
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
		leftSegs = append(leftSegs, Segment{BG: surface1, Text: pink.FG() + "  " + label + " "})
	}

	// ── Lines changed segment ───────────────────────────────────────────────
	if input.Cost.TotalLinesAdded > 0 || input.Cost.TotalLinesRemoved > 0 {
		leftSegs = append(leftSegs, Segment{
			BG:   surface0,
			Text: green.FG() + fmt.Sprintf("  +%d ", input.Cost.TotalLinesAdded) + red.FG() + fmt.Sprintf("-%d ", input.Cost.TotalLinesRemoved),
		})
	}

	// ── Compact warning ─────────────────────────────────────────────────────
	if pct >= 85 {
		leftSegs = append(leftSegs, Segment{BG: red, Text: base.FG() + " ⚡ COMPACT "})
	}

	// ── Assemble right segments ─────────────────────────────────────────────
	rightSegs := []Segment{ctxSeg, barSeg, tokensSeg, modelSeg}

	// ── Layout ──────────────────────────────────────────────────────────────
	tw := termWidth() - 6 // nerd font icon margin
	leftW := segmentsWidth(leftSegs)
	rightW := segmentsWidth(rightSegs)

	var out strings.Builder
	out.WriteString(reset) // Clear any ANSI state from Claude Code

	if leftW+rightW+2 <= tw {
		// Single line
		if len(leftSegs) > 0 {
			renderLeft(leftSegs, theme, &out)
		}
		gap := tw - leftW - rightW
		if gap < 1 {
			gap = 1
		}
		out.WriteString(strings.Repeat(" ", gap))
		renderRight(rightSegs, theme, &out)
	} else {
		// Two lines
		if len(leftSegs) > 0 {
			renderLeft(leftSegs, theme, &out)
		}
		out.WriteString("\n")
		gap := tw - rightW
		if gap < 0 {
			gap = 0
		}
		out.WriteString(strings.Repeat(" ", gap))
		renderRight(rightSegs, theme, &out)
	}
	out.WriteString("\n")

	os.Stdout.WriteString(out.String())

	// Write context metrics bridge file for the context-monitor PostToolUse hook.
	// The hook reads this to inject agent-facing warnings when context is low.
	if input.SessionID != "" {
		bridgePath := filepath.Join(os.TempDir(), "claude-ctx-"+input.SessionID+".json")
		bridgeData, _ := json.Marshal(map[string]any{
			"session_id":           input.SessionID,
			"remaining_percentage": input.ContextWindow.RemainingPercentage,
			"used_pct":             pct,
			"timestamp":            time.Now().Unix(),
		})
		_ = os.WriteFile(bridgePath, bridgeData, 0644)
	}

	writeMonitorBridge(input)
}
