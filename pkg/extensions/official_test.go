package extensions

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestOfficialExtension_Repository(t *testing.T) {
	ext := &OfficialExtension{Name: "stack", Owner: "github", Repo: "gh-stack"}
	repo := ext.Repository()
	assert.Equal(t, "github", repo.RepoOwner())
	assert.Equal(t, "gh-stack", repo.RepoName())
	assert.Equal(t, "github.com", repo.RepoHost())
}
