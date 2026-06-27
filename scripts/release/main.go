package main

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

const (
	checksumFileName = "checksums.txt"
	commandPackage   = "./cmd/veta/."
	distDirName      = "dist"
	projectName      = "veta"
	versionPackage   = "github.com/varavelio/veta/internal/version"
)

var releaseTargets = []target{
	{OS: "linux", Arch: "amd64"},
	{OS: "linux", Arch: "arm64"},
	{OS: "darwin", Arch: "amd64"},
	{OS: "darwin", Arch: "arm64"},
	{OS: "windows", Arch: "amd64"},
}

type target struct {
	Arch string
	OS   string
}

type releaseMetadata struct {
	Commit  string
	Date    string
	Version string
}

// main runs the release builder and owns process exit behavior.
func main() {
	if err := run(context.Background()); err != nil {
		fmt.Fprintf(os.Stderr, "release build failed: %s\n", err)
		os.Exit(1)
	}
}

// run builds all release archives and writes checksums into dist/.
func run(ctx context.Context) error {
	root, err := findProjectRoot()
	if err != nil {
		return err
	}
	metadata := detectReleaseMetadata(ctx, root)
	distDir := filepath.Join(root, distDirName)

	fmt.Printf("Project root: %s\n", root)
	fmt.Printf("Release: %s commit %s built %s\n", metadata.Version, metadata.Commit, metadata.Date)

	if err := os.RemoveAll(distDir); err != nil {
		return fmt.Errorf("clean dist directory: %w", err)
	}
	if err := os.MkdirAll(distDir, 0o755); err != nil {
		return fmt.Errorf("create dist directory: %w", err)
	}

	for _, target := range releaseTargets {
		if err := buildArchive(ctx, root, distDir, metadata, target); err != nil {
			return err
		}
	}
	if err := writeChecksums(distDir); err != nil {
		return err
	}

	fmt.Printf("Release artifacts written to %s\n", distDir)
	return nil
}

// findProjectRoot returns the repository root by walking up from the cwd.
func findProjectRoot() (string, error) {
	directory, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("get working directory: %w", err)
	}

	for {
		if _, err := os.Stat(filepath.Join(directory, "go.mod")); err == nil {
			if _, err := os.Stat(filepath.Join(directory, "Taskfile.yml")); err == nil {
				return directory, nil
			}
		}
		parent := filepath.Dir(directory)
		if parent == directory {
			return "", fmt.Errorf("project root not found")
		}
		directory = parent
	}
}

// detectReleaseMetadata determines version data from environment or git.
func detectReleaseMetadata(ctx context.Context, root string) releaseMetadata {
	version := normalizeVersion(firstNonEmpty(
		os.Getenv("VETA_VERSION"),
		tagLike(os.Getenv("GITHUB_REF_NAME")),
		gitOutput(ctx, root, "describe", "--tags", "--abbrev=0"),
		"0.0.0-dev",
	))
	commit := firstNonEmpty(
		shortCommit(os.Getenv("VETA_COMMIT")),
		shortCommit(os.Getenv("GITHUB_SHA")),
		gitOutput(ctx, root, "rev-parse", "--short", "HEAD"),
		"unknown",
	)
	date := firstNonEmpty(os.Getenv("VETA_DATE"), time.Now().UTC().Format(time.RFC3339))

	return releaseMetadata{Commit: commit, Date: date, Version: version}
}

