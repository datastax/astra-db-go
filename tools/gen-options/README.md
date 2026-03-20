# gen-options

Code generator that produces boilerplate for the `options` package. It inspects the options structs and emits two files:

- **`builders_gen.go`** — `Builder` implementations (`Setters()`, `Validate()`, `Set*` methods) for every `XxxOptions` struct that has a corresponding `XxxOptionsBuilder`. Also generates type aliases (`XxxOption = Builder[XxxOptions]`) for cleaner public signatures.
- **`children_gen.go`** — `Children() []Validator` methods for structs that contain `*Validator` fields, enabling recursive validation in `MergeAndValidate`.

Hand-written convenience methods (e.g. `SetIndexingAllow`) are left untouched and layer on top of the generated setters. Similarly, if you implement `Validator`, we won't code-gen a boilerplate `Validator` func.

## Usage

Assuming you create a struct in the options package like this:

```go
type TestOptions struct {
	// TestVersion is for testing. Mostly to illustrate how codegen
	// works and how it copies doc comments. It is required.
	TestVersion *string `json:"testVersion,omitempty"`
}
```

This tool will codegen the boilerplate:

```go
// TestOption is a convenience alias for Builder[TestOptions].
type TestOption = Builder[TestOptions]

// Setters implements Builder[TestOptions] allowing the raw struct to be
// passed directly to methods that accept ...Builder[TestOptions].
func (o *TestOptions) Setters() []func(*TestOptions) {
	return NoopBuilder(o)
}

// Validate implements Validator for TestOptions.
func (o *TestOptions) Validate() error { return nil }

// TestOptionsBuilder is a builder for TestOptions.
type TestOptionsBuilder struct {
	Opts []func(*TestOptions)
}

// Test creates a new TestOptionsBuilder.
func Test() *TestOptionsBuilder {
	return &TestOptionsBuilder{}
}

// Setters implements Builder[TestOptions].
func (b *TestOptionsBuilder) Setters() []func(*TestOptions) {
	return b.Opts
}

// SetTestVersion sets the TestVersion option.
// TestVersion is for testing. Mostly to illustrate how codegen
// works and how it copies doc comments. It is required.
func (b *TestOptionsBuilder) SetTestVersion(v string) *TestOptionsBuilder {
	b.Opts = append(b.Opts, func(o *TestOptions) { o.TestVersion = &v })
	return b
}
```

If you implement `Validator`, it will be excluded from boilerplate:

```go
// Validate implements Validator for TestOptions. Implementing this by hand
// will exclude it from being generated.
func (o *TestOptions) Validate() error {
	if o.TestVersion == nil {
		return errors.New("TestVersion is required")
	}
	return nil
}
```

## Running the code generator

```bash
# From main repo folder
go generate ./...
```

Note the following in [options.go](./../../options/options.go):

```go
//go:generate go run -modfile=../tools/gen-options/go.mod ../tools/gen-options/main.go -pkg .
```
