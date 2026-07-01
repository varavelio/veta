package dev

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/varavelio/veta/internal/build"
	toolconfig "github.com/varavelio/veta/internal/config"
)

const (
	// DefaultHost is the local interface used by veta dev.
	DefaultHost = toolconfig.DefaultDevHost

	// DefaultPort is the TCP port used by veta dev.
	DefaultPort = toolconfig.DefaultDevPort

	defaultPollInterval = 500 * time.Millisecond
	defaultDebounce     = 200 * time.Millisecond
	shutdownTimeout     = 5 * time.Second
)

// Config contains the local development server settings.
type Config struct {
	ConfigFile   string
	Debounce     time.Duration
	PollInterval time.Duration
	Stderr       io.Writer
	Stdout       io.Writer
}

type server struct {
	broadcaster *broadcaster
	config      Config
	outputDir   string
}

// Run starts the development workflow until ctx is canceled or the server fails.
func Run(ctx context.Context, config Config) (err error) {
	if ctx == nil {
		ctx = context.Background()
	}

	config, err = normalizeConfig(config)
	if err != nil {
		return err
	}

	outputDir, err := os.MkdirTemp("", "veta-dev-*")
	if err != nil {
		return fmt.Errorf("create dev output directory: %w", err)
	}
	defer func() {
		if removeErr := os.RemoveAll(outputDir); err == nil && removeErr != nil {
			err = fmt.Errorf("remove dev output directory %s: %w", outputDir, removeErr)
		}
	}()

	server := server{
		broadcaster: newBroadcaster(),
		config:      config,
		outputDir:   outputDir,
	}

	return server.run(ctx)
}

// normalizeConfig applies defaults and validates user-facing dev settings.
func normalizeConfig(config Config) (Config, error) {
	config.Stdout = defaultWriter(config.Stdout)
	config.Stderr = defaultWriter(config.Stderr)
	if config.PollInterval <= 0 {
		config.PollInterval = defaultPollInterval
	}
	if config.Debounce <= 0 {
		config.Debounce = defaultDebounce
	}

	return config, nil
}

// run performs the build, server, watcher, and rebuild loop.
func (server server) run(ctx context.Context) (err error) {
	result, err := server.rebuild(ctx, "Building site")
	if err != nil {
		return err
	}
	devConfig := result.Config.Dev

	listenConfig := net.ListenConfig{}
	listener, err := listenConfig.Listen(ctx, "tcp", listenAddress(devConfig))
	if err != nil {
		return fmt.Errorf("%w: %s: %w", ErrListenFailed, listenAddress(devConfig), err)
	}

	generatedHTML := newGeneratedHTMLFiles(result.GeneratedFiles)
	httpServer := &http.Server{
		Handler:           newHandler(server.outputDir, server.broadcaster, generatedHTML),
		ReadHeaderTimeout: 5 * time.Second,
	}
	serverErrors := make(chan error, 1)
	go func() {
		serveErr := httpServer.Serve(listener)
		if errors.Is(serveErr, http.ErrServerClosed) {
			serveErr = nil
		}
		serverErrors <- serveErr
	}()
	defer func() {
		if shutdownErr := shutdownHTTPServer(httpServer); err == nil && shutdownErr != nil {
			err = shutdownErr
		}
	}()

	watchCtx, stopWatching := context.WithCancel(ctx)
	defer stopWatching()
	changes, watcherErrors := watchProject(
		watchCtx,
		result.Root,
		server.watchPaths(result.Root, devConfig.Watch),
		server.config.PollInterval,
		server.config.Debounce,
	)

	if err := server.printStartup(listener.Addr()); err != nil {
		return err
	}

	for {
		select {
		case <-ctx.Done():
			if _, err := fmt.Fprintln(server.config.Stdout, "Stopping dev server..."); err != nil {
				return err
			}
			return nil
		case serveErr := <-serverErrors:
			if serveErr == nil {
				return nil
			}

			return fmt.Errorf("serve dev server: %w", serveErr)
		case watcherErr, ok := <-watcherErrors:
			if !ok {
				watcherErrors = nil
				continue
			}
			if watcherErr != nil {
				return watcherErr
			}
		case _, ok := <-changes:
			if !ok {
				changes = nil
				continue
			}

			rebuildResult, err := server.rebuild(ctx, "Rebuilding site")
			if err != nil {
				if ctx.Err() != nil {
					return nil
				}
				rebuildErr := err
				if _, writeErr := fmt.Fprintf(
					server.config.Stderr,
					"Rebuild failed: %s\n",
					rebuildErr,
				); writeErr != nil {
					return writeErr
				}
				continue
			}

			generatedHTML.update(rebuildResult.GeneratedFiles)
			server.broadcaster.broadcastReload()
		}
	}
}

