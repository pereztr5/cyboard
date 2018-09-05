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

func TestIPish(t *testing.T) {
	cases := []struct {
		name    string
		prefix  string
		success bool
	}{
		// You could argue whether some of these cases *should* cause errors,
		// but the IPish validator catches the problems that could be common.
		{"subnet addr", "192.168.0.", true},
		{"home addr", "127.0.0.", true},
		{"netmask", "255.255.255.", true},

		{"missing last dot", "192.168.0", false},
		{"too few", "192.0.", false},
		{"empty", "", false},
		{"full ip", "10.230.0.5", false},
		{"too many octets", "10.10.10.10.10", false},
		{"malformed", "apple.", false},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			got := IPish(tt.prefix)
			assert.Equal(t, tt.success, got)
		})
	}
}
