package build

import (
	"context"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/varavelio/veta/internal/components"
	"github.com/varavelio/veta/internal/config"
	"github.com/varavelio/veta/internal/data"
	"github.com/varavelio/veta/internal/filters"
	"github.com/varavelio/veta/internal/js"
	"github.com/varavelio/veta/internal/markdown"
	"github.com/varavelio/veta/internal/output"
	"github.com/varavelio/veta/internal/pages"
	"github.com/varavelio/veta/internal/render"
	"github.com/varavelio/veta/internal/theme"
	"github.com/varavelio/veta/internal/tmpl"
)

const (
	// DefaultOutputDir is the default directory used for build output.
	DefaultOutputDir = "dist"

	defaultRoot = "."
)

// Result summarizes a completed site build.
type Result struct {
	Config    config.Config
	Documents int
	OutputDir string
	Pages     int
}

// Option configures a build run.
type Option func(*runConfig) error

type runConfig struct {
	clean         bool
	configFile    string
	consoleOutput io.Writer
	debug         bool
	outputDir     string
	root          string
	themeOptions  []theme.Option
}

type filterScriptRunner struct {
	runner *js.Runner
}

// WithRoot configures the project root directory.
func WithRoot(root string) Option {
	return func(config *runConfig) error {
		root = strings.TrimSpace(root)
		if root == "" || strings.ContainsRune(root, 0) {
			return ErrRootInvalid
		}

		config.root = root
		return nil
	}
}

// WithOutputDir configures the output directory.
func WithOutputDir(outputDir string) Option {
	return func(config *runConfig) error {
		outputDir = strings.TrimSpace(outputDir)
		if outputDir == "" || strings.ContainsRune(outputDir, 0) {
			return ErrOutputDirInvalid
		}

		config.outputDir = outputDir
		return nil
	}
}

// WithConfigFile configures an explicit configuration file path inside root.
func WithConfigFile(name string) Option {
	return func(config *runConfig) error {
		name = strings.TrimSpace(name)
		if strings.ContainsRune(name, 0) {
			return ErrConfigFileInvalid
		}

		config.configFile = name
		return nil
	}
}

// WithClean configures whether output is removed before writing.
func WithClean(clean bool) Option {
	return func(config *runConfig) error {
		config.clean = clean
		return nil
	}
}

// WithDebug configures debug mode for template rendering.
func WithDebug(debug bool) Option {
	return func(config *runConfig) error {
		config.debug = debug
		return nil
	}
}

// WithConsoleOutput configures where JavaScript console messages are written.
func WithConsoleOutput(consoleOutput io.Writer) Option {
	return func(config *runConfig) error {
		config.consoleOutput = consoleOutput
		return nil
	}
}

// WithThemeOptions configures theme resolver options.
func WithThemeOptions(options ...theme.Option) Option {
	return func(config *runConfig) error {
		for _, option := range options {
			if option == nil {
				continue
			}

			config.themeOptions = append(config.themeOptions, option)
		}

		return nil
	}
}

// Run builds a Veta site from the configured root into the configured output.
func Run(ctx context.Context, options ...Option) (Result, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if err := checkContext(ctx); err != nil {
		return Result{}, err
	}

	config, err := newRunConfig(options)
	if err != nil {
		return Result{}, err
	}

	projectFiles := os.DirFS(config.root)
	toolConfig, err := loadToolConfig(projectFiles, config.configFile)
	if err != nil {
		return Result{}, err
	}
	if toolConfig.TailwindCSS.Enabled() {
		return Result{}, ErrTailwindUnsupported
	}

	themeOptions := append(
		[]theme.Option{theme.WithRoot(config.root), theme.WithContext(ctx)},
		config.themeOptions...,
	)
	site, err := theme.Resolve(projectFiles, toolConfig.Theme.Source, themeOptions...)
	if err != nil {
		return Result{}, fmt.Errorf("resolve theme: %w", err)
	}

	markdownRenderer := markdown.New()
	siteData, err := data.Load(site.Files, data.WithJSOptions(baseJSOptions(config, nil)...))
	if err != nil {
		return Result{}, fmt.Errorf("load data: %w", err)
	}
	siteContext := newSiteContext(siteData)
	runtime := js.Runtime{"data": map[string]any(siteData), "site": siteContext}

	manifest, err := pages.Load(site.Files, pages.WithJSOptions(baseJSOptions(config, runtime)...))
	if err != nil {
		return Result{}, fmt.Errorf("load pages: %w", err)
	}
	if err := checkContext(ctx); err != nil {
		return Result{}, err
	}

	templateRenderer, err := newTemplateRenderer(site.Files, markdownRenderer, config, runtime)
	if err != nil {
		return Result{}, err
	}
	componentProcessor, err := components.New(
		site.Files,
		templateRenderer,
		components.WithSlotRenderer(func(content string, _ any) (string, error) {
			return markdownRenderer.Render(content)
		}),
	)
	if err != nil {
		return Result{}, fmt.Errorf("load components: %w", err)
	}
	documentRenderer, err := render.New(
		render.WithContentProcessor(componentProcessor),
		render.WithMarkdownRenderer(markdownRenderer),
		render.WithTemplateRenderer(templateRenderer),
	)
	if err != nil {
		return Result{}, fmt.Errorf("create renderer: %w", err)
	}

	documents, err := documentRenderer.RenderPages(renderPages(manifest.Pages), siteContext)
	if err != nil {
		return Result{}, fmt.Errorf("render pages: %w", err)
	}
	if err := checkContext(ctx); err != nil {
		return Result{}, err
	}

	outputDir := outputRoot(config.root, config.outputDir)
	writer, err := output.New(outputDir, output.WithClean(config.clean))
	if err != nil {
		return Result{}, fmt.Errorf("create output writer: %w", err)
	}
	if err := writer.WriteSite(outputFiles(documents), site.Files); err != nil {
		return Result{}, fmt.Errorf("write output: %w", err)
	}

	return Result{
		Config:    toolConfig,
		Documents: len(documents),
		OutputDir: outputDir,
		Pages:     len(manifest.Pages),
	}, nil
}

