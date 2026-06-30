package cli

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/alexflint/go-arg"
	"github.com/varavelio/tinta"
	"github.com/varavelio/veta/internal/build"
	"github.com/varavelio/veta/internal/components"
	"github.com/varavelio/veta/internal/config"
	"github.com/varavelio/veta/internal/data"
	"github.com/varavelio/veta/internal/dev"
	"github.com/varavelio/veta/internal/output"
	"github.com/varavelio/veta/internal/pages"
	"github.com/varavelio/veta/internal/scaffold"
	"github.com/varavelio/veta/internal/tailwindcss"
	"github.com/varavelio/veta/internal/template"
	"github.com/varavelio/veta/internal/theme"
	"github.com/varavelio/veta/internal/version"
)

const programName = "veta"

type arguments struct {
	Build       *buildCommand   `arg:"subcommand:build"   help:"build the site"`
	Dev         *devCommand     `arg:"subcommand:dev"     help:"start the local development server"`
	Init        *initCommand    `arg:"subcommand:init"    help:"create a starter Veta project"`
	VersionCmd  *versionCommand `arg:"subcommand:version" help:"print version information"`
	VersionFlag bool            `arg:"-v,--"              help:"display version and exit"`
}

type buildCommand struct {
	ConfigFile string `arg:"-c,--config" help:"configuration file to use" placeholder:"FILE"`
}

type devCommand struct {
	ConfigFile string `arg:"-c,--config" help:"configuration file to use"         placeholder:"FILE"`
	Host       string `arg:"--host"      help:"host to bind (default: 127.0.0.1)" placeholder:"HOST"`
	Port       int    `arg:"--port"      help:"port to bind (default: 3000)"      placeholder:"PORT"`
}

type initCommand struct {
	Force bool   `arg:"--force"    help:"overwrite starter files that already exist"`
	Path  string `arg:"positional" help:"project directory (default: current directory)" placeholder:"PATH"`
}

type versionCommand struct{}

// Description returns the top-level help description.
func (arguments) Description() string {
	return "Veta static site generator"
}

// Version returns the metadata block used by top-level help.
func (arguments) Version() string {
	return helpMetadataBlock()
}

// Epilogue returns concise examples for top-level help.
func (arguments) Epilogue() string {
	return strings.TrimSpace(`Examples:
  veta init my-site
  veta dev
  veta build
  veta build --config ./veta.yaml`)
}

// Run parses args and executes the requested Veta command.
func Run(ctx context.Context, args []string, stdout, stderr io.Writer) error {
	stdout = defaultWriter(stdout)
	stderr = defaultWriter(stderr)

	parsed := arguments{}
	parser, err := arg.NewParser(arg.Config{Program: programName, Out: stderr}, &parsed)
	if err != nil {
		return writeError(stderr, fmt.Errorf("create command parser: %w", err))
	}

	if len(args) == 0 {
		parser.WriteHelp(stdout)
		return nil
	}
	if err := parser.Parse(args); err != nil {
		return handleParseError(parser, stdout, stderr, err)
	}
	if parsed.VersionFlag || parsed.VersionCmd != nil {
		return writeVersion(stdout)
	}
	if parsed.Init != nil {
		return runInit(parsed.Init, stdout, stderr)
	}
	if parsed.Build != nil {
		return runBuild(ctx, parsed.Build, stdout, stderr)
	}
	if parsed.Dev != nil {
		return runDev(ctx, parsed.Dev, stdout, stderr)
	}

	parser.WriteHelp(stdout)
	return nil
}

