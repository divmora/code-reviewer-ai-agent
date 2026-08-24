package provider

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/divmora/code-reviewer-ai-agent/pkg/git"
	"github.com/divmora/code-reviewer-ai-agent/pkg/model"
)

// GitHubProvider implements RepoProvider for GitHub.
type GitHubProvider struct {
	client *http.Client
}

// NewGitHubProvider creates a GitHub provider client.
func NewGitHubProvider() *GitHubProvider {
	return &GitHubProvider{
		client: &http.Client{Timeout: 30 * time.Second},
	}
}

func (p *GitHubProvider) Type() string {
	return "github"
}

func (p *GitHubProvider) apiURL(baseURL, path string) string {
	if baseURL == "" || strings.Contains(baseURL, "github.com") {
		return fmt.Sprintf("https://api.github.com/%s", strings.TrimLeft(path, "/"))
	}
	// GitHub Enterprise Server support
	baseURL = strings.TrimRight(baseURL, "/")
	return fmt.Sprintf("%s/api/v3/%s", baseURL, strings.TrimLeft(path, "/"))
}

func (p *GitHubProvider) doRequest(ctx context.Context, method, reqURL, token string, body any, customAccept string) (*http.Response, error) {
	var bodyReader io.Reader
	if body != nil {
		jsonBytes, err := json.Marshal(body)
		if err != nil {
			return nil, err
		}
		bodyReader = bytes.NewReader(jsonBytes)
	}

	req, err := http.NewRequestWithContext(ctx, method, reqURL, bodyReader)
	if err != nil {
		return nil, err
	}

	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	accept := "application/vnd.github+json"
	if customAccept != "" {
		accept = customAccept
	}
	req.Header.Set("Accept", accept)
	req.Header.Set("User-Agent", "code-reviewer-ai-agent")

	return p.client.Do(req)
}

