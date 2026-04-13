package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"time"
)

// safeID strips any characters from s that are not alphanumeric, hyphens, or
// underscores before embedding it in a /tmp filename.  This prevents a
// malicious or malformed session_id from escaping the intended directory.
var safeIDRe = regexp.MustCompile(`[^a-zA-Z0-9\-_]`)

func safeID(s string) string { return safeIDRe.ReplaceAllString(s, "") }

// cacheAccum holds per-session efficiency metrics written to
// /tmp/claude-cache-<sessionID>.json. LastTotalInput is used as a monotonic
// turn marker so we don't double-count when the statusline fires multiple
// times within the same turn.
type cacheAccum struct {
	LastTotalInput     int `json:"last_total_input"`
	SessionInput       int `json:"session_input"`
	SessionCacheRead   int `json:"session_cache_read"`
	SessionCacheCreate int `json:"session_cache_create"`

	// Waste factor: first 5 turns establish the baseline; last 5 track current.
	TurnCount    int   `json:"turn_count"`
	BaselineSum  int   `json:"baseline_sum"` // total tokens across first 5 turns
	BaselineN    int   `json:"baseline_n"`   // turns counted in baseline (max 5)
	RecentTurns  []int `json:"recent_turns"` // per-turn token totals, last 5
}

// wasteFactor returns (current_avg / baseline_avg, true) once enough turns have
// accumulated (≥3 baseline turns, ≥1 recent turn).
func (a cacheAccum) wasteFactor() (float64, bool) {
	if a.BaselineN < 3 || len(a.RecentTurns) == 0 {
		return 0, false
	}
	baselineAvg := float64(a.BaselineSum) / float64(a.BaselineN)
	if baselineAvg == 0 {
		return 0, false
	}
	recentSum := 0
	for _, t := range a.RecentTurns {
		recentSum += t
	}
	recentAvg := float64(recentSum) / float64(len(a.RecentTurns))
	return recentAvg / baselineAvg, true
}

// updateCacheAccum reads, updates, and writes the per-session cache accumulator.
// Returns the updated totals so main can compute session-wide efficiency metrics.
func updateCacheAccum(input StatusInput) cacheAccum {
	if input.SessionID == "" {
		return cacheAccum{}
	}
	path := filepath.Join(os.TempDir(), "claude-cache-"+safeID(input.SessionID)+".json")

	var acc cacheAccum
	if data, err := os.ReadFile(path); err == nil {
		_ = json.Unmarshal(data, &acc)
	}

	// Only accumulate when TotalInputTokens has grown (i.e. a new turn).
	if input.ContextWindow.TotalInputTokens > acc.LastTotalInput {
		cu := input.ContextWindow.CurrentUsage
		acc.SessionInput += cu.InputTokens
		acc.SessionCacheRead += cu.CacheReadInputTokens
		acc.SessionCacheCreate += cu.CacheCreationInputTokens
		acc.LastTotalInput = input.ContextWindow.TotalInputTokens

		// Per-turn token total (all tokens charged this turn).
		turnTokens := cu.InputTokens + cu.OutputTokens +
			cu.CacheReadInputTokens + cu.CacheCreationInputTokens

		acc.TurnCount++
		if acc.BaselineN < 5 {
			acc.BaselineSum += turnTokens
			acc.BaselineN++
		}
		acc.RecentTurns = append(acc.RecentTurns, turnTokens)
		if len(acc.RecentTurns) > 5 {
			acc.RecentTurns = acc.RecentTurns[len(acc.RecentTurns)-5:]
		}
	}

	data, _ := json.Marshal(acc)
	_ = os.WriteFile(path, data, 0600)
	return acc
}

func writeMonitorBridge(input StatusInput) {
	if input.SessionID == "" {
		return
	}
	bridgePath := filepath.Join(os.TempDir(), "claude-monitor-"+safeID(input.SessionID)+".json")
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
	_ = os.WriteFile(bridgePath, bridgeData, 0600)
}

func writeCtxBridge(input StatusInput, usedPct int) {
	if input.SessionID == "" {
		return
	}
	bridgePath := filepath.Join(os.TempDir(), "claude-ctx-"+safeID(input.SessionID)+".json")
	bridgeData, _ := json.Marshal(map[string]any{
		"session_id":           input.SessionID,
		"remaining_percentage": input.ContextWindow.RemainingPercentage,
		"used_pct":             usedPct,
		"timestamp":            time.Now().Unix(),
	})
	_ = os.WriteFile(bridgePath, bridgeData, 0600)
}