// runDev starts the local development server from parsed command options.
func runDev(ctx context.Context, command *devCommand, stdout, stderr io.Writer) error {
	port := command.Port
	if port == 0 {
		port = dev.DefaultPort
	}

	if err := dev.Run(ctx, dev.Config{
		ConfigFile: command.ConfigFile,
		Host:       command.Host,
		Port:       port,
		Stderr:     stderr,
		Stdout:     stdout,
	}); err != nil {
		return writeError(stderr, err)
	}

	return nil
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
	_, writeErr := fmt.Fprintf(stderr, "%s: %s\n\n", errorLabel(), humanError(wrapped))
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
	startedAt := time.Now()
	result, err := build.Run(
		ctx,
		build.WithConfigFile(command.ConfigFile),
		build.WithConsoleOutput(stderr),
	)
	duration := time.Since(startedAt)
	if err != nil {
		return writeError(stderr, err)
	}

	_, err = fmt.Fprintln(stdout, buildSuccessMessage(result, duration))
	return err
}

// buildSuccessMessage returns the styled message printed after a successful build.
func buildSuccessMessage(result build.Result, duration time.Duration) string {
	return tinta.Text().Green().Bold().Sprintf(
		"Veta built %d %s to %s in %s",
		result.Pages,
		pageLabel(result.Pages),
		result.Config.Build.Output,
		buildDuration(duration),
	)
}

// pageLabel returns the correctly pluralized page noun.
func pageLabel(pages int) string {
	if pages == 1 {
		return "page"
	}

	return "pages"
}

// buildDuration returns a readable ASCII duration for CLI output.
func buildDuration(duration time.Duration) string {
	if duration <= 0 {
		return "0s"
	}

	rounded := duration.Round(time.Millisecond)
	if rounded == 0 {
		return "1ms"
	}

	return rounded.String()
}

// runInit creates a starter project from parsed command options.
func runInit(command *initCommand, stdout, stderr io.Writer) error {
	result, err := scaffold.Create(scaffold.Config{Force: command.Force, Root: command.Path})
	if err != nil {
		return writeError(stderr, err)
	}

	_, err = fmt.Fprintf(stdout, strings.TrimLeft(`Initialized Veta project in %s

Next steps:
  cd %s
  veta dev
  veta build

Build settings live in veta.yaml.
`, "\n"), result.Root, result.Root)
	return err
}

// writeVersion writes the current CLI version.
func writeVersion(output io.Writer) error {
	_, err := fmt.Fprintln(output, version.Detailed())
	return err
}

// writeError writes a human-readable CLI error and returns the original error.
func writeError(output io.Writer, err error) error {
	if err == nil {
		return nil
	}
	if _, writeErr := fmt.Fprintf(
		output,
		"%s: %s\n",
		errorLabel(),
		humanError(err),
	); writeErr != nil {
		return writeErr
	}

	return err
}

