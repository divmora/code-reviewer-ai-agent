package provider

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/divmora/code-reviewer-ai-agent/pkg/git"
	"github.com/divmora/code-reviewer-ai-agent/pkg/model"
)

// GitLabProvider implements RepoProvider for GitLab (SaaS and Self-Hosted).
type GitLabProvider struct {
	client *http.Client
}

// NewGitLabProvider creates a GitLab provider with optional TLS verification skipping.
func NewGitLabProvider(skipTLSVerify bool) *GitLabProvider {
	transport := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: skipTLSVerify},
	}
	return &GitLabProvider{
		client: &http.Client{
			Transport: transport,
			Timeout:   30 * time.Second,
		},
	}
}

func (p *GitLabProvider) Type() string {
	return "gitlab"
}

func escapeProjectPath(projectPath string) string {
	return url.QueryEscape(projectPath)
}

func (p *GitLabProvider) apiURL(baseURL, path string) string {
	baseURL = strings.TrimRight(baseURL, "/")
	if !strings.HasPrefix(baseURL, "http://") && !strings.HasPrefix(baseURL, "https://") {
		baseURL = "https://" + baseURL
	}
	return fmt.Sprintf("%s/api/v4/%s", baseURL, strings.TrimLeft(path, "/"))
}

func (p *GitLabProvider) doRequest(ctx context.Context, method, reqURL, token string, body any) (*http.Response, error) {
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

	req.Header.Set("PRIVATE-TOKEN", token)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "code-reviewer-ai-agent")

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

