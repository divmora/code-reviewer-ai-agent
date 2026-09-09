package rules

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/divmora/code-reviewer-ai-agent/pkg/model"
	"gopkg.in/yaml.v3"
)

// IngestionOptions specifies all potential rule sources.
type IngestionOptions struct {
	Workspace    string
	CustomPrompt string // From --prompt or Zenith UI
	RulesJSON    string // From --rules-json
	RulesFile    string // From --rules
	InlineRule   string // From --rule
	Profile      string // From --profile
}

// LoadRules aggregates and parses rules from repo files and CLI / Zenith inputs.
func LoadRules(opts IngestionOptions) (*model.RuleConfig, error) {
	config := &model.RuleConfig{
		Language: "en-US",
		Reviews: model.ReviewConfigOptions{
			Profile:          model.ProfileBalanced,
			HighLevelSummary: true,
			FileWalkthrough:  true,
			Poem:             true,
			AutoReview: model.AutoReviewConfig{
				Enabled: true,
				Drafts:  false,
				IgnorePaths: []string{
					"**/vendor/**",
					"**/node_modules/**",
					"**/*.pb.go",
					"**/*_gen.go",
					"**/*.min.js",
					"**/*.min.css",
					"**/*.lock",
					"**/go.sum",
				},
			},
			Rules: DefaultRules,
		},
	}

	// 1. Auto-discover repo-level configs if workspace provided
	if opts.Workspace != "" {
		candidates := []string{
			filepath.Join(opts.Workspace, ".coderabbit.yaml"),
			filepath.Join(opts.Workspace, ".coderabbit.yml"),
			filepath.Join(opts.Workspace, ".zenith.yaml"),
			filepath.Join(opts.Workspace, ".zenith.yml"),
			filepath.Join(opts.Workspace, ".code-reviewer.yaml"),
			filepath.Join(opts.Workspace, ".code-reviewer.yml"),
		}

		for _, path := range candidates {
			if _, err := os.Stat(path); err == nil {
				if err := loadFileIntoConfig(path, config); err == nil {
					break
				}
			}
		}
	}

	// 2. Explicit rules file from --rules
	if opts.RulesFile != "" {
		if err := loadFileIntoConfig(opts.RulesFile, config); err != nil {
			return nil, fmt.Errorf("failed to load rules file %q: %w", opts.RulesFile, err)
		}
	}

	// 3. Inline JSON from --rules-json (Zenith UI payload)
	if opts.RulesJSON != "" {
		var inlineCfg model.RuleConfig
		if err := json.Unmarshal([]byte(opts.RulesJSON), &inlineCfg); err == nil {
			mergeConfigs(config, &inlineCfg)
		} else {
			// Might be a direct list of rules
			var inlineRules []model.CustomRule
			if err := json.Unmarshal([]byte(opts.RulesJSON), &inlineRules); err == nil {
				config.Reviews.Rules = append(config.Reviews.Rules, inlineRules...)
			}
		}
	}

	// 4. Custom prompt from --prompt or environment variable
	if opts.CustomPrompt != "" {
		config.CustomPrompt = opts.CustomPrompt
	} else if envPrompt := os.Getenv("ZENITH_CUSTOM_PROMPT"); envPrompt != "" {
		config.CustomPrompt = envPrompt
	}

	// 5. Ad-hoc inline rule from --rule
	if opts.InlineRule != "" {
		config.Reviews.Rules = append(config.Reviews.Rules, model.CustomRule{
			ID:          "cli-adhoc-rule",
			Name:        "Custom CLI Rule",
			Severity:    model.SeverityMedium,
			Category:    model.CategoryBestPractice,
			Description: opts.InlineRule,
		})
	}

	// 6. Profile override from --profile
	if opts.Profile != "" {
		config.Reviews.Profile = model.Profile(opts.Profile)
	}

	return config, nil
}

func loadFileIntoConfig(path string, target *model.RuleConfig) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	var parsed model.RuleConfig
	if err := yaml.Unmarshal(data, &parsed); err != nil {
		if err := json.Unmarshal(data, &parsed); err != nil {
			return err
		}
	}

	mergeConfigs(target, &parsed)
	return nil
}

func mergeConfigs(base, override *model.RuleConfig) {
	if override.CustomPrompt != "" {
		base.CustomPrompt = override.CustomPrompt
	}
	if override.Reviews.Profile != "" {
		base.Reviews.Profile = override.Reviews.Profile
	}
	if len(override.Reviews.AutoReview.IgnorePaths) > 0 {
		base.Reviews.AutoReview.IgnorePaths = append(base.Reviews.AutoReview.IgnorePaths, override.Reviews.AutoReview.IgnorePaths...)
	}
	if len(override.Reviews.Rules) > 0 {
		base.Reviews.Rules = append(base.Reviews.Rules, override.Reviews.Rules...)
	}
}
