package defaults

import (
	"os"
	"path/filepath"
)

// ArtifactsDir returns path of artifacts dir.
func ArtifactsDir() string {
	return envVarWithDefault("INSOLAR_ARTIFACTS_DIR", ".artifacts")
}

// LaunchnetDir returns path of launchnet's artifacts dir.
func LaunchnetDir() string {
	return envVarWithDefault("LAUNCHNET_BASE_DIR", filepath.Join(ArtifactsDir(), "launchnet"))
}

// LaunchnetDiscoveryNodesLogsDir returns path to dir with launchnet's discovery nodes logs.
func LaunchnetDiscoveryNodesLogsDir() string {
	return filepath.Join(LaunchnetDir(), "logs", "discoverynodes")
}

func envVarWithDefault(name string, defaultValue string) string {
	value := os.Getenv(name)
	if value != "" {
		return value
	}
	return defaultValue
}

// PathWithBaseDir adds base path to path if path is not absolute.
func PathWithBaseDir(path string, base string) string {
	if filepath.IsAbs(path) {
		return path
	}
	return filepath.Join(base, path)
}
