package setup

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

var (
	opencodeEnvProvider     = defaultOpencodeEnv
	errOpencodeHooksMissing = errors.New("opencode hooks not installed")
)

type opencodeEnv struct {
	stdout     io.Writer
	stderr     io.Writer
	homeDir    string
	projectDir string
	ensureDir  func(string, os.FileMode) error
	readFile   func(string) ([]byte, error)
	writeFile  func(string, []byte) error
}

func defaultOpencodeEnv() (opencodeEnv, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return opencodeEnv{}, fmt.Errorf("home directory: %w", err)
	}
	workDir, err := os.Getwd()
	if err != nil {
		return opencodeEnv{}, fmt.Errorf("working directory: %w", err)
	}
	return opencodeEnv{
		stdout:     os.Stdout,
		stderr:     os.Stderr,
		homeDir:    home,
		projectDir: workDir,
		ensureDir:  EnsureDir,
		readFile:   os.ReadFile,
		writeFile: func(path string, data []byte) error {
			return atomicWriteFile(path, data)
		},
	}, nil
}

// opencodeConfigNames lists config file names in priority order for reading.
// OpenCode reads config.json, opencode.json, opencode.jsonc (later files override earlier).
// For writing, we prefer opencode.json as it's the canonical name.
var opencodeConfigNames = []string{"config.json", "opencode.json", "opencode.jsonc"}

// projectOpencodeConfigPaths returns all possible project config paths in priority order.
// Includes both root-level and .opencode/ subdirectory locations.
func projectOpencodeConfigPaths(base string) []string {
	var paths []string
	// Check .opencode/ subdirectory first (more specific)
	for _, name := range opencodeConfigNames {
		paths = append(paths, filepath.Join(base, ".opencode", name))
	}
	// Then check root level
	for _, name := range opencodeConfigNames {
		paths = append(paths, filepath.Join(base, name))
	}
	return paths
}

// globalOpencodeConfigPaths returns all possible global config paths in priority order.
func globalOpencodeConfigPaths(home string) []string {
	configDir := filepath.Join(home, ".config", "opencode")
	var paths []string
	for _, name := range opencodeConfigNames {
		paths = append(paths, filepath.Join(configDir, name))
	}
	return paths
}

// findExistingOpencodeConfig finds the first existing config file from the given paths.
// Returns empty string if none exist.
func findExistingOpencodeConfig(readFile func(string) ([]byte, error), paths []string) string {
	for _, p := range paths {
		if _, err := readFile(p); err == nil {
			return p
		}
	}
	return ""
}

// preferredOpencodeConfigPath returns the preferred path for writing a new config.
// For project configs, prefers .opencode/opencode.json.
// For global configs, prefers ~/.config/opencode/opencode.json.
func preferredOpencodeConfigPath(paths []string) string {
	// Find opencode.json in the list (preferred for writing)
	for _, p := range paths {
		if filepath.Base(p) == "opencode.json" {
			return p
		}
	}
	// Fallback to first path
	if len(paths) > 0 {
		return paths[0]
	}
	return ""
}

// InstallOpencode installs OpenCode hooks
func InstallOpencode(project bool, stealth bool) {
	env, err := opencodeEnvProvider()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		setupExit(1)
		return
	}
	if err := installOpencode(env, project, stealth); err != nil {
		setupExit(1)
	}
}

