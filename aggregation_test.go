// Copyright 2025 Bob Vawter (bob@vawter.org)
// SPDX-License-Identifier: Apache-2.0

package notify

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestAggregationDrain(t *testing.T) {
	r := require.New(t)

	agg := NewAggregation()
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	vars := make([]*Var[int], 10)
	for i := range vars {
		vars[i] = VarOf(i)
		r.Equal(i, Aggregate(agg, vars[i]))
		// Ensure double-registration is a no-op.
		r.Equal(i, Aggregate(agg, vars[i]))
	}

	r.Equal(len(vars), agg.Len())

	found, ok := agg.Choose()
	r.Nil(found)
	r.False(ok)

	ch := agg.Updated(ctx)
	select {
	case <-ch:
		r.Fail("channel should be open")
	default:
	}

	for i, v := range vars {
		_, _, err := v.Update(func(old int) (int, error) { return old + len(vars), nil })
		r.NoError(err)
		select {
		case <-ch:
		case <-ctx.Done():
			r.NoError(ctx.Err())
		}

		found, ok := agg.Choose()
		r.True(ok)
		r.Same(v, found.(*Var[int]))
		r.Equal(len(vars)-i-1, agg.Len())
	}
}

func TestAggregationImmediate(t *testing.T) {
	r := require.New(t)

	agg := NewAggregation()
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	vars := make([]*Var[int], 10)
	for i := range vars {
		vars[i] = VarOf(i)
		r.Equal(i, Aggregate(agg, vars[i]))
		vars[i].Set(99)
	}

	r.Equal(len(vars), agg.Len())

	select {
	case <-agg.Updated(ctx):
	default:
		r.Fail("should already have a change notification")
	}

	count := 0
	for {
		found, ok := agg.Choose()
		if !ok {
			break
		}
		count++

		value, _ := found.(*Var[int]).Get()
		r.Equal(99, value)
	}
	r.Equal(len(vars), count)
}
