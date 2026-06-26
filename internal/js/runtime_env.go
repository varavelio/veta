package js

import (
	"fmt"
	"maps"
	"os"
	"strings"

	"github.com/dop251/goja"
)

// Environment contains string environment variables exposed through Veta.env.
type Environment map[string]string

// defaultEnvironment returns a snapshot of the process environment.
func defaultEnvironment() Environment {
	environment := Environment{}
	for _, entry := range os.Environ() {
		name, value, ok := strings.Cut(entry, "=")
		if !ok {
			continue
		}

		environment[name] = value
	}

	return environment
}

// newEnvironmentObject converts configured environment variables into a Goja
// object.
func (r *Runner) newEnvironmentObject(vm *goja.Runtime) (*goja.Object, error) {
	environment := vm.NewObject()
	for name, value := range r.environmentSnapshot() {
		if err := environment.Set(name, value); err != nil {
			return nil, fmt.Errorf("set %s.env.%s: %w", GlobalName, name, err)
		}
	}

	return environment, nil
}

// environmentSnapshot returns a copy of the environment configured on the
// runner.
func (r *Runner) environmentSnapshot() Environment {
	if r == nil || r.environment == nil {
		return defaultEnvironment()
	}

	return cloneEnvironment(r.environment)
}

// cloneEnvironment copies an environment map so executions cannot mutate runner
// configuration accidentally.
func cloneEnvironment(environment Environment) Environment {
	clone := make(Environment, len(environment))
	maps.Copy(clone, environment)

	return clone
}
