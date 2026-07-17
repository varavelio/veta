package js

import (
	"fmt"

	"github.com/dop251/goja"
	"github.com/varavelio/veta/internal/parsecontent"
)

// newParseAPI returns explicit text parsers exposed through context.parse.
func (r *Runner) newParseAPI(vm *goja.Runtime, runtime Runtime) (*goja.Object, error) {
	api := &parseAPI{
		componentRenderer: r.componentRenderer,
		markdownRenderer:  r.markdownRenderer,
		runtime:           runtime,
		vm:                vm,
	}
	parse := vm.NewObject()
	for name, value := range (Runtime{
		"json":             api.json,
		"markdown":         api.markdown,
		"renderComponents": api.renderComponents,
		"toml":             api.toml,
		"yaml":             api.yaml,
	}) {
		if err := parse.Set(name, value); err != nil {
			return nil, fmt.Errorf("set %s.parse.%s: %w", runtimeObjectName, name, err)
		}
	}

	return parse, nil
}

type parseAPI struct {
	componentRenderer ComponentRenderer
	markdownRenderer  MarkdownRenderer
	runtime           Runtime
	vm                *goja.Runtime
}

func (api *parseAPI) json(call goja.FunctionCall) goja.Value {
	content, err := requiredStringArgument(call.Argument(0), "parse.json content")
	if err != nil {
		panic(api.vm.NewGoError(err))
	}

	value, err := parsecontent.JSON(content)
	if err != nil {
		panic(api.vm.NewGoError(fmt.Errorf("parse json: %w", err)))
	}

	return api.vm.ToValue(value)
}

func (api *parseAPI) yaml(call goja.FunctionCall) goja.Value {
	content, err := requiredStringArgument(call.Argument(0), "parse.yaml content")
	if err != nil {
		panic(api.vm.NewGoError(err))
	}

	value, err := parsecontent.YAML(content)
	if err != nil {
		panic(api.vm.NewGoError(fmt.Errorf("parse yaml: %w", err)))
	}

	return api.vm.ToValue(value)
}

func (api *parseAPI) toml(call goja.FunctionCall) goja.Value {
	content, err := requiredStringArgument(call.Argument(0), "parse.toml content")
	if err != nil {
		panic(api.vm.NewGoError(err))
	}

	value, err := parsecontent.TOML(content)
	if err != nil {
		panic(api.vm.NewGoError(fmt.Errorf("parse toml: %w", err)))
	}

	return api.vm.ToValue(value)
}

func (api *parseAPI) markdown(call goja.FunctionCall) goja.Value {
	content, err := requiredStringArgument(call.Argument(0), "parse.markdown content")
	if err != nil {
		panic(api.vm.NewGoError(err))
	}

	document, err := parsecontent.Markdown(content)
	if err != nil {
		panic(api.vm.NewGoError(fmt.Errorf("parse markdown: %w", err)))
	}
	if api.markdownRenderer == nil {
		panic(api.vm.NewGoError(ErrMarkdownRendererRequired))
	}

	html, err := api.markdownRenderer.Render(document.Content)
	if err != nil {
		panic(api.vm.NewGoError(fmt.Errorf("render parsed markdown: %w", err)))
	}

	return api.vm.ToValue(map[string]any{
		"content":     document.Content,
		"frontmatter": document.Frontmatter,
		"html":        html,
	})
}

// renderComponents expands registered component tags in a string without
// applying any other content transformation.
func (api *parseAPI) renderComponents(call goja.FunctionCall) goja.Value {
	content, err := requiredStringArgument(call.Argument(0), "parse.renderComponents content")
	if err != nil {
		panic(api.vm.NewGoError(err))
	}
	if api.componentRenderer == nil {
		panic(api.vm.NewGoError(ErrComponentRendererRequired))
	}

	output, err := api.componentRenderer.Render(content, componentRenderContext(api.runtime))
	if err != nil {
		panic(api.vm.NewGoError(fmt.Errorf("render components: %w", err)))
	}

	return api.vm.ToValue(output)
}

// componentRenderContext returns the root values available to component
// templates for the current JavaScript execution.
func componentRenderContext(runtime Runtime) map[string]any {
	return map[string]any{
		"data":  runtime["data"],
		"page":  runtime["page"],
		"pages": runtime["pages"],
		"props": runtime["props"],
	}
}
