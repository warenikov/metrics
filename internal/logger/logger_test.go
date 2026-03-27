package logger

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestInitialize(t *testing.T) {
	tests := []struct {
		name    string
		level   string
		wantErr bool
	}{
		{name: "info level", level: "info"},
		{name: "debug level", level: "debug"},
		{name: "warn level", level: "warn"},
		{name: "error level", level: "error"},
		{name: "invalid level", level: "superverbose", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := Initialize(tt.level)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.NotNil(t, Log)
			assert.IsType(t, &zap.Logger{}, Log)
		})
	}
}
