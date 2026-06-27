package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

const (
	baseURL      = "https://github.com/tailwindlabs/tailwindcss/releases/download"
	binDir       = "internal/tailwindcss/bin"
	fileMode     = 0o644
	execFileMode = 0o755
	timeout      = 10 * time.Minute
	version      = "v4.3.1"
)

type asset struct {
	Name       string
	SHA256     string
	Executable bool
}

var assets = []asset{
	{
		Name:   "sha256sums.txt",
		SHA256: "b44b08f3c490ea50432072a3ba6ef4ec1deda260a30a4f39cda2267a38f2252e",
	},
	{
		Name:       "tailwindcss-linux-arm64",
		SHA256:     "3d662377a86d71c43b549dc06b90db4586b4acd412bf827a3268e951661e5adf",
		Executable: true,
	},
	{
		Name:       "tailwindcss-linux-arm64-musl",
		SHA256:     "7ed72712429166d869dc8472e0cd8c61cd46e565a5bc1ba8810612bedfe61e7b",
		Executable: true,
	},
	{
		Name:       "tailwindcss-linux-x64",
		SHA256:     "2526d063ba03b71f9a3ea7d5cee14f0aec147f117f222d5adc97b1d736d45999",
		Executable: true,
	},
	{
		Name:       "tailwindcss-linux-x64-musl",
		SHA256:     "daeabe94235912b3773273053d5c8a16325af3fa513aa03b7295d6f445093cf2",
		Executable: true,
	},
	{
		Name:       "tailwindcss-macos-arm64",
		SHA256:     "a27c43626185953ee19bdace1939c7601e55da654e0b2fc4461e3e29957aa739",
		Executable: true,
	},
	{
		Name:       "tailwindcss-macos-x64",
		SHA256:     "e9e830ceb3e70b7e0775a3dd79eee8ec82c6b31270f08f2fa2857d0077045ac3",
		Executable: true,
	},
	{
		Name:       "tailwindcss-windows-x64.exe",
		SHA256:     "dc4fd46acd354d976df2a31b6425fbe865a38229a06bc005a4c59f2b3d24ab4a",
		Executable: true,
	},
}

func main() {
	if err := run(context.Background()); err != nil {
		fmt.Fprintf(os.Stderr, "download tailwindcss: %s\n", err)
		os.Exit(1)
	}
}

func run(ctx context.Context) error {
	if err := os.MkdirAll(binDir, 0o755); err != nil {
		return fmt.Errorf("create bin directory %s: %w", binDir, err)
	}

	client := &http.Client{Timeout: timeout}
	for _, asset := range assets {
		if err := ensureAsset(ctx, client, asset); err != nil {
			return err
		}
	}

	return nil
}

func ensureAsset(ctx context.Context, client *http.Client, asset asset) error {
	path := filepath.Join(binDir, asset.Name)
	if ok, err := fileHashMatches(path, asset.SHA256); err != nil {
		return err
	} else if ok {
		if asset.Executable {
			return chmodExecutable(path)
		}

		fmt.Printf("tailwindcss %s already verified\n", asset.Name)
		return nil
	}

	return downloadAsset(ctx, client, asset, path)
}

func downloadAsset(ctx context.Context, client *http.Client, asset asset, path string) error {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, assetURL(asset), nil)
	if err != nil {
		return fmt.Errorf("create request for %s: %w", asset.Name, err)
	}
	request.Header.Set("User-Agent", "veta-tailwind-downloader")

	response, err := client.Do(request)
	if err != nil {
		return fmt.Errorf("download %s: %w", asset.Name, err)
	}
	defer func() {
		_ = response.Body.Close()
	}()

	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		_, _ = io.Copy(io.Discard, response.Body)
		return fmt.Errorf("download %s: %s", asset.Name, response.Status)
	}

	tempPath := path + ".tmp"
	tempFile, err := os.OpenFile(tempPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, fileMode)
	if err != nil {
		return fmt.Errorf("create temporary file %s: %w", tempPath, err)
	}
	defer func() {
		_ = os.Remove(tempPath)
	}()

	hash := sha256.New()
	if _, err := io.Copy(io.MultiWriter(tempFile, hash), response.Body); err != nil {
		_ = tempFile.Close()
		return fmt.Errorf("write %s: %w", asset.Name, err)
	}
	if err := tempFile.Close(); err != nil {
		return fmt.Errorf("close %s: %w", tempPath, err)
	}

	actualHash := hex.EncodeToString(hash.Sum(nil))
	if !strings.EqualFold(actualHash, asset.SHA256) {
		return fmt.Errorf(
			"%s checksum mismatch: got %s want %s",
			asset.Name,
			actualHash,
			asset.SHA256,
		)
	}
	if asset.Executable {
		if err := chmodExecutable(tempPath); err != nil {
			return err
		}
	}
	if err := os.Rename(tempPath, path); err != nil {
		return fmt.Errorf("install %s: %w", asset.Name, err)
	}

	fmt.Printf("tailwindcss %s downloaded and verified\n", asset.Name)
	return nil
}

func fileHashMatches(path, wantHash string) (bool, error) {
	file, err := os.Open(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return false, nil
		}

		return false, fmt.Errorf("open %s: %w", path, err)
	}
	defer func() {
		_ = file.Close()
	}()

	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return false, fmt.Errorf("hash %s: %w", path, err)
	}

	return strings.EqualFold(hex.EncodeToString(hash.Sum(nil)), wantHash), nil
}

func chmodExecutable(path string) error {
	if runtime.GOOS == "windows" {
		return nil
	}
	if err := os.Chmod(path, execFileMode); err != nil {
		return fmt.Errorf("chmod executable %s: %w", path, err)
	}

	return nil
}

func assetURL(asset asset) string {
	return fmt.Sprintf("%s/%s/%s", baseURL, version, asset.Name)
}
