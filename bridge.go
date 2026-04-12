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
