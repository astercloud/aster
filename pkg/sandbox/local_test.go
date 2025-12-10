package sandbox

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/astercloud/aster/pkg/types"
)

func TestLocalSandbox_Basic(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "sandbox-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	sb, err := NewLocalSandbox(&LocalSandboxConfig{
		WorkDir: tmpDir,
	})
	if err != nil {
		t.Fatalf("failed to create sandbox: %v", err)
	}
	defer sb.Dispose()

	if sb.Kind() != "local" {
		t.Errorf("expected kind='local', got '%s'", sb.Kind())
	}

	if sb.WorkDir() != tmpDir {
		t.Errorf("expected workDir='%s', got '%s'", tmpDir, sb.WorkDir())
	}
}

func TestLocalSandbox_Exec(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "sandbox-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	sb, err := NewLocalSandbox(&LocalSandboxConfig{
		WorkDir: tmpDir,
	})
	if err != nil {
		t.Fatalf("failed to create sandbox: %v", err)
	}
	defer sb.Dispose()

	ctx := context.Background()

	// Test simple command
	result, err := sb.Exec(ctx, "echo hello", nil)
	if err != nil {
		t.Fatalf("exec failed: %v", err)
	}
	if result.Code != 0 {
		t.Errorf("expected exit code 0, got %d", result.Code)
	}
	if result.Stdout != "hello\n" {
		t.Errorf("expected stdout='hello\\n', got '%s'", result.Stdout)
	}
}

func TestLocalSandbox_DangerousCommand(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "sandbox-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	sb, err := NewLocalSandbox(&LocalSandboxConfig{
		WorkDir: tmpDir,
	})
	if err != nil {
		t.Fatalf("failed to create sandbox: %v", err)
	}
	defer sb.Dispose()

	ctx := context.Background()

	// Test dangerous command is blocked
	result, err := sb.Exec(ctx, "rm -rf /", nil)
	if err != nil {
		t.Fatalf("exec failed: %v", err)
	}
	if result.Code != 1 {
		t.Errorf("expected exit code 1 for dangerous command, got %d", result.Code)
	}
	if result.Stderr == "" {
		t.Error("expected error message for dangerous command")
	}
}

func TestLocalSandbox_ExcludedCommands(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "sandbox-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	sb, err := NewLocalSandbox(&LocalSandboxConfig{
		WorkDir: tmpDir,
		Settings: &types.SandboxSettings{
			Enabled:          true,
			ExcludedCommands: []string{"git", "docker"},
		},
	})
	if err != nil {
		t.Fatalf("failed to create sandbox: %v", err)
	}
	defer sb.Dispose()

	// Test excluded command detection
	if !sb.isExcludedCommand("git status") {
		t.Error("expected 'git status' to be excluded")
	}
	if !sb.isExcludedCommand("docker ps") {
		t.Error("expected 'docker ps' to be excluded")
	}
	if sb.isExcludedCommand("ls -la") {
		t.Error("expected 'ls -la' to NOT be excluded")
	}
}

func TestLocalSandbox_NetworkAccess(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "sandbox-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	sb, err := NewLocalSandbox(&LocalSandboxConfig{
		WorkDir: tmpDir,
		Settings: &types.SandboxSettings{
			Enabled: true,
			Network: &types.NetworkSandboxSettings{
				AllowLocalBinding: false,
				AllowedHosts:      []string{"api.example.com"},
				BlockedHosts:      []string{"malicious.com"},
			},
		},
	})
	if err != nil {
		t.Fatalf("failed to create sandbox: %v", err)
	}
	defer sb.Dispose()

	// Test network access checks
	if sb.CheckNetworkAccess("localhost", 8080) {
		t.Error("expected localhost binding to be blocked")
	}
	if !sb.CheckNetworkAccess("api.example.com", 443) {
		t.Error("expected api.example.com to be allowed")
	}
	if sb.CheckNetworkAccess("malicious.com", 80) {
		t.Error("expected malicious.com to be blocked")
	}
	if sb.CheckNetworkAccess("other.com", 80) {
		t.Error("expected other.com to be blocked (not in allowed list)")
	}
}

