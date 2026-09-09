package provider

// TargetContext holds all resolved target parameters for a review.
type TargetContext struct {
	Provider      string // "gitlab", "github", "bitbucket", "local"
	BaseURL       string // e.g., "https://gitlab.com", "https://gitlab.corp.internal"
	Owner         string // Org / Group / Username
	Repo          string // Project name / Repo slug
	ProjectPath   string // Full path with namespace (e.g. "group/subgroup/project")
	PRID          int    // Pull Request / Merge Request number
	HeadSHA       string // Target commit SHA
	BaseSHA       string // Base commit SHA
	StartSHA      string // Start commit SHA
	FromSHA       string // Incremental compare start SHA
	Token         string // API access token
	SkipTLSVerify bool   // For internal self-signed TLS certs
	IsDraft       bool   // Whether the PR is in draft/WIP state
	Title         string // Current PR title
	Description   string // Current PR description
	WebURL        string // Web browser URL of the PR
	Workspace     string // Local workspace directory path
}