// buildArchive cross-compiles one target and archives its binary.
func buildArchive(
	ctx context.Context,
	root string,
	distDir string,
	metadata releaseMetadata,
	target target,
) error {
	fmt.Printf("Building %s/%s...\n", target.OS, target.Arch)
	tempDir, err := os.MkdirTemp(distDir, ".build-*")
	if err != nil {
		return fmt.Errorf("create temporary build directory: %w", err)
	}
	defer func() {
		_ = os.RemoveAll(tempDir)
	}()

	rawBinary := filepath.Join(tempDir, binaryName(target.OS))
	command := exec.CommandContext(
		ctx,
		"go",
		"build",
		"-trimpath",
		"-ldflags",
		ldflags(metadata),
		"-o",
		rawBinary,
		commandPackage,
	)
	command.Dir = root
	command.Env = append(os.Environ(),
		"CGO_ENABLED=0",
		"GOOS="+target.OS,
		"GOARCH="+target.Arch,
	)
	command.Stdout = os.Stdout
	command.Stderr = os.Stderr
	if err := command.Run(); err != nil {
		return fmt.Errorf("build %s/%s: %w", target.OS, target.Arch, err)
	}

	format := archiveFormat(target.OS)
	archivePath := filepath.Join(distDir, archiveName(target))
	files := archiveFiles(root, rawBinary, target.OS)
	if format == "zip" {
		err = createZip(archivePath, files)
	} else {
		err = createTarGz(archivePath, files)
	}
	if err != nil {
		return fmt.Errorf("archive %s/%s: %w", target.OS, target.Arch, err)
	}

	return nil
}

// ldflags returns release metadata flags for the Veta binary.
func ldflags(metadata releaseMetadata) string {
	return strings.Join([]string{
		"-s -w",
		"-X " + versionPackage + ".Version=" + metadata.Version,
		"-X " + versionPackage + ".Commit=" + metadata.Commit,
		"-X " + versionPackage + ".Date=" + metadata.Date,
	}, " ")
}

// archiveFiles returns source files mapped to archive-relative names.
func archiveFiles(root, rawBinary, goos string) map[string]string {
	files := map[string]string{rawBinary: binaryName(goos)}
	for _, name := range []string{"README.md", "LICENSE"} {
		path := filepath.Join(root, name)
		if _, err := os.Stat(path); err == nil {
			files[path] = name
		}
	}

	return files
}

// writeChecksums writes SHA-256 checksums for dist/ files.
func writeChecksums(distDir string) error {
	entries, err := os.ReadDir(distDir)
	if err != nil {
		return fmt.Errorf("read dist directory: %w", err)
	}

	names := make([]string, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() || entry.Name() == checksumFileName {
			continue
		}
		names = append(names, entry.Name())
	}
	sort.Strings(names)

	checksumPath := filepath.Join(distDir, checksumFileName)
	checksumFile, err := os.Create(checksumPath)
	if err != nil {
		return fmt.Errorf("create checksums file: %w", err)
	}
	defer func() {
		_ = checksumFile.Close()
	}()

	for _, name := range names {
		hash, err := fileSHA256(filepath.Join(distDir, name))
		if err != nil {
			return err
		}
		if _, err := fmt.Fprintf(checksumFile, "%s  %s\n", hash, name); err != nil {
			return fmt.Errorf("write checksum for %s: %w", name, err)
		}
	}

	return nil
}

// createZip writes a zip archive from source paths to archive names.
func createZip(target string, files map[string]string) error {
	archiveFile, err := os.Create(target)
	if err != nil {
		return fmt.Errorf("create zip archive: %w", err)
	}
	defer func() {
		_ = archiveFile.Close()
	}()

	archive := zip.NewWriter(archiveFile)
	defer func() {
		_ = archive.Close()
	}()

	for _, source := range sortedKeys(files) {
		if err := addFileToZip(archive, source, files[source]); err != nil {
			return err
		}
	}

	return nil
}

// addFileToZip adds one file to a zip archive.
func addFileToZip(archive *zip.Writer, source, name string) error {
	info, err := os.Stat(source)
	if err != nil {
		return fmt.Errorf("stat zip source %s: %w", source, err)
	}
	header, err := zip.FileInfoHeader(info)
	if err != nil {
		return fmt.Errorf("create zip header %s: %w", name, err)
	}
	header.Name = name
	header.Method = zip.Deflate

	writer, err := archive.CreateHeader(header)
	if err != nil {
		return fmt.Errorf("create zip entry %s: %w", name, err)
	}

	return copyFileContent(writer, source)
}

