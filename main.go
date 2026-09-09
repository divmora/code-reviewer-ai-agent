// Code Reviewer — AI-powered context-aware code review agent.
//
// Powered by Divmora LocalHarness SDK (github.com/divmora/localharness/adk).
//
// Usage:
//
//	# Review GitLab Merge Request (Cloud or Self-Hosted)
//	go run . --url https://gitlab.corp.internal/group/repo/-/merge_requests/42 --token $GITLAB_TOKEN --post
//
//	# Review GitHub Pull Request
//	go run . --url https://github.com/org/repo/pull/123 --token $GITHUB_TOKEN --post
//
//	# Review local git workspace diff
//	go run . --workspace . --diff
package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/divmora/code-reviewer-ai-agent/pkg/provider"
	"github.com/divmora/code-reviewer-ai-agent/pkg/reviewer"
	"github.com/divmora/code-reviewer-ai-agent/pkg/version"
)

func main() {
	targetURL := flag.String("url", "", "Pull Request / Merge Request URL (GitLab, GitHub, Bitbucket)")
	workspace := flag.String("workspace", ".", "Local workspace directory to review")
	diffFlag := flag.Bool("diff", false, "Review uncommitted local changes")
	againstBranch := flag.String("against", "", "Branch to compare against (e.g., origin/main)")
	providerFlag := flag.String("provider", "auto", "VCS provider: gitlab, github, bitbucket, local, auto")
	baseURL := flag.String("base-url", "", "Custom host URL for self-hosted GitLab / GitHub Enterprise")
	tokenFlag := flag.String("token", "", "VCS API access token (or set GITHUB_TOKEN / GITLAB_TOKEN / BITBUCKET_TOKEN)")
	skipTLSVerify := flag.Bool("skip-tls-verify", false, "Skip SSL/TLS verification (for internal self-hosted GitLab)")
	headSHA := flag.String("head-sha", "", "Expected target commit SHA")
	force := flag.Bool("force", false, "Force re-review even if commit SHA was already reviewed")

	postReview := flag.Bool("post", false, "Post review comments, description, and status to the PR/MR")
	updateDesc := flag.Bool("update-description", true, "Update MR/PR description with Scorecard, Walkthrough, and SHA watermark")
	updateLabels := flag.Bool("update-labels", true, "Update MR/PR labels with Quality, Security, and Maintainability scores")
	updateTitle := flag.Bool("update-title", false, "Update MR/PR title with suggested conventional commit title")
	autoApprove := flag.Bool("auto-approve", false, "Automatically approve PR if all scores >= 80 and 0 critical bugs")
	ignoreDrafts := flag.Bool("ignore-drafts", true, "Skip reviewing Draft / WIP pull requests")
	maxComments := flag.Int("max-comments", 15, "Maximum number of inline comments to post")

	format := flag.String("format", "terminal", "Output format: terminal, markdown, json")
	outputFile := flag.String("output", "", "File path to save the review report (e.g., review.md)")
	profile := flag.String("profile", "balanced", "Review sensitivity profile: chill, balanced, assertive")

	customPrompt := flag.String("prompt", "", "Custom project prompt (passed from Zenith UI or CLI)")
	rulesJSON := flag.String("rules-json", "", "Inline JSON structured rules payload from Zenith UI")
	rulesFile := flag.String("rules", "", "Path to custom YAML/JSON rules file")
	inlineRule := flag.String("rule", "", "Ad-hoc inline rule instruction")
	verbose := flag.Bool("verbose", false, "Enable verbose debug logging")

	litellmBaseURL := flag.String("litellm-base-url", "", "Custom LiteLLM proxy base URL (or env LITELLM_BASE_URL)")
	litellmAPIKey := flag.String("litellm-api-key", "", "Custom LiteLLM API key (or env LITELLM_API_KEY)")
	litellmModel := flag.String("litellm-model", "", "Custom LiteLLM model identifier (or env LITELLM_MODEL)")
	litellmEndpoint := flag.String("litellm-endpoint", "", "Named endpoint in ~/.divmora/config/litellm.json (or env LITELLM_ENDPOINT)")

	versionFlag := flag.Bool("version", false, "Print version information and exit")
	shortVersionFlag := flag.Bool("v", false, "Print version information and exit")

	flag.Parse()

	if *versionFlag || *shortVersionFlag {
		fmt.Println(version.Get().String())
		os.Exit(0)
	}

	// Setup logger
	logLevel := slog.LevelInfo
	if *verbose {
		logLevel = slog.LevelDebug
	}
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: logLevel}))

	ctx := context.Background()

	// 1. Resolve Target Context & Provider
	var target *provider.TargetContext
	var p provider.RepoProvider

	if *targetURL != "" {
		parsed, err := provider.ParseTargetURL(*targetURL)
		if err != nil {
			logger.Error("failed to parse URL", "error", err)
			os.Exit(1)
		}
		target = parsed

		if *baseURL != "" {
			target.BaseURL = *baseURL
		}
		if *providerFlag != "auto" {
			target.Provider = *providerFlag
		}
		target.Token = provider.ResolveToken(target.Provider, *tokenFlag)
		target.SkipTLSVerify = *skipTLSVerify

		switch target.Provider {
		case "gitlab":
			p = provider.NewGitLabProvider(*skipTLSVerify)
		case "github":
			p = provider.NewGitHubProvider()
		case "bitbucket":
			p = provider.NewBitbucketProvider()
		default:
			logger.Error("unsupported provider", "provider", target.Provider)
			os.Exit(1)
		}

		// Fetch PR / MR details
		logger.Info("fetching PR details", "provider", target.Provider, "repo", target.ProjectPath, "pr_id", target.PRID)
		if err := p.FetchPRDetails(ctx, target); err != nil {
			logger.Error("failed to fetch PR details", "error", err)
			os.Exit(1)
		}

		// Draft check
		if *ignoreDrafts && target.IsDraft {
			fmt.Printf("⏭️  Skipping Draft PR #%d: marked as draft/WIP (use --ignore-drafts=false to override)\n", target.PRID)
			os.Exit(0)
		}

		// Already-reviewed check
		if *headSHA != "" {
			target.HeadSHA = *headSHA
		}
		if shouldSkip, reason := provider.ShouldSkipReview(target.HeadSHA, target.Description, *force); shouldSkip {
			fmt.Printf("⏭️  %s\n", reason)
			os.Exit(0)
		}

	} else {
		// Local Workspace Review
		absWorkspace, err := filepath.Abs(*workspace)
		if err != nil {
			logger.Error("failed to resolve workspace path", "error", err)
			os.Exit(1)
		}

		target = &provider.TargetContext{
			Provider:  "local",
			Owner:     filepath.Base(absWorkspace),
			Repo:      filepath.Base(absWorkspace),
			Workspace: absWorkspace,
		}
		_ = diffFlag // uncommitted diff is default when againstBranch is empty
		p = provider.NewLocalProvider(absWorkspace, *againstBranch, "", "")

		if err := p.FetchPRDetails(ctx, target); err != nil {
			logger.Warn("could not fetch local branch details", "error", err)
		}
	}

	// 2. Fetch Diff
	fmt.Printf("🔍 Fetching diff for %s (commit: %s)...\n", target.ProjectPath, target.HeadSHA)
	files, _, err := p.FetchDiff(ctx, target)
	if err != nil {
		logger.Error("failed to fetch diff", "error", err)
		os.Exit(1)
	}

	if len(files) == 0 {
		fmt.Println("✅ No changed files found in diff.")
		os.Exit(0)
	}

	fmt.Printf("📦 Found %d changed file(s). Starting AI review...\n", len(files))

	// Post running status if requested
	if *postReview && target.HeadSHA != "" {
		_ = p.PostCommitStatus(ctx, target, target.HeadSHA, "running", "AI code review in progress...")
	}

	// 3. Run AI Reviewer
	rev := reviewer.NewReviewer(logger)
	report, err := rev.RunReview(ctx, p, target, files, reviewer.ReviewOptions{
		Workspace:       target.Workspace,
		ProjectPath:     target.ProjectPath,
		CustomPrompt:    *customPrompt,
		RulesJSON:       *rulesJSON,
		RulesFile:       *rulesFile,
		InlineRule:      *inlineRule,
		Profile:         *profile,
		HeadSHA:         target.HeadSHA,
		MaxComments:     *maxComments,
		Verbose:         *verbose,
		LitellmBaseURL:  *litellmBaseURL,
		LitellmAPIKey:   *litellmAPIKey,
		LitellmModel:    *litellmModel,
		LitellmEndpoint: *litellmEndpoint,
	})

	if err != nil {
		logger.Error("review failed", "error", err)
		if *postReview && target.HeadSHA != "" {
			_ = p.PostCommitStatus(ctx, target, target.HeadSHA, "failed", "AI code review failed")
		}
		os.Exit(1)
	}

	// 4. Output / Format Review Report
	var formattedOutput string
	switch strings.ToLower(*format) {
	case "markdown", "md":
		formattedOutput = provider.FormatMarkdown(report)
	case "json":
		formattedOutput, _ = provider.FormatJSON(report)
	default:
		formattedOutput = provider.FormatTerminal(report)
	}

	fmt.Print(formattedOutput)

	// Save to file if --output requested
	if *outputFile != "" {
		if err := os.WriteFile(*outputFile, []byte(formattedOutput), 0644); err != nil {
			logger.Error("failed to write output file", "path", *outputFile, "error", err)
		} else {
			fmt.Printf("\n📄 Review report saved to %s\n", *outputFile)
		}
	}

	// 5. Post to VCS if --post requested
	if *postReview {
		fmt.Println("\n🚀 Posting review to VCS provider...")
		res, err := p.PostReview(ctx, target, report, provider.PostReviewOptions{
			PostInlines:       true,
			UpdateDescription: *updateDesc,
			UpdateLabels:      *updateLabels,
			UpdateTitle:       *updateTitle,
			AutoApprove:       *autoApprove,
			MaxComments:       *maxComments,
		})

		if err != nil {
			logger.Error("posting review failed", "error", err)
		} else {
			fmt.Printf("   ✅ Posted %d inline comment(s)\n", res.CommentsPosted)
			if res.NotesPosted > 0 {
				fmt.Printf("   ℹ️ Posted %d fallback note(s)\n", res.NotesPosted)
			}
			if res.UpdatedDesc {
				fmt.Println("   ✅ Updated PR/MR description with Scorecard & Walkthrough")
			}
			if res.UpdatedLabels {
				fmt.Println("   ✅ Updated PR/MR labels with Score badges")
			}
			if res.Approved {
				fmt.Println("   🎉 Auto-approved PR/MR")
			}
		}

		// Update Commit Status
		statusState := "success"
		statusDesc := fmt.Sprintf("Review Complete — Score: %d/100 (%s)", report.CodeScore, report.Verdict)
		if report.Verdict == "REQUEST_CHANGES" {
			statusState = "failed"
		}
		_ = p.PostCommitStatus(ctx, target, target.HeadSHA, statusState, statusDesc)
	}

	fmt.Println("\n🏁 Review process completed.")
}
