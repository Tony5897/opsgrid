// Package buildinfo exposes the version and commit baked in at build time:
//
//	go build -ldflags "-X github.com/Tony5897/opsgrid/internal/platform/buildinfo.Version=v1.2.3 \
//	                   -X github.com/Tony5897/opsgrid/internal/platform/buildinfo.Commit=$(git rev-parse HEAD)"
//
// Without ldflags it falls back to the VCS stamp Go embeds automatically.
package buildinfo

import "runtime/debug"

var (
	Version = "dev"
	Commit  = ""
)

func init() {
	if Commit != "" {
		return
	}
	info, ok := debug.ReadBuildInfo()
	if !ok {
		return
	}
	var dirty bool
	for _, s := range info.Settings {
		switch s.Key {
		case "vcs.revision":
			Commit = s.Value
		case "vcs.modified":
			dirty = s.Value == "true"
		}
	}
	if dirty && Commit != "" {
		Commit += "-dirty"
	}
}

// Short returns the abbreviated commit.
func Short() string {
	if len(Commit) > 12 {
		return Commit[:12]
	}
	return Commit
}
