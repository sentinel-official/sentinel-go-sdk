package version

import (
	"fmt"
)

var (
	GitCommit = ""
	Version   = ""
)

type Info struct {
	GitCommit string `json:"git_commit"`
	Version   string `json:"version"`
}

func (i *Info) String() string {
	return fmt.Sprintf("Git Commit: %s\nVersion: %s", i.GitCommit, i.Version)
}

func Get() *Info {
	return &Info{
		GitCommit: GitCommit,
		Version:   Version,
	}
}
