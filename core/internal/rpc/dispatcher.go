package rpc

import (
	"encoding/json"
	"fmt"
	"sync"
)

// HandlerFunc is the function signature for RPC method handlers.
type HandlerFunc func(params map[string]interface{}) (interface{}, error)

// Dispatcher routes JSON-RPC method calls to registered handlers.
type Dispatcher struct {
	mu       sync.RWMutex
	handlers map[string]HandlerFunc
}

// NewDispatcher creates a new Dispatcher.
func NewDispatcher() *Dispatcher {
	return &Dispatcher{
		handlers: make(map[string]HandlerFunc),
	}
}

// Register registers a handler for the given method name.
func (d *Dispatcher) Register(method string, handler HandlerFunc) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.handlers[method] = handler
}

// Dispatch calls the handler for the given method with the provided params.
func (d *Dispatcher) Dispatch(method string, rawParams json.RawMessage) (interface{}, error) {
	d.mu.RLock()
	handler, ok := d.handlers[method]
	d.mu.RUnlock()

	if !ok {
		return nil, fmt.Errorf("method not found: %s", method)
	}

	var params map[string]interface{}
	if len(rawParams) > 0 {
		if err := json.Unmarshal(rawParams, &params); err != nil {
			return nil, fmt.Errorf("invalid params: %w", err)
		}
	}
	if params == nil {
		params = make(map[string]interface{})
	}

	return handler(params)
}
