package js

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/dop251/goja"
)

// newHTTPClientAPI returns the synchronous HTTP client exposed through
// Veta.httpClient.
func (r *Runner) newHTTPClientAPI(vm *goja.Runtime) (*goja.Object, error) {
	api := &httpClientAPI{
		defaultTimeout: r.defaultHTTPTimeout(),
		vm:             vm,
	}

	httpClient := vm.NewObject()
	for name, value := range map[string]any{
		"delete":  api.method("DELETE"),
		"get":     api.method("GET"),
		"head":    api.method("HEAD"),
		"patch":   api.method("PATCH"),
		"post":    api.method("POST"),
		"put":     api.method("PUT"),
		"request": api.request,
	} {
		if err := httpClient.Set(name, value); err != nil {
			return nil, fmt.Errorf("set %s.httpClient.%s: %w", GlobalName, name, err)
		}
	}

	return httpClient, nil
}

// defaultHTTPTimeout returns the timeout used when a request does not override
// it.
func (r *Runner) defaultHTTPTimeout() time.Duration {
	if r == nil || r.httpTimeout <= 0 {
		return defaultHTTPTimeout
	}

	return r.httpTimeout
}

// httpClientAPI owns synchronous HTTP callbacks exposed to JavaScript.
type httpClientAPI struct {
	defaultTimeout time.Duration
	vm             *goja.Runtime
}

// method returns a shortcut method callback such as get or post.
func (api *httpClientAPI) method(method string) func(goja.FunctionCall) goja.Value {
	return func(call goja.FunctionCall) goja.Value {
		return api.do(method, call.Argument(0), call.Argument(1))
	}
}

// request executes an HTTP request with an explicit method argument.
func (api *httpClientAPI) request(call goja.FunctionCall) goja.Value {
	method, err := requiredStringArgument(call.Argument(0), "Veta.httpClient.request method")
	if err != nil {
		panic(api.vm.NewGoError(err))
	}

	return api.do(method, call.Argument(1), call.Argument(2))
}

// do executes one synchronous HTTP request and returns a JavaScript object.
func (api *httpClientAPI) do(method string, rawURL, rawOptions goja.Value) goja.Value {
	requestURL, err := requiredStringArgument(rawURL, "Veta.httpClient URL")
	if err != nil {
		panic(api.vm.NewGoError(err))
	}

	response, err := api.fetch(method, requestURL, rawOptions)
	if err != nil {
		panic(api.vm.NewGoError(err))
	}

	return api.vm.ToValue(response)
}

// fetch executes one HTTP request and returns the Veta response shape.
func (api *httpClientAPI) fetch(
	method, rawURL string,
	rawOptions goja.Value,
) (map[string]any, error) {
	method, err := cleanHTTPMethod(method)
	if err != nil {
		return nil, err
	}

	requestURL, err := cleanHTTPURL(rawURL)
	if err != nil {
		return nil, err
	}

	options, err := api.requestOptions(rawOptions)
	if err != nil {
		return nil, err
	}

	ctx := context.Background()
	request, err := http.NewRequestWithContext(
		ctx,
		method,
		requestURL,
		bytes.NewReader(options.body),
	)
	if err != nil {
		return nil, fmt.Errorf("create http request: %w", err)
	}

	request.Header = options.headers
	client := &http.Client{Timeout: options.timeout}
	response, err := client.Do(request)
	if err != nil {
		return nil, fmt.Errorf("execute http request: %w", err)
	}
	defer func() {
		_ = response.Body.Close()
	}()

	body, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, fmt.Errorf("read http response body: %w", err)
	}

	return map[string]any{
		"body":       string(body),
		"headers":    mapHTTPHeaders(response.Header),
		"ok":         200 <= response.StatusCode && response.StatusCode <= 299,
		"status":     response.StatusCode,
		"statusText": http.StatusText(response.StatusCode),
		"url":        response.Request.URL.String(),
	}, nil
}

// httpRequestOptions contains the normalized options for one HTTP request.
type httpRequestOptions struct {
	body    []byte
	headers http.Header
	timeout time.Duration
}

