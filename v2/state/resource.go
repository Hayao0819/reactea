// Package state holds application data that components load and keep.
package state

import (
	"context"

	tea "charm.land/bubbletea/v2"
	"github.com/Hayao0819/reactea/v2"
)

// Resource is a value loaded in the background. Its zero value is ready to use.
// Only the latest load lands; an earlier one still in flight is cancelled.
type Resource[T any] struct {
	value   T
	err     error
	loading bool
	loaded  bool

	generation uint64
	cancel     context.CancelFunc
}

type loadedMsg[T any] struct {
	target     *Resource[T]
	generation uint64
	value      T
	err        error
}

// Load starts fetch under ctx's scope, so unmounting the component cancels it.
// Call it from Init or Update and pass the result to Handle.
func (r *Resource[T]) Load(ctx *reactea.Ctx, fetch func(context.Context) (T, error)) tea.Cmd {
	if r.cancel != nil {
		r.cancel()
	}

	background, cancel := context.WithCancel(ctx.Context())
	r.cancel = cancel
	r.generation++
	r.loading = true

	generation := r.generation

	return func() tea.Msg {
		defer cancel()

		value, err := fetch(background)

		return loadedMsg[T]{target: r, generation: generation, value: value, err: err}
	}
}

// Handle stores the result of this resource's latest load and reports whether
// msg belonged to this resource. A failed load keeps the last value.
func (r *Resource[T]) Handle(msg tea.Msg) bool {
	loaded, ok := msg.(loadedMsg[T])
	if !ok || loaded.target != r {
		return false
	}

	if loaded.generation != r.generation {
		return true
	}

	r.loading, r.cancel = false, nil
	r.err = loaded.err

	if loaded.err == nil {
		r.value, r.loaded = loaded.value, true
	}

	return true
}

// Value is the last successfully loaded value.
func (r *Resource[T]) Value() T { return r.value }

// Err is the latest load's error, or nil after a success.
func (r *Resource[T]) Err() error { return r.err }

// Loading reports whether a load is in flight.
func (r *Resource[T]) Loading() bool { return r.loading }

// Loaded reports whether any load has succeeded.
func (r *Resource[T]) Loaded() bool { return r.loaded }
