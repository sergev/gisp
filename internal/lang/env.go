package lang

import "fmt"

// Env implements a lexical environment chain.
type Env struct {
	parent *Env
	values map[string]Value
}

// NewEnv creates an environment with optional parent.
func NewEnv(parent *Env) *Env {
	return &Env{
		parent: parent,
		values: make(map[string]Value),
	}
}

// Define binds name to value in current frame.
func (e *Env) Define(name string, val Value) {
	e.values[name] = val
}

// Set updates an existing binding, searching parents if needed.
func (e *Env) Set(name string, val Value) error {
	if _, ok := e.values[name]; ok {
		e.values[name] = val
		return nil
	}
	if e.parent != nil {
		return e.parent.Set(name, val)
	}
	return fmt.Errorf("unbound variable: %s", name)
}

// Get retrieves a binding, searching parents if necessary.
func (e *Env) Get(name string) (Value, error) {
	if val, ok := e.values[name]; ok {
		return val, nil
	}
	if e.parent != nil {
		return e.parent.Get(name)
	}
	return Value{}, fmt.Errorf("unbound variable: %s", name)
}

// Parent returns the parent environment.
func (e *Env) Parent() *Env {
	return e.parent
}
