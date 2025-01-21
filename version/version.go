package version

import (
	"fmt"
)

var (
	Commit = ""
	Tag    = ""
)

type Info struct {
	Commit string `json:"commit"`
	Tag    string `json:"tag"`
}

func (i *Info) String() string {
	return fmt.Sprintf("Commit: %s\nTag: %s", i.Commit, i.Tag)
}

func Get() *Info {
	return &Info{
		Commit: Commit,
		Tag:    Tag,
	}
}
