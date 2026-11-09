// Package cliargs contains side-effect-free CLI classification helpers.
// It must not import the application bootstrap, config, database or network.
package cliargs

// IsSyncDatabaseCommand returns true only for the exact explicit database
// schema-upgrade command. No flags, aliases or partial forms are accepted.
func IsSyncDatabaseCommand(args []string) bool {
	return len(args) == 2 && args[0] == "sync" && args[1] == "db"
}
