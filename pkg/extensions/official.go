package extensions

import (
	"github.com/cli/cli/v2/internal/ghrepo"
)

// OfficialExtension describes a GitHub-owned CLI extension that can be
// suggested to users when they invoke an unknown command.
type OfficialExtension struct {
	Name  string
	Owner string
	Repo  string
}

// Repository returns a ghrepo.Interface pinned to github.com so that GHES
// users install from github.com rather than their enterprise host.
func (e *OfficialExtension) Repository() ghrepo.Interface {
	return ghrepo.NewWithHost(e.Owner, e.Repo, "github.com")
}

// OfficialExtensions is the registry of GitHub-owned extensions that gh will
// offer to install when the user invokes the corresponding command name.
var OfficialExtensions = []OfficialExtension{
	{Name: "aw", Owner: "github", Repo: "gh-aw"},
	{Name: "stack", Owner: "github", Repo: "gh-stack"},
}
