package config

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"path"
	"path/filepath"
	"slices"
	"strings"

	"gopkg.in/yaml.v3"
)

// FileName is the default Veta configuration file name.
const FileName = "veta.yaml"

var fileNames = []string{"veta.yaml", "veta.yml", ".veta.yaml", ".veta.yml"}

// DefaultBuildOutput is the default build output directory.
const DefaultBuildOutput = "dist"

// Config contains Veta's tool behavior settings.
type Config struct {
	Build       Build       `yaml:"build"`
	HTML        HTML        `yaml:"html"`
	Theme       Theme       `yaml:"theme"`
	TailwindCSS TailwindCSS `yaml:"tailwindcss"`
}

// Build contains settings for the site build workflow.
type Build struct {
	Clean  bool   `yaml:"clean"`
	Debug  bool   `yaml:"debug"`
	Output string `yaml:"output"`
}

// HTML contains generated HTML output settings.
type HTML struct {
	Minify bool `yaml:"minify"`
}

// Theme contains theme resolution settings.
type Theme struct {
	Source string `yaml:"source"`
}

// Enabled reports whether a theme source was configured.
func (theme Theme) Enabled() bool {
	return strings.TrimSpace(theme.Source) != ""
}

// TailwindCSS contains Tailwind CSS wrapper settings.
type TailwindCSS struct {
	Stylesheet string `yaml:"stylesheet"`
	Minify     bool   `yaml:"minify"`
}

// Enabled reports whether Tailwind CSS should run.
func (tailwind TailwindCSS) Enabled() bool {
	return strings.TrimSpace(tailwind.Stylesheet) != ""
}

// Default returns Veta's default tool configuration.
func Default() Config {
	return Config{Build: Build{Output: DefaultBuildOutput}}
}

// FileNames returns supported Veta configuration file names in priority order.
func FileNames() []string {
	return append([]string(nil), fileNames...)
}

// Load reads the first supported Veta configuration file from files. Missing
// configuration returns Default.
func Load(files fs.FS) (Config, error) {
	if files == nil {
		return Config{}, ErrFSRequired
	}

	for _, name := range fileNames {
		config, found, err := loadExistingFile(files, name)
		if err != nil {
			return Config{}, err
		}
		if found {
			return config, nil
		}
	}

	return Default(), nil
}

// LoadFile reads a Veta configuration file from files. Missing configuration
// returns Default.
func LoadFile(files fs.FS, name string) (Config, error) {
	if files == nil {
		return Config{}, ErrFSRequired
	}

	config, found, err := loadExistingFile(files, name)
	if err != nil {
		return Config{}, err
	}
	if !found {
		return Default(), nil
	}

	return config, nil
}

// LoadRequiredFile reads a required Veta configuration file from files.
func LoadRequiredFile(files fs.FS, name string) (Config, error) {
	if files == nil {
		return Config{}, ErrFSRequired
	}

	config, found, err := loadExistingFile(files, name)
	if err != nil {
		return Config{}, err
	}
	if !found {
		return Config{}, fmt.Errorf("read configuration %s: %w", name, fs.ErrNotExist)
	}

	return config, nil
}

func loadExistingFile(files fs.FS, name string) (Config, bool, error) {
	cleanName, err := cleanConfigPath(name)
	if err != nil {
		return Config{}, false, err
	}

	content, err := fs.ReadFile(files, cleanName)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return Config{}, false, nil
		}

		return Config{}, false, fmt.Errorf("read configuration %s: %w", cleanName, err)
	}

	config, err := Parse(content)
	if err != nil {
		return Config{}, false, fmt.Errorf("parse configuration %s: %w", cleanName, err)
	}

	return config, true, nil
}

// Parse parses Veta configuration from YAML bytes.
func Parse(content []byte) (Config, error) {
	if len(bytes.TrimSpace(content)) == 0 {
		return Default(), nil
	}

	config := Default()
	decoder := yaml.NewDecoder(bytes.NewReader(content))
	decoder.KnownFields(true)
	if err := decoder.Decode(&config); err != nil {
		return Config{}, fmt.Errorf("%w: decode yaml: %w", ErrInvalid, err)
	}

	var extra any
	if err := decoder.Decode(&extra); !errors.Is(err, io.EOF) {
		if err == nil {
			return Config{}, fmt.Errorf("%w: multiple yaml documents are not supported", ErrInvalid)
		}

		return Config{}, fmt.Errorf("%w: decode yaml: %w", ErrInvalid, err)
	}

	return normalize(config)
}

func normalize(config Config) (Config, error) {
	config.Build.Output = strings.TrimSpace(config.Build.Output)
	if config.Build.Output == "" {
		config.Build.Output = DefaultBuildOutput
	}
	config.Theme.Source = strings.TrimSpace(config.Theme.Source)
	config.TailwindCSS.Stylesheet = strings.TrimSpace(config.TailwindCSS.Stylesheet)

	if err := validateBuild(config.Build); err != nil {
		return Config{}, err
	}
	if err := validateTheme(config.Theme); err != nil {
		return Config{}, err
	}

	if err := validateTailwindCSS(config.TailwindCSS); err != nil {
		return Config{}, err
	}

	return config, nil
}

// validateBuild checks build configuration values.
func validateBuild(build Build) error {
	if strings.ContainsRune(build.Output, 0) {
		return fmt.Errorf("%w: build.output cannot contain NUL", ErrInvalid)
	}
	if err := validateProjectPath("build.output", build.Output); err != nil {
		return err
	}

	return nil
}

// validateTheme checks theme configuration values.
func validateTheme(theme Theme) error {
	if strings.ContainsRune(theme.Source, 0) {
		return fmt.Errorf("%w: theme.source cannot contain NUL", ErrInvalid)
	}

	return nil
}

func validateTailwindCSS(tailwind TailwindCSS) error {
	if tailwind.Stylesheet == "" {
		return nil
	}

	if err := validateProjectPath("tailwindcss.stylesheet", tailwind.Stylesheet); err != nil {
		return err
	}

	return nil
}

func validateProjectPath(field, value string) error {
	if _, err := cleanConfigPath(value); err != nil {
		return fmt.Errorf("%w: %s must be a relative project path: %w", ErrInvalid, field, err)
	}

	return nil
}

func cleanConfigPath(name string) (string, error) {
	rawName := strings.TrimSpace(name)
	if rawName == "" || strings.ContainsRune(rawName, 0) || filepath.VolumeName(rawName) != "" ||
		hasWindowsVolumeName(rawName) ||
		filepath.IsAbs(rawName) {
		return "", ErrPathInvalid
	}

	rawName = strings.ReplaceAll(rawName, "\\", "/")
	if path.IsAbs(rawName) {
		return "", ErrPathInvalid
	}

	if slices.Contains(strings.Split(rawName, "/"), "..") {
		return "", ErrPathInvalid
	}

	cleanName := path.Clean(rawName)
	if cleanName == "." || !fs.ValidPath(cleanName) {
		return "", ErrPathInvalid
	}

	return cleanName, nil
}

func hasWindowsVolumeName(name string) bool {
	return len(name) >= 2 && name[1] == ':' &&
		('A' <= name[0] && name[0] <= 'Z' || 'a' <= name[0] && name[0] <= 'z')
}
