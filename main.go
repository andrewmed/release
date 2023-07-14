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
	"github.com/go-git/go-git/v5/plumbing/transport"
	"github.com/google/go-github/v53/github"
)

const upstream = "origin"
const teamcityTpl = `https://teamcity.propellerdev.com/searchResults.html?query=revision%%3A%s&buildTypeId=&byTime=true`

var skipped = [...]string{
	"merge",
}

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)

	token := os.Getenv("GITHUB_AUTH_TOKEN")
	if token == "" {
		log.Fatal("missing GITHUB_AUTH_TOKEN")
	}

	ctx := context.Background()
	client := github.NewTokenClient(ctx, token)

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

	releases, _, err := client.Repositories.ListReleases(ctx, owner, name, nil)
	if err != nil {
		log.Fatal(err)
	}
	lastTag := "none"
	lastMsg := "none"
	if len(releases) > 0 {
		lastTag = *releases[0].TagName
		lastMsg = *releases[0].Name
	}

	// check staleness
	err = repo.Fetch(&git.FetchOptions{
		RemoteName:      "",
		RemoteURL:       "",
		RefSpecs:        nil,
		Depth:           1,
		Auth:            nil,
		Progress:        nil,
		Tags:            0,
		Force:           false,
		InsecureSkipTLS: false,
		CABundle:        nil,
		ProxyOptions:    transport.ProxyOptions{},
	})
	lastCommitLocal := lastCommit(err, repo, false)
	hash := lastCommitLocal.Hash.String()
	msgs := strings.Split(lastCommitLocal.Message, "\n")
	var msg string

top:
	for _, m := range msgs {
		normalized := strings.ToLower(strings.TrimSpace(m))
		if normalized == "" {
			continue
		}
		for _, s := range skipped {
			if strings.HasPrefix(normalized, s) {
				continue top
			}
		}
		msg = m
		break
	}
	if msg == "" {
		msg = msgs[0]
	}

	fmt.Printf("last tag: %s\n", lastTag)
	fmt.Printf("last msg: %s\n", lastMsg)

	if lastCommitGlobal := lastCommit(err, repo, true); hash != lastCommitGlobal.Hash.String() {
		fmt.Println("WARN: LOCAL HEAD IS BEHIND REMOTE BRANCH !!!!")
	}

	fmt.Printf("enter new tag:")
	reader := bufio.NewReader(os.Stdin)
	tag, err := reader.ReadString('\n')
	if err != nil {
		log.Fatal(err)
	}
	tag = strings.TrimSpace(tag)
	fmt.Println()

	if err != nil && err != git.NoErrAlreadyUpToDate {
		log.Fatalf("error fetching: %s", err)
	}

	fmt.Println("===========================")
	fmt.Printf("creating release: %s\n", tag)
	fmt.Printf("with title: %s\n", msg)
	fmt.Printf("last commit: %s\n", hash)

	fmt.Printf("hit any key to proceed...")
	reader.ReadByte()
	fmt.Println()

	rc := strings.Contains(tag, "-rc")
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
	fmt.Printf(teamcityTpl, hash)
}
