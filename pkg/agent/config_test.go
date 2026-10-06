package agent

import (
	"testing"
)

func TestNewReviewAgentConfig(t *testing.T) {
	opts := ConfigOptions{
		Workspace:       "/tmp/test-workspace",
		Verbose:         true,
		LitellmBaseURL:  "http://localhost:4000",
		LitellmAPIKey:   "test-key",
		LitellmModel:    "gpt-4o",
		LitellmEndpoint: "default",
	}

	cfg := NewReviewAgentConfig(opts)

	if cfg == nil {
		t.Fatal("expected non-nil LocalAgentConfig")
	}

	if len(cfg.Workspaces) != 1 || cfg.Workspaces[0].Directory != "/tmp/test-workspace" {
		t.Errorf("unexpected workspace: %+v", cfg.Workspaces)
	}

	if cfg.LitellmBaseURL != "http://localhost:4000" {
		t.Errorf("expected LitellmBaseURL http://localhost:4000, got %s", cfg.LitellmBaseURL)
	}

	if cfg.LitellmAPIKey != "test-key" {
		t.Errorf("expected LitellmAPIKey test-key, got %s", cfg.LitellmAPIKey)
	}

	if cfg.LitellmModel != "gpt-4o" {
		t.Errorf("expected LitellmModel gpt-4o, got %s", cfg.LitellmModel)
	}

	if cfg.LitellmEndpoint != "default" {
		t.Errorf("expected LitellmEndpoint default, got %s", cfg.LitellmEndpoint)
	}

	if cfg.Capabilities.RunCommand {
		t.Errorf("expected RunCommand capability to be false for security")
	}

	if len(cfg.Middlewares) != 2 {
		t.Errorf("expected 2 middlewares configured, got %d", len(cfg.Middlewares))
	}
}
