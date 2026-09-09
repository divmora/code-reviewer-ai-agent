package provider

import (
	"fmt"
	"net/url"
	"os"
	"regexp"
	"strconv"
	"strings"
)

var (
	gitLabMRRegex    = regexp.MustCompile(`^(https?://[^/]+)/(.+)/-/merge_requests/(\d+)`)
	gitHubPRRegex    = regexp.MustCompile(`^(https?://[^/]+)/([^/]+)/([^/]+)/pull/(\d+)`)
	bitbucketPRRegex = regexp.MustCompile(`^(https?://[^/]+)/([^/]+)/([^/]+)/pull-requests/(\d+)`)
)

// ParseTargetURL extracts provider, host, project/repo, and PR ID from a web URL.
func ParseTargetURL(rawURL string) (*TargetContext, error) {
	cleanURL := strings.TrimSpace(rawURL)
	cleanURL = strings.Split(cleanURL, "?")[0]
	cleanURL = strings.TrimRight(cleanURL, "/")

	// 1. Check GitLab URL: https://gitlab.corp.com/group/subgroup/project/-/merge_requests/123
	if match := gitLabMRRegex.FindStringSubmatch(cleanURL); len(match) == 4 {
		mrID, _ := strconv.Atoi(match[3])
		baseURL := match[1]
		projectPath := match[2]

		parts := strings.Split(projectPath, "/")
		repo := parts[len(parts)-1]
		owner := strings.Join(parts[:len(parts)-1], "/")

		return &TargetContext{
			Provider:    "gitlab",
			BaseURL:     baseURL,
			Owner:       owner,
			Repo:        repo,
			ProjectPath: projectPath,
			PRID:        mrID,
			WebURL:      cleanURL,
		}, nil
	}

	// 2. Check GitHub URL: https://github.com/owner/repo/pull/123
	if match := gitHubPRRegex.FindStringSubmatch(cleanURL); len(match) == 5 {
		prID, _ := strconv.Atoi(match[4])
		baseURL := match[1]
		owner := match[2]
		repo := match[3]

		return &TargetContext{
			Provider:    "github",
			BaseURL:     baseURL,
			Owner:       owner,
			Repo:        repo,
			ProjectPath: fmt.Sprintf("%s/%s", owner, repo),
			PRID:        prID,
			WebURL:      cleanURL,
		}, nil
	}

	// 3. Check Bitbucket URL: https://bitbucket.org/workspace/repo/pull-requests/123
	if match := bitbucketPRRegex.FindStringSubmatch(cleanURL); len(match) == 5 {
		prID, _ := strconv.Atoi(match[4])
		baseURL := match[1]
		owner := match[2]
		repo := match[3]

		return &TargetContext{
			Provider:    "bitbucket",
			BaseURL:     baseURL,
			Owner:       owner,
			Repo:        repo,
			ProjectPath: fmt.Sprintf("%s/%s", owner, repo),
			PRID:        prID,
			WebURL:      cleanURL,
		}, nil
	}

	// Check generic URL domain
	u, err := url.Parse(cleanURL)
	if err == nil && u.Host != "" {
		if strings.Contains(u.Host, "github") {
			return &TargetContext{Provider: "github", BaseURL: fmt.Sprintf("%s://%s", u.Scheme, u.Host), WebURL: cleanURL}, nil
		} else if strings.Contains(u.Host, "bitbucket") {
			return &TargetContext{Provider: "bitbucket", BaseURL: fmt.Sprintf("%s://%s", u.Scheme, u.Host), WebURL: cleanURL}, nil
		}
		// Default custom domain to gitlab (common for self-hosted instances)
		return &TargetContext{Provider: "gitlab", BaseURL: fmt.Sprintf("%s://%s", u.Scheme, u.Host), WebURL: cleanURL}, nil
	}

	return nil, fmt.Errorf("unable to parse PR/MR target URL: %q", rawURL)
}

// ResolveToken returns the access token from explicit flag or environment variables.
func ResolveToken(provider, explicitToken string) string {
	if explicitToken != "" {
		return explicitToken
	}

	switch strings.ToLower(provider) {
	case "gitlab":
		if t := os.Getenv("GITLAB_TOKEN"); t != "" {
			return t
		}
		if t := os.Getenv("CI_JOB_TOKEN"); t != "" {
			return t
		}
	case "github":
		if t := os.Getenv("GITHUB_TOKEN"); t != "" {
			return t
		}
		if t := os.Getenv("GH_TOKEN"); t != "" {
			return t
		}
	case "bitbucket":
		if t := os.Getenv("BITBUCKET_TOKEN"); t != "" {
			return t
		}
		if t := os.Getenv("BITBUCKET_APP_PASSWORD"); t != "" {
			return t
		}
	}

	return os.Getenv("VCS_TOKEN")
}
