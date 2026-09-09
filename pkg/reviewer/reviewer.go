package reviewer

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/divmora/code-reviewer-ai-agent/pkg/agent"
	"github.com/divmora/code-reviewer-ai-agent/pkg/ast"
	"github.com/divmora/code-reviewer-ai-agent/pkg/git"
	"github.com/divmora/code-reviewer-ai-agent/pkg/model"
	"github.com/divmora/code-reviewer-ai-agent/pkg/provider"
	"github.com/divmora/code-reviewer-ai-agent/pkg/rules"
	"github.com/divmora/localharness/adk"
)

const maxBatchChars = 100000
const maxSingleDiffChars = 15000

// Reviewer coordinates AST analysis, rule matching, and LocalHarness agent execution.
type Reviewer struct {
	logger *slog.Logger
}

// NewReviewer creates a new Reviewer instance.
func NewReviewer(logger *slog.Logger) *Reviewer {
	if logger == nil {
		logger = slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelInfo}))
	}
	return &Reviewer{logger: logger}
}

// ReviewOptions holds all parameters for a review execution.
type ReviewOptions struct {
	Workspace       string
	ProjectPath     string
	CustomPrompt    string
	RulesJSON       string
	RulesFile       string
	InlineRule      string
	Profile         string
	HeadSHA         string
	MaxComments     int
	Verbose         bool
	LitellmBaseURL  string
	LitellmAPIKey   string
	LitellmModel    string
	LitellmEndpoint string
}

type fileContextBlock struct {
	path  string
	block string
}

