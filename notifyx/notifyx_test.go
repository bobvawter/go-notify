// Copyright 2024 The Cockroach Authors
// Copyright 2025 Bob Vawter (bob@vawter.org)
// SPDX-License-Identifier: Apache-2.0

package notifyx

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"vawter.tech/notify"
	"vawter.tech/stopper"
)

func TestDoWhenChanged(t *testing.T) {
	r := require.New(t)
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	var called atomic.Bool
	var v notify.Var[int]

	stop := stopper.WithContext(ctx)
	stop.Go(func(stop *stopper.Context) error {
		_, err := DoWhenChanged(stop, -1, &v, func(ctx *stopper.Context, old, new int) error {
			switch new {
			case 0:
				// This can happen if the goroutine executes before the
				// call to Set(1) below.
				r.Equal(-1, old)
			case 1:
				// Or statement to account for early-execution case.
				if old == -1 || old == 0 {
					v.Set(2) // This should cause us to loop around.
				} else {
					r.Failf("unexpected old value", "%d", old)
				}
			case 2:
				r.Equal(1, old)
				called.Store(true)
				stop.Stop(time.Minute)
			}
			return nil
		})
		return err
	})

	v.Set(1)
	r.NoError(stop.Wait())
	r.True(called.Load())
}

func TestDoWhenChangedOrInterval(t *testing.T) {
	r := require.New(t)
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	var called atomic.Bool
	sawMinusOne := make(chan struct{})
	var v notify.Var[int]

	stop := stopper.WithContext(ctx)
	stop.Go(func(stop *stopper.Context) error {
		_, err := DoWhenChangedOrInterval(stop, -1, &v, time.Millisecond,
			func(ctx *stopper.Context, old, new int) error {
				// The expected sequence old (old, new):
				// (-1, 0)
				// (0, 0) ...
				// (0, 1)
				// (1, 1) ...
				// (1, 2)
				// (2, 2) ...

				switch {
				case old == new:
					// Looping due to timer.
				case old == -1 && new == 0:
					close(sawMinusOne)
				case old == 0 && new == 1:
					v.Set(2)
				case old == 1 && new == 2:
					called.Store(true)
					stop.Stop(time.Minute)
				default:
					r.Failf("unexpected state", "old=%d, new=%d", old, new)
				}

				return nil
			})
		return err
	})

	<-sawMinusOne
	v.Set(1)
	r.NoError(stop.Wait())
	r.True(called.Load())
}

func TestWaitForValue(t *testing.T) {
	r := require.New(t)
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	var called atomic.Bool
	var v notify.Var[int]

	stop := stopper.WithContext(ctx)
	stop.Go(func(stop *stopper.Context) error {
		if err := WaitForValue(stop, 1, &v); err != nil {
			return err
		}
		called.Store(true)
		stop.Stop(time.Minute)
		return nil
	})

	v.Set(1)
	r.NoError(stop.Wait())
	r.True(called.Load())

}
