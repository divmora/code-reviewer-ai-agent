package provider

import (
	"context"
	"os"
	"path/filepath"

	"github.com/divmora/code-reviewer-ai-agent/pkg/git"
	"github.com/divmora/code-reviewer-ai-agent/pkg/model"
)

// LocalProvider runs code reviews against a local git repository workspace.
type LocalProvider struct {
	Workspace     string
	AgainstBranch string
	FromSHA       string
	ToSHA         string
	runner        *git.Runner
}

// NewLocalProvider creates a local git repository provider.
func NewLocalProvider(workspace, againstBranch, fromSHA, toSHA string) *LocalProvider {
	return &LocalProvider{
		Workspace:     workspace,
		AgainstBranch: againstBranch,
		FromSHA:       fromSHA,
		ToSHA:         toSHA,
		runner:        git.NewRunner(workspace),
	}
}

func (p *LocalProvider) Type() string {
	return "local"
}

func (p *LocalProvider) FetchPRDetails(ctx context.Context, target *TargetContext) error {
	headSHA, err := p.runner.GetHeadSHA(ctx)
	if err == nil {
		target.HeadSHA = headSHA
	}
	branch, err := p.runner.GetCurrentBranch(ctx)
	if err == nil {
		target.Title = "Local Review on branch: " + branch
	}
	return nil
}

func (p *LocalProvider) FetchDiff(ctx context.Context, target *TargetContext) ([]*git.FileDiff, string, error) {
	var rawDiff string
	var err error

	if p.FromSHA != "" && p.ToSHA != "" {
		rawDiff, err = p.runner.DiffCommits(ctx, p.FromSHA, p.ToSHA)
	} else if p.AgainstBranch != "" {
		rawDiff, err = p.runner.DiffAgainst(ctx, p.AgainstBranch)
	} else {
		rawDiff, err = p.runner.DiffUncommitted(ctx)
	}

	if err != nil {
		return nil, "", err
	}

	files := git.ParseUnifiedDiff(rawDiff)
	return files, rawDiff, nil
}

func (p *LocalProvider) FetchFileContent(ctx context.Context, target *TargetContext, filePath, ref string) (string, error) {
	fullPath := filepath.Join(p.Workspace, filePath)
	data, err := os.ReadFile(fullPath)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func (p *LocalProvider) PostReview(ctx context.Context, target *TargetContext, report *model.ReviewReport, opts PostReviewOptions) (*PostReviewResult, error) {
	return &PostReviewResult{
		CommentsPosted: len(report.Issues),
	}, nil
}

func (p *LocalProvider) PostCommitStatus(ctx context.Context, target *TargetContext, sha, state, description string) error {
	return nil
}

func (p *LocalProvider) ApprovePR(ctx context.Context, target *TargetContext) error {
	return nil
}
