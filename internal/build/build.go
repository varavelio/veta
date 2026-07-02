package build

import (
	"context"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/varavelio/veta/internal/components"
	"github.com/varavelio/veta/internal/config"
	"github.com/varavelio/veta/internal/data"
	"github.com/varavelio/veta/internal/filters"
	"github.com/varavelio/veta/internal/js"
	"github.com/varavelio/veta/internal/loaddata"
	"github.com/varavelio/veta/internal/markdown"
	"github.com/varavelio/veta/internal/output"
	"github.com/varavelio/veta/internal/pages"
	"github.com/varavelio/veta/internal/render"
	"github.com/varavelio/veta/internal/tailwindcss"
	"github.com/varavelio/veta/internal/template"
	"github.com/varavelio/veta/internal/theme"
)

const (
	// DefaultOutputDir is the default directory used for build output.
	DefaultOutputDir = config.DefaultBuildOutput

	defaultRoot = "."

	templatesDirName = "templates"
)

// Result summarizes a completed site build.
type Result struct {
	Config         config.Config
	Documents      int
	GeneratedFiles []string
	OutputDir      string
	Pages          int
	Root           string
}

// Option configures a build run.
type Option func(*runConfig) error

type runConfig struct {
	configFile      string
	consoleOutput   io.Writer
	cleanOverride   *bool
	outputDir       string
	outputDirSet    bool
	root            string
	tailwindOptions []tailwindcss.Option
	themeOptions    []theme.Option
}

type filterScriptRunner struct {
	runner *js.Runner
}

// WithRoot configures the directory where config discovery starts.
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

// WithConfigFile configures an explicit configuration file path.
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

// WithConsoleOutput configures where JavaScript console messages are written.
func WithConsoleOutput(consoleOutput io.Writer) Option {
	return func(config *runConfig) error {
		config.consoleOutput = consoleOutput
		return nil
	}
}

// WithOutputDir configures the output directory for this build run.
func WithOutputDir(outputDir string) Option {
	return func(config *runConfig) error {
		outputDir = strings.TrimSpace(outputDir)
		if outputDir == "" || strings.ContainsRune(outputDir, 0) {
			return ErrOutputDirInvalid
		}

		config.outputDir = outputDir
		config.outputDirSet = true
		return nil
	}
}

