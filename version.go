package fbgo

import "runtime"

var (
	Version = "dev"
	Commit  = "unknown"
)

type VersionInfo struct {
	Version   string `json:"version"`
	Commit    string `json:"commit"`
	GoVersion string `json:"goVersion"`
}

func BuildVersion() VersionInfo {
	return VersionInfo{Version: Version, Commit: Commit, GoVersion: runtime.Version()}
}
