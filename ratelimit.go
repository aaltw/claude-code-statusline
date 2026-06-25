package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"
)

// rlSample is a timestamped snapshot of the account-wide rate-limit usage.
// The *Reset fields double as window identity: when they change (or a used
// percentage drops) the window has rolled over and a fresh slope begins.
type rlSample struct {
	T          int64   `json:"t"`           // unix seconds
	FiveH      float64 `json:"five_h"`      // used_percentage
	FiveReset  float64 `json:"five_reset"`  // resets_at (window identity)
	SevenD     float64 `json:"seven_d"`     // used_percentage
	SevenReset float64 `json:"seven_reset"` // resets_at (window identity)
}

type rlHistory struct {
	Samples []rlSample `json:"samples"`
}

const (
	rlMinInterval int64 = 60          // don't append more than once a minute unless a value changed
	rlMaxAge      int64 = 26 * 60 * 60 // prune samples older than ~26h (covers the 7d "today" delta)
	// slopeThreshold is the 5h burn rate (percent/sec) below which usage reads as
	// idle/steady rather than actively burning. ~0.0005 %/s ≈ 1.8%/hour.
	slopeThreshold = 0.0005
	// fiveHourBurnCeiling is the burn rate that would spend the whole 5h window
	// evenly over its duration (100% / 5h ≈ 0.00556 %/s ≈ 20%/hour). Above it the
	// 5h burn rate is "too high" — faster than the window lasts.
	fiveHourBurnCeiling = 100.0 / (5 * 60 * 60)
)

// rlPick selects (used_percentage, resets_at) for one window from a sample.
type rlPick func(rlSample) (used, reset float64)

func pickFive(s rlSample) (float64, float64) { return s.FiveH, s.FiveReset }

// rateHistoryPath is the single account-global sample store. Rate limits are
// account-wide, not per-session, so there is no session id in the filename.
func rateHistoryPath() string {
	return filepath.Join(os.TempDir(), "claude-ratelimit-history.json")
}

// updateRateHistory reads the persisted history, appends a fresh sample (when
// enough has changed to be worth recording), prunes stale samples and writes
// the result back. All file I/O is best-effort, mirroring the bridge code.
// Returns rlHistory{} without touching disk when no rate-limit data is present.
func updateRateHistory(input StatusInput) rlHistory {
	five := input.RateLimits.FiveHour
	seven := input.RateLimits.SevenDay
	if five.ResetsAt == 0 && seven.ResetsAt == 0 {
		return rlHistory{}
	}

	path := rateHistoryPath()
	var hist rlHistory
	if data, err := os.ReadFile(path); err == nil {
		_ = json.Unmarshal(data, &hist)
	}

	now := time.Now().Unix()
	sample := rlSample{
		T:          now,
		FiveH:      five.UsedPercentage,
		FiveReset:  five.ResetsAt,
		SevenD:     seven.UsedPercentage,
		SevenReset: seven.ResetsAt,
	}

	// Throttle file growth: only append if the last sample is old or a value moved.
	appendSample := true
	if n := len(hist.Samples); n > 0 {
		last := hist.Samples[n-1]
		unchanged := last.FiveH == sample.FiveH && last.SevenD == sample.SevenD &&
			last.FiveReset == sample.FiveReset && last.SevenReset == sample.SevenReset
		if unchanged && now-last.T < rlMinInterval {
			appendSample = false
		}
	}
	if appendSample {
		hist.Samples = append(hist.Samples, sample)
	}

	// Prune samples older than the retention window.
	cutoff := now - rlMaxAge
	kept := hist.Samples[:0]
	for _, s := range hist.Samples {
		if s.T >= cutoff {
			kept = append(kept, s)
		}
	}
	hist.Samples = kept

	if data, err := json.Marshal(hist); err == nil {
		_ = os.WriteFile(path, data, 0600)
	}
	return hist
}

