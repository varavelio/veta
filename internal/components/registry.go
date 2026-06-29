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
}

// scan discovers component templates and resolves tag conflicts.
func scan(files fs.FS) (map[string]Component, []Conflict, error) {
	candidates := []componentCandidate{}
	if err := fs.WalkDir(files, DirName, func(name string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if name == DirName {
			return nil
		}
		if entry.IsDir() {
			if componentFileNameIgnored(entry.Name()) {
				return fs.SkipDir
			}

			return nil
		}
		if componentPathIgnored(name) {
			return nil
		}

		candidate, err := componentCandidateFor(name)
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
		if leftComponent.Tag == rightComponent.Tag {
			leftHasExtension := path.Ext(leftComponent.Path) != ""
			rightHasExtension := path.Ext(rightComponent.Path) != ""
			if leftHasExtension != rightHasExtension {
				return leftHasExtension
			}
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
func componentCandidateFor(name string) (componentCandidate, error) {
	relativeName := strings.TrimPrefix(name, DirName+"/")
	stem := fileStem(relativeName)
	tag := strings.ReplaceAll(stem, "/", "-")
	if err := validateTagName(tag); err != nil {
		return componentCandidate{}, fmt.Errorf("%w: %s", err, name)
	}

	return componentCandidate{
		component: Component{
			Depth:    strings.Count(stem, "/"),
			Path:     name,
			Tag:      tag,
			Template: name,
		},
	}, nil
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

// fileStem returns the file name without its final extension.
func fileStem(name string) string {
	return strings.TrimSuffix(name, path.Ext(name))
}

// componentPathIgnored reports whether a path contains an ignored file segment.
func componentPathIgnored(name string) bool {
	return slices.ContainsFunc(strings.Split(name, "/"), componentFileNameIgnored)
}

// componentFileNameIgnored reports whether a component file should be skipped.
func componentFileNameIgnored(name string) bool {
	lowerName := strings.ToLower(name)

	return strings.HasPrefix(name, ".") ||
		strings.HasSuffix(name, "~") ||
		strings.HasSuffix(lowerName, ".tmp")
}