// Run executes a JavaScript filter source with explicit filter arguments.
func (runner filterScriptRunner) Run(source filters.Source, input, parameter any) (any, error) {
	result, err := runner.runner.Call(
		js.Source{Name: source.Name, Code: source.Code},
		input,
		parameter,
	)
	if err != nil {
		return nil, err
	}

	return result.Export(), nil
}

// newRunConfig applies build options and defaults.
func newRunConfig(options []Option) (runConfig, error) {
	config := runConfig{outputDir: DefaultOutputDir, root: defaultRoot}
	for _, option := range options {
		if option == nil {
			continue
		}
		if err := option(&config); err != nil {
			return runConfig{}, err
		}
	}

	return config, nil
}

// loadToolConfig loads the default or explicit Veta config file.
func loadToolConfig(files fs.FS, configFile string) (config.Config, error) {
	if strings.TrimSpace(configFile) == "" {
		return config.Load(files)
	}

	return config.LoadFile(files, configFile)
}

// baseJSOptions returns JavaScript runtime options shared by build loaders.
func baseJSOptions(config runConfig, runtime js.Runtime) []js.Option {
	options := []js.Option{js.WithRoot(config.root), js.WithConsoleOutput(config.consoleOutput)}
	if runtime != nil {
		options = append(options, js.WithRuntime(runtime))
	}

	return options
}

// newSiteContext returns the site namespace exposed to templates and scripts.
func newSiteContext(values data.Values) map[string]any {
	return map[string]any{"data": map[string]any(values)}
}

// newTemplateRenderer creates a template renderer with native and script filters.
func newTemplateRenderer(
	files fs.FS,
	markdownRenderer *markdown.Renderer,
	config runConfig,
	runtime js.Runtime,
) (*tmpl.Renderer, error) {
	filterRunner := filterScriptRunner{runner: js.New(baseJSOptions(config, runtime)...)}
	filterSet, err := filters.Load(
		files,
		filters.WithMarkdownRenderer(markdownRenderer),
		filters.WithScriptRunner(filterRunner),
	)
	if err != nil {
		return nil, fmt.Errorf("load filters: %w", err)
	}

	templateOptions := []tmpl.Option{tmpl.WithDebug(config.debug)}
	for name, filter := range filterSet.Functions() {
		templateOptions = append(templateOptions, tmpl.WithFilter(name, tmpl.FilterFunc(filter)))
	}

	templateRenderer, err := tmpl.New(files, templateOptions...)
	if err != nil {
		return nil, fmt.Errorf("create template renderer: %w", err)
	}

	return templateRenderer, nil
}

// renderPages converts page manifest pages into renderer pages.
func renderPages(manifestPages []pages.Page) []render.Page {
	renderPages := make([]render.Page, 0, len(manifestPages))
	for _, page := range manifestPages {
		renderPages = append(renderPages, render.Page{
			Content:    page.Content,
			Data:       page.Data,
			Date:       page.Date,
			Layout:     page.Layout,
			OutputPath: page.OutputPath,
			Permalink:  page.Permalink,
			Title:      page.Title,
		})
	}

	return renderPages
}

// outputFiles converts rendered documents into output files.
func outputFiles(documents []render.Document) []output.File {
	files := make([]output.File, 0, len(documents))
	for _, document := range documents {
		files = append(files, output.File{Content: document.Content, Path: document.OutputPath})
	}

	return files
}

// outputRoot returns the output directory resolved against root when relative.
func outputRoot(root, outputDir string) string {
	if filepath.IsAbs(outputDir) {
		return filepath.Clean(outputDir)
	}

	return filepath.Clean(filepath.Join(root, outputDir))
}

// checkContext reports whether the build context has been canceled.
func checkContext(ctx context.Context) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
		return nil
	}
}
