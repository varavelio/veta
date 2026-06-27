package theme

import (
	"archive/zip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/varavelio/veta/internal/dirs"
	"github.com/varavelio/veta/internal/vfs"
)

const (
	archiveFileName   = "archive.zip"
	cacheCompleteName = ".complete"
	defaultGitHubURL  = "https://codeload.github.com"
	filesDirName      = "files"
	githubCacheDir    = "github"
	repositoryPrefix  = "veta-theme-"
	remoteTimeout     = 60 * time.Second
)

var allowedThemeDirs = []string{"templates", "components", "filters", "data", "public"}

// HTTPClient performs HTTP requests for remote theme downloads.
type HTTPClient interface {
	Do(request *http.Request) (*http.Response, error)
}

// Site contains the project and optional theme filesystems composed for a build.
type Site struct {
	Files   fs.FS
	Project fs.FS
	Theme   fs.FS
	Source  string
}

// Option configures theme resolution.
type Option func(*resolverConfig) error

type resolverConfig struct {
	cacheDir      string
	context       context.Context
	githubBaseURL string
	httpClient    HTTPClient
	root          string
}

type remoteReference struct {
	Owner string
	Ref   string
	Repo  string
}

// WithRoot configures the project root used to resolve relative theme sources.
func WithRoot(root string) Option {
	return func(config *resolverConfig) error {
		root = strings.TrimSpace(root)
		if root == "" || strings.ContainsRune(root, 0) {
			return ErrRootInvalid
		}

		config.root = root
		return nil
	}
}

// WithCacheDir configures the directory used to cache remote themes.
func WithCacheDir(cacheDir string) Option {
	return func(config *resolverConfig) error {
		cacheDir = strings.TrimSpace(cacheDir)
		if cacheDir == "" || strings.ContainsRune(cacheDir, 0) {
			return ErrCacheDirInvalid
		}

		config.cacheDir = normalizePath(cacheDir)
		return nil
	}
}

// WithContext configures the context used for remote theme downloads.
func WithContext(ctx context.Context) Option {
	return func(config *resolverConfig) error {
		if ctx == nil {
			ctx = context.Background()
		}

		config.context = ctx
		return nil
	}
}

// WithGitHubBaseURL configures the GitHub archive base URL.
func WithGitHubBaseURL(baseURL string) Option {
	return func(config *resolverConfig) error {
		baseURL = strings.TrimSpace(baseURL)
		if baseURL == "" || strings.ContainsRune(baseURL, 0) {
			return ErrSourceInvalid
		}

		config.githubBaseURL = strings.TrimRight(baseURL, "/")
		return nil
	}
}

// WithHTTPClient configures the HTTP client used for remote theme downloads.
func WithHTTPClient(client HTTPClient) Option {
	return func(config *resolverConfig) error {
		if client == nil {
			return ErrSourceInvalid
		}

		config.httpClient = client
		return nil
	}
}

// Resolve composes projectFiles with the optional local or remote theme source.
func Resolve(projectFiles fs.FS, source string, options ...Option) (Site, error) {
	if projectFiles == nil {
		return Site{}, ErrProjectFSRequired
	}

	config, err := newResolverConfig(options)
	if err != nil {
		return Site{}, err
	}

	source = strings.TrimSpace(source)
	if source == "" {
		return Site{Files: projectFiles, Project: projectFiles}, nil
	}
	if strings.ContainsRune(source, 0) {
		return Site{}, fmt.Errorf("%w: source cannot contain NUL", ErrSourceInvalid)
	}
	if strings.Contains(source, "://") {
		return Site{}, fmt.Errorf("%w: %s", ErrRemoteUnsupported, source)
	}
	if remoteSource(source) {
		themeRoot, err := resolveRemoteTheme(config, source)
		if err != nil {
			return Site{}, err
		}

		return compose(projectFiles, themeRoot, source)
	}

	themeRoot := localThemeRoot(config.root, source)
	return compose(projectFiles, themeRoot, source)
}