// RunReview performs the full AST-scoped code review against a provider and diff set.
func (r *Reviewer) RunReview(ctx context.Context, p provider.RepoProvider, target *provider.TargetContext, rawFiles []*git.FileDiff, opts ReviewOptions) (*model.ReviewReport, error) {
	// 1. Load and aggregate rules
	ruleConfig, err := rules.LoadRules(rules.IngestionOptions{
		Workspace:    opts.Workspace,
		CustomPrompt: opts.CustomPrompt,
		RulesJSON:    opts.RulesJSON,
		RulesFile:    opts.RulesFile,
		InlineRule:   opts.InlineRule,
		Profile:      opts.Profile,
	})
	if err != nil {
		r.logger.Warn("could not load custom rules", "error", err)
	}

	// 2. Filter noise (lockfiles, binaries, minified code)
	filteredFiles, binaryWarnings := FilterDiffFiles(rawFiles, ruleConfig.Reviews.AutoReview.IgnorePaths)
	if len(filteredFiles) == 0 && len(binaryWarnings) == 0 {
		return &model.ReviewReport{
			Summary:              "No reviewable code changes detected in diff.",
			CodeScore:            100,
			SecurityScore:        100,
			MaintainabilityScore: 100,
			Verdict:              "APPROVE",
			HeadSHA:              opts.HeadSHA,
		}, nil
	}

	// 3. Pre-scan for hardcoded secrets
	secretIssues := ScanForSecrets(filteredFiles)

	// 4. Build AST-Scoped decorated context for each file
	var modifiedPaths []string
	var blocks []fileContextBlock

	for _, file := range filteredFiles {
		path := file.NewPath
		if path == "" {
			path = file.OldPath
		}
		modifiedPaths = append(modifiedPaths, path)

		var b strings.Builder
		b.WriteString(fmt.Sprintf("\n%s\n", strings.Repeat("=", 60)))
		b.WriteString(fmt.Sprintf("FILE: %s\n", path))

		if file.IsDeleted {
			b.WriteString("(DELETED FILE)\n")
			diffText := file.RawDiff
			if len(diffText) > maxSingleDiffChars {
				diffText = diffText[:maxSingleDiffChars] + "\n# ... remaining deleted lines truncated ...\n"
			}
			b.WriteString(fmt.Sprintf("DIFF:\n%s\n", diffText))
			blocks = append(blocks, fileContextBlock{path: path, block: b.String()})
			continue
		}

		if file.IsRenamed {
			b.WriteString(fmt.Sprintf("(RENAMED from %s)\n", file.OldPath))
		}

		// Fetch file content at HeadSHA
		fileContent, err := p.FetchFileContent(ctx, target, path, opts.HeadSHA)
		if err == nil && fileContent != "" {
			scopedContent := ast.SliceFileContext(fileContent, path, file.ChangedLines)
			b.WriteString("CODE (DECORATED WITH LINE NUMBERS):\n")
			b.WriteString(scopedContent + "\n")
		} else {
			b.WriteString("(Full source content unavailable; analyzing raw diff)\n")
		}

		diffText := file.RawDiff
		if len(diffText) > maxSingleDiffChars {
			diffText = diffText[:maxSingleDiffChars] + "\n# ... remaining diff truncated ...\n"
		}
		b.WriteString(fmt.Sprintf("DIFF:\n%s\n", diffText))

		blocks = append(blocks, fileContextBlock{path: path, block: b.String()})
	}

	// 5. Partition blocks into batches that fit comfortably within LLM context
	var batches [][]fileContextBlock
	var currentBatch []fileContextBlock
	currentBatchLen := 0

	for _, b := range blocks {
		if len(currentBatch) > 0 && currentBatchLen+len(b.block) > maxBatchChars {
			batches = append(batches, currentBatch)
			currentBatch = nil
			currentBatchLen = 0
		}
		currentBatch = append(currentBatch, b)
		currentBatchLen += len(b.block)
	}
	if len(currentBatch) > 0 {
		batches = append(batches, currentBatch)
	}

	// Cap max batches for very large MRs (focus on top 5 most important batches)
	if len(batches) > 5 {
		r.logger.Info("large MR detected, reviewing primary batches", "total_batches", len(batches), "reviewing", 5)
		batches = batches[:5]
	}

	// 6. Match active rules
	activeRules := rules.FilterApplicableRules(ruleConfig.Reviews.Rules, modifiedPaths)

	projectPath := opts.ProjectPath
	if projectPath == "" {
		projectPath = filepath.Base(opts.Workspace)
	}

	// 7. Execute AI review for each batch
	var batchReports []*model.ReviewReport

	for i, batch := range batches {
		var contextBuilder strings.Builder
		for _, b := range batch {
			contextBuilder.WriteString(b.block)
		}

		// Attach binary warnings to first batch
		if i == 0 {
			for _, binPath := range binaryWarnings {
				contextBuilder.WriteString(fmt.Sprintf("\n%s\n", strings.Repeat("=", 60)))
				contextBuilder.WriteString(fmt.Sprintf("FILE: %s (NEW BINARY ASSET COMMITTED)\n", binPath))
				contextBuilder.WriteString("NOTE: Binary files must be uploaded to CDN/S3 instead of git repository.\n")
			}
		}

		if len(batches) > 1 {
			r.logger.Info(fmt.Sprintf("reviewing batch %d/%d (%d files)...", i+1, len(batches), len(batch)))
		}

		reviewPrompt := agent.BuildReviewPrompt(projectPath, ruleConfig.CustomPrompt, ruleConfig.Reviews.Profile, activeRules, contextBuilder.String())

		rep, err := r.executeSingleTurn(ctx, opts, reviewPrompt, modifiedPaths)
		if err != nil {
			r.logger.Error("batch review failed", "batch", i+1, "error", err)
			continue
		}
		batchReports = append(batchReports, rep)
	}

	if len(batchReports) == 0 {
		return nil, fmt.Errorf("all review batches failed to return valid reports")
	}

	// 8. Merge batch reports into a single unified report
	finalReport := mergeReports(batchReports, modifiedPaths, opts.HeadSHA)

	// Merge pre-scanned secret issues
	if len(secretIssues) > 0 {
		finalReport.Issues = append(secretIssues, finalReport.Issues...)
		finalReport.SecurityScore = min(finalReport.SecurityScore, 40)
	}

	// 9. Apply comment budgeting
	topIssues, _ := ApplyCommentBudget(finalReport.Issues, opts.MaxComments)
	finalReport.Issues = topIssues

	finalReport.CalculateVerdict()
	return finalReport, nil
}

