package js

import (
	"fmt"

	"github.com/dop251/goja"
	"github.com/varavelio/veta/internal/parsecontent"
)

// newParseAPI returns explicit text parsers exposed through context.parse.
func (r *Runner) newParseAPI(vm *goja.Runtime) (*goja.Object, error) {
	api := &parseAPI{vm: vm}
	parse := vm.NewObject()
	for name, value := range (Runtime{
		"json":     api.json,
		"markdown": api.markdown,
		"toml":     api.toml,
		"yaml":     api.yaml,
	}) {
		if err := parse.Set(name, value); err != nil {
			return nil, fmt.Errorf("set %s.parse.%s: %w", runtimeObjectName, name, err)
		}
	}

	return parse, nil
}

type parseAPI struct {
	vm *goja.Runtime
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

	value, err := parsecontent.MarkdownMap(content)
	if err != nil {
		panic(api.vm.NewGoError(fmt.Errorf("parse markdown: %w", err)))
	}

	return api.vm.ToValue(value)
}