func installOpencode(env opencodeEnv, project bool, stealth bool) error {
	var configPaths []string
	if project {
		configPaths = projectOpencodeConfigPaths(env.projectDir)
		_, _ = fmt.Fprintln(env.stdout, "Installing OpenCode hooks for this project...")
	} else {
		configPaths = globalOpencodeConfigPaths(env.homeDir)
		_, _ = fmt.Fprintln(env.stdout, "Installing OpenCode hooks globally...")
	}

	// Find existing config or use preferred path for new installs
	configPath := findExistingOpencodeConfig(env.readFile, configPaths)
	if configPath == "" {
		configPath = preferredOpencodeConfigPath(configPaths)
	}

	if err := env.ensureDir(filepath.Dir(configPath), 0o755); err != nil {
		_, _ = fmt.Fprintf(env.stderr, "Error: %v\n", err)
		return err
	}

	config := make(map[string]interface{})
	if data, err := env.readFile(configPath); err == nil {
		if err := json.Unmarshal(data, &config); err != nil {
			_, _ = fmt.Fprintf(env.stderr, "Error: failed to parse %s: %v\n", filepath.Base(configPath), err)
			return err
		}
	}

	experimental, ok := config["experimental"].(map[string]interface{})
	if !ok {
		experimental = make(map[string]interface{})
		config["experimental"] = experimental
	}

	hook, ok := experimental["hook"].(map[string]interface{})
	if !ok {
		hook = make(map[string]interface{})
		experimental["hook"] = hook
	}

	command := []string{"bd", "prime"}
	if stealth {
		command = []string{"bd", "prime", "--stealth"}
	}

	if addOpencodeHookCommand(hook, "session_start", command) {
		_, _ = fmt.Fprintln(env.stdout, "✓ Registered session_start hook")
	}
	if addOpencodeHookCommand(hook, "pre_compact", command) {
		_, _ = fmt.Fprintln(env.stdout, "✓ Registered pre_compact hook")
	}

	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		_, _ = fmt.Fprintf(env.stderr, "Error: marshal config: %v\n", err)
		return err
	}

	if err := env.writeFile(configPath, data); err != nil {
		_, _ = fmt.Fprintf(env.stderr, "Error: write config: %v\n", err)
		return err
	}

	_, _ = fmt.Fprintln(env.stdout, "\n✓ OpenCode integration installed")
	_, _ = fmt.Fprintf(env.stdout, "  Config: %s\n", configPath)
	_, _ = fmt.Fprintln(env.stdout, "\nRestart OpenCode for changes to take effect.")
	return nil
}

// CheckOpencode checks if OpenCode integration is installed
func CheckOpencode() {
	env, err := opencodeEnvProvider()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		setupExit(1)
		return
	}
	if err := checkOpencode(env); err != nil {
		setupExit(1)
	}
}

func checkOpencode(env opencodeEnv) error {
	globalPaths := globalOpencodeConfigPaths(env.homeDir)
	projectPaths := projectOpencodeConfigPaths(env.projectDir)

	// Check global configs
	for _, configPath := range globalPaths {
		if hasOpencodeBeadsHooks(configPath) {
			_, _ = fmt.Fprintf(env.stdout, "✓ Global hooks installed: %s\n", configPath)
			return nil
		}
	}

	// Check project configs
	for _, configPath := range projectPaths {
		if hasOpencodeBeadsHooks(configPath) {
			_, _ = fmt.Fprintf(env.stdout, "✓ Project hooks installed: %s\n", configPath)
			return nil
		}
	}

	_, _ = fmt.Fprintln(env.stdout, "✗ No hooks installed")
	_, _ = fmt.Fprintln(env.stdout, "  Run: bd setup opencode")
	return errOpencodeHooksMissing
}

// RemoveOpencode removes OpenCode hooks
func RemoveOpencode(project bool) {
	env, err := opencodeEnvProvider()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		setupExit(1)
		return
	}
	if err := removeOpencode(env, project); err != nil {
		setupExit(1)
	}
}

func removeOpencode(env opencodeEnv, project bool) error {
	var configPaths []string
	if project {
		configPaths = projectOpencodeConfigPaths(env.projectDir)
		_, _ = fmt.Fprintln(env.stdout, "Removing OpenCode hooks from project...")
	} else {
		configPaths = globalOpencodeConfigPaths(env.homeDir)
		_, _ = fmt.Fprintln(env.stdout, "Removing OpenCode hooks globally...")
	}

	// Find existing config file
	configPath := findExistingOpencodeConfig(env.readFile, configPaths)
	if configPath == "" {
		_, _ = fmt.Fprintln(env.stdout, "No config file found")
		return nil
	}

	data, err := env.readFile(configPath)
	if err != nil {
		_, _ = fmt.Fprintln(env.stdout, "No config file found")
		return nil
	}

	var config map[string]interface{}
	if err := json.Unmarshal(data, &config); err != nil {
		_, _ = fmt.Fprintf(env.stderr, "Error: failed to parse %s: %v\n", filepath.Base(configPath), err)
		return err
	}

	experimental, ok := config["experimental"].(map[string]interface{})
	if !ok {
		_, _ = fmt.Fprintln(env.stdout, "No hooks found")
		return nil
	}

	hook, ok := experimental["hook"].(map[string]interface{})
	if !ok {
		_, _ = fmt.Fprintln(env.stdout, "No hooks found")
		return nil
	}

	removeOpencodeHookCommand(hook, "session_start")
	removeOpencodeHookCommand(hook, "pre_compact")

	data, err = json.MarshalIndent(config, "", "  ")
	if err != nil {
		_, _ = fmt.Fprintf(env.stderr, "Error: marshal config: %v\n", err)
		return err
	}

	if err := env.writeFile(configPath, data); err != nil {
		_, _ = fmt.Fprintf(env.stderr, "Error: write config: %v\n", err)
		return err
	}

	_, _ = fmt.Fprintln(env.stdout, "✓ OpenCode hooks removed")
	return nil
}

