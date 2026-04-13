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
	cu := input.ContextWindow.CurrentUsage
	cacheT := cu.CacheReadInputTokens + cu.CacheCreationInputTokens
	tokensSeg := Segment{
		BG: p.Surface0,
		Text: p.Green.FG() + " ↑" + fmtTokens(cu.InputTokens) +
			"/" + fmtTokens(input.ContextWindow.TotalInputTokens) + " " +
			p.Yellow.FG() + "↓" + fmtTokens(cu.OutputTokens) +
			"/" + fmtTokens(input.ContextWindow.TotalOutputTokens) + " " +
			p.Teal.FG() + "~" + fmtTokens(cacheT) + " ",
	}

	// ── Cache efficiency segment ──────────────────────────────────────────────
	// Enabled via STATUSLINE_CACHE_EFFICIENCY=1.
	// Session-wide cache hit ratio accumulated across all turns in this session.
	// High ratio = cache working well; low = wasteful fresh sends.
	acc := updateCacheAccum(input)
	sessionTotal := acc.SessionInput + acc.SessionCacheRead + acc.SessionCacheCreate
	var cacheSeg *Segment
	if os.Getenv("STATUSLINE_CACHE_EFFICIENCY") == "1" && sessionTotal > 500 {
		cacheRatio := acc.SessionCacheRead * 100 / sessionTotal
		var cacheColor Color
		switch {
		case cacheRatio >= 50:
			cacheColor = p.Sapphire
		case cacheRatio >= 20:
			cacheColor = p.Yellow
		default:
			cacheColor = p.Red
		}
		cs := Segment{
			BG:   p.Surface0,
			Text: cacheColor.FG() + fmt.Sprintf(" 󰆼 %d%% ", cacheRatio),
		}
		cacheSeg = &cs
	}

	// ── Waste factor segment ─────────────────────────────────────────────────
	// Enabled via STATUSLINE_CACHE_EFFICIENCY=1 (shares the same flag).
	// Ratio of recent avg tokens/turn to baseline avg (first 5 turns).
	// 1.0x = steady; >2x = context bloat is burning significantly more quota.
	var wasteSeg *Segment
	if os.Getenv("STATUSLINE_CACHE_EFFICIENCY") == "1" {
		if wf, ok := acc.wasteFactor(); ok {
			var wColor Color
			switch {
			case wf < 1.5:
				wColor = p.Peach
			case wf < 2.5:
				wColor = p.Yellow
			default:
				wColor = p.Red
			}
			ws := Segment{
				BG:   p.Surface1,
				Text: wColor.FG() + fmt.Sprintf(" ↗ %.1fx ", wf),
			}
			wasteSeg = &ws
		}
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
		if filled > 0 && filled < 10 {
			// For inverted themes SepRight points /, so use SepLeft (\) instead.
			sep := theme.SepRight
			if theme.InvertSep {
				sep = theme.SepLeft
			}
			barStr.WriteString(barFill.FG() + barDim.BG() + sep)
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
	rightSegs := []Segment{ctxSeg, barSeg, tokensSeg}
	if wasteSeg != nil {
		rightSegs = append(rightSegs, *wasteSeg)
	}
	if cacheSeg != nil {
		rightSegs = append(rightSegs, *cacheSeg)
	}
	rightSegs = append(rightSegs, modelSeg)
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