// WithClean configures whether this build run cleans its output directory.
func WithClean(clean bool) Option {
	return func(config *runConfig) error {
		config.cleanOverride = &clean
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

// WithTailwindOptions configures Tailwind CSS build options.
func WithTailwindOptions(options ...tailwindcss.Option) Option {
	return func(config *runConfig) error {
		for _, option := range options {
			if option == nil {
				continue
			}

			config.tailwindOptions = append(config.tailwindOptions, option)
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

	runConfig, err := newRunConfig(options)
	if err != nil {
		return Result{}, err
	}

	projectRoot, toolConfig, err := loadToolConfig(runConfig)
	if err != nil {
		return Result{}, err
	}
	toolConfig = applyRuntimeOverrides(toolConfig, runConfig)
	runConfig.root = projectRoot
	projectFiles := os.DirFS(projectRoot)
	themeOptions := append(
		[]theme.Option{
			theme.WithRoot(projectRoot),
			theme.WithContext(ctx),
		},
		runConfig.themeOptions...,
	)
	site, err := theme.Resolve(projectFiles, toolConfig.Theme.Source, themeOptions...)
	if err != nil {
		return Result{}, fmt.Errorf("resolve theme: %w", err)
	}

	markdownRenderer := markdown.New()
	siteData, err := data.Load(site.Files, data.WithJSOptions(baseJSOptions(runConfig, nil)...))
	if err != nil {
		return Result{}, fmt.Errorf("load data: %w", err)
	}
	dataContext := map[string]any(siteData)
	runtime := js.Runtime{"data": dataContext}

	manifest, err := pages.Load(
		site.Files,
		pages.WithJSOptions(baseJSOptions(runConfig, runtime)...),
	)
	if err != nil {
		return Result{}, fmt.Errorf("load pages: %w", err)
	}
	if err := checkContext(ctx); err != nil {
		return Result{}, err
	}

	templateRenderer, err := newTemplateRenderer(
		site.Files,
		markdownRenderer,
		runConfig,
		runtime,
	)
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
		render.WithTemplateRenderer(pageTemplateRenderer{renderer: templateRenderer}),
	)
	if err != nil {
		return Result{}, fmt.Errorf("create renderer: %w", err)
	}

	documents, err := documentRenderer.RenderPages(renderPages(manifest.Pages), dataContext)
	if err != nil {
		return Result{}, fmt.Errorf("render pages: %w", err)
	}
	if err := checkContext(ctx); err != nil {
		return Result{}, err
	}

	generatedFiles := outputFiles(documents)
	outputDir := outputRoot(projectRoot, toolConfig.Build.Output)
	writer, err := output.New(
		outputDir,
		output.WithClean(toolConfig.Build.Clean),
		output.WithHTMLMinify(toolConfig.HTML.Minify),
	)
	if err != nil {
		return Result{}, fmt.Errorf("create output writer: %w", err)
	}
	if err := writer.WriteSite(generatedFiles, site.Files); err != nil {
		return Result{}, fmt.Errorf("write output: %w", err)
	}
	if err := buildTailwindCSS(ctx, site.Files, outputDir, toolConfig, runConfig); err != nil {
		return Result{}, err
	}

	return Result{
		Config:         toolConfig,
		Documents:      len(documents),
		GeneratedFiles: outputFilePaths(generatedFiles),
		OutputDir:      outputDir,
		Pages:          len(manifest.Pages),
		Root:           projectRoot,
	}, nil
}

// pageTemplateRenderer resolves page template names from the templates directory.
type pageTemplateRenderer struct {
	renderer render.TemplateRenderer
}

// Render renders one page template by resolving the name under templates/.
func (renderer pageTemplateRenderer) Render(name string, context any) (string, error) {
	return renderer.renderer.Render(path.Join(templatesDirName, name), context)
}

// buildTailwindCSS builds CSS when Tailwind CSS is enabled.
func buildTailwindCSS(
	ctx context.Context,
	files fs.FS,
	outputDir string,
	toolConfig config.Config,
	runConfig runConfig,
) error {
	if !toolConfig.TailwindCSS.Enabled() {
		return nil
	}

	for _, stylesheet := range toolConfig.TailwindCSS.Stylesheets {
		if err := tailwindcss.Build(
			ctx,
			files,
			tailwindcss.Config{
				Input:   path.Join(output.PublicDirName, stylesheet),
				Minify:  toolConfig.TailwindCSS.Minify,
				Output:  filepath.Join(outputDir, filepath.FromSlash(stylesheet)),
				WorkDir: outputDir,
			},
			runConfig.tailwindOptions...,
		); err != nil {
			return fmt.Errorf("build tailwindcss %s: %w", stylesheet, err)
		}
	}

	return nil
}

// Run executes a JavaScript filter source with a runtime context followed by
// explicit filter arguments.
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
	config := runConfig{root: defaultRoot}
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

// applyRuntimeOverrides applies per-run settings after the config file is loaded.
func applyRuntimeOverrides(toolConfig config.Config, runConfig runConfig) config.Config {
	if runConfig.outputDirSet {
		toolConfig.Build.Output = runConfig.outputDir
	}
	if runConfig.cleanOverride != nil {
		toolConfig.Build.Clean = *runConfig.cleanOverride
	}

	return toolConfig
}

// loadToolConfig discovers and loads the Veta config file for a build.
func loadToolConfig(runConfig runConfig) (string, config.Config, error) {
	configPath, err := resolveConfigPath(runConfig.root, runConfig.configFile)
	if err != nil {
		return "", config.Config{}, err
	}

	root := filepath.Dir(configPath)
	toolConfig, err := config.LoadRequiredFile(os.DirFS(root), filepath.Base(configPath))
	if err != nil {
		return "", config.Config{}, err
	}

	return root, toolConfig, nil
}

// resolveConfigPath returns the explicit or discovered configuration file path.
func resolveConfigPath(root, configFile string) (string, error) {
	root, err := normalizeRoot(root)
	if err != nil {
		return "", err
	}
	configFile = strings.TrimSpace(configFile)
	if configFile != "" {
		return explicitConfigPath(root, configFile)
	}

	return discoverConfigPath(root)
}

// explicitConfigPath resolves a caller-provided configuration file path.
func explicitConfigPath(root, configFile string) (string, error) {
	if strings.ContainsRune(configFile, 0) {
		return "", ErrConfigFileInvalid
	}
	if !filepath.IsAbs(configFile) {
		configFile = filepath.Join(root, configFile)
	}
	configFile = filepath.Clean(configFile)

	info, err := os.Stat(configFile)
	if err != nil {
		return "", fmt.Errorf("%w: %s: %w", ErrConfigFileInvalid, configFile, err)
	}
	if info.IsDir() {
		return "", fmt.Errorf("%w: %s is a directory", ErrConfigFileInvalid, configFile)
	}

	return configFile, nil
}

// discoverConfigPath searches root and its ancestors for a Veta config file.
func discoverConfigPath(root string) (string, error) {
	directory := root
	for {
		configPath, found, err := configInDirectory(directory)
		if err != nil {
			return "", err
		}
		if found {
			return configPath, nil
		}

		parent := filepath.Dir(directory)
		if parent == directory {
			return "", fmt.Errorf("%w: searched from %s", ErrConfigNotFound, root)
		}
		directory = parent
	}
}

// configInDirectory returns the highest-priority Veta config file in directory.
func configInDirectory(directory string) (string, bool, error) {
	for _, name := range config.FileNames() {
		path := filepath.Join(directory, name)
		info, err := os.Stat(path)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}

			return "", false, fmt.Errorf("inspect configuration %s: %w", path, err)
		}
		if info.IsDir() {
			continue
		}

		return path, true, nil
	}

	return "", false, nil
}

// normalizeRoot returns the absolute config search root.
func normalizeRoot(root string) (string, error) {
	root = strings.TrimSpace(root)
	if root == "" || strings.ContainsRune(root, 0) {
		return "", ErrRootInvalid
	}
	root, err := filepath.Abs(root)
	if err != nil {
		return "", fmt.Errorf("%w: %w", ErrRootInvalid, err)
	}
	info, err := os.Stat(root)
	if err != nil {
		return "", fmt.Errorf("%w: %s: %w", ErrRootInvalid, root, err)
	}
	if !info.IsDir() {
		return "", fmt.Errorf("%w: %s is not a directory", ErrRootInvalid, root)
	}

	return filepath.Clean(root), nil
}

// baseJSOptions returns JavaScript runtime options shared by build loaders.
func baseJSOptions(config runConfig, runtime js.Runtime) []js.Option {
	options := []js.Option{js.WithRoot(config.root), js.WithConsoleOutput(config.consoleOutput)}
	if runtime != nil {
		options = append(options, js.WithRuntime(runtime))
	}

	return options
}

// newTemplateRenderer creates a template renderer with native and script filters.
func newTemplateRenderer(
	files fs.FS,
	markdownRenderer *markdown.Renderer,
	config runConfig,
	runtime js.Runtime,
) (*template.Renderer, error) {
	filterRunner := filterScriptRunner{runner: js.New(baseJSOptions(config, runtime)...)}
	filterSet, err := filters.Load(
		files,
		filters.WithMarkdownRenderer(markdownRenderer),
		filters.WithScriptRunner(filterRunner),
	)
	if err != nil {
		return nil, fmt.Errorf("load filters: %w", err)
	}

	dataLoader, err := loaddata.New(files)
	if err != nil {
		return nil, fmt.Errorf("create load_data loader: %w", err)
	}
	templateOptions := []template.Option{
		template.WithLoadData(func(request template.LoadDataRequest) (any, error) {
			return dataLoader.Load(loaddata.Request{
				Path: request.Path,
				URL:  request.URL,
			})
		}),
	}
	for name, filter := range filterSet.Functions() {
		templateOptions = append(
			templateOptions,
			template.WithFilter(name, template.FilterFunc(filter)),
		)
	}

	templateRenderer, err := template.New(files, templateOptions...)
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
			Fields:     page.Fields,
			OutputPath: page.OutputPath,
			Permalink:  page.Permalink,
			Template:   page.Template,
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

// outputFilePaths returns the generated output paths from files.
func outputFilePaths(files []output.File) []string {
	paths := make([]string, 0, len(files))
	for _, file := range files {
		paths = append(paths, file.Path)
	}

	return paths
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
