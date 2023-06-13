package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
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
	repo = strings.TrimSuffix(repo, ".git")
	return owner, repo
}

func lastCommit(err error, repo *git.Repository, all bool) *object.Commit {
	logs, err := repo.Log(&git.LogOptions{
		From:       plumbing.Hash{},
		Order:      0,
		FileName:   nil,
		PathFilter: nil,
		All:        all,
		Since:      nil,
		Until:      nil,
	})
	if err != nil {
		log.Fatal(err)
	}
	commit, err := logs.Next()
	if err != nil {
		log.Fatal(err)
	}
	return commit
}
