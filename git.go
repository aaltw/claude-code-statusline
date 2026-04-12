package main

import (
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

type GitInfo struct {
	Branch string
	Dirty  string
	Ahead  int
	Behind int
}

func gitCachePath(sessionID string) string {
	if sessionID != "" {
		return "/tmp/claude-statusline-git-" + sessionID
	}
	return "/tmp/claude-statusline-git-cache"
}

func getGitInfo(cwd, sessionID string) GitInfo {
	cachePath := gitCachePath(sessionID)
	if info, err := os.Stat(cachePath); err == nil {
		if time.Since(info.ModTime()) < 5*time.Second {
			if data, err := os.ReadFile(cachePath); err == nil {
				lines := strings.Split(string(data), "\n")
				if len(lines) >= 4 {
					ahead, _ := strconv.Atoi(lines[2])
					behind, _ := strconv.Atoi(lines[3])
					return GitInfo{Branch: lines[0], Dirty: lines[1], Ahead: ahead, Behind: behind}
				}
			}
		}
	}

	gi := GitInfo{}
	if cwd == "" {
		cwd, _ = os.Getwd()
	}

	if out, err := exec.Command("git", "-C", cwd, "--no-optional-locks", "rev-parse", "--abbrev-ref", "HEAD").Output(); err == nil {
		gi.Branch = strings.TrimSpace(string(out))
	}
	if gi.Branch == "" {
		writeGitCache(gi, cachePath)
		return gi
	}

	err1 := exec.Command("git", "-C", cwd, "--no-optional-locks", "diff", "--quiet").Run()
	err2 := exec.Command("git", "-C", cwd, "--no-optional-locks", "diff", "--cached", "--quiet").Run()
	if err1 != nil || err2 != nil {
		gi.Dirty = " ✦"
	}

	if out, err := exec.Command("git", "-C", cwd, "--no-optional-locks", "rev-list", "--count", "@{u}..HEAD").Output(); err == nil {
		gi.Ahead, _ = strconv.Atoi(strings.TrimSpace(string(out)))
	}
	if out, err := exec.Command("git", "-C", cwd, "--no-optional-locks", "rev-list", "--count", "HEAD..@{u}").Output(); err == nil {
		gi.Behind, _ = strconv.Atoi(strings.TrimSpace(string(out)))
	}

	writeGitCache(gi, cachePath)
	return gi
}

func writeGitCache(gi GitInfo, cachePath string) {
	data := fmt.Sprintf("%s\n%s\n%d\n%d\n", gi.Branch, gi.Dirty, gi.Ahead, gi.Behind)
	_ = os.WriteFile(cachePath, []byte(data), 0644)
}
