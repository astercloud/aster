package config

import (
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"testing"
)

func TestConfigDir(t *testing.T) {
	// Reset paths for testing
	pathsOnce = sync.Once{}

	dir := ConfigDir()
	if dir == "" {
		t.Error("ConfigDir should not be empty")
	}

	switch runtime.GOOS {
	case "darwin":
		if !contains(dir, "Library/Application Support/Aster") {
			t.Errorf("macOS ConfigDir should contain 'Library/Application Support/Aster', got %s", dir)
		}
	case "windows":
		if !contains(dir, "Aster") {
			t.Errorf("Windows ConfigDir should contain 'Aster', got %s", dir)
		}
	default:
		if !contains(dir, ".config/aster") && !contains(dir, "aster") {
			t.Errorf("Linux ConfigDir should contain '.config/aster', got %s", dir)
		}
	}
}

func TestDataDir(t *testing.T) {
	pathsOnce = sync.Once{}

	dir := DataDir()
	if dir == "" {
		t.Error("DataDir should not be empty")
	}
}

func TestLogDir(t *testing.T) {
	pathsOnce = sync.Once{}

	dir := LogDir()
	if dir == "" {
		t.Error("LogDir should not be empty")
	}
}

func TestCacheDir(t *testing.T) {
	pathsOnce = sync.Once{}

	dir := CacheDir()
	if dir == "" {
		t.Error("CacheDir should not be empty")
	}
}

func TestDatabaseFile(t *testing.T) {
	pathsOnce = sync.Once{}

	file := DatabaseFile()
	if filepath.Base(file) != "aster.db" {
		t.Errorf("DatabaseFile should be named 'aster.db', got %s", filepath.Base(file))
	}
}

func TestEnsureDir(t *testing.T) {
	tempDir := t.TempDir()
	testDir := filepath.Join(tempDir, "test", "nested", "dir")

	if err := EnsureDir(testDir); err != nil {
		t.Fatalf("EnsureDir failed: %v", err)
	}

	if _, err := os.Stat(testDir); os.IsNotExist(err) {
		t.Error("Directory was not created")
	}
}

func TestProjectConfigFile(t *testing.T) {
	tempDir := t.TempDir()

	// Create a project structure
	projectDir := filepath.Join(tempDir, "myproject")
	asterDir := filepath.Join(projectDir, ".aster")
	subDir := filepath.Join(projectDir, "src", "components")

	if err := os.MkdirAll(asterDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(subDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Create config file
	configFile := filepath.Join(asterDir, "config.yaml")
	if err := os.WriteFile(configFile, []byte("test: true"), 0644); err != nil {
		t.Fatal(err)
	}

	// Test from subdirectory
	found := ProjectConfigFile(subDir)
	if found != configFile {
		t.Errorf("Expected %s, got %s", configFile, found)
	}

	// Test from project root
	found = ProjectConfigFile(projectDir)
	if found != configFile {
		t.Errorf("Expected %s, got %s", configFile, found)
	}

	// Test from directory without config
	found = ProjectConfigFile(tempDir)
	if found != "" {
		t.Errorf("Expected empty string, got %s", found)
	}
}

func TestResolveConfigFile(t *testing.T) {
	pathsOnce = sync.Once{}
	tempDir := t.TempDir()

	// Without project config, should return global
	resolved := ResolveConfigFile(tempDir)
	if resolved != ConfigFile() {
		t.Errorf("Expected global config, got %s", resolved)
	}

	// With project config
	projectDir := filepath.Join(tempDir, "myproject")
	asterDir := filepath.Join(projectDir, ".aster")
	if err := os.MkdirAll(asterDir, 0755); err != nil {
		t.Fatal(err)
	}

	configFile := filepath.Join(asterDir, "config.yaml")
	if err := os.WriteFile(configFile, []byte("test: true"), 0644); err != nil {
		t.Fatal(err)
	}

	resolved = ResolveConfigFile(projectDir)
	if resolved != configFile {
		t.Errorf("Expected project config %s, got %s", configFile, resolved)
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsImpl(s, substr))
}

func containsImpl(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