// addOpencodeHookCommand adds a hook command to an event if not already present
// Returns true if hook was added, false if already exists
func addOpencodeHookCommand(hook map[string]interface{}, event string, command []string) bool {
	// Get or create event array
	eventHooks, ok := hook[event].([]interface{})
	if !ok {
		eventHooks = []interface{}{}
	}

	// Check if bd hook already registered
	for _, h := range eventHooks {
		hookMap, ok := h.(map[string]interface{})
		if !ok {
			continue
		}
		cmdArray, ok := hookMap["command"].([]interface{})
		if !ok {
			continue
		}
		if commandArrayContainsBdPrime(cmdArray) {
			fmt.Printf("✓ Hook already registered: %s\n", event)
			return false
		}
	}

	// Add bd hook to array
	newHook := map[string]interface{}{
		"command": command,
	}

	eventHooks = append(eventHooks, newHook)
	hook[event] = eventHooks
	return true
}

// removeOpencodeHookCommand removes bd prime hooks from an event
func removeOpencodeHookCommand(hook map[string]interface{}, event string) {
	eventHooks, ok := hook[event].([]interface{})
	if !ok {
		return
	}

	// Filter out bd prime hooks
	var filtered []interface{}
	for _, h := range eventHooks {
		hookMap, ok := h.(map[string]interface{})
		if !ok {
			filtered = append(filtered, h)
			continue
		}

		cmdArray, ok := hookMap["command"].([]interface{})
		if !ok {
			filtered = append(filtered, h)
			continue
		}

		if commandArrayContainsBdPrime(cmdArray) {
			fmt.Printf("✓ Removed %s hook\n", event)
			continue
		}

		filtered = append(filtered, h)
	}

	hook[event] = filtered
}

// commandArrayContainsBdPrime checks if a command array contains "bd" and "prime"
func commandArrayContainsBdPrime(cmdArray []interface{}) bool {
	if len(cmdArray) < 2 {
		return false
	}
	first, ok1 := cmdArray[0].(string)
	second, ok2 := cmdArray[1].(string)
	if !ok1 || !ok2 {
		return false
	}
	return first == "bd" && second == "prime"
}

// hasOpencodeBeadsHooks checks if a config file has bd prime hooks
func hasOpencodeBeadsHooks(configPath string) bool {
	data, err := os.ReadFile(configPath) // #nosec G304 -- configPath is constructed from known safe locations (user home/.config/opencode), not user input
	if err != nil {
		return false
	}

	var config map[string]interface{}
	if err := json.Unmarshal(data, &config); err != nil {
		return false
	}

	experimental, ok := config["experimental"].(map[string]interface{})
	if !ok {
		return false
	}

	hook, ok := experimental["hook"].(map[string]interface{})
	if !ok {
		return false
	}

	// Check session_start and pre_compact for "bd" "prime"
	for _, event := range []string{"session_start", "pre_compact"} {
		eventHooks, ok := hook[event].([]interface{})
		if !ok {
			continue
		}

		for _, h := range eventHooks {
			hookMap, ok := h.(map[string]interface{})
			if !ok {
				continue
			}
			cmdArray, ok := hookMap["command"].([]interface{})
			if !ok {
				continue
			}
			if commandArrayContainsBdPrime(cmdArray) {
				return true
			}
		}
	}

	return false
}