// slopePerSec estimates the recent burn rate (percent/sec) for one window. It
// finds the earliest sample within windowSec that still belongs to the current
// window (same reset value, value not decreased) and divides the rise by the
// run. Returns 0 — treated as idle/green — when there is no usable earlier sample.
func slopePerSec(hist rlHistory, pick rlPick, windowSec int64) float64 {
	n := len(hist.Samples)
	if n < 2 {
		return 0
	}
	last := hist.Samples[n-1]
	usedNow, resetNow := pick(last)
	tNow := last.T

	for i := 0; i < n-1; i++ {
		s := hist.Samples[i]
		usedThen, resetThen := pick(s)
		if tNow-s.T > windowSec {
			continue // outside the look-back window
		}
		if resetThen != resetNow {
			continue // belongs to a previous window
		}
		if usedThen > usedNow {
			continue // value dropped → window boundary, skip
		}
		dt := tNow - s.T
		if dt <= 0 {
			return 0
		}
		return (usedNow - usedThen) / float64(dt)
	}
	return 0
}

// projectedAtReset linearly extrapolates current usage to the reset instant.
func projectedAtReset(usedNow, slopePerSec, resetsAt, now float64) float64 {
	return usedNow + slopePerSec*(resetsAt-now)
}

// windowColor maps a projected-at-reset percentage to a burn-rate color:
// green below 90, Peach 90–100, red at/above 100 (will exhaust before reset).
func windowColor(p Palette, projected float64) Color {
	switch {
	case projected >= 100:
		return p.Red
	case projected >= 90:
		return p.Peach
	default:
		return p.Green
	}
}

// burnArrow classifies the 5h burn rate (percent/sec) into a colored trend
// arrow whose steepness rises with the rate: steady (idle) → green "→",
// acceptable (up to the full-window pace) → Peach "↗", too high (faster than
// the window lasts) → red "↑".
func burnArrow(p Palette, slopePerSec float64) (glyph string, c Color) {
	switch {
	case slopePerSec > fiveHourBurnCeiling:
		return "↑", p.Red
	case slopePerSec > slopeThreshold:
		return "↗", p.Peach
	default:
		return "→", p.Green
	}
}

// fiveHourBudget returns the even-pace 5h quota you're "allowed" to have spent
// by now: the fraction of the 5-hour window elapsed, as a percentage (0–100).
// The window starts 5h before resetsAt; spending 100% evenly across it lands you
// exactly here right now. Mirrors workingDayBudget for the 7d window.
func fiveHourBudget(resetsAt, now float64) float64 {
	const windowSec = 5 * 60 * 60
	pct := (now - (resetsAt - windowSec)) / windowSec * 100
	if pct < 0 {
		return 0
	}
	if pct > 100 {
		return 100
	}
	return pct
}

// percentPerWorkingDay is the 7d quota a 5-day work-week allows per working day
// (100% / 5 days = 20%). Cumulated across the window's working days it forms the
// "budget" you're allowed by today.
const percentPerWorkingDay = 20.0

// workingDayBudget returns the cumulative 7d quota you're allowed to have spent
// by the end of today: percentPerWorkingDay times the number of working days
// (Mon–Fri) from the window's start through today, capped at 100%. The window
// starts 7 days before resetsAt; any 7-day span holds exactly 5 weekdays, so a
// full window budgets to 100%.
func workingDayBudget(resetsAt float64, now time.Time) float64 {
	loc := now.Location()
	windowStart := time.Unix(int64(resetsAt), 0).In(loc).AddDate(0, 0, -7)
	day := time.Date(windowStart.Year(), windowStart.Month(), windowStart.Day(), 0, 0, 0, 0, loc)
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, loc)

	workingDays := 0
	for !day.After(today) {
		if wd := day.Weekday(); wd != time.Saturday && wd != time.Sunday {
			workingDays++
		}
		day = day.AddDate(0, 0, 1)
	}
	budget := float64(workingDays) * percentPerWorkingDay
	if budget > 100 {
		budget = 100
	}
	return budget
}

// budgetColor compares 7d usage against the working-day budget allowed by today:
// green below 90% of budget, Peach 90–100%, red at/over budget (ahead of plan).
// When the budget is 0 (reset day / weekend start) any usage is over plan → red.
func budgetColor(p Palette, used, budget float64) Color {
	if budget <= 0 {
		if used > 0 {
			return p.Red
		}
		return p.Green
	}
	switch ratio := used / budget * 100; {
	case ratio >= 100:
		return p.Red
	case ratio >= 90:
		return p.Peach
	default:
		return p.Green
	}
}
