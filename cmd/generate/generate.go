package generate

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"

	"github.com/pelletier/go-toml"
	"github.com/peterbourgon/ff/v3/ffcli"

	"github.com/sig-0/chigui-cifras/internal/config"
)

type generateCfg struct {
	outputPath string
}

// NewGenerateCmd creates the generate command
func NewGenerateCmd() *ffcli.Command {
	cfg := &generateCfg{}

	fs := flag.NewFlagSet("generate", flag.ExitOnError)
	cfg.registerFlags(fs)

	return &ffcli.Command{
		Name:       "generate",
		ShortUsage: "generate [flags]",
		LongHelp:   "Generates and outputs the default configuration",
		FlagSet:    fs,
		Exec:       cfg.exec,
	}
}

func (c *generateCfg) registerFlags(fs *flag.FlagSet) {
	fs.StringVar(
		&c.outputPath,
		"output-path",
		"./config.toml",
		"the output path for the TOML configuration file",
	)
}

func (c *generateCfg) exec(_ context.Context, _ []string) error {
	if c.outputPath == "" {
		return errors.New("output path not set")
	}

	cfg := config.DefaultConfig()

	encoded, err := toml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("unable to encode config: %w", err)
	}

	outputFile, err := os.Create(c.outputPath)
	if err != nil {
		return fmt.Errorf("unable to create output file: %w", err)
	}
	defer outputFile.Close()

	if _, err = outputFile.Write(encoded); err != nil {
		return fmt.Errorf("unable to write output file: %w", err)
	}

	return nil
}
