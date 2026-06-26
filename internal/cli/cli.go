package cli

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"strings"

	"github.com/varavelio/veta/internal/build"
)

// Run parses args and executes the requested Veta command.
func Run(ctx context.Context, args []string, stdout, stderr io.Writer) error {
	stdout = defaultWriter(stdout)
	stderr = defaultWriter(stderr)

	if len(args) == 0 {
		return runBuild(ctx, nil, stdout, stderr)
	}

	switch args[0] {
	case "build":
		return runBuild(ctx, args[1:], stdout, stderr)
	case "help", "--help", "-h":
		return writeUsage(stdout)
	default:
		if strings.HasPrefix(args[0], "-") {
			return runBuild(ctx, args, stdout, stderr)
		}

		return fmt.Errorf("%w: %s", ErrUnknownCommand, args[0])
	}
}

// runBuild parses build flags and starts a site build.
func runBuild(ctx context.Context, args []string, stdout, stderr io.Writer) error {
	flags := flag.NewFlagSet("veta build", flag.ContinueOnError)
	flags.SetOutput(stderr)

	root := flags.String("root", ".", "project root directory")
	outputDir := flags.String("out", build.DefaultOutputDir, "output directory")
	configFile := flags.String("config", "", "configuration file inside the project root")
	clean := flags.Bool("clean", false, "clean output before writing")
	debug := flags.Bool("debug", false, "disable template caching")
	flags.Usage = func() {
		_ = writeBuildUsage(stderr)
	}

	if err := flags.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}

		return err
	}
	if flags.NArg() > 0 {
		return fmt.Errorf("%w: %s", ErrUnknownCommand, strings.Join(flags.Args(), " "))
	}

	result, err := build.Run(
		ctx,
		build.WithRoot(*root),
		build.WithOutputDir(*outputDir),
		build.WithConfigFile(*configFile),
		build.WithClean(*clean),
		build.WithDebug(*debug),
		build.WithConsoleOutput(stderr),
	)
	if err != nil {
		return err
	}

	_, err = fmt.Fprintf(stdout, "Built %d page(s) to %s\n", result.Pages, result.OutputDir)
	return err
}

// writeUsage writes top-level command help.
func writeUsage(output io.Writer) error {
	_, err := fmt.Fprint(output, strings.TrimLeft(`
Usage:
  veta [build] [flags]
  veta help

Commands:
  build    Build the site
  help     Show this help

Use "veta build --help" for build flags.
`, "\n"))
	return err
}

// writeBuildUsage writes build command help.
func writeBuildUsage(output io.Writer) error {
	_, err := fmt.Fprint(output, strings.TrimLeft(`
Usage:
  veta build [flags]

Flags:
  --root string     project root directory (default ".")
  --out string      output directory (default "dist")
  --config string   configuration file inside the project root
  --clean           clean output before writing
  --debug           disable template caching
`, "\n"))
	return err
}

// defaultWriter returns io.Discard when writer is nil.
func defaultWriter(writer io.Writer) io.Writer {
	if writer == nil {
		return io.Discard
	}

	return writer
}