// FetchPRDetails retrieves PR title, description, head SHA, and draft status.
func (p *GitHubProvider) FetchPRDetails(ctx context.Context, target *TargetContext) error {
	reqURL := p.apiURL(target.BaseURL, fmt.Sprintf("repos/%s/%s/pulls/%d", target.Owner, target.Repo, target.PRID))

	resp, err := p.doRequest(ctx, "GET", reqURL, target.Token, nil, "")
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("GitHub API returned status %d: %s", resp.StatusCode, string(body))
	}

	var data struct {
		Title       string `json:"title"`
		Body        string `json:"body"`
		Draft       bool   `json:"draft"`
		HTMLURL     string `json:"html_url"`
		Head struct {
			SHA string `json:"sha"`
		} `json:"head"`
		Base struct {
			SHA string `json:"sha"`
		} `json:"base"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return err
	}

	target.Title = data.Title
	target.Description = data.Body
	target.HeadSHA = data.Head.SHA
	target.BaseSHA = data.Base.SHA
	target.IsDraft = data.Draft
	target.WebURL = data.HTMLURL

	return nil
}

// FetchDiff retrieves the unified diff from GitHub.
func (p *GitHubProvider) FetchDiff(ctx context.Context, target *TargetContext) ([]*git.FileDiff, string, error) {
	reqURL := p.apiURL(target.BaseURL, fmt.Sprintf("repos/%s/%s/pulls/%d", target.Owner, target.Repo, target.PRID))

	resp, err := p.doRequest(ctx, "GET", reqURL, target.Token, nil, "application/vnd.github.v3.diff")
	if err != nil {
		return nil, "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, "", fmt.Errorf("GitHub diff API returned status %d: %s", resp.StatusCode, string(body))
	}

	diffBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, "", err
	}
	rawDiff := string(diffBytes)
	fileDiffs := git.ParseUnifiedDiff(rawDiff)

	return fileDiffs, rawDiff, nil
}

// FetchFileContent fetches file source from GitHub repository at a given ref/SHA.
func (p *GitHubProvider) FetchFileContent(ctx context.Context, target *TargetContext, filePath, ref string) (string, error) {
	reqURL := p.apiURL(target.BaseURL, fmt.Sprintf("repos/%s/%s/contents/%s?ref=%s", target.Owner, target.Repo, filePath, ref))

	resp, err := p.doRequest(ctx, "GET", reqURL, target.Token, nil, "application/vnd.github.raw+json")
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("failed to fetch file %s (status %d)", filePath, resp.StatusCode)
	}

	content, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	return string(content), nil
}

// PostReview submits a formal PR review with inline comments and updates description.
func (p *GitHubProvider) PostReview(ctx context.Context, target *TargetContext, report *model.ReviewReport, opts PostReviewOptions) (*PostReviewResult, error) {
	res := &PostReviewResult{}

	// 1. Update PR Description
	if opts.UpdateDescription {
		newDesc := ComposeMRDescription(target.Description, report, target.HeadSHA)
		updatePayload := map[string]any{"body": newDesc}
		if opts.UpdateTitle && report.SuggestedTitle != "" {
			updatePayload["title"] = report.SuggestedTitle
		}

		reqURL := p.apiURL(target.BaseURL, fmt.Sprintf("repos/%s/%s/pulls/%d", target.Owner, target.Repo, target.PRID))
		resp, err := p.doRequest(ctx, "PATCH", reqURL, target.Token, updatePayload, "")
		if err == nil {
			resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				res.UpdatedDesc = true
			}
		}
	}

	// 2. Prepare Comments
	var comments []map[string]any
	if opts.PostInlines && len(report.Issues) > 0 {
		for _, issue := range report.Issues {
			var body strings.Builder
			body.WriteString(fmt.Sprintf("**[%s] %s**\n\n%s\n", strings.ToUpper(string(issue.Severity)), issue.Category, issue.Description))

			if issue.Suggestion != "" {
				body.WriteString(fmt.Sprintf("\n```suggestion\n%s\n```\n", issue.Suggestion))
			}

			comments = append(comments, map[string]any{
				"path": issue.Filepath,
				"line": issue.Line,
				"side": "RIGHT",
				"body": body.String(),
			})
		}
	}

	// 3. Post Formal Review
	event := "COMMENT"
	if report.CalculateVerdict() == "APPROVE" {
		event = "APPROVE"
	} else if report.CalculateVerdict() == "REQUEST_CHANGES" {
		event = "REQUEST_CHANGES"
	}

	reviewPayload := map[string]any{
		"commit_id": target.HeadSHA,
		"event":     event,
		"body":      fmt.Sprintf("## 🤖 AI Code Review\n\n**Verdict:** `%s`\n\n%s", report.Verdict, report.Summary),
		"comments":  comments,
	}

	reviewURL := p.apiURL(target.BaseURL, fmt.Sprintf("repos/%s/%s/pulls/%d/reviews", target.Owner, target.Repo, target.PRID))
	resp, err := p.doRequest(ctx, "POST", reviewURL, target.Token, reviewPayload, "")
	if err == nil && (resp.StatusCode == http.StatusOK || resp.StatusCode == http.StatusCreated) {
		resp.Body.Close()
		res.CommentsPosted = len(comments)
		if event == "APPROVE" {
			res.Approved = true
		}
	} else if resp != nil {
		resp.Body.Close()
	}

	return res, nil
}

// PostCommitStatus creates a commit status on GitHub.
func (p *GitHubProvider) PostCommitStatus(ctx context.Context, target *TargetContext, sha, state, description string) error {
	reqURL := p.apiURL(target.BaseURL, fmt.Sprintf("repos/%s/%s/statuses/%s", target.Owner, target.Repo, sha))

	stateMap := map[string]string{
		"running": "pending",
		"success": "success",
		"failed":  "failure",
	}
	ghState := stateMap[state]
	if ghState == "" {
		ghState = "pending"
	}

	payload := map[string]string{
		"state":       ghState,
		"context":     "ai-code-reviewer",
		"description": description,
	}
	if target.WebURL != "" {
		payload["target_url"] = target.WebURL
	}

	resp, err := p.doRequest(ctx, "POST", reqURL, target.Token, payload, "")
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return nil
}

// ApprovePR submits an APPROVE review to GitHub.
func (p *GitHubProvider) ApprovePR(ctx context.Context, target *TargetContext) error {
	reviewURL := p.apiURL(target.BaseURL, fmt.Sprintf("repos/%s/%s/pulls/%d/reviews", target.Owner, target.Repo, target.PRID))
	payload := map[string]any{
		"commit_id": target.HeadSHA,
		"event":     "APPROVE",
		"body":      "Approved by AI Code Reviewer.",
	}
	resp, err := p.doRequest(ctx, "POST", reviewURL, target.Token, payload, "")
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return nil
}