// FetchPRDetails fetches MR info, title, description, and diff_refs.
func (p *GitLabProvider) FetchPRDetails(ctx context.Context, target *TargetContext) error {
	encodedPath := escapeProjectPath(target.ProjectPath)
	reqURL := p.apiURL(target.BaseURL, fmt.Sprintf("projects/%s/merge_requests/%d", encodedPath, target.PRID))

	resp, err := p.doRequest(ctx, "GET", reqURL, target.Token, nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("GitLab API returned status %d: %s", resp.StatusCode, string(body))
	}

	var data struct {
		Title          string `json:"title"`
		Description    string `json:"description"`
		SHA            string `json:"sha"`
		Draft          bool   `json:"draft"`
		WorkInProgress bool   `json:"work_in_progress"`
		WebURL         string `json:"web_url"`
		DiffRefs       struct {
			BaseSHA  string `json:"base_sha"`
			HeadSHA  string `json:"head_sha"`
			StartSHA string `json:"start_sha"`
		} `json:"diff_refs"`
		Labels []string `json:"labels"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return err
	}

	target.Title = data.Title
	target.Description = data.Description
	target.HeadSHA = data.SHA
	target.BaseSHA = data.DiffRefs.BaseSHA
	target.StartSHA = data.DiffRefs.StartSHA
	target.IsDraft = data.Draft || data.WorkInProgress || strings.HasPrefix(strings.ToLower(data.Title), "draft:") || strings.HasPrefix(strings.ToLower(data.Title), "wip:")
	if data.WebURL != "" {
		target.WebURL = data.WebURL
	}

	return nil
}

// FetchDiff retrieves raw or parsed diffs from the MR matching the native GitLab diff view.
func (p *GitLabProvider) FetchDiff(ctx context.Context, target *TargetContext) ([]*git.FileDiff, string, error) {
	encodedPath := escapeProjectPath(target.ProjectPath)

	// 1. Fetch paginated MR diffs
	type diffItem struct {
		OldPath     string `json:"old_path"`
		NewPath     string `json:"new_path"`
		Diff        string `json:"diff"`
		NewFile     bool   `json:"new_file"`
		DeletedFile bool   `json:"deleted_file"`
		RenamedFile bool   `json:"renamed_file"`
	}

	var allDiffs []diffItem
	page := 1

	for {
		diffsURL := p.apiURL(target.BaseURL, fmt.Sprintf("projects/%s/merge_requests/%d/diffs?per_page=100&page=%d", encodedPath, target.PRID, page))
		resp, err := p.doRequest(ctx, "GET", diffsURL, target.Token, nil)
		if err != nil {
			break
		}

		var pageDiffs []diffItem
		if err := json.NewDecoder(resp.Body).Decode(&pageDiffs); err != nil {
			resp.Body.Close()
			break
		}
		resp.Body.Close()

		if len(pageDiffs) == 0 {
			break
		}
		allDiffs = append(allDiffs, pageDiffs...)
		if len(pageDiffs) < 100 {
			break
		}
		page++
	}

	// 2. Fetch MR Commits to identify author non-merge commits
	commitsURL := p.apiURL(target.BaseURL, fmt.Sprintf("projects/%s/merge_requests/%d/commits?per_page=100", encodedPath, target.PRID))
	hasMergeCommits := false
	authorTouchedFiles := make(map[string]bool)

	if resp, err := p.doRequest(ctx, "GET", commitsURL, target.Token, nil); err == nil {
		var mrCommits []struct {
			ID        string   `json:"id"`
			ParentIDs []string `json:"parent_ids"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&mrCommits); err == nil {
			for _, c := range mrCommits {
				if len(c.ParentIDs) > 1 {
					hasMergeCommits = true
				} else {
					// Non-merge commit: fetch its touched files
					cDiffURL := p.apiURL(target.BaseURL, fmt.Sprintf("projects/%s/repository/commits/%s/diff", encodedPath, c.ID))
					if cResp, cErr := p.doRequest(ctx, "GET", cDiffURL, target.Token, nil); cErr == nil {
						var commitDiffs []diffItem
						if err := json.NewDecoder(cResp.Body).Decode(&commitDiffs); err == nil {
							for _, cd := range commitDiffs {
								path := cd.NewPath
								if path == "" {
									path = cd.OldPath
								}
								authorTouchedFiles[path] = true
							}
						}
						cResp.Body.Close()
					}
				}
			}
		}
		resp.Body.Close()
	}

	// 3. Fallback to /changes endpoint if /diffs was empty
	if len(allDiffs) == 0 {
		reqURL := p.apiURL(target.BaseURL, fmt.Sprintf("projects/%s/merge_requests/%d/changes", encodedPath, target.PRID))
		resp, err := p.doRequest(ctx, "GET", reqURL, target.Token, nil)
		if err != nil {
			return nil, "", err
		}
		defer resp.Body.Close()

		if resp.StatusCode == http.StatusOK {
			var data struct {
				Changes  []diffItem `json:"changes"`
				DiffRefs struct {
					BaseSHA  string `json:"base_sha"`
					HeadSHA  string `json:"head_sha"`
					StartSHA string `json:"start_sha"`
				} `json:"diff_refs"`
			}
			if err := json.NewDecoder(resp.Body).Decode(&data); err == nil {
				allDiffs = data.Changes
				if data.DiffRefs.HeadSHA != "" {
					target.HeadSHA = data.DiffRefs.HeadSHA
				}
				if data.DiffRefs.BaseSHA != "" {
					target.BaseSHA = data.DiffRefs.BaseSHA
				}
				if data.DiffRefs.StartSHA != "" {
					target.StartSHA = data.DiffRefs.StartSHA
				}
			}
		}
	}

	var fileDiffs []*git.FileDiff
	var combinedRawDiff strings.Builder

	for _, c := range allDiffs {
		path := c.NewPath
		if path == "" {
			path = c.OldPath
		}

		// If the branch had intermediate merge commits and author touched files were discovered,
		// filter out files that were only introduced by the external merge
		if hasMergeCommits && len(authorTouchedFiles) > 0 && !authorTouchedFiles[path] {
			continue
		}

		parsed := git.ParseUnifiedDiff(c.Diff)
		if len(parsed) > 0 {
			fd := parsed[0]
			fd.OldPath = c.OldPath
			fd.NewPath = c.NewPath
			fd.IsNew = c.NewFile
			fd.IsDeleted = c.DeletedFile
			fd.IsRenamed = c.RenamedFile
			fileDiffs = append(fileDiffs, fd)
		} else {
			fileDiffs = append(fileDiffs, &git.FileDiff{
				OldPath:   c.OldPath,
				NewPath:   c.NewPath,
				IsNew:     c.NewFile,
				IsDeleted: c.DeletedFile,
				IsRenamed: c.RenamedFile,
				RawDiff:   c.Diff,
			})
		}
		combinedRawDiff.WriteString(c.Diff + "\n")
	}

	return fileDiffs, combinedRawDiff.String(), nil
}