// newResolverConfig applies options and defaults.
func newResolverConfig(options []Option) (resolverConfig, error) {
	config := resolverConfig{
		context:       context.Background(),
		githubBaseURL: defaultGitHubURL,
		httpClient:    &http.Client{Timeout: remoteTimeout},
		root:          ".",
	}
	for _, option := range options {
		if option == nil {
			continue
		}
		if err := option(&config); err != nil {
			return resolverConfig{}, err
		}
	}

	return config, nil
}

// compose returns the project and theme filesystems as a Veta site.
func compose(projectFiles fs.FS, themeRoot, source string) (Site, error) {
	themeInfo, err := os.Stat(themeRoot)
	if err != nil {
		return Site{}, fmt.Errorf("%w: %s: %w", ErrSourceInvalid, source, err)
	}
	if !themeInfo.IsDir() {
		return Site{}, fmt.Errorf("%w: %s is not a directory", ErrSourceInvalid, source)
	}

	themeFiles, err := vfs.AllowTopDirs(os.DirFS(themeRoot), allowedThemeDirs...)
	if err != nil {
		return Site{}, fmt.Errorf("filter theme %s: %w", source, err)
	}
	overlay, err := vfs.NewOverlay(
		vfs.Layer{Name: "theme", FS: themeFiles},
		vfs.Layer{Name: "project", FS: projectFiles},
	)
	if err != nil {
		return Site{}, fmt.Errorf("compose theme %s: %w", source, err)
	}

	return Site{Files: overlay, Project: projectFiles, Theme: themeFiles, Source: themeRoot}, nil
}

// resolveRemoteTheme returns an extracted cached remote theme path.
func resolveRemoteTheme(config resolverConfig, source string) (string, error) {
	reference, err := parseRemoteReference(source)
	if err != nil {
		return "", err
	}

	cacheDir := config.cacheDir
	if cacheDir == "" {
		cacheDir, err = dirs.GetThemesCacheDir()
		if err != nil {
			return "", err
		}
	}

	cacheRoot := remoteCacheRoot(cacheDir, reference)
	filesRoot := filepath.Join(cacheRoot, filesDirName)
	if cachedThemeReady(cacheRoot, filesRoot) {
		return filesRoot, nil
	}

	if err := downloadRemoteTheme(config, reference, cacheRoot); err != nil {
		return "", err
	}

	return filesRoot, nil
}

// localThemeRoot returns the filesystem path for a local theme source.
func localThemeRoot(root, source string) string {
	if filepath.IsAbs(source) {
		return filepath.Clean(source)
	}

	return filepath.Clean(filepath.Join(root, source))
}

// parseRemoteReference parses a GitHub owner/repository@ref theme source.
func parseRemoteReference(source string) (remoteReference, error) {
	path, ref, ok := strings.Cut(source, "@")
	if !ok || strings.TrimSpace(ref) == "" {
		return remoteReference{}, fmt.Errorf("%w: remote theme must include @ref", ErrSourceInvalid)
	}
	owner, repo, ok := strings.Cut(path, "/")
	if !ok || strings.TrimSpace(owner) == "" || strings.TrimSpace(repo) == "" {
		return remoteReference{}, fmt.Errorf(
			"%w: remote theme must be owner/repo@ref",
			ErrSourceInvalid,
		)
	}
	if strings.Contains(repo, "/") {
		return remoteReference{}, fmt.Errorf(
			"%w: remote theme must be owner/repo@ref",
			ErrSourceInvalid,
		)
	}

	owner = strings.TrimSpace(owner)
	repo = strings.TrimSpace(repo)
	ref = strings.TrimSpace(ref)
	if !validGitHubName(owner) || !validGitHubName(repo) || !validGitReference(ref) {
		return remoteReference{}, fmt.Errorf("%w: %s", ErrSourceInvalid, source)
	}
	if !strings.HasPrefix(repo, repositoryPrefix) {
		return remoteReference{}, fmt.Errorf(
			"%w: remote theme repository %q must start with %q",
			ErrSourceInvalid,
			repo,
			repositoryPrefix,
		)
	}

	return remoteReference{Owner: owner, Ref: ref, Repo: repo}, nil
}

