package functions

import (
	"fmt"
	"path"
	"path/filepath"
	"strings"
)

func urlFunction(context Context, args ...any) (any, error) {
	if len(args) != 1 {
		return nil, fmt.Errorf("url expects 1 argument")
	}

	return relativeURL(fmt.Sprint(args[0]), pagePermalink(context)), nil
}

func relativeURL(target, currentPermalink string) string {
	if target == "" || !strings.HasPrefix(target, "/") || strings.HasPrefix(target, "//") {
		return target
	}

	pathPart, suffix := splitURLSuffix(target)
	targetPath := strings.TrimPrefix(path.Clean(pathPart), "/")
	if targetPath == "" {
		targetPath = "."
	}
	if strings.HasSuffix(pathPart, "/") && targetPath != "" {
		targetPath += "/"
	}

	fromDir := outputDirFromPermalink(currentPermalink)
	result, err := filepath.Rel(filepath.FromSlash(fromDir), filepath.FromSlash(targetPath))
	if err != nil || result == "" {
		return "." + suffix
	}
	result = filepath.ToSlash(result)
	if strings.HasSuffix(pathPart, "/") && !strings.HasSuffix(result, "/") && result != "." {
		result += "/"
	}

	return result + suffix
}

func splitURLSuffix(value string) (string, string) {
	queryIndex := strings.IndexAny(value, "?#")
	if queryIndex == -1 {
		return value, ""
	}

	return value[:queryIndex], value[queryIndex:]
}

func outputDirFromPermalink(permalink string) string {
	cleanPermalink := path.Clean("/" + strings.TrimPrefix(permalink, "/"))
	if cleanPermalink == "/" {
		return "."
	}
	if strings.HasSuffix(permalink, "/") {
		return strings.TrimPrefix(cleanPermalink, "/")
	}

	dir := path.Dir(strings.TrimPrefix(cleanPermalink, "/"))
	if dir == "/" || dir == "." {
		return "."
	}

	return dir
}

func pagePermalink(context Context) string {
	page, _ := context["page"].(map[string]any)
	permalink, _ := page["permalink"].(string)
	return permalink
}
