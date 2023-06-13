package main

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

func findGitRoot(path string) (string, error) {
	for {
		isGitDir, err := isGitDirectory(path)
		if err != nil {
			return "", err
		}
		if isGitDir {
			return path, nil
		}

		parent := filepath.Dir(path)
		if parent == path {
			return "", fmt.Errorf("reached filesystem root, but .git directory not found")
		}

		path = parent
	}
}

func isGitDirectory(path string) (bool, error) {
	fi, err := os.Stat(filepath.Join(path, ".git"))
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, err
	}
	return fi.IsDir(), nil
}

func parseRemote(s string) (string, string) {
	re := regexp.MustCompile(`.*[\/|:](\w+)\/(.+)$`)
	match := re.FindStringSubmatch(s)
	owner := match[1]
	repo := match[2]
	repo = strings.TrimPrefix(repo, ".git")
	return owner, repo
}
