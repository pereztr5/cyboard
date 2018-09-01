package server

import (
	"path/filepath"
	"testing"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_Configuration_Validate(t *testing.T) {
	const testpath = "testdata/config_tests"

	cases := []struct {
		configfile      string
		expectedErrText string
	}{
		{"ends_before_start", "Event starts after it ends"},
		{"neg_timeout", "Timeout must be positive"},
		{"breaks_out_of_order", "Breaks must be ordered earliest to latest"},
		{"negative_breaktime", "Breaks must go for a positive amount of time"},
		{"break_before_event", "Breaks must start after the event has started"},
		{"break_after_event", "Breaks must end before the event has ended"},
		{"valid", ""},
	}

	for _, tt := range cases {
		t.Run(tt.configfile, func(t *testing.T) {
			tomlfile := tt.configfile + ".toml"
			path := filepath.Join(testpath, tomlfile)

			v := viper.New()
			v.SetConfigFile(path)
			require.NoError(t, v.ReadInConfig())

			cfg := new(Configuration)
			require.NoError(t, v.Unmarshal(cfg))

			err := cfg.Validate()
			if tt.expectedErrText == "" {
				assert.NoError(t, err)
			} else if assert.Error(t, err) {
				assert.Contains(t, err.Error(), tt.expectedErrText)
			}
		})
	}
}
