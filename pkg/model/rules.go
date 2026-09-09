package model

// Profile sets the reviewer's sensitivity and tone.
type Profile string

const (
	ProfileChill     Profile = "chill"
	ProfileBalanced  Profile = "balanced"
	ProfileAssertive Profile = "assertive"
)

// CustomRule represents an individual custom rule definition.
type CustomRule struct {
	ID           string   `json:"id" yaml:"id"`
	Name         string   `json:"name" yaml:"name"`
	Severity     Severity `json:"severity" yaml:"severity"`
	Category     Category `json:"category" yaml:"category"`
	Description  string   `json:"description" yaml:"description"`
	Languages    []string `json:"languages,omitempty" yaml:"languages,omitempty"`
	FilePatterns []string `json:"file_patterns,omitempty" yaml:"file_patterns,omitempty"`
}

// AutoReviewConfig defines path exclusion and triggering options.
type AutoReviewConfig struct {
	Enabled     bool     `json:"enabled" yaml:"enabled"`
	Drafts      bool     `json:"drafts" yaml:"drafts"`
	IgnorePaths []string `json:"ignore_paths,omitempty" yaml:"ignore_paths,omitempty"`
}

// ReviewConfigOptions contains settings for PR reviews.
type ReviewConfigOptions struct {
	Profile          Profile          `json:"profile" yaml:"profile"`
	HighLevelSummary bool             `json:"high_level_summary" yaml:"high_level_summary"`
	FileWalkthrough  bool             `json:"file_walkthrough" yaml:"file_walkthrough"`
	Poem             bool             `json:"poem" yaml:"poem"`
	AutoReview       AutoReviewConfig `json:"auto_review,omitempty" yaml:"auto_review,omitempty"`
	Rules            []CustomRule     `json:"rules,omitempty" yaml:"rules,omitempty"`
}

// RuleConfig represents the top-level structure of a .coderabbit.yaml or .zenith.yaml file.
type RuleConfig struct {
	Version      string              `json:"version" yaml:"version"`
	Language     string              `json:"language" yaml:"language"`
	CustomPrompt string              `json:"custom_prompt,omitempty" yaml:"custom_prompt,omitempty"`
	Reviews      ReviewConfigOptions `json:"reviews,omitempty" yaml:"reviews,omitempty"`
}
