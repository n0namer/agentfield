package harness

import (
	"runtime"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestPlatformDefaults(t *testing.T) {
	tests := []struct {
		name     string
		goos     string
		provider string
		model    string
		variant  string
	}{
		{name: "windows", goos: "windows", provider: ProviderCodex, model: DefaultWindowsModel, variant: DefaultWindowsVariant},
		{name: "non-windows", goos: "linux", provider: ProviderOpenCode, model: "broker/fast-coding", variant: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv(ProviderEnvVar, "")
			got := platformDefaultsFor(tt.goos, Options{})
			require.Equal(t, tt.provider, got.Provider)
			require.Equal(t, tt.model, got.Model)
			require.Equal(t, tt.variant, got.Variant)
		})
	}
}

func TestRunnerResolutionAppliesPlatformDefaults(t *testing.T) {
	t.Setenv(ProviderEnvVar, "")

	windows := resolveRunnerOptionsFor("windows", Options{}, Options{})
	require.Equal(t, ProviderCodex, windows.Provider)
	require.Equal(t, "gpt-5.6-luna", windows.Model)
	require.Equal(t, "low", windows.Variant)

	unix := resolveRunnerOptionsFor("linux", Options{}, Options{})
	require.Equal(t, ProviderOpenCode, unix.Provider)
	require.Equal(t, "broker/fast-coding", unix.Model)
	require.Empty(t, unix.Variant)
}

func TestRunnerResolutionPreservesExplicitOptionsAndProviderEnv(t *testing.T) {
	t.Setenv(ProviderEnvVar, ProviderOpenCode)
	got := resolveRunnerOptionsFor("windows", Options{Model: "default-model"}, Options{
		Provider: ProviderCodex,
		Model:    "explicit-model",
		Variant:  "explicit-variant",
	})
	require.Equal(t, ProviderCodex, got.Provider)
	require.Equal(t, "explicit-model", got.Model)
	require.Equal(t, "explicit-variant", got.Variant)

	got = resolveRunnerOptionsFor("windows", Options{}, Options{})
	require.Equal(t, ProviderOpenCode, got.Provider)
	require.Empty(t, got.Model)
	require.Empty(t, got.Variant)
}

func TestPlatformDefaultsExplicitValuesWin(t *testing.T) {
	t.Setenv(ProviderEnvVar, ProviderOpenCode)
	got := platformDefaultsFor("windows", Options{
		Provider: ProviderCodex,
		Model:    "custom/model",
		Variant:  "custom-variant",
	})
	require.Equal(t, Options{Provider: ProviderCodex, Model: "custom/model", Variant: "custom-variant"}, got)

	got = platformDefaultsFor(runtime.GOOS, Options{Provider: ProviderOpenCode, Model: "custom/model"})
	require.Equal(t, ProviderOpenCode, got.Provider)
	require.Equal(t, "custom/model", got.Model)
}

func TestPlatformDefaultsDoNotApplyToExplicitProvider(t *testing.T) {
	t.Setenv(ProviderEnvVar, "")
	t.Setenv(ModelEnvVar, "")
	t.Setenv(VariantEnvVar, "")
	t.Setenv("HARNESS_MODEL", "")
	t.Setenv("HARNESS_VARIANT", "")

	got := platformDefaultsFor("windows", Options{Provider: ProviderGemini})
	require.Equal(t, ProviderGemini, got.Provider)
	require.Empty(t, got.Model)
	require.Empty(t, got.Variant)

	t.Setenv(ProviderEnvVar, ProviderOpenCode)
	got = platformDefaultsFor("windows", Options{})
	require.Equal(t, ProviderOpenCode, got.Provider)
	require.Empty(t, got.Model)
	require.Empty(t, got.Variant)
}

func TestPlatformDefaultsHonorModelVariantEnvironment(t *testing.T) {
	t.Setenv(ProviderEnvVar, "")
	t.Setenv(ModelEnvVar, "env/model")
	t.Setenv(VariantEnvVar, "high")
	got := platformDefaultsFor("windows", Options{})
	require.Equal(t, "env/model", got.Model)
	require.Equal(t, "high", got.Variant)

	got = platformDefaultsFor("windows", Options{Model: "explicit/model", Variant: "medium"})
	require.Equal(t, "explicit/model", got.Model)
	require.Equal(t, "medium", got.Variant)

	t.Setenv(ModelEnvVar, "")
	t.Setenv(VariantEnvVar, "")
	t.Setenv("HARNESS_MODEL", "alias/model")
	t.Setenv("HARNESS_VARIANT", "minimal")
	got = platformDefaultsFor("linux", Options{})
	require.Equal(t, "alias/model", got.Model)
	require.Equal(t, "minimal", got.Variant)
}
