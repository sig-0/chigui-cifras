package sql

import (
	"context"
	"flag"

	"github.com/peterbourgon/ff/v3/ffcli"
)

// NewSQLCmd creates the sql parent subcommand
func NewSQLCmd() *ffcli.Command {
	fs := flag.NewFlagSet("sql", flag.ExitOnError)

	return &ffcli.Command{
		Name:       "sql",
		ShortUsage: "sql <subcommand>",
		LongHelp:   "SQL database management commands",
		FlagSet:    fs,
		Exec: func(_ context.Context, _ []string) error {
			return flag.ErrHelp
		},
		Subcommands: []*ffcli.Command{
			newMigrateCmd(),
		},
	}
}
