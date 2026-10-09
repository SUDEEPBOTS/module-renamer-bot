package vcs

import (
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"strings"
)

func CloneRepo(repoURL, destDir string) error {
	_ = os.RemoveAll(destDir)
	cmd := exec.Command("git", "clone", "--depth", "1", repoURL, destDir)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("git clone failed: %s (%v)", string(output), err)
	}
	return nil
}

func PushToGitHub(dir, repoURL, token, commitMsg string) error {
	_ = os.RemoveAll(fmt.Sprintf("%s/.git", dir))

	if commitMsg == "" {
		commitMsg = "feat: project rebranded via SUDEEPBOTS Module Renamer"
	}

	authURL := repoURL
	if token != "" {
		trimmed := strings.TrimPrefix(repoURL, "https://")
		trimmed = strings.TrimPrefix(trimmed, "http://")
		authURL = fmt.Sprintf("https://%s@%s", url.QueryEscape(token), trimmed)
	}

	commands := [][]string{
		{"git", "init", "-b", "main"},
		{"git", "config", "user.name", "SUDEEPBOTS"},
		{"git", "config", "user.email", "sudeepbots@users.noreply.github.com"},
		{"git", "add", "-A"},
		{"git", "commit", "-m", commitMsg},
		{"git", "remote", "add", "origin", authURL},
		{"git", "push", "-u", "origin", "main", "--force"},
	}

	for _, args := range commands {
		cmd := exec.Command(args[0], args[1:]...)
		cmd.Dir = dir
		output, err := cmd.CombinedOutput()
		if err != nil {
			return fmt.Errorf("command '%s' failed: %s (%v)", strings.Join(args, " "), string(output), err)
		}
	}

	return nil
}
