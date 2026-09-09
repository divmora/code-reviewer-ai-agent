package git

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"strings"
)

// Runner executes git commands inside a repository workspace.
type Runner struct {
	Workspace string
}

// NewRunner creates a new git runner for a workspace directory.
func NewRunner(workspace string) *Runner {
	return &Runner{Workspace: workspace}
}

func (r *Runner) run(ctx context.Context, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, "git", args...)
	if r.Workspace != "" {
		cmd.Dir = r.Workspace
	}
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		return "", fmt.Errorf("git %s failed: %w (stderr: %s)", strings.Join(args, " "), err, stderr.String())
	}
	return stdout.String(), nil
}

// GetHeadSHA returns the current HEAD commit hash.
func (r *Runner) GetHeadSHA(ctx context.Context) (string, error) {
	out, err := r.run(ctx, "rev-parse", "HEAD")
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(out), nil
}

// GetCurrentBranch returns the current branch name.
func (r *Runner) GetCurrentBranch(ctx context.Context) (string, error) {
	out, err := r.run(ctx, "rev-parse", "--abbrev-ref", "HEAD")
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(out), nil
}

// DiffUncommitted returns the unified diff of unstaged and staged changes.
func (r *Runner) DiffUncommitted(ctx context.Context) (string, error) {
	// First check staged diff
	staged, err := r.run(ctx, "diff", "--cached")
	if err != nil {
		return "", err
	}
	// Check unstaged diff
	unstaged, err := r.run(ctx, "diff")
	if err != nil {
		return "", err
	}

	if staged != "" && unstaged != "" {
		return staged + "\n" + unstaged, nil
	} else if staged != "" {
		return staged, nil
	}
	return unstaged, nil
}

// DiffAgainst returns the diff between the current branch and a target branch.
func (r *Runner) DiffAgainst(ctx context.Context, targetBranch string) (string, error) {
	return r.run(ctx, "diff", targetBranch+"...HEAD")
}

// DiffCommits returns the diff between two commit hashes.
func (r *Runner) DiffCommits(ctx context.Context, fromSHA, toSHA string) (string, error) {
	return r.run(ctx, "diff", fromSHA+"..."+toSHA)
}

// ShowFileContent returns the content of a file at a specific commit ref.
func (r *Runner) ShowFileContent(ctx context.Context, ref, filePath string) (string, error) {
	return r.run(ctx, "show", fmt.Sprintf("%s:%s", ref, filePath))
}
