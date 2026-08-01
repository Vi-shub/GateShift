package gitops

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/go-github/v66/github"
	"golang.org/x/oauth2"
)

func createGitHubPR(ctx context.Context, token string, req PRRequest, branch, body string) (*PRResult, error) {
	owner, repo, err := splitRepo(req.Repo)
	if err != nil {
		return nil, err
	}
	ts := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: token})
	gh := github.NewClient(oauth2.NewClient(ctx, ts))

	baseRef, _, err := gh.Git.GetRef(ctx, owner, repo, "refs/heads/"+req.BaseBranch)
	if err != nil {
		return nil, fmt.Errorf("get base ref: %w", err)
	}
	baseSHA := baseRef.GetObject().GetSHA()

	refName := "refs/heads/" + branch
	_, _, err = gh.Git.CreateRef(ctx, owner, repo, &github.Reference{
		Ref:    github.String(refName),
		Object: &github.GitObject{SHA: github.String(baseSHA)},
	})
	if err != nil {
		return nil, fmt.Errorf("create branch: %w", err)
	}

	path := fmt.Sprintf("gateway-api/%s.yaml", sanitize(req.IngressName))
	_, _, err = gh.Repositories.CreateFile(ctx, owner, repo, path, &github.RepositoryContentFileOptions{
		Message: github.String(req.Title),
		Content: req.ManifestYAML,
		Branch:  github.String(branch),
	})
	if err != nil {
		return nil, fmt.Errorf("commit manifest: %w", err)
	}

	pr, _, err := gh.PullRequests.Create(ctx, owner, repo, &github.NewPullRequest{
		Title: github.String(req.Title),
		Head:  github.String(branch),
		Base:  github.String(req.BaseBranch),
		Body:  github.String(body),
	})
	if err != nil {
		return nil, fmt.Errorf("create PR: %w", err)
	}
	return &PRResult{
		URL:    pr.GetHTMLURL(),
		Branch: branch,
		Body:   body,
		DryRun: false,
	}, nil
}

func splitRepo(repo string) (owner, name string, err error) {
	parts := strings.Split(repo, "/")
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return "", "", fmt.Errorf("repo must be owner/name, got %q", repo)
	}
	return parts[0], parts[1], nil
}
