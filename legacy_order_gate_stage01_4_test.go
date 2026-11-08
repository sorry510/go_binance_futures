package main

import "testing"

// The archived order schema must be registered only for the exact supported
// database migration CLI. This same helper is used by registerModels and
// runCommand; any future subcommand flag change needs an explicit review.
func TestStage014LegacyOrderSchemaCLISelection(t *testing.T) {
	for _, tc := range []struct {
		name string
		args []string
		want bool
	}{
		{"normal_server", nil, false},
		{"ordinary_start", []string{"serve"}, false},
		{"doctor", []string{"doctor"}, false},
		{"cleanup", []string{"cleanup"}, false},
		{"sync_missing_db", []string{"sync"}, false},
		{"sync_wrong_subcommand", []string{"sync", "orders"}, false},
		{"sync_with_unapproved_flags", []string{"sync", "db", "--force"}, false},
		{"sync_db", []string{"sync", "db"}, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got := isSyncDatabaseCommand(tc.args)
			if got != tc.want {
				t.Fatalf("legacy archived schema gate args=%q got=%t want=%t", tc.args, got, tc.want)
			}
		})
	}
}