// humanError converts internal errors into CLI-facing messages.
func humanError(err error) string {
	if errors.Is(err, context.Canceled) {
		return "Operation canceled."
	}
	if errors.Is(err, build.ErrConfigNotFound) {
		return strings.TrimSpace(`Could not find a Veta config file.

Veta looks for veta.yaml, veta.yml, .veta.yaml, or .veta.yml in the current directory and then walks up through its ancestors.

Run ` + "`veta init`" + ` to create a project, run ` + "`veta build`" + ` or ` + "`veta dev`" + ` from inside an existing project, or pass an explicit config file with ` + "`veta build --config ./veta.yaml`" + `.`)
	}
	if errors.Is(err, build.ErrConfigFileInvalid) {
		return "The config file passed with --config could not be used. Check that the file exists and is not a directory.\n\nDetails: " + err.Error()
	}
	if errors.Is(err, build.ErrOutputDirInvalid) {
		return "The build output directory is invalid. Check the output path and run the command again.\n\nDetails: " + err.Error()
	}
	if errors.Is(err, dev.ErrAddressInvalid) || errors.Is(err, dev.ErrListenFailed) {
		return "The dev server could not start. Check --host and --port, then run `veta dev` again.\n\nDetails: " + err.Error()
	}
	if errors.Is(err, build.ErrRootInvalid) {
		return "The config search directory is invalid. Run Veta from an existing directory or pass an explicit config file with `veta build --config ./veta.yaml`.\n\nDetails: " + err.Error()
	}
	if errors.Is(err, config.ErrInvalid) || errors.Is(err, config.ErrPathInvalid) {
		return "The Veta configuration is invalid. Fix veta.yaml and run the command again.\n\nDetails: " + err.Error()
	}
	if errors.Is(err, pages.ErrPageInvalid) || errors.Is(err, pages.ErrGeneratorInvalid) ||
		errors.Is(err, pages.ErrOutputPathDuplicate) || errors.Is(err, pages.ErrPermalinkInvalid) ||
		errors.Is(err, pages.ErrNestedUnsupported) || errors.Is(err, pages.ErrFormatUnsupported) {
		return "Page generation failed. Check the files in pages/ and the page objects they return.\n\nDetails: " + err.Error()
	}
	if errors.Is(err, template.ErrTemplateNotFound) ||
		errors.Is(err, template.ErrTemplateNameInvalid) ||
		errors.Is(err, template.ErrTemplateAmbiguous) {
		return "Template rendering failed. Check that the page template points to an existing file in templates/.\n\nDetails: " + err.Error()
	}
	if errors.Is(err, tailwindcss.ErrConfigInvalid) || errors.Is(err, tailwindcss.ErrRunFailed) ||
		errors.Is(
			err,
			tailwindcss.ErrPlatformUnsupported,
		) || errors.Is(err, tailwindcss.ErrBinaryUnavailable) {
		return "Tailwind CSS failed. Check tailwindcss.stylesheet in veta.yaml.\n\nDetails: " + err.Error()
	}
	if errors.Is(err, output.ErrDirInvalid) || errors.Is(err, output.ErrPathInvalid) ||
		errors.Is(err, output.ErrPathDuplicate) || errors.Is(err, output.ErrMinifyFailed) {
		return "Writing output failed. Check build.output in veta.yaml and any generated output paths.\n\nDetails: " + err.Error()
	}

	if errors.Is(err, theme.ErrSourceInvalid) || errors.Is(err, theme.ErrDownloadFailed) ||
		errors.Is(err, theme.ErrRemoteUnsupported) {
		return "Theme resolution failed. Check theme.source in veta.yaml.\n\nDetails: " + err.Error()
	}
	if errors.Is(err, data.ErrInvalid) || errors.Is(err, data.ErrKeyDuplicate) ||
		errors.Is(err, data.ErrKeyInvalid) || errors.Is(err, data.ErrFormatUnsupported) ||
		errors.Is(err, data.ErrValueUnsupported) {
		return "Data loading failed. Check the files in data/ and make sure they produce JSON-compatible values.\n\nDetails: " + err.Error()
	}
	if errors.Is(err, components.ErrComponentNameInvalid) ||
		errors.Is(err, components.ErrFormatUnsupported) ||
		errors.Is(err, components.ErrSyntax) ||
		errors.Is(err, components.ErrAttributeInvalid) {
		return "Component processing failed. Check component filenames and component tags in your content.\n\nDetails: " + err.Error()
	}

	var existingFiles scaffold.ExistingFilesError
	if errors.As(err, &existingFiles) {
		return existingFilesMessage(existingFiles)
	}

	message := err.Error()
	message = strings.TrimPrefix(message, ErrUsage.Error()+": ")
	return message
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

// helpMetadataBlock returns the top-level help metadata shown after the description.
func helpMetadataBlock() string {
	lines := []string{
		metadataLine("Version", version.Number()),
		metadataLine("Commit", version.CommitHash()),
	}
	lines = append(lines, metadataLine("Repository", version.Repository))

	return "\n" + strings.Join(lines, "\n") + "\n"
}

// metadataLine returns one aligned top-level help metadata line.
func metadataLine(label, value string) string {
	label += ":"
	return label + strings.Repeat(" ", max(1, 12-len(label))) + value
}

// errorLabel returns the styled CLI error prefix.
func errorLabel() string {
	return tinta.Text().Red().Bold().String("error")
}

// defaultWriter returns io.Discard when writer is nil.
func defaultWriter(writer io.Writer) io.Writer {
	if writer == nil {
		return io.Discard
	}

	return writer
}
