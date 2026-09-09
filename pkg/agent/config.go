package agent

import (
	"log/slog"
	"os"

	"github.com/divmora/localharness/adk"
	"github.com/divmora/localharness/adk/connection"
	"github.com/divmora/localharness/adk/middleware"
	"github.com/divmora/localharness/adk/policy"
)

const localharnessVersion = "0.3.0"

// ConfigOptions holds settings for agent initialization.
type ConfigOptions struct {
	Workspace       string
	Verbose         bool
	Logger          *slog.Logger
	LitellmBaseURL  string
	LitellmAPIKey   string
	LitellmModel    string
	LitellmEndpoint string
}

// NewReviewAgentConfig creates a LocalAgentConfig configured for code reviews.
func NewReviewAgentConfig(opts ConfigOptions) *adk.LocalAgentConfig {
	cfg := adk.NewLocalAgentConfig()
	cfg.Workspaces = []adk.WorkspaceDef{{Directory: opts.Workspace}}
	cfg.Policies = []policy.Policy{policy.AllowAll()}
	cfg.Capabilities.RunCommand = false // Read-only safety for code review

	// 1. Resolve Base URL: CLI Flag -> CODE_REVIEWER_LITELLM_BASE_URL -> LITELLM_BASE_URL
	baseURL := opts.LitellmBaseURL
	if baseURL == "" {
		baseURL = os.Getenv("CODE_REVIEWER_LITELLM_BASE_URL")
	}
	if baseURL == "" {
		baseURL = os.Getenv("LITELLM_BASE_URL")
	}

	// 2. Resolve API Key: CLI Flag -> CODE_REVIEWER_LITELLM_API_KEY -> LITELLM_API_KEY
	apiKey := opts.LitellmAPIKey
	if apiKey == "" {
		apiKey = os.Getenv("CODE_REVIEWER_LITELLM_API_KEY")
	}
	if apiKey == "" {
		apiKey = os.Getenv("LITELLM_API_KEY")
	}

	// 3. Resolve Model: CLI Flag -> CODE_REVIEWER_LITELLM_MODEL -> LITELLM_MODEL
	modelName := opts.LitellmModel
	if modelName == "" {
		modelName = os.Getenv("CODE_REVIEWER_LITELLM_MODEL")
	}
	if modelName == "" {
		modelName = os.Getenv("LITELLM_MODEL")
	}

	// 4. Resolve Named Endpoint: CLI Flag -> CODE_REVIEWER_LITELLM_ENDPOINT -> LITELLM_ENDPOINT
	endpoint := opts.LitellmEndpoint
	if endpoint == "" {
		endpoint = os.Getenv("CODE_REVIEWER_LITELLM_ENDPOINT")
	}
	if endpoint == "" {
		endpoint = os.Getenv("LITELLM_ENDPOINT")
	}

	if baseURL != "" {
		cfg.LitellmBaseURL = baseURL
	}
	if apiKey != "" {
		cfg.LitellmAPIKey = apiKey
	}
	if modelName != "" {
		cfg.LitellmModel = modelName
	}
	if endpoint != "" {
		cfg.LitellmEndpoint = endpoint
	}

	logger := opts.Logger
	if logger == nil {
		level := slog.LevelInfo
		if opts.Verbose {
			level = slog.LevelDebug
		}
		logger = slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: level}))
	}
	cfg.Logger = logger
	cfg.Verbose = opts.Verbose

	// Auto-resolve localharness binary
	resolver := &connection.BinaryResolver{
		Version: localharnessVersion,
		Logger:  logger,
	}
	if binPath, err := resolver.Resolve(""); err == nil {
		cfg.BinaryPath = binPath
	}

	// Middleware Pipeline
	cfg.Middlewares = []middleware.Middleware{
		middleware.NewTokenGuard(120000, 0.8, logger),
		middleware.NewPatchToolArgs(logger),
	}

	cfg.MaxSubagentDepth = 1
	cfg.MaxAutoWakeTurns = 5

	return cfg
}