// validGitHubName reports whether name is safe for owner or repository segments.
func validGitHubName(name string) bool {
	if name == "" || strings.Contains(name, "..") {
		return false
	}
	for _, char := range name {
		if 'A' <= char && char <= 'Z' || 'a' <= char && char <= 'z' ||
			'0' <= char && char <= '9' || char == '-' || char == '_' || char == '.' {
			continue
		}

		return false
	}

	return true
}

// validGitReference reports whether ref is safe for a GitHub archive URL.
func validGitReference(ref string) bool {
	if ref == "" || strings.ContainsAny(ref, "\x00\\ \t\r\n") || strings.Contains(ref, "..") ||
		strings.HasPrefix(ref, "/") || strings.HasSuffix(ref, "/") || strings.Contains(ref, "//") {
		return false
	}

	return true
}

// remoteCacheRoot returns the cache directory for a remote theme reference.
func remoteCacheRoot(cacheDir string, reference remoteReference) string {
	hash := sha256.Sum256([]byte(reference.Ref))
	return filepath.Join(
		cacheDir,
		githubCacheDir,
		reference.Owner,
		reference.Repo,
		hex.EncodeToString(hash[:]),
	)
}

// cachedThemeReady reports whether a remote theme cache entry is complete.
func cachedThemeReady(cacheRoot, filesRoot string) bool {
	if _, err := os.Stat(filepath.Join(cacheRoot, cacheCompleteName)); err != nil {
		return false
	}
	info, err := os.Stat(filesRoot)
	return err == nil && info.IsDir()
}

// downloadRemoteTheme downloads and extracts a remote theme into cacheRoot.
func downloadRemoteTheme(config resolverConfig, reference remoteReference, cacheRoot string) error {
	parentDir := filepath.Dir(cacheRoot)
	if err := os.MkdirAll(parentDir, 0o755); err != nil {
		return fmt.Errorf("create theme cache parent %s: %w", parentDir, err)
	}

	tempRoot, err := os.MkdirTemp(parentDir, ".download-*")
	if err != nil {
		return fmt.Errorf("create temporary theme cache: %w", err)
	}
	installed := false
	defer func() {
		if !installed {
			_ = os.RemoveAll(tempRoot)
		}
	}()

	archivePath := filepath.Join(tempRoot, archiveFileName)
	if err := downloadArchive(config, reference, archivePath); err != nil {
		return err
	}
	if err := extractArchive(archivePath, filepath.Join(tempRoot, filesDirName)); err != nil {
		return err
	}
	if err := os.WriteFile(
		filepath.Join(tempRoot, cacheCompleteName),
		[]byte(reference.Ref),
		0o644,
	); err != nil {
		return fmt.Errorf("write theme cache marker: %w", err)
	}

	if err := os.RemoveAll(cacheRoot); err != nil {
		return fmt.Errorf("replace theme cache %s: %w", cacheRoot, err)
	}
	if err := os.Rename(tempRoot, cacheRoot); err != nil {
		return fmt.Errorf("install theme cache %s: %w", cacheRoot, err)
	}

	installed = true
	return nil
}

// downloadArchive downloads a GitHub ZIP archive to archivePath.
func downloadArchive(config resolverConfig, reference remoteReference, archivePath string) error {
	request, err := http.NewRequestWithContext(
		config.context,
		http.MethodGet,
		archiveURL(config.githubBaseURL, reference),
		nil,
	)
	if err != nil {
		return fmt.Errorf("%w: create request: %w", ErrDownloadFailed, err)
	}
	request.Header.Set("Accept", "application/zip")
	request.Header.Set("User-Agent", "veta")

	response, err := config.httpClient.Do(request)
	if err != nil {
		return fmt.Errorf("%w: %w", ErrDownloadFailed, err)
	}
	defer func() {
		_ = response.Body.Close()
	}()

	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		_, _ = io.Copy(io.Discard, response.Body)
		return fmt.Errorf("%w: github returned %s", ErrDownloadFailed, response.Status)
	}

	archiveFile, err := os.Create(archivePath)
	if err != nil {
		return fmt.Errorf("create theme archive %s: %w", archivePath, err)
	}

	if _, err := io.Copy(archiveFile, response.Body); err != nil {
		_ = archiveFile.Close()
		return fmt.Errorf("%w: write archive: %w", ErrDownloadFailed, err)
	}
	if err := archiveFile.Close(); err != nil {
		return fmt.Errorf("close theme archive %s: %w", archivePath, err)
	}

	return nil
}

