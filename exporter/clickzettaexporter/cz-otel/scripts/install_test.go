// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package scripts

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// TestInstallScript_SyntaxValid verifies that install.sh is syntactically valid bash.
func TestInstallScript_SyntaxValid(t *testing.T) {
	scriptPath := filepath.Join("install.sh")
	if _, err := os.Stat(scriptPath); os.IsNotExist(err) {
		t.Skip("install.sh not found in scripts directory")
	}

	cmd := exec.Command("bash", "-n", scriptPath)
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("install.sh has syntax errors: %v\nOutput: %s", err, output)
	}
}

// TestInstallScript_ContainsExpectedCommands verifies that install.sh contains
// the expected commands and patterns for a remote installer.
func TestInstallScript_ContainsExpectedCommands(t *testing.T) {
	scriptPath := filepath.Join("install.sh")
	data, err := os.ReadFile(scriptPath)
	if err != nil {
		t.Fatalf("failed to read install.sh: %v", err)
	}
	content := string(data)

	expectedPatterns := []struct {
		pattern     string
		description string
	}{
		{"mkdir -p", "directory creation command"},
		{"/bin", "bin directory path"},
		{"/logs", "logs directory path"},
		{"chmod +x", "executable permission setting"},
		{"otelcol-clickzetta", "collector binary name"},
		{"cz-otel", "CLI binary name"},
		{"PATH", "PATH instructions"},
		{"config init", "next steps guidance"},
		{"detect_os", "OS detection function"},
		{"detect_arch", "architecture detection function"},
		{"tar -xzf", "archive extraction command"},
		{"github.com", "GitHub download URL"},
		{"CZ_OTEL_VERSION", "version override variable"},
		{"INSTALL_DIR", "install directory override variable"},
	}

	for _, p := range expectedPatterns {
		if !strings.Contains(content, p.pattern) {
			t.Errorf("install.sh missing expected %s (pattern: %q)", p.description, p.pattern)
		}
	}
}

// TestInstallScript_HasMainFunction verifies the script defines and calls a main function.
func TestInstallScript_HasMainFunction(t *testing.T) {
	scriptPath := filepath.Join("install.sh")
	data, err := os.ReadFile(scriptPath)
	if err != nil {
		t.Fatalf("failed to read install.sh: %v", err)
	}
	content := string(data)

	if !strings.Contains(content, "main()") {
		t.Error("install.sh missing main() function definition")
	}
	if !strings.Contains(content, `main "$@"`) {
		t.Error("install.sh missing main invocation at end of script")
	}
}

// TestInstallScript_HasErrorHandling verifies the script has proper error handling.
func TestInstallScript_HasErrorHandling(t *testing.T) {
	scriptPath := filepath.Join("install.sh")
	data, err := os.ReadFile(scriptPath)
	if err != nil {
		t.Fatalf("failed to read install.sh: %v", err)
	}
	content := string(data)

	if !strings.Contains(content, "set -e") {
		t.Error("install.sh missing 'set -e' for error handling")
	}
	if !strings.Contains(content, "trap") {
		t.Error("install.sh missing trap for cleanup")
	}
	if !strings.Contains(content, "error()") || !strings.Contains(content, "exit 1") {
		t.Error("install.sh missing error function with exit 1")
	}
}

// TestInstallScript_SupportsAllPlatforms verifies the script handles all supported platforms.
func TestInstallScript_SupportsAllPlatforms(t *testing.T) {
	scriptPath := filepath.Join("install.sh")
	data, err := os.ReadFile(scriptPath)
	if err != nil {
		t.Fatalf("failed to read install.sh: %v", err)
	}
	content := string(data)

	platforms := []string{"linux", "darwin", "windows"}
	for _, p := range platforms {
		if !strings.Contains(content, p) {
			t.Errorf("install.sh missing platform support for %q", p)
		}
	}

	architectures := []string{"amd64", "arm64"}
	for _, a := range architectures {
		if !strings.Contains(content, a) {
			t.Errorf("install.sh missing architecture support for %q", a)
		}
	}
}
