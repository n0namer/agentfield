package harness

import (
	"runtime"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestResolveProviderName(t *testing.T) {
	platformDefault := ProviderOpenCode
	if runtime.GOOS == "windows" {
		platformDefault = ProviderCodex
	}
	tests := []struct {
		name     string
		explicit string
		env      string
		want     string
	}{
		{name: "default", want: platformDefault},
		{name: "environment", env: ProviderCodex, want: ProviderCodex},
		{name: "explicit wins", explicit: ProviderGemini, env: ProviderCodex, want: ProviderGemini},
		{name: "blank explicit", explicit: "   ", want: platformDefault},
		{name: "blank environment", env: "  ", want: platformDefault},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv(ProviderEnvVar, tt.env)
			assert.Equal(t, tt.want, ResolveProviderName(tt.explicit))
		})
	}
}

func TestBuildProvider_DefaultsToPlatformProvider(t *testing.T) {
	t.Setenv(ProviderEnvVar, "")
	provider, err := BuildProvider("", "")
	require.NoError(t, err)
	require.NotNil(t, provider)
	if runtime.GOOS == "windows" {
		assert.IsType(t, &CodexProvider{}, provider)
	} else {
		assert.IsType(t, &OpenCodeProvider{}, provider)
	}
}

func TestBuildProvider_RejectsUnknownName(t *testing.T) {
	provider, err := BuildProvider("nope", "")
	assert.Nil(t, provider)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "nope")
	assert.True(t, strings.Contains(err.Error(), "supported: aforge, claude-code, codex, gemini, opencode, pi, omp, cursor"))
}
