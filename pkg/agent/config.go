package agent

import (
	"log/slog"
	"os"

	"github.com/divmora/localharness/adk"
	"github.com/divmora/localharness/adk/connection"
	"github.com/divmora/localharness/adk/middleware"
	"github.com/divmora/localharness/adk/policy"
)

const localharnessVersion = "2.0.0"

// ConfigOptions holds settings for agent initialization.
type ConfigOptions struct {
	Workspace string
	Verbose   bool
	Logger    *slog.Logger
}

// NewReviewAgentConfig creates a LocalAgentConfig configured for code reviews.
func NewReviewAgentConfig(opts ConfigOptions) *adk.LocalAgentConfig {
	cfg := adk.NewLocalAgentConfig()
	cfg.Workspaces = []adk.WorkspaceDef{{Directory: opts.Workspace}}
	cfg.Policies = []policy.Policy{policy.AllowAll()}
	cfg.Capabilities.RunCommand = false // Read-only safety for code review

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