// printStartup writes the dev server startup summary.
func (server server) printStartup(address net.Addr) error {
	if _, err := fmt.Fprintf(
		server.config.Stdout,
		"Serving at %s\n",
		localURL(address),
	); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(
		server.config.Stdout,
		"Development output: %s\n",
		server.outputDir,
	); err != nil {
		return err
	}
	if _, err := fmt.Fprintln(
		server.config.Stdout,
		"Watching for changes. Press Ctrl+C to stop.",
	); err != nil {
		return err
	}

	return nil
}

// rebuild runs a full clean build into the development output directory.
func (server server) rebuild(ctx context.Context, label string) (build.Result, error) {
	if _, err := fmt.Fprintf(server.config.Stdout, "%s...\n", label); err != nil {
		return build.Result{}, err
	}
	startedAt := time.Now()
	result, err := build.Run(ctx, server.buildOptions()...)
	if err != nil {
		return build.Result{}, err
	}

	if _, err := fmt.Fprintf(
		server.config.Stdout,
		"Built %d %s in %s\n",
		result.Pages,
		pageLabel(result.Pages),
		buildDuration(time.Since(startedAt)),
	); err != nil {
		return build.Result{}, err
	}
	return result, nil
}

// buildOptions returns the full-build options used by the dev workflow.
func (server server) buildOptions() []build.Option {
	return []build.Option{
		build.WithConfigFile(server.config.ConfigFile),
		build.WithConsoleOutput(server.config.Stderr),
		build.WithOutputDir(server.outputDir),
		build.WithClean(true),
	}
}

// watchPaths returns project paths observed for rebuild triggers.
func (server server) watchPaths(root string, configuredPaths []string) []string {
	paths := defaultWatchPaths()
	paths = append(paths, configuredPaths...)
	if configFile, ok := explicitConfigWatchPath(root, server.config.ConfigFile); ok {
		paths = append(paths, configFile)
	}

	return paths
}

// explicitConfigWatchPath returns the explicit config path relative to root.
func explicitConfigWatchPath(root, configFile string) (string, bool) {
	configFile = strings.TrimSpace(configFile)
	if configFile == "" {
		return "", false
	}
	if !filepath.IsAbs(configFile) {
		absolute, err := filepath.Abs(configFile)
		if err != nil {
			return "", false
		}
		configFile = absolute
	}

	relativePath, err := filepath.Rel(root, configFile)
	if err != nil {
		return "", false
	}

	return filepath.ToSlash(relativePath), true
}

// listenAddress returns the TCP address used by the HTTP server.
func listenAddress(config toolconfig.Dev) string {
	return net.JoinHostPort(config.Host, strconv.Itoa(config.Port))
}

// localURL returns the browser URL for a listener address.
func localURL(address net.Addr) string {
	host, port, err := net.SplitHostPort(address.String())
	if err != nil {
		return "http://" + address.String() + "/"
	}
	if host == "" || host == "::" || host == "0.0.0.0" {
		host = "localhost"
	}

	return (&url.URL{Scheme: "http", Host: net.JoinHostPort(host, port), Path: "/"}).String()
}

// shutdownHTTPServer stops the development HTTP server gracefully.
func shutdownHTTPServer(server *http.Server) error {
	if server == nil {
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()
	if err := server.Shutdown(ctx); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return fmt.Errorf("shut down dev server: %w", err)
	}

	return nil
}

// pageLabel returns the correctly pluralized page noun.
func pageLabel(pages int) string {
	if pages == 1 {
		return "page"
	}

	return "pages"
}

// buildDuration returns a readable ASCII duration for dev server output.
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

// defaultWriter returns io.Discard when writer is nil.
func defaultWriter(writer io.Writer) io.Writer {
	if writer == nil {
		return io.Discard
	}

	return writer
}
