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