func (r *Reviewer) executeSingleTurn(ctx context.Context, opts ReviewOptions, prompt string, modifiedPaths []string) (*model.ReviewReport, error) {
	agentCfg := agent.NewReviewAgentConfig(agent.ConfigOptions{
		Workspace:       opts.Workspace,
		Verbose:         opts.Verbose,
		Logger:          r.logger,
		LitellmBaseURL:  opts.LitellmBaseURL,
		LitellmAPIKey:   opts.LitellmAPIKey,
		LitellmModel:    opts.LitellmModel,
		LitellmEndpoint: opts.LitellmEndpoint,
	})

	aiAgent, err := adk.NewAgent(agentCfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create LocalHarness agent: %w", err)
	}
	defer aiAgent.Close()

	if err := aiAgent.Start(ctx); err != nil {
		return nil, fmt.Errorf("failed to start LocalHarness agent: %w", err)
	}

	events, err := aiAgent.ChatStream(ctx, prompt)
	if err != nil {
		return nil, fmt.Errorf("agent chat stream failed: %w", err)
	}

	var rawResponse strings.Builder
	for event := range events {
		switch event.Type {
		case adk.EventTextDelta:
			rawResponse.WriteString(event.TextDelta)
			if opts.Verbose {
				fmt.Print(event.TextDelta)
			}
		case adk.EventToolCallStart:
			if opts.Verbose {
				fmt.Printf("\n🔧 Tool: %s\n", event.Step.ToolName)
			}
		case adk.EventToolCallDone:
			if opts.Verbose {
				fmt.Println("   ✅ done")
			}
		case adk.EventError:
			r.logger.Error("agent turn error", "error", event.Step.ErrorMessage)
		case adk.EventTurnComplete:
			if event.Error != nil {
				r.logger.Error("turn completed with error", "error", event.Error)
			}
		}
	}

	report, err := extractJSONReport(rawResponse.String())
	if err != nil {
		r.logger.Warn("could not extract strict JSON from LLM output, building fallback report", "error", err)
		report = &model.ReviewReport{
			Summary:              rawResponse.String(),
			CodeScore:            80,
			SecurityScore:        85,
			MaintainabilityScore: 80,
			FilesAnalyzed:        modifiedPaths,
		}
	}

	return report, nil
}

func mergeReports(reports []*model.ReviewReport, allFiles []string, headSHA string) *model.ReviewReport {
	if len(reports) == 1 {
		rep := reports[0]
		rep.HeadSHA = headSHA
		rep.FilesAnalyzed = allFiles
		return rep
	}

	merged := &model.ReviewReport{
		HeadSHA:       headSHA,
		FilesAnalyzed: allFiles,
	}

	var summaries []string
	var poems []string
	totalCodeScore := 0
	totalSecurityScore := 0
	totalMaintScore := 0

	for _, rep := range reports {
		if rep.SuggestedTitle != "" && merged.SuggestedTitle == "" {
			merged.SuggestedTitle = rep.SuggestedTitle
		}
		if rep.Summary != "" {
			summaries = append(summaries, rep.Summary)
		}
		if rep.Poem != "" {
			poems = append(poems, rep.Poem)
		}
		merged.Issues = append(merged.Issues, rep.Issues...)
		merged.PositiveFeedback = append(merged.PositiveFeedback, rep.PositiveFeedback...)

		totalCodeScore += rep.CodeScore
		totalSecurityScore += rep.SecurityScore
		totalMaintScore += rep.MaintainabilityScore
	}

	n := len(reports)
	merged.CodeScore = totalCodeScore / n
	merged.SecurityScore = totalSecurityScore / n
	merged.MaintainabilityScore = totalMaintScore / n

	merged.Summary = strings.Join(summaries, "\n\n")
	if len(poems) > 0 {
		merged.Poem = poems[0]
	}

	return merged
}

var jsonBlockRegex = regexp.MustCompile("(?s)```(?:json)?\\s*(\\{.*?\\})\\s*```")

func extractJSONReport(raw string) (*model.ReviewReport, error) {
	var report model.ReviewReport
	if err := json.Unmarshal([]byte(strings.TrimSpace(raw)), &report); err == nil {
		return &report, nil
	}

	if matches := jsonBlockRegex.FindStringSubmatch(raw); len(matches) >= 2 {
		if err := json.Unmarshal([]byte(matches[1]), &report); err == nil {
			return &report, nil
		}
	}

	firstBrace := strings.Index(raw, "{")
	lastBrace := strings.LastIndex(raw, "}")
	if firstBrace != -1 && lastBrace > firstBrace {
		slice := raw[firstBrace : lastBrace+1]
		if err := json.Unmarshal([]byte(slice), &report); err == nil {
			return &report, nil
		}
	}

	return nil, fmt.Errorf("no valid JSON review report found in response")
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
