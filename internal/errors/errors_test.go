package errors

import (
	"errors"
	"strings"
	"testing"
)

func TestCLIError_Error(t *testing.T) {
	err := New(CategoryRuntime, "test error")
	if err.Error() != "test error" {
		t.Errorf("expected 'test error', got '%s'", err.Error())
	}
}

func TestCLIError_Unwrap(t *testing.T) {
	cause := errors.New("underlying error")
	err := Wrap(CategoryRuntime, "wrapper", cause)

	if err.Unwrap() != cause {
		t.Error("Unwrap should return the cause")
	}
}

func TestFormat_CLIError(t *testing.T) {
	tests := []struct {
		name     string
		err      *CLIError
		contains []string
	}{
		{
			name:     "basic error",
			err:      New(CategoryRuntime, "something failed"),
			contains: []string{"Error:", "something failed"},
		},
		{
			name:     "usage error",
			err:      New(CategoryUsage, "invalid argument"),
			contains: []string{"Usage error:", "invalid argument"},
		},
		{
			name:     "config error",
			err:      New(CategoryConfig, "missing config"),
			contains: []string{"Configuration error:", "missing config"},
		},
		{
			name:     "connection error",
			err:      New(CategoryConnection, "daemon unreachable"),
			contains: []string{"Connection error:", "daemon unreachable"},
		},
		{
			name:     "not found error",
			err:      New(CategoryNotFound, "worker missing"),
			contains: []string{"Not found:", "worker missing"},
		},
		{
			name:     "error with cause",
			err:      Wrap(CategoryRuntime, "operation failed", errors.New("permission denied")),
			contains: []string{"operation failed", "permission denied"},
		},
		{
			name:     "error with suggestion",
			err:      New(CategoryConnection, "daemon offline").WithSuggestion("multiclaude daemon start"),
			contains: []string{"daemon offline", "Try:", "multiclaude daemon start"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			formatted := Format(tt.err)
			for _, s := range tt.contains {
				if !strings.Contains(formatted, s) {
					t.Errorf("expected formatted error to contain '%s', got: %s", s, formatted)
				}
			}
		})
	}
}

func TestFormat_RegularError(t *testing.T) {
	err := errors.New("regular error")
	formatted := Format(err)

	if !strings.Contains(formatted, "Error:") {
		t.Errorf("expected 'Error:' prefix, got: %s", formatted)
	}
	if !strings.Contains(formatted, "regular error") {
		t.Errorf("expected error message, got: %s", formatted)
	}
}

func TestFormat_Nil(t *testing.T) {
	if Format(nil) != "" {
		t.Error("Format(nil) should return empty string")
	}
}

func TestDaemonNotRunning(t *testing.T) {
	err := DaemonNotRunning()

	if err.Category != CategoryConnection {
		t.Error("DaemonNotRunning should have CategoryConnection")
	}
	if err.Suggestion == "" {
		t.Error("DaemonNotRunning should have a suggestion")
	}

	formatted := Format(err)
	if !strings.Contains(formatted, "daemon") {
		t.Errorf("expected 'daemon' in message, got: %s", formatted)
	}
	if !strings.Contains(formatted, "multiclaude daemon start") {
		t.Errorf("expected suggestion, got: %s", formatted)
	}
}

func TestDaemonCommunicationFailed(t *testing.T) {
	cause := errors.New("connection refused")
	err := DaemonCommunicationFailed("listing repos", cause)

	if err.Category != CategoryConnection {
		t.Error("should have CategoryConnection")
	}
	if err.Cause != cause {
		t.Error("should wrap cause")
	}

	formatted := Format(err)
	if !strings.Contains(formatted, "listing repos") {
		t.Errorf("expected operation in message, got: %s", formatted)
	}
	if !strings.Contains(formatted, "connection refused") {
		t.Errorf("expected cause in message, got: %s", formatted)
	}
}

