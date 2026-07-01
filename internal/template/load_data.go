package template

import (
	"fmt"

	"github.com/flosch/pongo2/v7"
)

const loadDataGlobalName = "__veta_load_data"

// LoadDataRequest describes one template load_data call.
type LoadDataRequest struct {
	Path      string
	URL       string
	Format    string
	TimeoutMs int
}

// LoadDataFunc loads data for the load_data template helper.
type LoadDataFunc func(LoadDataRequest) (any, error)

type loadDataTagNode struct {
	path      pongo2.IEvaluator
	url       pongo2.IEvaluator
	format    pongo2.IEvaluator
	timeoutMs pongo2.IEvaluator
	target    string
}

// WithLoadData registers the load_data function and tag for this renderer.
func WithLoadData(loader LoadDataFunc) Option {
	return func(config *rendererConfig) error {
		if loader == nil {
			return fmt.Errorf("%w: load_data", ErrGlobalNameInvalid)
		}

		config.globals[loadDataGlobalName] = loader
		config.globals["load_data"] = positionalLoadData(loader)
		return nil
	}
}

func positionalLoadData(loader LoadDataFunc) func(string, ...string) (any, error) {
	return func(source string, formats ...string) (any, error) {
		if len(formats) > 1 {
			return nil, fmt.Errorf("load_data accepts at most one format argument")
		}

		format := ""
		if len(formats) == 1 {
			format = formats[0]
		}

		request := LoadDataRequest{Path: source, Format: format}
		if isTemplateRemoteURL(source) {
			request.Path = ""
			request.URL = source
		}

		return loader(request)
	}
}

func parseLoadDataTag(
	doc *pongo2.Parser,
	_ *pongo2.Token,
	arguments *pongo2.Parser,
) (pongo2.INodeTag, error) {
	node := &loadDataTagNode{}

	for arguments.Remaining() > 0 {
		if arguments.Match(pongo2.TokenKeyword, "as") != nil {
			target := arguments.MatchType(pongo2.TokenIdentifier)
			if target == nil {
				return nil, arguments.Error("Expected identifier after 'as'.", nil)
			}
			node.target = target.Val
			break
		}

		name := arguments.MatchType(pongo2.TokenIdentifier)
		if name == nil {
			return nil, arguments.Error("Expected load_data option name.", nil)
		}
		if arguments.Match(pongo2.TokenSymbol, "=") == nil {
			return nil, arguments.Error("Expected '=' after load_data option name.", nil)
		}

		expression, err := arguments.ParseExpression()
		if err != nil {
			return nil, err
		}
		if err := node.setOption(name.Val, expression); err != nil {
			return nil, arguments.Error(err.Error(), name)
		}
	}

	if node.target == "" {
		return nil, arguments.Error("Expected 'as' in load_data tag.", nil)
	}
	if arguments.Remaining() > 0 {
		return nil, arguments.Error("Malformed load_data tag arguments.", nil)
	}

	return node, nil
}

func (node *loadDataTagNode) setOption(name string, expression pongo2.IEvaluator) error {
	switch name {
	case "path":
		if node.path != nil {
			return fmt.Errorf("load_data path was already set")
		}
		node.path = expression
	case "url":
		if node.url != nil {
			return fmt.Errorf("load_data url was already set")
		}
		node.url = expression
	case "format":
		if node.format != nil {
			return fmt.Errorf("load_data format was already set")
		}
		node.format = expression
	case "timeout_ms":
		if node.timeoutMs != nil {
			return fmt.Errorf("load_data timeout_ms was already set")
		}
		node.timeoutMs = expression
	default:
		return fmt.Errorf("unknown load_data option %q", name)
	}

	return nil
}

func (node *loadDataTagNode) Execute(ctx *pongo2.ExecutionContext, _ pongo2.TemplateWriter) error {
	loader, ok := ctx.Public[loadDataGlobalName].(LoadDataFunc)
	if !ok || loader == nil {
		return ctx.Error("load_data is not configured", nil)
	}

	request, err := node.request(ctx)
	if err != nil {
		return ctx.OrigError(err, nil)
	}
	value, err := loader(request)
	if err != nil {
		return ctx.OrigError(err, nil)
	}

	ctx.Private[node.target] = pongo2.AsValue(value)
	return nil
}

func (node *loadDataTagNode) request(ctx *pongo2.ExecutionContext) (LoadDataRequest, error) {
	pathValue, err := evaluateString(ctx, node.path)
	if err != nil {
		return LoadDataRequest{}, fmt.Errorf("load_data path: %w", err)
	}
	urlValue, err := evaluateString(ctx, node.url)
	if err != nil {
		return LoadDataRequest{}, fmt.Errorf("load_data url: %w", err)
	}
	formatValue, err := evaluateString(ctx, node.format)
	if err != nil {
		return LoadDataRequest{}, fmt.Errorf("load_data format: %w", err)
	}
	timeoutMs, err := evaluateInt(ctx, node.timeoutMs)
	if err != nil {
		return LoadDataRequest{}, fmt.Errorf("load_data timeout_ms: %w", err)
	}

	return LoadDataRequest{
		Path:      pathValue,
		URL:       urlValue,
		Format:    formatValue,
		TimeoutMs: timeoutMs,
	}, nil
}

func evaluateString(ctx *pongo2.ExecutionContext, expression pongo2.IEvaluator) (string, error) {
	if expression == nil {
		return "", nil
	}

	value, err := expression.Evaluate(ctx)
	if err != nil {
		return "", err
	}

	return value.String(), nil
}

func evaluateInt(ctx *pongo2.ExecutionContext, expression pongo2.IEvaluator) (int, error) {
	if expression == nil {
		return 0, nil
	}

	value, err := expression.Evaluate(ctx)
	if err != nil {
		return 0, err
	}

	return value.Integer(), nil
}

func isTemplateRemoteURL(source string) bool {
	return len(source) >= len("http://") && (source[:len("http://")] == "http://" ||
		len(source) >= len("https://") && source[:len("https://")] == "https://")
}
