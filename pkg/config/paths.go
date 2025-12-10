// Package config provides cross-platform configuration path management.
// It follows platform conventions:
//   - macOS: ~/Library/Application Support/Aster
//   - Linux: ~/.config/aster (XDG_CONFIG_HOME)
//   - Windows: %APPDATA%/Aster
package config

import (
	"os"
	"path/filepath"
	"runtime"
	"sync"
)

var (
	// pathsOnce ensures paths are computed once
	pathsOnce sync.Once
	// cached paths
	configDir string
	dataDir   string
	logDir    string
	cacheDir  string
)

// initPaths initializes all paths based on the current platform
func initPaths() {
	pathsOnce.Do(func() {
		switch runtime.GOOS {
		case "darwin":
			initDarwinPaths()
		case "windows":
			initWindowsPaths()
		default:
			initLinuxPaths()
		}
	})
}

func initDarwinPaths() {
	home := os.Getenv("HOME")
	appSupport := filepath.Join(home, "Library", "Application Support", "Aster")
	configDir = appSupport
	dataDir = appSupport
	logDir = filepath.Join(home, "Library", "Logs", "Aster")
	cacheDir = filepath.Join(home, "Library", "Caches", "Aster")
}

func initWindowsPaths() {
	appData := os.Getenv("APPDATA")
	localAppData := os.Getenv("LOCALAPPDATA")
	if localAppData == "" {
		localAppData = appData
	}

	configDir = filepath.Join(appData, "Aster")
	dataDir = filepath.Join(appData, "Aster", "data")
	logDir = filepath.Join(localAppData, "Aster", "logs")
	cacheDir = filepath.Join(localAppData, "Aster", "cache")
}

func initLinuxPaths() {
	home := os.Getenv("HOME")

	// XDG Base Directory Specification
	xdgConfig := os.Getenv("XDG_CONFIG_HOME")
	if xdgConfig == "" {
		xdgConfig = filepath.Join(home, ".config")
	}

	xdgData := os.Getenv("XDG_DATA_HOME")
	if xdgData == "" {
		xdgData = filepath.Join(home, ".local", "share")
	}

	xdgCache := os.Getenv("XDG_CACHE_HOME")
	if xdgCache == "" {
		xdgCache = filepath.Join(home, ".cache")
	}

	xdgState := os.Getenv("XDG_STATE_HOME")
	if xdgState == "" {
		xdgState = filepath.Join(home, ".local", "state")
	}

	configDir = filepath.Join(xdgConfig, "aster")
	dataDir = filepath.Join(xdgData, "aster")
	logDir = filepath.Join(xdgState, "aster", "logs")
	cacheDir = filepath.Join(xdgCache, "aster")
}

// ConfigDir returns the configuration directory.
// This is where config.yaml, recipes, and extensions are stored.
func ConfigDir() string {
	initPaths()
	return configDir
}

// DataDir returns the data directory.
// This is where sessions, memories, and persistent data are stored.
func DataDir() string {
	initPaths()
	return dataDir
}

// LogDir returns the log directory.
func LogDir() string {
	initPaths()
	return logDir
}

// CacheDir returns the cache directory.
// This is for temporary data that can be safely deleted.
func CacheDir() string {
	initPaths()
	return cacheDir
}

// ConfigFile returns the path to the main configuration file.
func ConfigFile() string {
	return filepath.Join(ConfigDir(), "config.yaml")
}

// SessionsDir returns the path to the sessions directory.
func SessionsDir() string {
	return filepath.Join(DataDir(), "sessions")
}

// RecipesDir returns the path to the recipes directory.
func RecipesDir() string {
	return filepath.Join(ConfigDir(), "recipes")
}

// ExtensionsDir returns the path to the extensions directory.
func ExtensionsDir() string {
	return filepath.Join(ConfigDir(), "extensions")
}

// MemoriesDir returns the path to the memories directory.
func MemoriesDir() string {
	return filepath.Join(DataDir(), "memories")
}

// DatabaseFile returns the path to the SQLite database file.
func DatabaseFile() string {
	return filepath.Join(DataDir(), "aster.db")
}

// EnsureDir creates a directory if it doesn't exist.
func EnsureDir(path string) error {
	return os.MkdirAll(path, 0755)
}

// EnsureAllDirs creates all necessary directories.
func EnsureAllDirs() error {
	dirs := []string{
		ConfigDir(),
		DataDir(),
		LogDir(),
		CacheDir(),
		SessionsDir(),
		RecipesDir(),
		ExtensionsDir(),
		MemoriesDir(),
	}

	for _, dir := range dirs {
		if err := EnsureDir(dir); err != nil {
			return err
		}
	}

	return nil
}

// ProjectConfigFile returns the path to project-level config file.
// It searches from the current directory upward for .aster/config.yaml
func ProjectConfigFile(startDir string) string {
	dir := startDir
	for {
		configPath := filepath.Join(dir, ".aster", "config.yaml")
		if _, err := os.Stat(configPath); err == nil {
			return configPath
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			// Reached root
			return ""
		}
		dir = parent
	}
}

// ResolveConfigFile returns the effective config file path.
// Priority: project config > global config
func ResolveConfigFile(workDir string) string {
	if projectConfig := ProjectConfigFile(workDir); projectConfig != "" {
		return projectConfig
	}
	return ConfigFile()
}