func TestNotInRepo(t *testing.T) {
	err := NotInRepo()
	formatted := Format(err)

	if !strings.Contains(formatted, "not in a tracked repository") {
		t.Errorf("expected message, got: %s", formatted)
	}
	if !strings.Contains(formatted, "multiclaude repo init") {
		t.Errorf("expected repo init suggestion, got: %s", formatted)
	}
}

func TestMultipleRepos(t *testing.T) {
	err := MultipleRepos()
	formatted := Format(err)

	if !strings.Contains(formatted, "--repo") {
		t.Errorf("expected --repo flag suggestion, got: %s", formatted)
	}
}

func TestAgentNotFound(t *testing.T) {
	err := AgentNotFound("worker", "test-worker", "my-repo")
	formatted := Format(err)

	if !strings.Contains(formatted, "test-worker") {
		t.Errorf("expected agent name, got: %s", formatted)
	}
	if !strings.Contains(formatted, "my-repo") {
		t.Errorf("expected repo name, got: %s", formatted)
	}
	if !strings.Contains(formatted, "multiclaude worker list") {
		t.Errorf("expected list suggestion, got: %s", formatted)
	}
}

func TestInvalidPRURL(t *testing.T) {
	err := InvalidPRURL()
	formatted := Format(err)

	if !strings.Contains(formatted, "github.com") {
		t.Errorf("expected example URL format, got: %s", formatted)
	}
}

func TestClaudeNotFound(t *testing.T) {
	err := ClaudeNotFound(errors.New("not found"))
	formatted := Format(err)

	if !strings.Contains(formatted, "claude") {
		t.Errorf("expected claude mention, got: %s", formatted)
	}
	if !strings.Contains(formatted, "install") || !strings.Contains(formatted, "anthropic") {
		t.Errorf("expected install suggestion, got: %s", formatted)
	}
}

func TestMissingArgument(t *testing.T) {
	err := MissingArgument("repo", "string")
	formatted := Format(err)

	if !strings.Contains(formatted, "repo") {
		t.Errorf("expected argument name, got: %s", formatted)
	}
	if !strings.Contains(formatted, "string") {
		t.Errorf("expected type hint, got: %s", formatted)
	}
}

func TestInvalidArgument(t *testing.T) {
	err := InvalidArgument("count", "abc", "integer")
	formatted := Format(err)

	if !strings.Contains(formatted, "count") {
		t.Errorf("expected argument name, got: %s", formatted)
	}
	if !strings.Contains(formatted, "abc") {
		t.Errorf("expected value, got: %s", formatted)
	}
	if !strings.Contains(formatted, "integer") {
		t.Errorf("expected expected type, got: %s", formatted)
	}
}

func TestUnknownCommand(t *testing.T) {
	err := UnknownCommand("foobar")
	formatted := Format(err)

	if !strings.Contains(formatted, "foobar") {
		t.Errorf("expected command name, got: %s", formatted)
	}
	if !strings.Contains(formatted, "--help") {
		t.Errorf("expected help suggestion, got: %s", formatted)
	}
}

func TestWithSuggestion_Chaining(t *testing.T) {
	err := New(CategoryRuntime, "failed").WithSuggestion("try again")

	if err.Suggestion != "try again" {
		t.Errorf("expected suggestion to be set, got: %s", err.Suggestion)
	}
}

func TestTmuxOperationFailed_SpecificSuggestions(t *testing.T) {
	tests := []struct {
		name         string
		operation    string
		causeMsg     string
		wantContains string
	}{
		{
			name:         "tmux not found",
			operation:    "create session",
			causeMsg:     "executable file not found in $PATH",
			wantContains: "could not find 'tmux' binary in PATH",
		},
		{
			name:         "duplicate session",
			operation:    "create session",
			causeMsg:     "duplicate session: mc-repo",
			wantContains: "tmux kill-session -t",
		},
		{
			name:         "generic error has no suggestion",
			operation:    "create session",
			causeMsg:     "exit status 1",
			wantContains: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var cause error
			if tt.causeMsg != "" {
				cause = errors.New(tt.causeMsg)
			}

			err := TmuxOperationFailed(tt.operation, cause)

			if tt.wantContains == "" {
				if err.Suggestion != "" {
					t.Errorf("expected no suggestion, got %q", err.Suggestion)
				}
			} else if !strings.Contains(err.Suggestion, tt.wantContains) {
				t.Errorf("suggestion %q should contain %q", err.Suggestion, tt.wantContains)
			}
		})
	}
}

