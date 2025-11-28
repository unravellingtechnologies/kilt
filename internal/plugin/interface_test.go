package plugin

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestExecutionPhase_String(t *testing.T) {
	tests := []struct {
		phase ExecutionPhase
		want  string
	}{
		{PhasePreSync, "PreSync"},
		{PhaseCore, "Core"},
		{PhaseRunOnce, "RunOnce"},
		{PhaseOnChange, "OnChange"},
		{PhaseIntegration, "Integration"},
		{PhasePostSync, "PostSync"},
		{ExecutionPhase(99), "Unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			assert.Equal(t, tt.want, tt.phase.String())
		})
	}
}
