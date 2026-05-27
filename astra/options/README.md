# Options
Options are implemented using a `Builder` interface. Options themselves implement the `Builder` interface so they can be passed as structs directly. So - we define our options as simple structs with pointer values:

```go
type SimpleOptions struct {
	Blocking *bool
}
```

Since our options must implement the `Builder` interface, we have a `NoopBuilder` that returns a func that automatically copies non-nil fields:

```go
func (o *SimpleOptions) Setters() []func(*SimpleOptions) {
	return NoopBuilder(o)
}
```

Options must also implement `Validate`.

```go
func (o SimpleOptions) Validate() error {
    // Do validation logic and return error if appropriate.
	return nil
}
```

And here's the `builder` pattern:

```go
// SimpleOptionsBuilder is a builder for SimpleOptions.
type SimpleOptionsBuilder struct {
	Opts []func(*SimpleOptions)
}

// Simple creates a new SimpleOptionsBuilder. This is so consumers can use:
// options.Simple().SetSomeProperty("hello")
//
// It makes more sense with real names like:
// options.CreateIndex().SetName("idx_123")
func Simple() *SimpleOptionsBuilder {
	return &SimpleOptionsBuilder{}
}

// Setters implements Builder[SimpleOptions].
func (b *SimpleOptionsBuilder) Setters() []func(*SimpleOptions) {
	return b.Opts
}

// SetBlocking sets Blocking.
func (b *SimpleOptionsBuilder) SetBlocking(v bool) *SimpleOptionsBuilder {
	b.Opts = append(b.Opts, func(o *SimpleOptions) {
		o.Blocking = &v
	})
	return b
}
```

If you want to set option defaults, you can implement `Defaulter` on your options struct:

```go
// SetDefaults implements the Defaulter interface for SimpleOptions.
func (o *SimpleOptions) SetDefaults() {
    // Default blocking to true
	o.Blocking = ptr.To(true)
}
```