func TestLocalSandbox_UnixSocketAccess(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "sandbox-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	sb, err := NewLocalSandbox(&LocalSandboxConfig{
		WorkDir: tmpDir,
		Settings: &types.SandboxSettings{
			Enabled: true,
			Network: &types.NetworkSandboxSettings{
				AllowUnixSockets: []string{"/var/run/docker.sock"},
			},
		},
	})
	if err != nil {
		t.Fatalf("failed to create sandbox: %v", err)
	}
	defer sb.Dispose()

	if !sb.CheckUnixSocketAccess("/var/run/docker.sock") {
		t.Error("expected docker socket to be allowed")
	}
	if sb.CheckUnixSocketAccess("/var/run/other.sock") {
		t.Error("expected other socket to be blocked")
	}
}

func TestLocalSandbox_IgnoreViolations(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "sandbox-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	sb, err := NewLocalSandbox(&LocalSandboxConfig{
		WorkDir: tmpDir,
		Settings: &types.SandboxSettings{
			Enabled: true,
			IgnoreViolations: &types.SandboxIgnoreViolations{
				FilePatterns:    []string{"/tmp/*", "*.log"},
				NetworkPatterns: []string{"localhost:*"},
			},
		},
	})
	if err != nil {
		t.Fatalf("failed to create sandbox: %v", err)
	}
	defer sb.Dispose()

	// Test file violation ignoring
	if !sb.ShouldIgnoreViolation("file", "/tmp/test.txt") {
		t.Error("expected /tmp/test.txt to be ignored")
	}
	if !sb.ShouldIgnoreViolation("file", "app.log") {
		t.Error("expected app.log to be ignored")
	}
	if sb.ShouldIgnoreViolation("file", "/etc/passwd") {
		t.Error("expected /etc/passwd to NOT be ignored")
	}
}

func TestLocalSandbox_Timeout(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "sandbox-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	sb, err := NewLocalSandbox(&LocalSandboxConfig{
		WorkDir: tmpDir,
	})
	if err != nil {
		t.Fatalf("failed to create sandbox: %v", err)
	}
	defer sb.Dispose()

	ctx := context.Background()

	// Test command with short timeout
	result, err := sb.Exec(ctx, "sleep 5", &ExecOptions{
		Timeout: 100 * time.Millisecond,
	})
	if err != nil {
		t.Fatalf("exec failed: %v", err)
	}
	// Command should be killed due to timeout
	if result.Code == 0 {
		t.Error("expected non-zero exit code due to timeout")
	}
}

func TestLocalSandbox_FS(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "sandbox-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	sb, err := NewLocalSandbox(&LocalSandboxConfig{
		WorkDir:         tmpDir,
		EnforceBoundary: true,
	})
	if err != nil {
		t.Fatalf("failed to create sandbox: %v", err)
	}
	defer sb.Dispose()

	fs := sb.FS()

	// Test Resolve
	resolved := fs.Resolve("test.txt")
	expected := filepath.Join(tmpDir, "test.txt")
	if resolved != expected {
		t.Errorf("expected resolved='%s', got '%s'", expected, resolved)
	}

	// Test IsInside
	if !fs.IsInside(filepath.Join(tmpDir, "subdir", "file.txt")) {
		t.Error("expected path inside sandbox to return true")
	}
	if fs.IsInside("/etc/passwd") {
		t.Error("expected path outside sandbox to return false")
	}

	// Test Write and Read
	ctx := context.Background()
	testContent := "hello world"
	if err := fs.Write(ctx, "test.txt", testContent); err != nil {
		t.Fatalf("write failed: %v", err)
	}

	content, err := fs.Read(ctx, "test.txt")
	if err != nil {
		t.Fatalf("read failed: %v", err)
	}
	if content != testContent {
		t.Errorf("expected content='%s', got '%s'", testContent, content)
	}
}

func TestLocalSandbox_GetSettings(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "sandbox-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	settings := &types.SandboxSettings{
		Enabled:                  true,
		AutoAllowBashIfSandboxed: true,
		ExcludedCommands:         []string{"git"},
	}

	sb, err := NewLocalSandbox(&LocalSandboxConfig{
		WorkDir:  tmpDir,
		Settings: settings,
	})
	if err != nil {
		t.Fatalf("failed to create sandbox: %v", err)
	}
	defer sb.Dispose()

	if !sb.IsEnabled() {
		t.Error("expected sandbox to be enabled")
	}

	got := sb.GetSettings()
	if got == nil {
		t.Fatal("expected settings to be non-nil")
	}
	if !got.AutoAllowBashIfSandboxed {
		t.Error("expected AutoAllowBashIfSandboxed=true")
	}
}
