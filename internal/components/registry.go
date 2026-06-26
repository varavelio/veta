package components

import (
	"errors"
	"fmt"
	"io/fs"
	"path"
	"slices"
	"sort"
	"strings"
)

type componentCandidate struct {
	component Component
	extension string
}

// scan discovers component templates and resolves tag conflicts.
func scan(files fs.FS, extensions []string) (map[string]Component, []Conflict, error) {
	candidates := []componentCandidate{}
	if err := fs.WalkDir(files, DirName, func(name string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if name == DirName || entry.IsDir() {
			return nil
		}

		candidate, err := componentCandidateFor(name, extensions)
		if err != nil {
			return err
		}

		candidates = append(candidates, candidate)
		return nil
	}); err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return map[string]Component{}, nil, nil
		}

		return nil, nil, fmt.Errorf("scan components directory %s: %w", DirName, err)
	}

	sort.Slice(candidates, func(left, right int) bool {
		leftComponent := candidates[left].component
		rightComponent := candidates[right].component
		if leftComponent.Depth != rightComponent.Depth {
			return leftComponent.Depth < rightComponent.Depth
		}

		return leftComponent.Path < rightComponent.Path
	})

	components := map[string]Component{}
	conflicts := []Conflict{}
	for _, candidate := range candidates {
		component := candidate.component
		if previous, exists := components[component.Tag]; exists {
			conflicts = append(conflicts, Conflict{
				Ignored: component.Path,
				Tag:     component.Tag,
				Winner:  previous.Path,
			})
			continue
		}

		components[component.Tag] = component
	}

	return components, conflicts, nil
}

// componentCandidateFor converts one file path into a component candidate.
func componentCandidateFor(name string, extensions []string) (componentCandidate, error) {
	extension := strings.ToLower(path.Ext(name))
	if !hasExtension(extensions, extension) {
		return componentCandidate{}, fmt.Errorf("%w: %s", ErrFormatUnsupported, name)
	}

	relativeName := strings.TrimPrefix(name, DirName+"/")
	stem := strings.TrimSuffix(relativeName, path.Ext(relativeName))
	tag := strings.ReplaceAll(stem, "/", "-")
	if err := validateTagName(tag); err != nil {
		return componentCandidate{}, fmt.Errorf("%w: %s", err, name)
	}

	return componentCandidate{
		component: Component{
			Depth:    strings.Count(stem, "/"),
			Path:     name,
			Tag:      tag,
			Template: path.Join(DirName, stem),
		},
		extension: extension,
	}, nil
}

// normalizeExtensions validates component template extensions.
func normalizeExtensions(extensions []string) ([]string, error) {
	if len(extensions) == 0 {
		return nil, fmt.Errorf(
			"%w: at least one component extension is required",
			ErrFormatUnsupported,
		)
	}

	normalized := make([]string, 0, len(extensions))
	for _, extension := range extensions {
		extension = strings.TrimSpace(strings.ToLower(extension))
		if extension == "" || strings.ContainsAny(extension, "/\\") ||
			strings.ContainsRune(extension, 0) {
			return nil, fmt.Errorf(
				"%w: invalid component extension %q",
				ErrFormatUnsupported,
				extension,
			)
		}
		if !strings.HasPrefix(extension, ".") {
			extension = "." + extension
		}
		if extension == "." {
			return nil, fmt.Errorf(
				"%w: invalid component extension %q",
				ErrFormatUnsupported,
				extension,
			)
		}

		normalized = append(normalized, extension)
	}

	return normalized, nil
}

// hasExtension reports whether extension is allowed.
func hasExtension(extensions []string, extension string) bool {
	return slices.Contains(extensions, extension)
}

// validateTagName checks that a component tag follows Veta's tag syntax.
func validateTagName(tag string) error {
	if tag == "" || tag[0] < 'a' || tag[0] > 'z' || strings.Contains(tag, "--") {
		return ErrComponentNameInvalid
	}

	for _, char := range tag {
		if 'a' <= char && char <= 'z' || '0' <= char && char <= '9' || char == '-' {
			continue
		}

		return ErrComponentNameInvalid
	}

	return nil
}
