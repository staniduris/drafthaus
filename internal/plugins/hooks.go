package plugins

import (
	"context"
	"fmt"
	"sync"
)

// HookType defines when a plugin runs.
type HookType string

const (
	HookBeforeRender HookType = "before_render" // Before page render
	HookAfterRender  HookType = "after_render"  // After page render (can modify HTML)
	HookBeforeSave   HookType = "before_save"   // Before entity save
	HookAfterSave    HookType = "after_save"    // After entity save
	HookOnRequest    HookType = "on_request"    // Custom route handler
)

// HookRegistration maps a plugin to a hook.
type HookRegistration struct {
	PluginName string   `json:"plugin_name"`
	Hook       HookType `json:"hook"`
	Function   string   `json:"function"`        // WASM function name to call
	Route      string   `json:"route,omitempty"` // For on_request hooks
}

// HookManager manages plugin hooks.
type HookManager struct {
	runtime *Runtime
	hooks   map[HookType][]HookRegistration
	mu      sync.RWMutex
}

// NewHookManager creates a HookManager backed by a Runtime.
func NewHookManager(rt *Runtime) *HookManager {
	return &HookManager{
		runtime: rt,
		hooks:   make(map[HookType][]HookRegistration),
	}
}

// Register adds a hook registration. Returns an error if the plugin is not loaded.
func (m *HookManager) Register(reg HookRegistration) error {
	m.runtime.mu.RLock()
	_, ok := m.runtime.plugins[reg.PluginName]
	m.runtime.mu.RUnlock()
	if !ok {
		return fmt.Errorf("plugin %q is not loaded", reg.PluginName)
	}

	m.mu.Lock()
	m.hooks[reg.Hook] = append(m.hooks[reg.Hook], reg)
	m.mu.Unlock()
	return nil
}

// RunHooks executes all registered handlers for hookType in order.
// Output of each handler becomes input of the next (pipeline / chain).
func (m *HookManager) RunHooks(ctx context.Context, hookType HookType, data []byte) ([]byte, error) {
	m.mu.RLock()
	regs := make([]HookRegistration, len(m.hooks[hookType]))
	copy(regs, m.hooks[hookType])
	m.mu.RUnlock()

	current := data
	for _, reg := range regs {
		out, err := m.runtime.CallPlugin(ctx, reg.PluginName, reg.Function, current)
		if err != nil {
			return nil, fmt.Errorf("hook %s plugin %q function %q: %w", hookType, reg.PluginName, reg.Function, err)
		}
		current = out
	}
	return current, nil
}

// ListHooks returns all registrations for a given hook type.
func (m *HookManager) ListHooks(hookType HookType) []HookRegistration {
	m.mu.RLock()
	defer m.mu.RUnlock()
	regs := m.hooks[hookType]
	out := make([]HookRegistration, len(regs))
	copy(out, regs)
	return out
}
