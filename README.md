# Golang Channel-Based Data Notifications

[![Go Reference](https://pkg.go.dev/badge/vawter.tech/notify.svg)](https://pkg.go.dev/vawter.tech/notify)

```shell
go get vawter.tech/notify
```

This package provides a storage variable type with channel-based notifications of changes.
It is useful for values that need to be observed at runtime, such as configuration types.
The `notify.Var` type can be thought of as a cross between an `atomic.Value` coupled to
a `sync.Cond`, albeit using channels for control-flow composition.

```go
v := notify.VarOf(&ConfigurationType{})

// Somewhere in your run loop to respond to changes.
for cfg, cfgUpdated := v.Get(); ; {
    // Do something with the new configuration.
    select {
    case <-cfgUpdated:
        // The channel will be closed when the contents of the
        // variable have changed. This composes easily with multiple
        // channels.
        cfg, cfgUpdated = v.Get()

    case <-ctx.Done():
        return
    }
}

// Elsewhere to update the value.
v.Set(&Configuration{Updated: true})
```

## Project History

This repository was extracted from `github.com/cockroachdb/field-eng-powertools` using the command
`git filter-repo --path stopvar --subdirectory-filter notify --path LICENSE` by the code's original author.
