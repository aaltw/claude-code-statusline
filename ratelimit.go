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
	// slopeThreshold is the burn rate (percent/sec) above which the trend arrow
	// reads as "burning" rather than idle. ~0.0005 %/s ≈ 1.8%/hour.
	slopeThreshold = 0.0005
)

// rlPick selects (used_percentage, resets_at) for one window from a sample.
type rlPick func(rlSample) (used, reset float64)

func pickFive(s rlSample) (float64, float64)  { return s.FiveH, s.FiveReset }
func pickSeven(s rlSample) (float64, float64) { return s.SevenD, s.SevenReset }

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

// todayColor maps the 7d "today" usage to the per-working-day budget (20%/day):
// green below 18, Peach 18–20, red above 20.
func todayColor(p Palette, today float64) Color {
	switch {
	case today > 20:
		return p.Red
	case today >= 18:
		return p.Peach
	default:
		return p.Green
	}
}

// trendArrow returns "↑" when the burn rate is meaningfully positive, else "→".
func trendArrow(slopePerSec float64) string {
	if slopePerSec > slopeThreshold {
		return "↑"
	}
	return "→"
}

// todayUsage returns how much of the 7d quota was consumed since local midnight:
// usedNow minus the 7d value of the earliest sample at/after midnight that still
// belongs to the current window. Returns 0 when no such sample exists yet.
func todayUsage(hist rlHistory, usedNow, resetNow float64, now time.Time) float64 {
	midnight := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location()).Unix()
	for _, s := range hist.Samples {
		if s.T < midnight {
			continue
		}
		if s.SevenReset != resetNow {
			continue // different window
		}
		if s.SevenD > usedNow {
			continue // value dropped → window boundary
		}
		return usedNow - s.SevenD
	}
	return 0
}