// FetchFileContent fetches file source from GitLab repository at a given ref/SHA.
func (p *GitLabProvider) FetchFileContent(ctx context.Context, target *TargetContext, filePath, ref string) (string, error) {
	encodedPath := escapeProjectPath(target.ProjectPath)
	encodedFile := url.QueryEscape(filePath)
	reqURL := p.apiURL(target.BaseURL, fmt.Sprintf("projects/%s/repository/files/%s/raw?ref=%s", encodedPath, encodedFile, ref))

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

// PostReview posts inline comments, description updates, and labels to GitLab.
func (p *GitLabProvider) PostReview(ctx context.Context, target *TargetContext, report *model.ReviewReport, opts PostReviewOptions) (*PostReviewResult, error) {
	encodedPath := escapeProjectPath(target.ProjectPath)
	res := &PostReviewResult{}

	// 1. Update MR Description
	if opts.UpdateDescription {
		newDesc := ComposeMRDescription(target.Description, report, target.HeadSHA)
		updatePayload := map[string]any{"description": newDesc}
		if opts.UpdateTitle && report.SuggestedTitle != "" {
			updatePayload["title"] = report.SuggestedTitle
		}

		reqURL := p.apiURL(target.BaseURL, fmt.Sprintf("projects/%s/merge_requests/%d", encodedPath, target.PRID))
		resp, err := p.doRequest(ctx, "PUT", reqURL, target.Token, updatePayload)
		if err == nil {
			resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				res.UpdatedDesc = true
			}
		} else {
			res.Errors = append(res.Errors, fmt.Sprintf("Description update failed: %v", err))
		}
	}

	// 2. Update Labels
	if opts.UpdateLabels {
		labelsURL := p.apiURL(target.BaseURL, fmt.Sprintf("projects/%s/merge_requests/%d", encodedPath, target.PRID))
		resp, err := p.doRequest(ctx, "GET", labelsURL, target.Token, nil)
		if err == nil {
			var mrData struct {
				Labels []string `json:"labels"`
			}
			json.NewDecoder(resp.Body).Decode(&mrData)
			resp.Body.Close()

			updatedLabels := BuildUpdatedLabels(mrData.Labels, report)
			updateResp, err := p.doRequest(ctx, "PUT", labelsURL, target.Token, map[string]any{
				"labels": strings.Join(updatedLabels, ","),
			})
			if err == nil {
				updateResp.Body.Close()
				if updateResp.StatusCode == http.StatusOK {
					res.UpdatedLabels = true
				}
			}
		}
	}

	// 3. Post Inline Discussions with Fallback to Notes
	if opts.PostInlines && len(report.Issues) > 0 {
		for _, issue := range report.Issues {
			var body strings.Builder
			body.WriteString(fmt.Sprintf("**[%s] %s**\n\n%s\n", strings.ToUpper(string(issue.Severity)), issue.Category, issue.Description))

			if issue.Suggestion != "" {
				body.WriteString(fmt.Sprintf("\n```suggestion:-0+0\n%s\n```\n", issue.Suggestion))
			}

			payload := map[string]any{
				"body": body.String(),
			}

			if target.BaseSHA != "" && target.HeadSHA != "" && issue.Filepath != "" && issue.Line > 0 {
				payload["position"] = map[string]any{
					"base_sha":      target.BaseSHA,
					"start_sha":     target.StartSHA,
					"head_sha":      target.HeadSHA,
					"position_type": "text",
					"new_path":      issue.Filepath,
					"new_line":      issue.Line,
				}
			}

			discURL := p.apiURL(target.BaseURL, fmt.Sprintf("projects/%s/merge_requests/%d/discussions", encodedPath, target.PRID))
			resp, err := p.doRequest(ctx, "POST", discURL, target.Token, payload)
			if err == nil && resp.StatusCode == http.StatusCreated {
				resp.Body.Close()
				res.CommentsPosted++
			} else {
				if resp != nil {
					resp.Body.Close()
				}
				// Fallback to top-level note
				fallbackBody := fmt.Sprintf("**File:** `%s:%d`\n\n%s", issue.Filepath, issue.Line, body.String())
				noteURL := p.apiURL(target.BaseURL, fmt.Sprintf("projects/%s/merge_requests/%d/notes", encodedPath, target.PRID))
				noteResp, nErr := p.doRequest(ctx, "POST", noteURL, target.Token, map[string]any{"body": fallbackBody})
				if nErr == nil && noteResp.StatusCode == http.StatusCreated {
					noteResp.Body.Close()
					res.NotesPosted++
				} else if noteResp != nil {
					noteResp.Body.Close()
				}
			}
		}
	}

	// 4. Auto-Approve if enabled
	if opts.AutoApprove && report.CalculateVerdict() == "APPROVE" {
		if err := p.ApprovePR(ctx, target); err == nil {
			res.Approved = true
		}
	}

	return res, nil
}

// PostCommitStatus posts a commit status on GitLab.
func (p *GitLabProvider) PostCommitStatus(ctx context.Context, target *TargetContext, sha, state, description string) error {
	encodedPath := escapeProjectPath(target.ProjectPath)
	reqURL := p.apiURL(target.BaseURL, fmt.Sprintf("projects/%s/statuses/%s", encodedPath, sha))

	payload := map[string]string{
		"state":       state,
		"name":        "ai-code-reviewer",
		"description": description,
	}
	if target.WebURL != "" {
		payload["target_url"] = target.WebURL
	}

	resp, err := p.doRequest(ctx, "POST", reqURL, target.Token, payload)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return nil
}

// ApprovePR approves the Merge Request via GitLab API.
func (p *GitLabProvider) ApprovePR(ctx context.Context, target *TargetContext) error {
	encodedPath := escapeProjectPath(target.ProjectPath)
	reqURL := p.apiURL(target.BaseURL, fmt.Sprintf("projects/%s/merge_requests/%d/approve", encodedPath, target.PRID))

	resp, err := p.doRequest(ctx, "POST", reqURL, target.Token, nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return nil
}
