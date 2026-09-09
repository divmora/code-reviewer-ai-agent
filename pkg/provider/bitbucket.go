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

// BitbucketProvider implements RepoProvider for Bitbucket Cloud.
type BitbucketProvider struct {
	client *http.Client
}

// NewBitbucketProvider creates a Bitbucket client.
func NewBitbucketProvider() *BitbucketProvider {
	return &BitbucketProvider{
		client: &http.Client{Timeout: 30 * time.Second},
	}
}

func (p *BitbucketProvider) Type() string {
	return "bitbucket"
}

func (p *BitbucketProvider) apiURL(path string) string {
	return fmt.Sprintf("https://api.bitbucket.org/2.0/%s", strings.TrimLeft(path, "/"))
}

func (p *BitbucketProvider) doRequest(ctx context.Context, method, reqURL, token string, body any) (*http.Response, error) {
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
		if strings.Contains(token, ":") {
			parts := strings.SplitN(token, ":", 2)
			req.SetBasicAuth(parts[0], parts[1])
		} else {
			req.Header.Set("Authorization", "Bearer "+token)
		}
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "code-reviewer-ai-agent")

	return p.client.Do(req)
}

// FetchPRDetails retrieves Bitbucket PR metadata.
func (p *BitbucketProvider) FetchPRDetails(ctx context.Context, target *TargetContext) error {
	reqURL := p.apiURL(fmt.Sprintf("repositories/%s/%s/pullrequests/%d", target.Owner, target.Repo, target.PRID))

	resp, err := p.doRequest(ctx, "GET", reqURL, target.Token, nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("Bitbucket API returned status %d: %s", resp.StatusCode, string(body))
	}

	var data struct {
		Title       string `json:"title"`
		Description string `json:"description"`
		Source      struct {
			Commit struct {
				Hash string `json:"hash"`
			} `json:"commit"`
		} `json:"source"`
		Links struct {
			HTML struct {
				Href string `json:"href"`
			} `json:"html"`
		} `json:"links"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return err
	}

	target.Title = data.Title
	target.Description = data.Description
	target.HeadSHA = data.Source.Commit.Hash
	target.WebURL = data.Links.HTML.Href

	return nil
}

// FetchDiff retrieves diff text from Bitbucket.
func (p *BitbucketProvider) FetchDiff(ctx context.Context, target *TargetContext) ([]*git.FileDiff, string, error) {
	reqURL := p.apiURL(fmt.Sprintf("repositories/%s/%s/pullrequests/%d/diff", target.Owner, target.Repo, target.PRID))

	resp, err := p.doRequest(ctx, "GET", reqURL, target.Token, nil)
	if err != nil {
		return nil, "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, "", fmt.Errorf("Bitbucket diff API returned status %d: %s", resp.StatusCode, string(body))
	}

	diffBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, "", err
	}
	rawDiff := string(diffBytes)
	fileDiffs := git.ParseUnifiedDiff(rawDiff)

	return fileDiffs, rawDiff, nil
}

// FetchFileContent retrieves file content from Bitbucket.
func (p *BitbucketProvider) FetchFileContent(ctx context.Context, target *TargetContext, filePath, ref string) (string, error) {
	reqURL := p.apiURL(fmt.Sprintf("repositories/%s/%s/src/%s/%s", target.Owner, target.Repo, ref, filePath))

	resp, err := p.doRequest(ctx, "GET", reqURL, target.Token, nil)
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

// PostReview submits comments and updates PR description on Bitbucket.
func (p *BitbucketProvider) PostReview(ctx context.Context, target *TargetContext, report *model.ReviewReport, opts PostReviewOptions) (*PostReviewResult, error) {
	res := &PostReviewResult{}

	// 1. Update Description
	if opts.UpdateDescription {
		newDesc := ComposeMRDescription(target.Description, report, target.HeadSHA)
		updatePayload := map[string]any{"description": newDesc}
		if opts.UpdateTitle && report.SuggestedTitle != "" {
			updatePayload["title"] = report.SuggestedTitle
		}

		reqURL := p.apiURL(fmt.Sprintf("repositories/%s/%s/pullrequests/%d", target.Owner, target.Repo, target.PRID))
		resp, err := p.doRequest(ctx, "PUT", reqURL, target.Token, updatePayload)
		if err == nil {
			resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				res.UpdatedDesc = true
			}
		}
	}

	// 2. Post Inline Comments
	if opts.PostInlines && len(report.Issues) > 0 {
		for _, issue := range report.Issues {
			var body strings.Builder
			body.WriteString(fmt.Sprintf("**[%s] %s**\n\n%s\n", strings.ToUpper(string(issue.Severity)), issue.Category, issue.Description))
			if issue.Suggestion != "" {
				body.WriteString(fmt.Sprintf("\n```\n%s\n```\n", issue.Suggestion))
			}

			payload := map[string]any{
				"content": map[string]string{"raw": body.String()},
			}
			if issue.Filepath != "" && issue.Line > 0 {
				payload["inline"] = map[string]any{
					"to":   issue.Line,
					"path": issue.Filepath,
				}
			}

			commURL := p.apiURL(fmt.Sprintf("repositories/%s/%s/pullrequests/%d/comments", target.Owner, target.Repo, target.PRID))
			resp, err := p.doRequest(ctx, "POST", commURL, target.Token, payload)
			if err == nil && resp.StatusCode == http.StatusCreated {
				resp.Body.Close()
				res.CommentsPosted++
			} else if resp != nil {
				resp.Body.Close()
			}
		}
	}

	return res, nil
}

// PostCommitStatus creates a build status in Bitbucket.
func (p *BitbucketProvider) PostCommitStatus(ctx context.Context, target *TargetContext, sha, state, description string) error {
	reqURL := p.apiURL(fmt.Sprintf("repositories/%s/%s/commit/%s/statuses/build", target.Owner, target.Repo, sha))

	stateMap := map[string]string{
		"running": "INPROGRESS",
		"success": "SUCCESSFUL",
		"failed":  "FAILED",
	}
	bbState := stateMap[state]
	if bbState == "" {
		bbState = "INPROGRESS"
	}

	payload := map[string]string{
		"state":       bbState,
		"key":         "ai-code-reviewer",
		"name":        "AI Code Reviewer",
		"description": description,
		"url":         target.WebURL,
	}

	resp, err := p.doRequest(ctx, "POST", reqURL, target.Token, payload)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return nil
}

// ApprovePR approves the Bitbucket PR.
func (p *BitbucketProvider) ApprovePR(ctx context.Context, target *TargetContext) error {
	reqURL := p.apiURL(fmt.Sprintf("repositories/%s/%s/pullrequests/%d/approve", target.Owner, target.Repo, target.PRID))
	resp, err := p.doRequest(ctx, "POST", reqURL, target.Token, nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return nil
}
