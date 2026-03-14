package plugins

import (
	"context"
	"fmt"
	"sync"

	"github.com/tetratelabs/wazero"
	"github.com/tetratelabs/wazero/api"
	"github.com/tetratelabs/wazero/imports/wasi_snapshot_preview1"
)

// Plugin represents a loaded WASM plugin.
type Plugin struct {
	Name    string `json:"name"`
	Version string `json:"version"`
	module  api.Module
}

// Runtime manages WASM plugin loading and execution.
type Runtime struct {
	rt      wazero.Runtime
	plugins map[string]*Plugin
	mu      sync.RWMutex
}

// NewRuntime creates a WASM runtime.
func NewRuntime(ctx context.Context) (*Runtime, error) {
	rt := wazero.NewRuntime(ctx)
	// Enable WASI for plugins that need stdio/filesystem
	if _, err := wasi_snapshot_preview1.Instantiate(ctx, rt); err != nil {
		rt.Close(ctx) //nolint:errcheck
		return nil, fmt.Errorf("instantiate wasi: %w", err)
	}
	return &Runtime{
		rt:      rt,
		plugins: make(map[string]*Plugin),
	}, nil
}

// Close shuts down the runtime.
func (r *Runtime) Close(ctx context.Context) error {
	return r.rt.Close(ctx)
}

// LoadPlugin compiles and instantiates a WASM plugin.
// The plugin must export "plugin_name" and "plugin_version" functions.
func (r *Runtime) LoadPlugin(ctx context.Context, name string, wasmBytes []byte) (*Plugin, error) {
	compiled, err := r.rt.CompileModule(ctx, wasmBytes)
	if err != nil {
		return nil, fmt.Errorf("compile plugin %q: %w", name, err)
	}

	cfg := wazero.NewModuleConfig().WithName(name)
	mod, err := r.rt.InstantiateModule(ctx, compiled, cfg)
	if err != nil {
		return nil, fmt.Errorf("instantiate plugin %q: %w", name, err)
	}

	// Validate required exports.
	for _, export := range []string{"plugin_name", "plugin_version"} {
		if fn := mod.ExportedFunction(export); fn == nil {
			mod.Close(ctx) //nolint:errcheck
			return nil, fmt.Errorf("plugin %q missing required export %q", name, export)
		}
	}

	pluginName, err := readStringResult(ctx, mod, "plugin_name")
	if err != nil {
		mod.Close(ctx) //nolint:errcheck
		return nil, fmt.Errorf("plugin %q: read name: %w", name, err)
	}

	pluginVersion, err := readStringResult(ctx, mod, "plugin_version")
	if err != nil {
		mod.Close(ctx) //nolint:errcheck
		return nil, fmt.Errorf("plugin %q: read version: %w", name, err)
	}

	p := &Plugin{
		Name:    pluginName,
		Version: pluginVersion,
		module:  mod,
	}

	r.mu.Lock()
	r.plugins[name] = p
	r.mu.Unlock()

	return p, nil
}

// CallPlugin invokes a WASM function in the named plugin.
// The function must have the signature: (ptr i32, len i32) -> (ptr i32, len i32).
func (r *Runtime) CallPlugin(ctx context.Context, name string, function string, input []byte) ([]byte, error) {
	r.mu.RLock()
	p, ok := r.plugins[name]
	r.mu.RUnlock()
	if !ok {
		return nil, fmt.Errorf("plugin %q not loaded", name)
	}

	fn := p.module.ExportedFunction(function)
	if fn == nil {
		return nil, fmt.Errorf("plugin %q has no export %q", name, function)
	}

	// Allocate memory in the WASM module for the input.
	allocFn := p.module.ExportedFunction("malloc")
	if allocFn == nil {
		allocFn = p.module.ExportedFunction("alloc")
	}
	if allocFn == nil {
		return nil, fmt.Errorf("plugin %q missing alloc/malloc export", name)
	}

	inputLen := uint64(len(input))
	results, err := allocFn.Call(ctx, inputLen)
	if err != nil {
		return nil, fmt.Errorf("plugin %q alloc: %w", name, err)
	}
	ptr := results[0]

	// Copy input bytes into WASM memory.
	if !p.module.Memory().Write(uint32(ptr), input) {
		return nil, fmt.Errorf("plugin %q: failed to write input to memory", name)
	}

	// Call the function.
	res, err := fn.Call(ctx, ptr, inputLen)
	if err != nil {
		return nil, fmt.Errorf("plugin %q function %q: %w", name, function, err)
	}
	if len(res) < 2 {
		return nil, fmt.Errorf("plugin %q function %q: expected 2 return values, got %d", name, function, len(res))
	}

	// Read result from WASM memory.
	outPtr := uint32(res[0])
	outLen := uint32(res[1])
	if outLen == 0 {
		return []byte{}, nil
	}

	out, ok := p.module.Memory().Read(outPtr, outLen)
	if !ok {
		return nil, fmt.Errorf("plugin %q function %q: failed to read result from memory", name, function)
	}

	// Copy because WASM memory can be invalidated.
	result := make([]byte, len(out))
	copy(result, out)
	return result, nil
}

// ListPlugins returns all loaded plugins.
func (r *Runtime) ListPlugins() []*Plugin {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]*Plugin, 0, len(r.plugins))
	for _, p := range r.plugins {
		out = append(out, p)
	}
	return out
}

// UnloadPlugin closes and removes a plugin from the runtime.
func (r *Runtime) UnloadPlugin(ctx context.Context, name string) error {
	r.mu.Lock()
	p, ok := r.plugins[name]
	if ok {
		delete(r.plugins, name)
	}
	r.mu.Unlock()

	if !ok {
		return fmt.Errorf("plugin %q not loaded", name)
	}
	if err := p.module.Close(ctx); err != nil {
		return fmt.Errorf("close plugin %q: %w", name, err)
	}
	return nil
}

// readStringResult calls a no-arg WASM function returning (ptr, len) and reads the string.
func readStringResult(ctx context.Context, mod api.Module, fnName string) (string, error) {
	fn := mod.ExportedFunction(fnName)
	if fn == nil {
		return "", fmt.Errorf("missing export %q", fnName)
	}
	res, err := fn.Call(ctx)
	if err != nil {
		return "", fmt.Errorf("call %q: %w", fnName, err)
	}
	if len(res) < 2 {
		return "", fmt.Errorf("%q: expected 2 return values, got %d", fnName, len(res))
	}
	ptr := uint32(res[0])
	length := uint32(res[1])
	b, ok := mod.Memory().Read(ptr, length)
	if !ok {
		return "", fmt.Errorf("%q: failed to read string from memory", fnName)
	}
	return string(b), nil
}