// createTarGz writes a tar.gz archive from source paths to archive names.
func createTarGz(target string, files map[string]string) error {
	archiveFile, err := os.Create(target)
	if err != nil {
		return fmt.Errorf("create tar.gz archive: %w", err)
	}
	defer func() {
		_ = archiveFile.Close()
	}()

	gzipWriter := gzip.NewWriter(archiveFile)
	defer func() {
		_ = gzipWriter.Close()
	}()

	tarWriter := tar.NewWriter(gzipWriter)
	defer func() {
		_ = tarWriter.Close()
	}()

	for _, source := range sortedKeys(files) {
		if err := addFileToTar(tarWriter, source, files[source]); err != nil {
			return err
		}
	}

	return nil
}

// addFileToTar adds one file to a tar archive.
func addFileToTar(archive *tar.Writer, source, name string) error {
	info, err := os.Stat(source)
	if err != nil {
		return fmt.Errorf("stat tar source %s: %w", source, err)
	}
	header, err := tar.FileInfoHeader(info, info.Name())
	if err != nil {
		return fmt.Errorf("create tar header %s: %w", name, err)
	}
	header.Name = name
	if err := archive.WriteHeader(header); err != nil {
		return fmt.Errorf("write tar header %s: %w", name, err)
	}

	return copyFileContent(archive, source)
}

// copyFileContent writes source file bytes into writer.
func copyFileContent(writer io.Writer, source string) error {
	file, err := os.Open(source)
	if err != nil {
		return fmt.Errorf("open %s: %w", source, err)
	}
	defer func() {
		_ = file.Close()
	}()

	if _, err := io.Copy(writer, file); err != nil {
		return fmt.Errorf("copy %s: %w", source, err)
	}

	return nil
}

// fileSHA256 returns the SHA-256 hex digest for path.
func fileSHA256(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", fmt.Errorf("open %s for hashing: %w", path, err)
	}
	defer func() {
		_ = file.Close()
	}()

	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", fmt.Errorf("hash %s: %w", path, err)
	}

	return hex.EncodeToString(hash.Sum(nil)), nil
}

// sortedKeys returns the sorted keys of values.
func sortedKeys(values map[string]string) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	return keys
}

// archiveName returns the release archive filename for target.
func archiveName(target target) string {
	return fmt.Sprintf("%s_%s_%s.%s", projectName, target.OS, target.Arch, archiveFormat(target.OS))
}

// archiveFormat returns the archive format used for an OS.
func archiveFormat(goos string) string {
	if goos == "windows" {
		return "zip"
	}

	return "tar.gz"
}

// binaryName returns the binary filename used inside archives.
func binaryName(goos string) string {
	if goos == "windows" {
		return projectName + ".exe"
	}

	return projectName
}

// normalizeVersion returns version without a leading v prefix.
func normalizeVersion(version string) string {
	version = strings.TrimSpace(strings.ToLower(version))
	version = strings.TrimPrefix(version, "refs/tags/")
	return strings.TrimPrefix(version, "v")
}

// tagLike returns value only when it looks like a release tag.
func tagLike(value string) string {
	value = strings.TrimSpace(value)
	if strings.HasPrefix(value, "v") {
		return value
	}

	return ""
}

// firstNonEmpty returns the first non-empty trimmed value.
func firstNonEmpty(values ...string) string {
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" {
			return value
		}
	}

	return ""
}

// shortCommit returns a short git commit hash when value is long enough.
func shortCommit(value string) string {
	value = strings.TrimSpace(value)
	if len(value) > 12 {
		return value[:12]
	}

	return value
}

// gitOutput returns trimmed stdout from a git command or an empty string.
func gitOutput(ctx context.Context, root string, args ...string) string {
	command := exec.CommandContext(ctx, "git", args...)
	command.Dir = root
	output, err := command.Output()
	if err != nil {
		return ""
	}

	return strings.TrimSpace(string(output))
}
