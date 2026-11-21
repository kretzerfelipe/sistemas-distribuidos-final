package db

import "testing"

func TestWithPgxSchemeConvertsPostgresURLs(t *testing.T) {
	scenarios := map[string]struct{ input, want string }{
		"postgres scheme":   {"postgres://u:p@localhost:5432/db", "pgx5://u:p@localhost:5432/db"},
		"postgresql scheme": {"postgresql://u:p@localhost:5432/db", "pgx5://u:p@localhost:5432/db"},
		"already pgx5":      {"pgx5://u:p@localhost:5432/db", "pgx5://u:p@localhost:5432/db"},
		"keeps query":       {"postgres://u:p@h:5432/db?sslmode=disable", "pgx5://u:p@h:5432/db?sslmode=disable"},
	}

	for name, scenario := range scenarios {
		t.Run(name, func(t *testing.T) {
			got := withPgxScheme(scenario.input)

			if got != scenario.want {
				t.Errorf("want %q, got %q", scenario.want, got)
			}
		})
	}
}
