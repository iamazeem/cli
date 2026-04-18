package discovery

import (
	"fmt"
	"sort"
	"strings"
)

// NameCollision represents a group of skills that share the same InstallName
// and would overwrite each other when installed to the same directory.
type NameCollision struct {
	Name         string   // the conflicting install name (may include namespace prefix)
	DisplayNames []string // display names of each conflicting skill
}

// FindNameCollisions detects skills that share the same InstallName and returns a
// sorted slice of collisions. Callers decide how to present the conflict to
// the user (different flows need different error messages).
func FindNameCollisions(skills []Skill) []NameCollision {
	byName := make(map[string][]Skill)
	for _, s := range skills {
		byName[s.InstallName()] = append(byName[s.InstallName()], s)
	}

	var collisions []NameCollision
	for name, group := range byName {
		if len(group) <= 1 {
			continue
		}
		names := make([]string, len(group))
		for i, s := range group {
			names[i] = s.DisplayName()
		}
		collisions = append(collisions, NameCollision{Name: name, DisplayNames: names})
	}

	sort.Slice(collisions, func(i, j int) bool {
		return collisions[i].Name < collisions[j].Name
	})
	return collisions
}

// FormatCollisions builds a human-readable string listing each collision,
// suitable for embedding in an error message. Each collision is formatted as
// "name: display1, display2" and collisions are separated by newlines with
// leading indentation.
func FormatCollisions(collisions []NameCollision) string {
	lines := make([]string, len(collisions))
	for i, c := range collisions {
		lines[i] = fmt.Sprintf("%s: %s", c.Name, strings.Join(c.DisplayNames, ", "))
	}
	return strings.Join(lines, "\n  ")
}
