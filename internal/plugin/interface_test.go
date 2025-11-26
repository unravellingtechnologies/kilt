package plugin

import (
	"testing"
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
			if got := tt.phase.String(); got != tt.want {
				t.Errorf("ExecutionPhase.String() = %v, want %v", got, tt.want)
			}
		})
	}
}

