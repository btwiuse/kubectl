//go:build js

/*
Copyright 2016 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package interrupt

import (
	"os"
	"sync"
)

// Handler guarantees execution of notifications after a critical section.
type Handler struct {
	notify []func()
	final  func(os.Signal)
	once   sync.Once
}

// Chain creates a new handler with all notify functions.
func Chain(handler *Handler, notify ...func()) *Handler {
	if handler == nil {
		return New(nil, notify...)
	}
	return New(handler.Signal, append(notify, handler.Close)...)
}

// New creates a new handler.
func New(final func(os.Signal), notify ...func()) *Handler {
	return &Handler{
		final:  final,
		notify: notify,
	}
}

// Close executes all notification handlers once.
func (h *Handler) Close() {
	h.once.Do(func() {
		for _, fn := range h.notify {
			fn()
		}
	})
}

// Signal executes notifications and the final callback.
func (h *Handler) Signal(s os.Signal) {
	h.Close()
	if h.final != nil {
		h.final(s)
	}
}

// Run executes fn and ensures notifications are invoked afterward.
func (h *Handler) Run(fn func() error) error {
	defer h.Close()
	return fn()
}
