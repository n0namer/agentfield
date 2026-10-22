package harness

import (
	"context"
	"os"
	"runtime"
	"strings"
)

const (
	// ProviderAforge is the provider name for the Aforge CLI.
	ProviderAforge = "aforge"
	// ProviderOpenCode is the provider name for OpenCode CLI.
	ProviderOpenCode = "opencode"
	// ProviderClaudeCode is the provider name for Claude Code CLI.
	ProviderClaudeCode = "claude-code"
	// ProviderCodex is the provider name for Codex CLI.
	ProviderCodex = "codex"
	// ProviderGemini is the provider name for Gemini CLI.
	ProviderGemini = "gemini"
	// ProviderPi is the provider name for the Pi coding-agent CLI.
	ProviderPi = "pi"
	// ProviderOMP is the provider name for the Oh My Pi coding-agent CLI.
	ProviderOMP = "omp"
	// ProviderCursor is the provider name for the Cursor CLI (`agent`).
	ProviderCursor = "cursor"
)

const (
	// DefaultProvider is retained for source compatibility with callers that
	// need a stable provider constant. Runtime resolution is platform-aware.
	DefaultProvider = ProviderAforge

	// ProviderEnvVar overrides the default provider process-wide. An
	// explicitly configured provider still wins over it.
	ProviderEnvVar = "AGENTFIELD_HARNESS_PROVIDER"
	// ModelEnvVar and VariantEnvVar override platform model/variant defaults.
	// Explicit Options.Model / Options.Variant still win over them.
	ModelEnvVar   = "AGENTFIELD_HARNESS_MODEL"
	VariantEnvVar = "AGENTFIELD_HARNESS_VARIANT"
)

const (
	DefaultWindowsModel   = "gpt-5.6-luna"
	DefaultWindowsVariant = "low"
	DefaultUnixModel      = "broker/fast-coding"
	// Unix/FCM keeps its existing reasoning default; no variant is invented.
	DefaultUnixVariant = ""
)

func platformDefaultsFor(goos string, options Options) Options {
	platformProvider := ProviderOpenCode
	if goos == "windows" {
		platformProvider = ProviderCodex
	}
	if provider := strings.TrimSpace(options.Provider); provider != "" {
		options.Provider = provider
	} else {
		if envProvider := strings.TrimSpace(os.Getenv(ProviderEnvVar)); envProvider != "" {
			options.Provider = envProvider
		} else {
			options.Provider = platformProvider
		}
	}

	if strings.TrimSpace(options.Model) == "" {
		if envModel := strings.TrimSpace(os.Getenv(ModelEnvVar)); envModel != "" {
			options.Model = envModel
		}
	}
	if strings.TrimSpace(options.Variant) == "" {
		if envVariant := strings.TrimSpace(os.Getenv(VariantEnvVar)); envVariant != "" {
			options.Variant = envVariant
		}
	}

	// Platform model/variant defaults belong to the implicit platform
	// provider. An explicitly selected non-platform provider keeps its own
	// defaults unless the caller supplied an AgentField-specific env override.
	if options.Provider != platformProvider {
		return options
	}

	switch options.Provider {
	case ProviderCodex:
		if strings.TrimSpace(options.Model) == "" {
			options.Model = DefaultWindowsModel
		}
		if strings.TrimSpace(options.Variant) == "" {
			options.Variant = DefaultWindowsVariant
		}
	case ProviderOpenCode:
		if strings.TrimSpace(options.Model) == "" {
			options.Model = DefaultUnixModel
		}
		if strings.TrimSpace(options.Variant) == "" {
			options.Variant = DefaultUnixVariant
		}
	}
	return options
}

func resolveRunnerOptionsFor(goos string, defaults, overrides Options) Options {
	runner := &Runner{DefaultOptions: defaults}
	return platformDefaultsFor(goos, runner.mergeOptions(overrides))
}

// ResolvePlatformDefaults fills unset provider, model, and variant fields.
// Explicit values win, then AgentField harness env overrides, then platform defaults.
func ResolvePlatformDefaults(options Options) Options {
	return platformDefaultsFor(runtime.GOOS, options)
}

