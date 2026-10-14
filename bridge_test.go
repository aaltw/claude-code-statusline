package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestWriteCtxBridge(t *testing.T) {
	cases := []struct {
		name      string
		json      string
		wantFile  bool
		wantUsed  int
		wantRemng float64
	}{
		{"no context_window", `{"session_id":"s"}`, false, 0, 0},
		{"null percentages", `{"session_id":"s","context_window":{"used_percentage":null,"remaining_percentage":null}}`, false, 0, 0},
		{"real zero", `{"session_id":"s","context_window":{"used_percentage":0,"remaining_percentage":100}}`, true, 0, 100},
		{"normal", `{"session_id":"s","context_window":{"used_percentage":42,"remaining_percentage":58}}`, true, 42, 58},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			dir := t.TempDir()
			t.Setenv("TMPDIR", dir)
			var in StatusInput
			if err := json.Unmarshal([]byte(c.json), &in); err != nil {
				t.Fatal(err)
			}
			writeCtxBridge(in)
			data, err := os.ReadFile(filepath.Join(dir, "claude-ctx-s.json"))
			if !c.wantFile {
				if err == nil {
					t.Fatalf("ctx file written without a reading: %s", data)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			var got struct {
				Remaining float64 `json:"remaining_percentage"`
				Used      int     `json:"used_pct"`
			}
			_ = json.Unmarshal(data, &got)
			if got.Used != c.wantUsed || got.Remaining != c.wantRemng {
				t.Errorf("got used=%d remaining=%v, want %d/%v", got.Used, got.Remaining, c.wantUsed, c.wantRemng)
			}
		})
	}
}