func TestWorktreeCreationFailed_SpecificSuggestions(t *testing.T) {
	tests := []struct {
		name            string
		causeMsg        string
		wantContains    []string
		wantNotContains []string
	}{
		{
			name:         "branch already exists with name",
			causeMsg:     "failed to create worktree: exit status 128\nOutput: fatal: a branch named 'work/nice-owl' already exists",
			wantContains: []string{"work/nice-owl", "multiclaude cleanup", "git branch -D work/nice-owl"},
		},
		{
			name:         "generic branch already exists",
			causeMsg:     "branch already exists",
			wantContains: []string{"multiclaude cleanup", "previous run"},
		},
		{
			name:         "worktree path exists",
			causeMsg:     "path already exists",
			wantContains: []string{"multiclaude cleanup", "worktree directory"},
		},
		{
			name:         "is a worktree error",
			causeMsg:     "is a worktree",
			wantContains: []string{"multiclaude cleanup"},
		},
		{
			name:         "not a valid reference",
			causeMsg:     "not a valid reference: invalid-branch",
			wantContains: []string{"start branch does not exist", "git branch -a"},
		},
		{
			name:         "branch already checked out",
			causeMsg:     "already checked out at",
			wantContains: []string{"multiclaude cleanup", "another worktree"},
		},
		{
			name:            "generic error falls back to default",
			causeMsg:        "some random error",
			wantContains:    []string{"disk space", "repository state"},
			wantNotContains: []string{"multiclaude cleanup", "git branch"},
		},
		{
			name:         "nil cause uses default suggestion",
			causeMsg:     "",
			wantContains: []string{"disk space", "repository state"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var cause error
			if tt.causeMsg != "" {
				cause = errors.New(tt.causeMsg)
			}

			err := WorktreeCreationFailed(cause)

			for _, want := range tt.wantContains {
				if !strings.Contains(err.Suggestion, want) {
					t.Errorf("suggestion should contain %q, got: %q", want, err.Suggestion)
				}
			}
			for _, notWant := range tt.wantNotContains {
				if strings.Contains(err.Suggestion, notWant) {
					t.Errorf("suggestion should NOT contain %q, got: %q", notWant, err.Suggestion)
				}
			}
		})
	}
}

