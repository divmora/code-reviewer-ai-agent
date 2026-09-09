package provider

import (
	"context"

	"github.com/divmora/code-reviewer-ai-agent/pkg/git"
	"github.com/divmora/code-reviewer-ai-agent/pkg/model"
)

// PostReviewOptions configures publishing actions on the VCS provider.
type PostReviewOptions struct {
	PostInlines       bool
	UpdateDescription bool
	UpdateLabels      bool
	UpdateTitle       bool
	AutoApprove       bool
	MaxComments       int
}

// PostReviewResult contains the results of publishing a review.
type PostReviewResult struct {
	CommentsPosted int
	NotesPosted    int
	UpdatedDesc    bool
	UpdatedLabels  bool
	Approved       bool
	Errors         []string
}

// RepoProvider defines the operations supported across GitHub, GitLab, and Bitbucket.
type RepoProvider interface {
	Type() string
	FetchPRDetails(ctx context.Context, target *TargetContext) error
	FetchDiff(ctx context.Context, target *TargetContext) ([]*git.FileDiff, string, error)
	FetchFileContent(ctx context.Context, target *TargetContext, filePath, ref string) (string, error)
	PostReview(ctx context.Context, target *TargetContext, report *model.ReviewReport, opts PostReviewOptions) (*PostReviewResult, error)
	PostCommitStatus(ctx context.Context, target *TargetContext, sha, state, description string) error
	ApprovePR(ctx context.Context, target *TargetContext) error
}
