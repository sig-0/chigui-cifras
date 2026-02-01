package main

import (
	"context"
	"flag"
	"fmt"
	"os"

	"github.com/peterbourgon/ff/v3/ffcli"

	"github.com/sig-0/chigui-cifras/cmd/generate"
	"github.com/sig-0/chigui-cifras/cmd/serve"
)

func main() {
	fs := flag.NewFlagSet("chigui", flag.ExitOnError)

	cmd := &ffcli.Command{
		ShortUsage: "chigui <subcommand> [flags]",
		LongHelp:   "ChiguiCifras Telegram bot for exchange rate queries and notifications",
		FlagSet:    fs,
		Exec: func(_ context.Context, _ []string) error {
			return flag.ErrHelp
		},
	}

	cmd.Subcommands = []*ffcli.Command{
		serve.NewServeCmd(),
		generate.NewGenerateCmd(),
	}

	if err := cmd.ParseAndRun(context.Background(), os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