// Provider is the interface that CLI-based harness providers implement.
// Each provider knows how to invoke a specific coding agent (opencode,
// claude-code, etc.) and return a RawResult.
type Provider interface {
	Execute(ctx context.Context, prompt string, options Options) (*RawResult, error)
}

// Options control a single harness invocation. Fields are optional;
// zero values mean "use default".
type Options struct {
	// Provider name: "aforge", "opencode", "claude-code", "codex",
	// "gemini", "pi", "omp", or "cursor". An explicit value wins over
	// AGENTFIELD_HARNESS_PROVIDER; when both are empty, the platform provider
	// is selected (Codex on Windows, OpenCode elsewhere).
	Provider string

	// Model identifier passed to the coding agent. It may carry a
	// reasoning-effort variant after a "#" separator (e.g.
	// "openrouter/z-ai/glm-5.2#high"); providers strip the suffix and map
	// the variant to their effort control when they have one.
	Model string

	// Variant is the provider-specific reasoning-effort variant (e.g.
	// "high", "max", "minimal"). When set it wins over a "#variant" suffix
	// on Model. Providers without an effort control drop it.
	Variant string

	// MaxTurns limits the number of agent iterations.
	MaxTurns int

	// PermissionMode controls tool permissions ("auto", "plan").
	PermissionMode string

	// SystemPrompt is prepended to the user prompt.
	SystemPrompt string

	// Env is additional environment variables for the subprocess.
	// Empty string values unset variables in the subprocess environment.
	Env map[string]string

	// Cwd is the working directory for the subprocess.
	Cwd string

	// ProjectDir is the project root the coding agent explores.
	// When set, the output file is placed inside this directory so the
	// coding agent's Write tool can reach it.
	ProjectDir string

	// Tools is a list of allowed tools for the agent.
	Tools []string

	// MaxBudgetUSD is a cost cap (provider-dependent).
	MaxBudgetUSD float64

	// ResumeSessionID resumes a previous session (provider-dependent).
	ResumeSessionID string

	// BinPath overrides the provider binary path.
	BinPath string

	// Timeout in seconds for the subprocess. 0 means provider default.
	Timeout int

	// MaxRetries for transient errors. Default 3.
	MaxRetries int

	// InitialDelay in seconds for retry backoff. Default 1.0.
	InitialDelay float64

	// MaxDelay in seconds for retry backoff. Default 30.0.
	MaxDelay float64

	// BackoffFactor for exponential backoff. Default 2.0.
	BackoffFactor float64

	// SchemaMaxRetries controls how many times to retry when schema
	// validation fails. Default 2.
	SchemaMaxRetries int

	// SchemaMode selects how schema-constrained output is produced:
	//
	//	"single" (default, or "") — the agent writes the whole JSON object in
	//	    one shot (cheapest).
	//	"incremental"             — the agent builds the object one top-level
	//	    field at a time, and recovery patches only the failing fields
	//	    (more robust for large or deeply nested schemas).
	//	"auto"                    — incremental only when the schema is large
	//	    (compact JSON token estimate exceeds the large-schema threshold).
	//
	// Mirrors the Python SDK's schema_mode argument to .harness().
	SchemaMode string

	// schemaOutputDir is runner-owned internal state for schema-constrained
	// OpenCode runs. Providers may observe durable completion markers here,
	// but it is intentionally not part of the public Options contract.
	schemaOutputDir string
}

func (o Options) maxRetries() int {
	if o.MaxRetries > 0 {
		return o.MaxRetries
	}
	return 3
}

func (o Options) initialDelay() float64 {
	if o.InitialDelay > 0 {
		return o.InitialDelay
	}
	return 1.0
}

func (o Options) maxDelay() float64 {
	if o.MaxDelay > 0 {
		return o.MaxDelay
	}
	return 30.0
}

func (o Options) backoffFactor() float64 {
	if o.BackoffFactor > 0 {
		return o.BackoffFactor
	}
	return 2.0
}

func (o Options) schemaMaxRetries() int {
	if o.SchemaMaxRetries > 0 {
		return o.SchemaMaxRetries
	}
	return 2
}

func (o Options) timeout() int {
	if o.Timeout > 0 {
		return o.Timeout
	}
	return 600
}