func TestExtractQuotedValue(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"fatal: a branch named 'work/nice-owl' already exists", "work/nice-owl"},
		{"some error 'value' here", "value"},
		{"no quotes here", ""},
		{"'only-one-quote", ""},
		{"''", ""},
		{"'test'", "test"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := extractQuotedValue(tt.input)
			if got != tt.want {
				t.Errorf("extractQuotedValue(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestInvalidUsage(t *testing.T) {
	err := InvalidUsage("command requires an argument")

	if err.Category != CategoryUsage {
		t.Errorf("expected CategoryUsage, got %v", err.Category)
	}
	if err.Message != "command requires an argument" {
		t.Errorf("expected message to match input, got: %s", err.Message)
	}

	formatted := Format(err)
	if !strings.Contains(formatted, "Usage error:") {
		t.Errorf("expected 'Usage error:' prefix, got: %s", formatted)
	}
	if !strings.Contains(formatted, "command requires an argument") {
		t.Errorf("expected message in output, got: %s", formatted)
	}
}

func TestGitOperationFailed(t *testing.T) {
	cause := errors.New("permission denied")
	err := GitOperationFailed("push", cause)

	if err.Category != CategoryRuntime {
		t.Errorf("expected CategoryRuntime, got %v", err.Category)
	}
	if err.Cause != cause {
		t.Error("should wrap cause")
	}
	if err.Suggestion == "" {
		t.Error("should have a suggestion")
	}

	formatted := Format(err)
	if !strings.Contains(formatted, "git push failed") {
		t.Errorf("expected operation in message, got: %s", formatted)
	}
	if !strings.Contains(formatted, "permission denied") {
		t.Errorf("expected cause in output, got: %s", formatted)
	}
	if !strings.Contains(formatted, "git status") {
		t.Errorf("expected git status suggestion, got: %s", formatted)
	}
}

func TestNotInAgentContext(t *testing.T) {
	err := NotInAgentContext()

	if err.Category != CategoryConfig {
		t.Errorf("expected CategoryConfig, got %v", err.Category)
	}
	if err.Suggestion == "" {
		t.Error("should have a suggestion")
	}

	formatted := Format(err)
	if !strings.Contains(formatted, "agent") {
		t.Errorf("expected 'agent' in message, got: %s", formatted)
	}
	if !strings.Contains(formatted, "tmux") {
		t.Errorf("expected tmux in suggestion, got: %s", formatted)
	}
}

func TestMissingArgument_WithoutType(t *testing.T) {
	err := MissingArgument("filename", "")

	formatted := Format(err)
	if !strings.Contains(formatted, "filename") {
		t.Errorf("expected argument name, got: %s", formatted)
	}
	// Should not contain parentheses when type is empty
	if strings.Contains(formatted, "()") {
		t.Errorf("should not show empty type parentheses, got: %s", formatted)
	}
}

func TestCategoryPrefix_DefaultCase(t *testing.T) {
	// Test that unknown category defaults to "Error:"
	err := &CLIError{
		Category: Category(999), // Invalid category
		Message:  "test message",
	}

	formatted := Format(err)
	if !strings.Contains(formatted, "Error:") {
		t.Errorf("expected default 'Error:' prefix for unknown category, got: %s", formatted)
	}
}

func TestNoRepositoriesFound(t *testing.T) {
	err := NoRepositoriesFound()

	if err.Category != CategoryNotFound {
		t.Errorf("expected CategoryNotFound, got %v", err.Category)
	}
	if err.Suggestion == "" {
		t.Error("should have a suggestion")
	}

	formatted := Format(err)
	if !strings.Contains(formatted, "no repositories found") {
		t.Errorf("expected 'no repositories found' in message, got: %s", formatted)
	}
	if !strings.Contains(formatted, "multiclaude repo init") {
		t.Errorf("expected repo init suggestion, got: %s", formatted)
	}
}

func TestNoWorkersFound(t *testing.T) {
	err := NoWorkersFound("my-repo")

	if err.Category != CategoryNotFound {
		t.Errorf("expected CategoryNotFound, got %v", err.Category)
	}
	if err.Suggestion == "" {
		t.Error("should have a suggestion")
	}

	formatted := Format(err)
	if !strings.Contains(formatted, "no workers found") {
		t.Errorf("expected 'no workers found' in message, got: %s", formatted)
	}
	if !strings.Contains(formatted, "my-repo") {
		t.Errorf("expected repo name in message, got: %s", formatted)
	}
	if !strings.Contains(formatted, "multiclaude worker create") {
		t.Errorf("expected worker create suggestion, got: %s", formatted)
	}
}

func TestNoWorkspacesFound(t *testing.T) {
	err := NoWorkspacesFound("my-repo")

	if err.Category != CategoryNotFound {
		t.Errorf("expected CategoryNotFound, got %v", err.Category)
	}
	if err.Suggestion == "" {
		t.Error("should have a suggestion")
	}

	formatted := Format(err)
	if !strings.Contains(formatted, "no workspaces found") {
		t.Errorf("expected 'no workspaces found' in message, got: %s", formatted)
	}
	if !strings.Contains(formatted, "my-repo") {
		t.Errorf("expected repo name in message, got: %s", formatted)
	}
	if !strings.Contains(formatted, "multiclaude workspace add") {
		t.Errorf("expected workspace add suggestion, got: %s", formatted)
	}
}

func TestNoAgentsFound(t *testing.T) {
	err := NoAgentsFound("my-repo")

	if err.Category != CategoryNotFound {
		t.Errorf("expected CategoryNotFound, got %v", err.Category)
	}
	if err.Suggestion == "" {
		t.Error("should have a suggestion")
	}

	formatted := Format(err)
	if !strings.Contains(formatted, "no agents found") {
		t.Errorf("expected 'no agents found' in message, got: %s", formatted)
	}
	if !strings.Contains(formatted, "my-repo") {
		t.Errorf("expected repo name in message, got: %s", formatted)
	}
	if !strings.Contains(formatted, "multiclaude worker list") {
		t.Errorf("expected worker list suggestion, got: %s", formatted)
	}
}

func TestWorkspaceNotFound(t *testing.T) {
	err := WorkspaceNotFound("my-workspace", "my-repo")

	if err.Category != CategoryNotFound {
		t.Errorf("expected CategoryNotFound, got %v", err.Category)
	}
	if err.Suggestion == "" {
		t.Error("should have a suggestion")
	}

	formatted := Format(err)
	if !strings.Contains(formatted, "my-workspace") {
		t.Errorf("expected workspace name in message, got: %s", formatted)
	}
	if !strings.Contains(formatted, "my-repo") {
		t.Errorf("expected repo name in message, got: %s", formatted)
	}
	if !strings.Contains(formatted, "multiclaude workspace list") {
		t.Errorf("expected workspace list suggestion, got: %s", formatted)
	}
}

// Tests for PR #340 structured error constructors

func TestRepoAlreadyExists(t *testing.T) {
	err := RepoAlreadyExists("my-repo")

	if err.Category != CategoryConfig {
		t.Errorf("expected CategoryConfig, got %v", err.Category)
	}
	if !strings.Contains(err.Message, "my-repo") {
		t.Errorf("expected repo name in message, got: %s", err.Message)
	}
	if !strings.Contains(err.Message, "already initialized") {
		t.Errorf("expected 'already initialized' in message, got: %s", err.Message)
	}
	if !strings.Contains(err.Suggestion, "multiclaude repo rm") {
		t.Errorf("expected rm suggestion, got: %s", err.Suggestion)
	}

	formatted := Format(err)
	if !strings.Contains(formatted, "Configuration error:") {
		t.Errorf("expected config error prefix, got: %s", formatted)
	}
}

func TestDirectoryAlreadyExists(t *testing.T) {
	err := DirectoryAlreadyExists("/tmp/test-dir")

	if err.Category != CategoryConfig {
		t.Errorf("expected CategoryConfig, got %v", err.Category)
	}
	if !strings.Contains(err.Message, "/tmp/test-dir") {
		t.Errorf("expected path in message, got: %s", err.Message)
	}
	if !strings.Contains(err.Suggestion, "remove") {
		t.Errorf("expected remove suggestion, got: %s", err.Suggestion)
	}
}

func TestWorkspaceAlreadyExists(t *testing.T) {
	err := WorkspaceAlreadyExists("dev", "my-repo")

	if err.Category != CategoryConfig {
		t.Errorf("expected CategoryConfig, got %v", err.Category)
	}
	if !strings.Contains(err.Message, "dev") {
		t.Errorf("expected workspace name in message, got: %s", err.Message)
	}
	if !strings.Contains(err.Message, "my-repo") {
		t.Errorf("expected repo name in message, got: %s", err.Message)
	}
	if !strings.Contains(err.Suggestion, "multiclaude workspace list") {
		t.Errorf("expected list suggestion, got: %s", err.Suggestion)
	}
}

func TestInvalidWorkspaceName(t *testing.T) {
	tests := []struct {
		name   string
		reason string
	}{
		{"", "cannot be empty"},
		{".", "cannot be '.' or '..'"},
		{".hidden", "cannot start with '.' or '-'"},
		{"bad..name", "cannot contain '..'"},
	}

	for _, tt := range tests {
		t.Run(tt.name+"_"+tt.reason, func(t *testing.T) {
			err := InvalidWorkspaceName(tt.name, tt.reason)

			if err.Category != CategoryUsage {
				t.Errorf("expected CategoryUsage, got %v", err.Category)
			}
			if !strings.Contains(err.Message, tt.reason) {
				t.Errorf("expected reason in message, got: %s", err.Message)
			}
			if !strings.Contains(err.Suggestion, "alphanumeric") {
				t.Errorf("expected naming guidance in suggestion, got: %s", err.Suggestion)
			}
		})
	}
}

func TestLogFileNotFound(t *testing.T) {
	err := LogFileNotFound("worker1", "my-repo")

	if err.Category != CategoryNotFound {
		t.Errorf("expected CategoryNotFound, got %v", err.Category)
	}
	if !strings.Contains(err.Message, "worker1") {
		t.Errorf("expected agent name in message, got: %s", err.Message)
	}
	if !strings.Contains(err.Message, "my-repo") {
		t.Errorf("expected repo name in message, got: %s", err.Message)
	}
	if err.Suggestion == "" {
		t.Error("should have a suggestion")
	}
}

func TestInvalidDuration(t *testing.T) {
	err := InvalidDuration("abc")

	if err.Category != CategoryUsage {
		t.Errorf("expected CategoryUsage, got %v", err.Category)
	}
	if !strings.Contains(err.Message, "abc") {
		t.Errorf("expected value in message, got: %s", err.Message)
	}
	if !strings.Contains(err.Suggestion, "7d") {
		t.Errorf("expected example format in suggestion, got: %s", err.Suggestion)
	}
}

func TestNoDefaultRepo(t *testing.T) {
	err := NoDefaultRepo()

	if err.Category != CategoryUsage {
		t.Errorf("expected CategoryUsage, got %v", err.Category)
	}
	if !strings.Contains(err.Message, "could not determine") {
		t.Errorf("expected message about repo determination, got: %s", err.Message)
	}
	if !strings.Contains(err.Suggestion, "--repo") {
		t.Errorf("expected --repo flag in suggestion, got: %s", err.Suggestion)
	}
	if !strings.Contains(err.Suggestion, "multiclaude repo use") {
		t.Errorf("expected repo use suggestion, got: %s", err.Suggestion)
	}
}

func TestStateLoadFailed(t *testing.T) {
	cause := errors.New("corrupted json")
	err := StateLoadFailed(cause)

	if err.Category != CategoryRuntime {
		t.Errorf("expected CategoryRuntime, got %v", err.Category)
	}
	if err.Cause != cause {
		t.Error("should wrap cause")
	}
	if !strings.Contains(err.Suggestion, "multiclaude repair") {
		t.Errorf("expected repair suggestion, got: %s", err.Suggestion)
	}

	formatted := Format(err)
	if !strings.Contains(formatted, "corrupted json") {
		t.Errorf("expected cause in formatted output, got: %s", formatted)
	}
}

func TestSessionIDGenerationFailed(t *testing.T) {
	cause := errors.New("entropy exhausted")
	err := SessionIDGenerationFailed("supervisor", cause)

	if err.Category != CategoryRuntime {
		t.Errorf("expected CategoryRuntime, got %v", err.Category)
	}
	if err.Cause != cause {
		t.Error("should wrap cause")
	}
	if !strings.Contains(err.Message, "supervisor") {
		t.Errorf("expected agent type in message, got: %s", err.Message)
	}
	if !strings.Contains(err.Suggestion, "try again") {
		t.Errorf("expected retry suggestion, got: %s", err.Suggestion)
	}
}

func TestPromptWriteFailed(t *testing.T) {
	cause := errors.New("disk full")
	err := PromptWriteFailed("worker", cause)

	if err.Category != CategoryRuntime {
		t.Errorf("expected CategoryRuntime, got %v", err.Category)
	}
	if err.Cause != cause {
		t.Error("should wrap cause")
	}
	if !strings.Contains(err.Message, "worker") {
		t.Errorf("expected agent type in message, got: %s", err.Message)
	}
	if !strings.Contains(err.Suggestion, "disk space") {
		t.Errorf("expected disk space suggestion, got: %s", err.Suggestion)
	}
}

func TestClaudeStartFailed(t *testing.T) {
	cause := errors.New("exit code 1")
	err := ClaudeStartFailed("merge-queue", cause)

	if err.Category != CategoryRuntime {
		t.Errorf("expected CategoryRuntime, got %v", err.Category)
	}
	if err.Cause != cause {
		t.Error("should wrap cause")
	}
	if !strings.Contains(err.Message, "merge-queue") {
		t.Errorf("expected agent type in message, got: %s", err.Message)
	}
	if !strings.Contains(err.Suggestion, "claude --version") {
		t.Errorf("expected version check suggestion, got: %s", err.Suggestion)
	}
}

func TestAgentRegistrationFailed(t *testing.T) {
	cause := errors.New("socket error")
	err := AgentRegistrationFailed("supervisor", cause)

	if err.Category != CategoryRuntime {
		t.Errorf("expected CategoryRuntime, got %v", err.Category)
	}
	if err.Cause != cause {
		t.Error("should wrap cause")
	}
	if !strings.Contains(err.Message, "supervisor") {
		t.Errorf("expected agent type in message, got: %s", err.Message)
	}
	if !strings.Contains(err.Suggestion, "multiclaude daemon status") {
		t.Errorf("expected daemon status suggestion, got: %s", err.Suggestion)
	}
}

func TestWorktreeCleanupNeeded(t *testing.T) {
	cause := errors.New("permission denied")
	err := WorktreeCleanupNeeded("/tmp/wt/worker1", cause)

	if err.Category != CategoryRuntime {
		t.Errorf("expected CategoryRuntime, got %v", err.Category)
	}
	if err.Cause != cause {
		t.Error("should wrap cause")
	}
	if !strings.Contains(err.Suggestion, "git worktree remove") {
		t.Errorf("expected worktree remove suggestion, got: %s", err.Suggestion)
	}
	if !strings.Contains(err.Suggestion, "/tmp/wt/worker1") {
		t.Errorf("expected path in suggestion, got: %s", err.Suggestion)
	}
}

func TestTmuxWindowCleanupNeeded(t *testing.T) {
	cause := errors.New("session not found")
	err := TmuxWindowCleanupNeeded("mc-repo", "worker1", cause)

	if err.Category != CategoryRuntime {
		t.Errorf("expected CategoryRuntime, got %v", err.Category)
	}
	if err.Cause != cause {
		t.Error("should wrap cause")
	}
	if !strings.Contains(err.Suggestion, "tmux kill-window") {
		t.Errorf("expected kill-window suggestion, got: %s", err.Suggestion)
	}
	if !strings.Contains(err.Suggestion, "mc-repo:worker1") {
		t.Errorf("expected session:window in suggestion, got: %s", err.Suggestion)
	}
}

func TestTmuxSessionCleanupNeeded(t *testing.T) {
	cause := errors.New("busy")
	err := TmuxSessionCleanupNeeded("mc-repo", cause)

	if err.Category != CategoryRuntime {
		t.Errorf("expected CategoryRuntime, got %v", err.Category)
	}
	if err.Cause != cause {
		t.Error("should wrap cause")
	}
	if !strings.Contains(err.Suggestion, "tmux kill-session") {
		t.Errorf("expected kill-session suggestion, got: %s", err.Suggestion)
	}
	if !strings.Contains(err.Suggestion, "mc-repo") {
		t.Errorf("expected session name in suggestion, got: %s", err.Suggestion)
	}
}

func TestInvalidTmuxSessionName(t *testing.T) {
	err := InvalidTmuxSessionName("repository name cannot be empty")

	if err.Category != CategoryUsage {
		t.Errorf("expected CategoryUsage, got %v", err.Category)
	}
	if !strings.Contains(err.Message, "repository name cannot be empty") {
		t.Errorf("expected reason in message, got: %s", err.Message)
	}
	if !strings.Contains(err.Suggestion, "must not be empty") {
		t.Errorf("expected naming guidance in suggestion, got: %s", err.Suggestion)
	}
}

func TestWorkerNotFound(t *testing.T) {
	err := WorkerNotFound("test-worker", "my-repo")

	if err.Category != CategoryNotFound {
		t.Errorf("expected CategoryNotFound, got %v", err.Category)
	}
	if !strings.Contains(err.Message, "test-worker") {
		t.Errorf("expected worker name in message, got: %s", err.Message)
	}
	if !strings.Contains(err.Message, "my-repo") {
		t.Errorf("expected repo name in message, got: %s", err.Message)
	}
	if !strings.Contains(err.Suggestion, "multiclaude worker list") {
		t.Errorf("expected list suggestion, got: %s", err.Suggestion)
	}
}

func TestAgentNoSessionID(t *testing.T) {
	err := AgentNoSessionID("supervisor")

	if err.Category != CategoryConfig {
		t.Errorf("expected CategoryConfig, got %v", err.Category)
	}
	if !strings.Contains(err.Message, "supervisor") {
		t.Errorf("expected agent name in message, got: %s", err.Message)
	}
	if !strings.Contains(err.Message, "no session ID") {
		t.Errorf("expected 'no session ID' in message, got: %s", err.Message)
	}
	if !strings.Contains(err.Suggestion, "removing and recreating") {
		t.Errorf("expected recreate suggestion, got: %s", err.Suggestion)
	}
}

// TestAllNewConstructorsFormat verifies all PR #340 constructors produce valid formatted output
func TestAllNewConstructorsFormat(t *testing.T) {
	cause := errors.New("test cause")

	constructors := []struct {
		name string
		err  *CLIError
	}{
		{"RepoAlreadyExists", RepoAlreadyExists("repo")},
		{"DirectoryAlreadyExists", DirectoryAlreadyExists("/tmp/dir")},
		{"WorkspaceAlreadyExists", WorkspaceAlreadyExists("ws", "repo")},
		{"InvalidWorkspaceName", InvalidWorkspaceName("bad", "reason")},
		{"LogFileNotFound", LogFileNotFound("agent", "repo")},
		{"InvalidDuration", InvalidDuration("xyz")},
		{"NoDefaultRepo", NoDefaultRepo()},
		{"StateLoadFailed", StateLoadFailed(cause)},
		{"SessionIDGenerationFailed", SessionIDGenerationFailed("worker", cause)},
		{"PromptWriteFailed", PromptWriteFailed("worker", cause)},
		{"ClaudeStartFailed", ClaudeStartFailed("worker", cause)},
		{"AgentRegistrationFailed", AgentRegistrationFailed("worker", cause)},
		{"WorktreeCleanupNeeded", WorktreeCleanupNeeded("/path", cause)},
		{"TmuxWindowCleanupNeeded", TmuxWindowCleanupNeeded("session", "window", cause)},
		{"TmuxSessionCleanupNeeded", TmuxSessionCleanupNeeded("session", cause)},
		{"InvalidTmuxSessionName", InvalidTmuxSessionName("reason")},
		{"WorkerNotFound", WorkerNotFound("name", "repo")},
		{"AgentNoSessionID", AgentNoSessionID("name")},
	}

	for _, tt := range constructors {
		t.Run(tt.name, func(t *testing.T) {
			// Verify it's a valid CLIError
			if tt.err == nil {
				t.Fatal("constructor returned nil")
			}

			// Verify Error() returns non-empty
			if tt.err.Error() == "" {
				t.Error("Error() should return non-empty string")
			}

			// Verify Format() produces output
			formatted := Format(tt.err)
			if formatted == "" {
				t.Error("Format() should return non-empty string")
			}

			// Verify formatted output contains the message
			if !strings.Contains(formatted, tt.err.Message) {
				t.Errorf("formatted output should contain message %q, got: %s", tt.err.Message, formatted)
			}

			// Verify suggestion is included when present
			if tt.err.Suggestion != "" {
				if !strings.Contains(formatted, "Try:") {
					t.Errorf("formatted output should contain 'Try:' for errors with suggestions, got: %s", formatted)
				}
			}
		})
	}
}
