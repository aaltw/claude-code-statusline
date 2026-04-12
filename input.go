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
