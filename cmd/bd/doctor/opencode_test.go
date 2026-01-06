package doctor

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCheckOpencode(t *testing.T) {
	// Verify CheckOpencode returns a valid DoctorCheck
	check := CheckOpencode()

	if check.Name != "OpenCode Integration" {
		t.Errorf("Expected check name 'OpenCode Integration', got %s", check.Name)
	}

	validStatuses := map[string]bool{"ok": true, "warning": true, "error": true}
	if !validStatuses[check.Status] {
		t.Errorf("Invalid status: %s", check.Status)
	}

	// If warning, should have fix message
	if check.Status == "warning" && check.Fix == "" {
		t.Error("Expected fix message for warning status")
	}
}

func TestHasOpencodeHooks(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		expected bool
	}{
		{
			name:     "invalid JSON",
			content:  "not valid json",
			expected: false,
		},
		{
			name:     "missing experimental section",
			content:  `{"other": "data"}`,
			expected: false,
		},
		{
			name:     "missing hook section",
			content:  `{"experimental": {"other": "data"}}`,
			expected: false,
		},
		{
			name: "valid hooks with bd prime",
			content: `{
				"experimental": {
					"hook": {
						"session_start": [
							{
								"command": ["bd", "prime"]
							}
						],
						"pre_compact": [
							{
								"command": ["bd", "prime"]
							}
						]
					}
				}
			}`,
			expected: true,
		},
		{
			name: "hooks without bd prime",
			content: `{
				"experimental": {
					"hook": {
						"session_start": [
							{
								"command": ["echo", "hello"]
							}
						],
						"pre_compact": [
							{
								"command": ["echo", "world"]
							}
						]
					}
				}
			}`,
			expected: false,
		},
		{
			name: "only session_start has bd prime",
			content: `{
				"experimental": {
					"hook": {
						"session_start": [
							{
								"command": ["bd", "prime"]
							}
						],
						"pre_compact": [
							{
								"command": ["echo", "world"]
							}
						]
					}
				}
			}`,
			expected: false,
		},
		{
			name: "only pre_compact has bd prime",
			content: `{
				"experimental": {
					"hook": {
						"session_start": [
							{
								"command": ["echo", "hello"]
							}
						],
						"pre_compact": [
							{
								"command": ["bd", "prime"]
							}
						]
					}
				}
			}`,
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			configPath := filepath.Join(tmpDir, "opencode.json")

			if err := os.WriteFile(configPath, []byte(tt.content), 0644); err != nil {
				t.Fatal(err)
			}

			result := hasOpencodeHooks(configPath)

			if result != tt.expected {
				t.Errorf("Expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestHasOpencodeHooksWithInvalidPath(t *testing.T) {
	result := hasOpencodeHooks("/nonexistent/path/to/opencode.json")

	if result != false {
		t.Error("Expected false for non-existent config file")
	}
}

func TestHasOpencodeHooksGlobal(t *testing.T) {
	// Sanity check for global hooks detection
	result := hasOpencodeHooksGlobal()

	// Just verify it returns a boolean without panicking
	if result != true && result != false {
		t.Error("Expected boolean result from hasOpencodeHooksGlobal")
	}
}

func TestHasOpencodeHooksProject(t *testing.T) {
	// Sanity check for project hooks detection
	result := hasOpencodeHooksProject()

	// Just verify it returns a boolean without panicking
	if result != true && result != false {
		t.Error("Expected boolean result from hasOpencodeHooksProject")
	}
}