// requestOptions converts a JavaScript options object into request options.
func (api *httpClientAPI) requestOptions(value goja.Value) (httpRequestOptions, error) {
	options := httpRequestOptions{
		headers: make(http.Header),
		timeout: api.defaultTimeout,
	}

	if isJavaScriptNullish(value) {
		return options, nil
	}

	if _, ok := value.Export().(map[string]any); !ok {
		return httpRequestOptions{}, ErrHTTPOptionsUnsupported
	}

	object := value.ToObject(api.vm)
	headers, err := httpHeaders(object.Get("headers"))
	if err != nil {
		return httpRequestOptions{}, err
	}
	options.headers = headers

	bodyValue := object.Get("body")
	jsonValue := object.Get("json")
	if !isJavaScriptNullish(bodyValue) && !isJavaScriptNullish(jsonValue) {
		return httpRequestOptions{}, ErrHTTPBodyConflict
	}

	if !isJavaScriptNullish(bodyValue) {
		body, err := httpBody(bodyValue)
		if err != nil {
			return httpRequestOptions{}, err
		}
		options.body = body
	}

	if !isJavaScriptNullish(jsonValue) {
		body, err := json.Marshal(jsonValue.Export())
		if err != nil {
			return httpRequestOptions{}, fmt.Errorf("serialize http json body: %w", err)
		}
		options.body = body
		if options.headers.Get("Content-Type") == "" {
			options.headers.Set("Content-Type", "application/json")
		}
	}

	timeoutValue := object.Get("timeoutMs")
	if !isJavaScriptNullish(timeoutValue) {
		timeout, err := positiveMilliseconds(timeoutValue, "http timeoutMs")
		if err != nil {
			return httpRequestOptions{}, fmt.Errorf("%w: %w", ErrHTTPTimeoutInvalid, err)
		}
		options.timeout = timeout
	}

	return options, nil
}

// httpBody converts a JavaScript body value into raw request bytes.
func httpBody(value goja.Value) ([]byte, error) {
	if isJavaScriptNullish(value) {
		return nil, nil
	}

	exported := value.Export()
	body, ok := exported.(string)
	if !ok {
		return nil, ErrHTTPBodyUnsupported
	}

	return []byte(body), nil
}

// httpHeaders converts a JavaScript headers object into http.Header.
func httpHeaders(value goja.Value) (http.Header, error) {
	headers := make(http.Header)
	if isJavaScriptNullish(value) {
		return headers, nil
	}

	exported, ok := value.Export().(map[string]any)
	if !ok {
		return nil, ErrHTTPHeadersUnsupported
	}

	for name, rawValue := range exported {
		name = strings.TrimSpace(name)
		if name == "" {
			return nil, fmt.Errorf("http header name cannot be empty")
		}

		for _, headerValue := range httpHeaderValues(rawValue) {
			headers.Add(name, headerValue)
		}
	}

	return headers, nil
}

// httpHeaderValues converts a JavaScript header value into one or more strings.
func httpHeaderValues(value any) []string {
	switch typedValue := value.(type) {
	case []any:
		values := make([]string, 0, len(typedValue))
		for _, item := range typedValue {
			values = append(values, fmt.Sprint(item))
		}
		return values
	case []string:
		return typedValue
	default:
		return []string{fmt.Sprint(typedValue)}
	}
}

// cleanHTTPMethod normalizes and validates an HTTP method.
func cleanHTTPMethod(method string) (string, error) {
	method = strings.ToUpper(strings.TrimSpace(method))
	if method == "" || strings.ContainsAny(method, " \t\r\n") {
		return "", ErrHTTPMethodInvalid
	}

	return method, nil
}

// cleanHTTPURL validates that a URL is absolute and uses HTTP or HTTPS.
func cleanHTTPURL(rawURL string) (string, error) {
	parsedURL, err := url.Parse(strings.TrimSpace(rawURL))
	if err != nil {
		return "", fmt.Errorf("parse http url: %w", err)
	}

	if parsedURL.Host == "" || (parsedURL.Scheme != "http" && parsedURL.Scheme != "https") {
		return "", ErrHTTPURLUnsupported
	}

	return parsedURL.String(), nil
}

// mapHTTPHeaders converts HTTP response headers into a JavaScript-friendly map.
func mapHTTPHeaders(headers http.Header) map[string][]string {
	mapped := make(map[string][]string, len(headers))
	for name, values := range headers {
		mapped[name] = append([]string(nil), values...)
	}

	return mapped
}
