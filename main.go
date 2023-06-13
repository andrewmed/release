package main

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	git "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/google/go-github/v53/github"
)

const upstream = "origin"

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)

	token := os.Getenv("GITHUB_AUTH_TOKEN")
	if token == "" {
		log.Fatal("missing GITHUB_AUTH_TOKEN")
	}

	dir, err := os.Getwd()
	if err != nil {
		log.Fatalf("failed to get current directory: %v", err)
	}

	path, err := findGitRoot(dir)
	if err != nil {
		log.Fatalf("failed to find Git root: %v", err)
	}

	repo, err := git.PlainOpen(path)
	if err != nil {
		log.Fatalf("failed to open repo: %v", err)
	}

	cfg, err := repo.Config()
	if err != nil {
		log.Fatalf("failed to access config: %v", err)
	}
	if len(cfg.Remotes) == 0 {
		log.Fatalf("no remotes for %s", path)
	}
	fmt.Printf("repo: %s\n", path)

	remote := cfg.Remotes[upstream]
	if len(remote.URLs) == 0 {
		log.Fatalf("no remotes for %s", remote.Name)
	}

	url := remote.URLs[0]
	fmt.Printf("remote: %s\n", url)
	owner, name := parseRemote(url)

	ctx := context.Background()
	client := github.NewTokenClient(ctx, token)

	releases, _, err := client.Repositories.ListReleases(ctx, owner, name, &github.ListOptions{
		Page:    0,
		PerPage: 0,
	})
	if err != nil {
		log.Fatal(err)
	}
	last := "none"
	if len(releases) > 0 {
		last = *releases[0].TagName
	}

	fmt.Printf("last release: %s\n", last)
	fmt.Printf("Enter new tag:")
	reader := bufio.NewReader(os.Stdin)
	tag, err := reader.ReadString('\n')
	if err != nil {
		log.Fatal(err)
	}
	tag = strings.TrimSpace(tag)
	fmt.Println()

	logs, err := repo.Log(&git.LogOptions{
		From:       plumbing.Hash{},
		Order:      0,
		FileName:   nil,
		PathFilter: nil,
		All:        false,
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
	hash := commit.Hash.String()
	msg := strings.Split(commit.Message, "\n")[0]

	fmt.Println("============================")
	fmt.Printf("creating release: %s\n", tag)
	fmt.Printf("with title: %s\n", msg)
	fmt.Printf("commit: %s\n", hash)
	fmt.Printf("hit any key to proceed...")
	reader.ReadByte()
	fmt.Println()

	rc := strings.HasSuffix(tag, "-rc")
	latest := "true"
	if rc {
		latest = "false"
	}

	now := github.Timestamp{
		time.Now(),
	}

	release, _, err := client.Repositories.CreateRelease(ctx, owner, name, &github.RepositoryRelease{
		TagName:                &tag,
		TargetCommitish:        &hash,
		Name:                   &msg,
		Body:                   &msg,
		Draft:                  nil,
		Prerelease:             &rc,
		MakeLatest:             &latest,
		DiscussionCategoryName: nil,
		GenerateReleaseNotes:   nil,
		ID:                     nil,
		CreatedAt:              &now,
		PublishedAt:            &now,
		URL:                    nil,
		HTMLURL:                nil,
		AssetsURL:              nil,
		Assets:                 nil,
		UploadURL:              nil,
		ZipballURL:             nil,
		TarballURL:             nil,
		Author:                 nil,
		NodeID:                 nil,
	})

	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("created release: %s\n", *release.HTMLURL)
}
