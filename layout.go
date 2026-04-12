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
