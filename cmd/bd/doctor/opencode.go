package doctor

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// CheckOpencode returns OpenCode integration verification as a DoctorCheck
func CheckOpencode() DoctorCheck {
	hasGlobalHooks := hasOpencodeHooksGlobal()
	hasProjectHooks := hasOpencodeHooksProject()

	if hasGlobalHooks && hasProjectHooks {
		return DoctorCheck{
			Name:    "OpenCode Integration",
			Status:  StatusOK,
			Message: "Hooks installed (global and project)",
			Detail:  "bd prime hooks enabled in session_start and pre_compact",
		}
	} else if hasGlobalHooks {
		return DoctorCheck{
			Name:    "OpenCode Integration",
			Status:  StatusOK,
			Message: "Hooks installed (global)",
			Detail:  "bd prime hooks enabled in session_start and pre_compact",
		}
	} else if hasProjectHooks {
		return DoctorCheck{
			Name:    "OpenCode Integration",
			Status:  StatusOK,
			Message: "Hooks installed (project)",
			Detail:  "bd prime hooks enabled in session_start and pre_compact",
		}
	}

	return DoctorCheck{
		Name:    "OpenCode Integration",
		Status:  StatusWarning,
		Message: "Not configured",
		Detail:  "OpenCode can use bd more effectively with session hooks",
		Fix: "Set up OpenCode integration:\n" +
			"  Run 'bd setup opencode' to add session_start/pre_compact hooks\n" +
			"\n" +
			"Benefits:\n" +
			"  • Auto-inject workflow context on session start\n" +
			"  • Automatic context recovery before compaction\n" +
			"\n" +
			"See: bd setup opencode --help",
	}
}

// hasOpencodeHooksGlobal checks if bd prime hooks exist in global OpenCode config
func hasOpencodeHooksGlobal() bool {
	home, err := os.UserHomeDir()
	if err != nil {
		return false
	}

	globalConfig := filepath.Join(home, ".config", "opencode", "opencode.json")
	return hasOpencodeHooks(globalConfig)
}

// hasOpencodeHooksProject checks if bd prime hooks exist in project OpenCode config
func hasOpencodeHooksProject() bool {
	projectConfig := ".opencode/opencode.json"
	return hasOpencodeHooks(projectConfig)
}

// hasOpencodeHooks checks if a config file has bd prime hooks in session_start and pre_compact
func hasOpencodeHooks(configPath string) bool {
	data, err := os.ReadFile(configPath) // #nosec G304 -- configPath is constructed from known safe locations, not user input
	if err != nil {
		return false
	}

	var config map[string]interface{}
	if err := json.Unmarshal(data, &config); err != nil {
		return false
	}

	// Navigate to experimental.hook
	experimental, ok := config["experimental"].(map[string]interface{})
	if !ok {
		return false
	}

	hook, ok := experimental["hook"].(map[string]interface{})
	if !ok {
		return false
	}

	// Check both session_start and pre_compact for "bd prime"
	hasSessionStart := hasBdPrimeInHook(hook, "session_start")
	hasPreCompact := hasBdPrimeInHook(hook, "pre_compact")

	return hasSessionStart && hasPreCompact
}

// hasBdPrimeInHook checks if a specific hook event contains a "bd prime" command
func hasBdPrimeInHook(hook map[string]interface{}, eventName string) bool {
	eventHooks, ok := hook[eventName].([]interface{})
	if !ok {
		return false
	}

	for _, hookEntry := range eventHooks {
		hookMap, ok := hookEntry.(map[string]interface{})
		if !ok {
			continue
		}

		command, ok := hookMap["command"].([]interface{})
		if !ok {
			continue
		}

		// Check if command is ["bd", "prime"]
		if len(command) >= 2 {
			first, ok1 := command[0].(string)
			second, ok2 := command[1].(string)
			if ok1 && ok2 && first == "bd" && second == "prime" {
				return true
			}
		}
	}

	return false
}