// archiveURL returns the GitHub codeload URL for reference.
func archiveURL(baseURL string, reference remoteReference) string {
	return strings.Join([]string{
		strings.TrimRight(baseURL, "/"),
		url.PathEscape(reference.Owner),
		url.PathEscape(reference.Repo),
		"zip",
		escapeReferencePath(reference.Ref),
	}, "/")
}

// escapeReferencePath escapes each path segment in a Git reference.
func escapeReferencePath(ref string) string {
	parts := strings.Split(ref, "/")
	for index, part := range parts {
		parts[index] = url.PathEscape(part)
	}

	return strings.Join(parts, "/")
}

// extractArchive extracts a GitHub archive under destination.
func extractArchive(archivePath, destination string) error {
	reader, err := zip.OpenReader(archivePath)
	if err != nil {
		return fmt.Errorf("%w: open archive: %w", ErrDownloadFailed, err)
	}
	defer func() {
		_ = reader.Close()
	}()

	for _, file := range reader.File {
		name, ok := archiveEntryName(file.Name)
		if !ok {
			continue
		}

		if err := extractArchiveEntry(
			file,
			filepath.Join(destination, filepath.FromSlash(name)),
		); err != nil {
			return err
		}
	}

	return nil
}

// archiveEntryName strips the GitHub archive root directory from an entry name.
func archiveEntryName(name string) (string, bool) {
	name = strings.ReplaceAll(name, "\\", "/")
	name = strings.TrimLeft(name, "/")
	_, rest, ok := strings.Cut(name, "/")
	if !ok || rest == "" {
		return "", false
	}
	if strings.Contains(rest, "..") || strings.HasPrefix(rest, "/") ||
		strings.ContainsRune(rest, 0) {
		return "", false
	}

	return rest, true
}

// extractArchiveEntry extracts one safe ZIP archive entry.
func extractArchiveEntry(file *zip.File, targetPath string) error {
	if file.FileInfo().IsDir() {
		return os.MkdirAll(targetPath, 0o755)
	}
	if file.FileInfo().Mode()&os.ModeSymlink != 0 {
		return nil
	}

	if err := os.MkdirAll(filepath.Dir(targetPath), 0o755); err != nil {
		return fmt.Errorf("create archive parent %s: %w", targetPath, err)
	}

	source, err := file.Open()
	if err != nil {
		return fmt.Errorf("open archive entry %s: %w", file.Name, err)
	}
	defer func() {
		_ = source.Close()
	}()

	target, err := os.OpenFile(
		targetPath,
		os.O_CREATE|os.O_WRONLY|os.O_TRUNC,
		file.FileInfo().Mode().Perm(),
	)
	if err != nil {
		return fmt.Errorf("create archive entry %s: %w", targetPath, err)
	}

	if _, err := io.Copy(target, source); err != nil {
		_ = target.Close()
		return fmt.Errorf("extract archive entry %s: %w", file.Name, err)
	}
	if err := target.Close(); err != nil {
		return fmt.Errorf("close archive entry %s: %w", targetPath, err)
	}

	return nil
}

// normalizePath converts path to an absolute clean path when possible.
func normalizePath(path string) string {
	if filepath.IsAbs(path) {
		return filepath.Clean(path)
	}
	absolutePath, err := filepath.Abs(path)
	if err != nil {
		return filepath.Clean(path)
	}

	return filepath.Clean(absolutePath)
}

// remoteSource reports whether source looks like a remote theme reference.
func remoteSource(source string) bool {
	return strings.Contains(source, "@") && !strings.HasPrefix(source, ".") &&
		!filepath.IsAbs(source)
}
