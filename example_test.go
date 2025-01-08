// Copyright 2025 Bob Vawter (bob@vawter.org)
// SPDX-License-Identifier: Apache-2.0

package notify_test

import (
	"context"
	"fmt"
	"slices"
	"strconv"
	"time"

	"vawter.tech/notify"
)

// This example shows how to receive notifications when some arbitrary
// number of variables have changed.
func ExampleAggregation() {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// This represents some dynamic number of variables to monitor.
	intVars := make([]*notify.Var[int], 2)
	for i := range intVars {
		intVars[i] = notify.VarOf(i + 1)
	}

	// The variables need not be of the same value type.
	stringVars := make([]*notify.Var[string], 2)
	for i := range intVars {
		stringVars[i] = notify.VarOf("X " + strconv.Itoa(i+1))
	}

	// Register the variables with the Aggregation, showing how the
	// current value may be observed. The Aggregate function should
	// be a method once Golang supports generic methods.
	//
	// We'll also update the variable from separate goroutines.
	agg := notify.NewAggregation()
	for _, intVar := range intVars {
		sampled := notify.Aggregate(agg, intVar)
		go intVar.Set(10 * sampled)
	}
	for _, stringVar := range stringVars {
		sampled := notify.Aggregate(agg, stringVar)
		go stringVar.Set(sampled + " Updated")
	}

	var seen []string

	// Variables are removed from the aggregation once updated. They
	// may be added back by calling Aggregate again.
	for agg.Len() > 0 {
		select {
		case <-agg.Updated(ctx):
			// This may coalesce multiple updates.
		case <-ctx.Done():
			panic(ctx.Err())
		}

		// Drain updated variables.
		found, ok := agg.Choose()
		if !ok {
			continue
		}
		switch t := found.(type) {
		case *notify.Var[int]:
			val, _ := t.Get()
			seen = append(seen, strconv.Itoa(val))
		case *notify.Var[string]:
			val, _ := t.Get()
			seen = append(seen, val)
		}
	}

	// Sort output for test.
	slices.Sort(seen)
	fmt.Println(seen)

	// Output:
	// [10 20 X 1 Updated X 2 Updated]
}

type MyConfiguration struct {
	Setting string
}

// This example demonstrates how the latest value within a variable
// can be sampled, which is useful for configurations that may be
// changed at runtime. This is not guaranteed to receive all values
// stored in the variable under high-update-rate conditions. If all
// values are required for correct operation, use a channel, callback,
// or other mechanism.
func ExampleVar() {
	ctx := context.Background()
	v := notify.VarOf(&MyConfiguration{Setting: "Foo"})

	for cfg, cfgUpdated := v.Get(); ; {
		// Do something with the value.
		fmt.Println(cfg.Setting)
		select {
		case <-cfgUpdated:
			// The channel will be closed when the contents of the
			// variable have changed.
			cfg, cfgUpdated = v.Get()

		case <-ctx.Done():
			// It's good practice to have a way to interrupt the wait.
			return
		}
	}
}
