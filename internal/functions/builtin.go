package functions

import (
	"fmt"
	"regexp"
	"strings"
)

// LoadDataRequest describes one template load_data call.
type LoadDataRequest struct {
	Path string
	URL  string
}

// LoadDataFunc loads data for the load_data template function.
type LoadDataFunc func(LoadDataRequest) (any, error)

func loadDataFunction(loadData LoadDataFunc) Func {
	return func(_ Context, args ...any) (any, error) {
		if loadData == nil {
			return nil, ErrLoadDataRequired
		}
		if len(args) != 1 {
			return nil, fmt.Errorf("load_data expects 1 argument")
		}

		source := fmt.Sprint(args[0])
		request := LoadDataRequest{Path: source}
		if isRemoteURL(source) {
			request.Path = ""
			request.URL = source
		}

		return loadData(request)
	}
}

func regexReplaceFunction(_ Context, args ...any) (any, error) {
	if len(args) != 3 {
		return nil, fmt.Errorf("regex_replace expects 3 arguments")
	}

	expression, err := regexp.Compile(fmt.Sprint(args[1]))
	if err != nil {
		return "", fmt.Errorf("compile regex_replace pattern: %w", err)
	}

	return expression.ReplaceAllString(fmt.Sprint(args[0]), fmt.Sprint(args[2])), nil
}

func isRemoteURL(source string) bool {
	return strings.HasPrefix(source, "http://") || strings.HasPrefix(source, "https://")
}
