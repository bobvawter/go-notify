// Copyright 2025 Bob Vawter (bob@vawter.org)
// SPDX-License-Identifier: Apache-2.0

package notify

import (
	"context"
	"maps"
	"reflect"
	"slices"
	"sync"
)

// An UntypedVar is returned from [Aggregation.Choose].
type UntypedVar interface {
	notifyLocked()
}

// An Aggregation allows an arbitrary number of variables, of
// potentially heterogeneous types, to be selected on.
type Aggregation struct {
	mu struct {
		sync.RWMutex
		m map[UntypedVar]<-chan struct{}
	}
}

// NewAggregation constructs an Aggregation.
func NewAggregation() *Aggregation {
	agg := &Aggregation{}
	agg.mu.m = make(map[UntypedVar]<-chan struct{})
	return agg
}

// Aggregate adds the variable to the [Aggregation] and returns the
// current value of the variable.
//
// This should be a method whenever Go supports generic methods.
func Aggregate[T any](agg *Aggregation, v *Var[T]) T {
	agg.mu.Lock()
	defer agg.mu.Unlock()

	ret, ch := v.Get()
	agg.mu.m[v] = ch

	return ret
}

// Choose selects one aggregated variable at random from the variables
// that have changed since the last time [Aggregate] was called. If the
// Aggregation is empty or no variables have changed, the returned
// bool will be false.
func (a *Aggregation) Choose() (UntypedVar, bool) {
	a.mu.Lock()
	defer a.mu.Unlock()

	for k, v := range a.mu.m {
		select {
		case <-v:
			delete(a.mu.m, k)
			return k, true
		default:
		}
	}

	return nil, false
}

// Len returns the number of aggregated variables.
func (a *Aggregation) Len() int {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return len(a.mu.m)
}

// Updated returns a channel that will be closed if any variable has
// changed since the last time [Aggregate] was called on it or the
// context is cancelled. The updated variable is retrieved by calling
// [Aggregation.Choose].
func (a *Aggregation) Updated(ctx context.Context) <-chan struct{} {
	a.mu.RLock()
	toWatch := slices.Collect(maps.Values(a.mu.m))
	a.mu.RUnlock()

	cases := make([]reflect.SelectCase, len(toWatch)+1)
	cases[0] = reflect.SelectCase{
		Chan: reflect.ValueOf(ctx.Done()),
		Dir:  reflect.SelectRecv,
	}

	ret := make(chan struct{})
	for i, ch := range toWatch {
		select {
		case <-ch:
			close(ret)
			return ret
		default:
			cases[i+1] = reflect.SelectCase{
				Chan: reflect.ValueOf(ch),
				Dir:  reflect.SelectRecv,
			}
		}
	}

	go func() {
		defer close(ret)
		reflect.Select(cases)
	}()
	return ret
}
