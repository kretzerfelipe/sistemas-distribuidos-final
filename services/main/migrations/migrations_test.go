package migrations

import (
	"regexp"
	"strings"
	"testing"
)

var migrationFileName = regexp.MustCompile(`^(\d+)_[a-z0-9_]+\.(up|down)\.sql$`)

func TestEveryMigrationHasAnUpAndADownFile(t *testing.T) {
	files, err := FS.ReadDir(".")
	if err != nil {
		t.Fatal(err)
	}

	hasUp := map[string]bool{}
	hasDown := map[string]bool{}
	for _, file := range files {
		name := file.Name()
		isSQLFile := strings.HasSuffix(name, ".sql")
		if !isSQLFile {
			continue
		}
		parts := migrationFileName.FindStringSubmatch(name)
		nameFollowsConvention := parts != nil
		if !nameFollowsConvention {
			t.Errorf("%q must be named <version>_<title>.up.sql or <version>_<title>.down.sql", name)
			continue
		}
		migration := strings.TrimSuffix(strings.TrimSuffix(name, ".up.sql"), ".down.sql")
		if parts[2] == "up" {
			hasUp[migration] = true
		} else {
			hasDown[migration] = true
		}
	}

	for migration := range hasUp {
		if !hasDown[migration] {
			t.Errorf("migration %q has an up file but no down file", migration)
		}
	}
	for migration := range hasDown {
		if !hasUp[migration] {
			t.Errorf("migration %q has a down file but no up file", migration)
		}
	}
}
