package version

import (
	"fmt"
)

var (
	// Commit holds the commit hash of the current build.
	Commit = ""

	// Tag holds the version tag of the current build.
	Tag = ""
)

// Info represents versioning information.
type Info struct {
	Commit string `json:"commit"` // Commit hash of the build.
	Tag    string `json:"tag"`    // Version tag of the build.
}

// String returns the version information as a formatted string.
func (i *Info) String() string {
	return fmt.Sprintf("Commit: %s\nTag: %s", i.Commit, i.Tag)
}

// Get returns the current version information.
func Get() *Info {
	return &Info{
		Commit: Commit,
		Tag:    Tag,
	}
}
