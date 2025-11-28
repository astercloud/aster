package aster

// Version represents the current version of aster
const Version = "v0.19.0"

// VersionInfo provides detailed version information
type VersionInfo struct {
	Version   string
	GoVersion string
	GitCommit string
	BuildTime string
}

// GetVersion returns the current version string
func GetVersion() string {
	return Version
}

// GetVersionInfo returns detailed version information
func GetVersionInfo() VersionInfo {
	return VersionInfo{
		Version:   Version,
		GoVersion: "go1.21+",
		GitCommit: "", // Will be set during build
		BuildTime: "", // Will be set during build
	}
}
