package cli

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/alexflint/go-arg"
	"github.com/varavelio/veta/internal/build"
	"github.com/varavelio/veta/internal/scaffold"
	"github.com/varavelio/veta/internal/theme"
)

const programName = "veta"

// Version is the CLI version printed by version commands and flags.
var Version = "dev"

type arguments struct {
	Build       *buildCommand   `arg:"subcommand:build"   help:"Build the site"`
	Init        *initCommand    `arg:"subcommand:init"    help:"Create a starter Veta project"`
	Version     *versionCommand `arg:"subcommand:version" help:"Print version information"`
	VersionFlag bool            `arg:"-v,--version"       help:"print version information and exit"`
}

type buildCommand struct {
	Clean      bool   `arg:"--clean"  help:"clean output before writing"`
	ConfigFile string `arg:"--config" help:"configuration file inside the project root" placeholder:"FILE"`
	Debug      bool   `arg:"--debug"  help:"disable template caching"`
	OutputDir  string `arg:"--out"    help:"output directory"                           placeholder:"DIR"  default:"dist"`
	Root       string `arg:"--root"   help:"project root directory"                     placeholder:"DIR"  default:"."`
}

type initCommand struct {
	Force bool   `arg:"--force"    help:"overwrite starter files that already exist"`
	Path  string `arg:"positional" help:"project directory (default: current directory)" placeholder:"PATH"`
}

type versionCommand struct{}

// Run parses args and executes the requested Veta command.
func Run(ctx context.Context, args []string, stdout, stderr io.Writer) error {
	stdout = defaultWriter(stdout)
	stderr = defaultWriter(stderr)

	parsed := arguments{}
	parser, err := arg.NewParser(arg.Config{Program: programName, Out: stderr}, &parsed)
	if err != nil {
		return writeError(stderr, fmt.Errorf("create command parser: %w", err))
	}

	if err := parser.Parse(normalizeArgs(args)); err != nil {
		return handleParseError(parser, stdout, stderr, err)
	}
	if parsed.VersionFlag || parsed.Version != nil {
		return writeVersion(stdout)
	}
	if parsed.Init != nil {
		return runInit(parsed.Init, stdout, stderr)
	}
	if parsed.Build != nil {
		return runBuild(ctx, parsed.Build, stdout, stderr)
	}

	return runBuild(
		ctx,
		&buildCommand{OutputDir: build.DefaultOutputDir, Root: "."},
		stdout,
		stderr,
	)
}

// handleParseError writes help, version, or a usage error for parser failures.
func handleParseError(parser *arg.Parser, stdout, stderr io.Writer, err error) error {
	if errors.Is(err, arg.ErrHelp) {
		return parser.WriteHelpForSubcommand(stdout, parser.SubcommandNames()...)
	}
	if errors.Is(err, arg.ErrVersion) {
		return writeVersion(stdout)
	}

	wrapped := fmt.Errorf("%w: %w", ErrUsage, err)
	_, writeErr := fmt.Fprintf(stderr, "error: %s\n\n", humanError(wrapped))
	if writeErr != nil {
		return writeErr
	}
	if err := parser.WriteUsageForSubcommand(stderr, parser.SubcommandNames()...); err != nil {
		return err
	}

	return wrapped
}

// runBuild starts a site build from parsed command options.
func runBuild(ctx context.Context, command *buildCommand, stdout, stderr io.Writer) error {
	result, err := build.Run(
		ctx,
		build.WithRoot(command.Root),
		build.WithOutputDir(command.OutputDir),
		build.WithConfigFile(command.ConfigFile),
		build.WithClean(command.Clean),
		build.WithDebug(command.Debug),
		build.WithConsoleOutput(stderr),
	)
	if err != nil {
		return writeError(stderr, err)
	}

	_, err = fmt.Fprintf(stdout, "Built %d page(s) to %s\n", result.Pages, result.OutputDir)
	return err
}

// runInit creates a starter project from parsed command options.
func runInit(command *initCommand, stdout, stderr io.Writer) error {
	result, err := scaffold.Create(scaffold.Config{Force: command.Force, Root: command.Path})
	if err != nil {
		return writeError(stderr, err)
	}

	_, err = fmt.Fprintf(stdout, "Initialized Veta project in %s\n", result.Root)
	return err
}

// writeVersion writes the current CLI version.
func writeVersion(output io.Writer) error {
	_, err := fmt.Fprintf(output, "%s %s\n", programName, Version)
	return err
}

// writeError writes a human-readable CLI error and returns the original error.
func writeError(output io.Writer, err error) error {
	if err == nil {
		return nil
	}
	if _, writeErr := fmt.Fprintf(output, "error: %s\n", humanError(err)); writeErr != nil {
		return writeErr
	}

	return err
}

// humanError converts internal errors into CLI-facing messages.
func humanError(err error) string {
	var integrityError *theme.IntegrityError
	if errors.As(err, &integrityError) {
		return themeIntegrityMessage(integrityError)
	}

	var existingFiles scaffold.ExistingFilesError
	if errors.As(err, &existingFiles) {
		return existingFilesMessage(existingFiles)
	}

	message := err.Error()
	message = strings.TrimPrefix(message, ErrUsage.Error()+": ")
	return message
}

// themeIntegrityMessage returns guidance for remote theme checksum failures.
func themeIntegrityMessage(err *theme.IntegrityError) string {
	if errors.Is(err, theme.ErrIntegrityRequired) {
		return strings.TrimSpace(fmt.Sprintf(`Remote theme integrity is not configured.

Veta downloaded %s, but remote themes must be pinned with theme.sha256 so your builds stay reproducible and theme code cannot change silently.

After you verify that this is the theme source and version you want to trust, add this checksum to veta.yaml:

theme:
  source: %s
  sha256: "%s"`, err.Source, err.Source, err.Actual))
	}

	return strings.TrimSpace(
		fmt.Sprintf(`The configured theme.sha256 does not match the downloaded remote theme.

Theme:   %s
Expected: %s
Actual:   %s

This can mean the theme version or branch changed, the archive was modified, or the checksum belongs to a different theme version. Verify the theme source before updating theme.sha256.`, err.Source, err.Expected, err.Actual),
	)
}

// existingFilesMessage returns guidance for safe init overwrite failures.
func existingFilesMessage(err scaffold.ExistingFilesError) string {
	paths := make([]string, 0, len(err.Paths))
	for _, path := range err.Paths {
		paths = append(paths, "  - "+path)
	}

	return "Cannot initialize the project because these starter files already exist:\n\n" +
		strings.Join(paths, "\n") +
		"\n\nRun `veta init --force` only if you want Veta to overwrite those starter files."
}

// normalizeArgs applies Veta's build-by-default command behavior.
func normalizeArgs(args []string) []string {
	if len(args) == 0 {
		return []string{"build"}
	}
	if buildAlias(args[0]) {
		normalized := make([]string, 0, len(args)+1)
		normalized = append(normalized, "build")
		normalized = append(normalized, args...)
		return normalized
	}

	return args
}

// buildAlias reports whether args use the historical implicit build command.
func buildAlias(firstArg string) bool {
	if !strings.HasPrefix(firstArg, "-") {
		return false
	}

	switch firstArg {
	case "-h", "--help", "-v", "--version":
		return false
	default:
		return true
	}
}

// defaultWriter returns io.Discard when writer is nil.
func defaultWriter(writer io.Writer) io.Writer {
	if writer == nil {
		return io.Discard
	}

	return writer
}
