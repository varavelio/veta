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
	version      = "v4.3.3"
)

type asset struct {
	Name       string
	SHA256     string
	Executable bool
}

var assets = []asset{
	{
		Name:   "sha256sums.txt",
		SHA256: "527b4fcd96950f9ae8f83bbbff27c61e4ff3596cb0b2eb760f9b3516de5d3c56",
	},
	{
		Name:       "tailwindcss-linux-arm64",
		SHA256:     "55fd0b241214eff3de1e8ee4f22796662f2d2e7a49bcfca7477cfd0bac398195",
		Executable: true,
	},
	{
		Name:       "tailwindcss-linux-arm64-musl",
		SHA256:     "71ea4be79c9de9827545682df3e040053fb535d37c71ed2cfdedf9385a0868e0",
		Executable: true,
	},
	{
		Name:       "tailwindcss-linux-x64",
		SHA256:     "dc61b3ac6b8c9ca874c0cc4c57b2409791a64c5540404ca5f5367360babc313a",
		Executable: true,
	},
	{
		Name:       "tailwindcss-linux-x64-musl",
		SHA256:     "a04d34ceacc8f52cbe8920ad846cdeb61d3d0021dba32db0d1f77c9d9fad7a6c",
		Executable: true,
	},
	{
		Name:       "tailwindcss-macos-arm64",
		SHA256:     "cdf646702987a743464dff4d9c60fd4480d1c1e73dd819a9a67f1078815dce9d",
		Executable: true,
	},
	{
		Name:       "tailwindcss-macos-x64",
		SHA256:     "7922e0953f2110c05976e3bf58f14e643d90427575e766b7d433f5f80cbee7e1",
		Executable: true,
	},
	{
		Name:       "tailwindcss-windows-x64.exe",
		SHA256:     "e0e260ce048014e9268f6237ff18f8ccf02cef521cbd0ae04e82c2cdf7aa3955",
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
