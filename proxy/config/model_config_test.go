package config

import (
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestConfig_ModelConfigSanitizedCommand(t *testing.T) {
	config := &ModelConfig{
		Cmd: `python model1.py \
    --arg1 value1 \
    --arg2 value2`,
	}

	args, err := config.SanitizedCommand()
	assert.NoError(t, err)
	assert.Equal(t, []string{"python", "model1.py", "--arg1", "value1", "--arg2", "value2"}, args)
}

func TestConfig_ModelFilters(t *testing.T) {
	content := `
macros:
  default_strip: "temperature, top_p"
models:
  model1:
    cmd: path/to/cmd --port ${PORT}
    filters:
      # macros inserted and list is cleaned of duplicates and empty strings
      stripParams: "model, top_k, top_k, temperature, ${default_strip}, , ,"
  # check for strip_params (legacy field name) compatibility
  legacy:
    cmd: path/to/cmd --port ${PORT}
    filters:
      strip_params: "model, top_k, top_k, temperature, ${default_strip}, , ,"
`
	config, err := LoadConfigFromReader(strings.NewReader(content))
	assert.NoError(t, err)
	for modelId, modelConfig := range config.Models {
		t.Run(fmt.Sprintf("Testing macros in filters for model %s", modelId), func(t *testing.T) {
			assert.Equal(t, "model, top_k, top_k, temperature, temperature, top_p, , ,", modelConfig.Filters.StripParams)
			sanitized, err := modelConfig.Filters.SanitizedStripParams()
			if assert.NoError(t, err) {
				// model has been removed
				// empty strings have been removed
				// duplicates have been removed
				assert.Equal(t, []string{"temperature", "top_k", "top_p"}, sanitized)
			}
		})
	}
}

func TestConfig_IsRemoteModel(t *testing.T) {
	tests := []struct {
		name     string
		cmd      string
		proxy    string
		expected bool
	}{
		// Local models (have cmd)
		{"local model with cmd", "python server.py", "http://localhost:8080", false},
		{"local model with cmd and remote proxy", "python server.py", "http://example.com:8080", false},

		// Remote models (no cmd, non-loopback proxy)
		{"remote model", "", "http://example.com:8080", true},
		{"remote model https", "", "https://api.openai.com/v1", true},
		{"remote model with port", "", "http://192.168.1.100:8080", true},
		{"remote model ipv6", "", "http://[2001:db8::1]:8080", true},

		// Loopback addresses (not remote)
		{"localhost", "", "http://localhost:8080", false},
		{"localhost no port", "", "http://localhost", false},
		{"127.0.0.1", "", "http://127.0.0.1:8080", false},
		{"127.0.0.2 (loopback range)", "", "http://127.0.0.2:8080", false},
		{"127.255.255.255 (loopback range)", "", "http://127.255.255.255:8080", false},
		{"ipv6 loopback", "", "http://[::1]:8080", false},

		// Edge cases
		{"invalid url", "", "not-a-url", true}, // url.Parse succeeds, hostname is "not-a-url", not IP, not localhost
		{"empty proxy", "", "", true},          // url.Parse succeeds with empty string
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := ModelConfig{
				Cmd:   tt.cmd,
				Proxy: tt.proxy,
			}
			assert.Equal(t, tt.expected, m.IsRemoteModel())
		})
	}
}

func TestConfig_ModelSendLoadingState(t *testing.T) {
	content := `
sendLoadingState: true
models:
  model1:
    cmd: path/to/cmd --port ${PORT}
    sendLoadingState: false
  model2:
    cmd: path/to/cmd --port ${PORT}
`
	config, err := LoadConfigFromReader(strings.NewReader(content))
	assert.NoError(t, err)
	assert.True(t, config.SendLoadingState)
	if assert.NotNil(t, config.Models["model1"].SendLoadingState) {
		assert.False(t, *config.Models["model1"].SendLoadingState)
	}
	if assert.NotNil(t, config.Models["model2"].SendLoadingState) {
		assert.True(t, *config.Models["model2"].SendLoadingState)
	}
}
