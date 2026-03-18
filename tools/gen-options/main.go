// Copyright DataStax, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
//
// gen-options generates boilerplate for the options package:
//
//  1. Children() methods — for each struct with *Validator fields, emits a
//     Children() []Validator method so MergeOptions can recursively validate them.
//
//  2. Builder implementations — for each XxxOptions struct that has a corresponding
//     XxxOptionsBuilder struct, emits the builder struct definition, constructor,
//     List(), and all Set* methods. Also emits the options struct's List() method
//     and trivial Validate() stubs (when no hand-written Validate exists).
//     Hand-written convenience methods (e.g. SetIndexingAllow) are left alone in their
//     original files and simply layer on top of the generated setters.
//
// Usage (via go:generate in options/options.go):
//
//	//go:generate go run -modfile=../tools/gen-options/go.mod ../tools/gen-options -pkg .
//
// Output files (never edit by hand):
//
//	children_gen.go
//	builders_gen.go
package main

import (
	"bytes"
	"flag"
	"fmt"
	"go/ast"
	"go/token"
	"go/types"
	"log"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	"golang.org/x/tools/go/packages"
	"golang.org/x/tools/imports"
)

func main() {
	pkgDir := flag.String("pkg", ".", "package directory to inspect")
	flag.Parse()
	// Since children relies on builders, we generate builders first
	// then reload symbols.
	loadAndGenerateBuilders(*pkgDir)
	loadAndGenerateChildren(*pkgDir)
}

func loadAndGenerateBuilders(pkgDir string) {
	pkg, err := load(pkgDir)
	if err != nil {
		log.Fatalf("load: %v", err)
	}

	if err := writeFile(pkgDir, "builders_gen.go", buildersSrc(pkg)); err != nil {
		log.Fatalf("builders: %v", err)
	}
}

func loadAndGenerateChildren(pkgDir string) {
	pkg, err := load(pkgDir)
	if err != nil {
		log.Fatalf("load: %v", err)
	}

	if err := writeFile(pkgDir, "children_gen.go", childrenSrc(pkg)); err != nil {
		log.Fatalf("children: %v", err)
	}
}

// ----- Package loading -----

type loadedPkg struct {
	name      string
	types     *types.Package
	validator *types.Interface // the Validator interface defined in this package
	syntax    []*ast.File
	fset      *token.FileSet
}

func load(dir string) (*loadedPkg, error) {
	fset := token.NewFileSet()
	cfg := &packages.Config{
		Mode: packages.NeedName | packages.NeedTypes | packages.NeedTypesInfo | packages.NeedSyntax,
		Dir:  dir,
		Fset: fset,
	}
	pkgs, err := packages.Load(cfg, ".")
	if err != nil {
		return nil, err
	}
	if len(pkgs) == 0 {
		return nil, fmt.Errorf("no packages found in %q", dir)
	}
	p := pkgs[0]

	iface, err := validatorInterface(p.Types)
	if err != nil {
		return nil, err
	}
	return &loadedPkg{name: p.Name, types: p.Types, validator: iface, syntax: p.Syntax, fset: fset}, nil
}

func validatorInterface(pkg *types.Package) (*types.Interface, error) {
	obj := pkg.Scope().Lookup("Validator")
	if obj == nil {
		return nil, fmt.Errorf("Validator not found in package %q", pkg.Name())
	}
	iface, ok := obj.Type().Underlying().(*types.Interface)
	if !ok {
		return nil, fmt.Errorf("Validator is not an interface")
	}
	return iface, nil
}

// handWrittenMethods scans non-generated source files and returns a set of
// "ReceiverType.MethodName" strings for all methods found.
func handWrittenMethods(pkg *loadedPkg) map[string]bool {
	methods := make(map[string]bool)
	for _, file := range pkg.syntax {
		filename := filepath.Base(pkg.fset.File(file.Pos()).Name())
		if strings.Contains(filename, "_gen") {
			continue
		}
		for _, decl := range file.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			if !ok || fn.Recv == nil || len(fn.Recv.List) == 0 {
				continue
			}
			var recvName string
			switch t := fn.Recv.List[0].Type.(type) {
			case *ast.StarExpr:
				if ident, ok := t.X.(*ast.Ident); ok {
					recvName = ident.Name
				}
			case *ast.Ident:
				recvName = t.Name
			}
			if recvName != "" {
				methods[recvName+"."+fn.Name.Name] = true
			}
		}
	}
	return methods
}

// ----- Children generation -----

// childStruct is an options struct that has at least one *Validator field.
type childStruct struct {
	Name   string   // e.g. "CreateCollectionOptions"
	Fields []string // names of *Validator pointer fields
}

func childrenSrc(pkg *loadedPkg) renderJob {
	var structs []childStruct
	for _, name := range pkg.types.Scope().Names() {
		s, ok := namedStruct(pkg.types.Scope().Lookup(name))
		if !ok {
			continue
		}
		fields := validatorPtrFields(s, pkg.validator)
		if len(fields) > 0 {
			structs = append(structs, childStruct{Name: name, Fields: fields})
		}
	}
	return renderJob{PkgName: pkg.name, Tmpl: childrenTmpl, Data: structs}
}

// validatorPtrFields returns field names whose type is *T where T implements Validator.
func validatorPtrFields(s *types.Struct, validator *types.Interface) []string {
	var names []string
	for i := range s.NumFields() {
		f := s.Field(i)
		if ptr, ok := f.Type().(*types.Pointer); ok && types.Implements(ptr, validator) {
			names = append(names, f.Name())
		}
	}
	return names
}

// ----- Builder generation -----

// optsDef describes an options type and its optional builder.
type optsDef struct {
	OptsType    string // e.g. "CreateCollectionOptions"
	GenValidate bool   // true → generate trivial Validate()
	HasBuilder  bool   // true → generate builder struct, constructor, setters
	BuilderType string // e.g. "CreateCollectionOptionsBuilder"
	Constructor string // e.g. "CreateCollection"
	Setters     []setterDef
}

// setterDef describes a single Set* method to generate.
type setterDef struct {
	Method            string // e.g. "SetLimit"
	Field             string // e.g. "Limit"
	ParamType         string // e.g. "int", "Builder[VectorOptions]", "map[string]any"
	IsVariadicBuilder bool   // true → takes ...Builder[T], calls MergeOptions
	InnerType         string // T in Builder[T] when IsVariadicBuilder is true
	IsMap             bool   // true → stored directly, no pointer wrapping
	IsSlice           bool   // true → variadic element setter, stored directly
	ElemType          string // element type when IsSlice is true
}

func buildersSrc(pkg *loadedPkg) renderJob {
	hwMethods := handWrittenMethods(pkg)
	scope := pkg.types.Scope()

	var defs []optsDef
	for _, name := range scope.Names() {
		if !strings.HasSuffix(name, "Options") || strings.HasSuffix(name, "OptionsBuilder") {
			continue
		}
		obj := scope.Lookup(name)
		optsStruct, ok := namedStruct(obj)
		if !ok {
			continue
		}

		builderName := name + "Builder"
		def := optsDef{
			OptsType:    name,
			GenValidate: !hwMethods[name+".Validate"],
			HasBuilder:  true,
			BuilderType: builderName,
			Constructor: strings.TrimSuffix(name, "Options"),
			Setters:     settersFor(optsStruct, pkg.validator),
		}

		defs = append(defs, def)
	}

	return renderJob{PkgName: pkg.name, Tmpl: buildersTmpl, Data: defs}
}

// settersFor inspects every field of s and returns a setterDef for each one we
// know how to generate. Unrecognised field kinds are silently skipped — they can
// be written by hand as convenience methods that delegate to the generated ones.
func settersFor(s *types.Struct, validator *types.Interface) []setterDef {
	var setters []setterDef
	for i := range s.NumFields() {
		if sd, ok := setterForField(s.Field(i), validator); ok {
			setters = append(setters, sd)
		}
	}
	return setters
}

func setterForField(f *types.Var, validator *types.Interface) (setterDef, bool) {
	method := "Set" + f.Name()

	switch t := f.Type().(type) {

	case *types.Pointer:
		// *Validator child → variadic builder setter using MergeOptions
		if types.Implements(t, validator) {
			inner := t.Elem().(*types.Named).Obj().Name()
			return setterDef{
				Method:            method,
				Field:             f.Name(),
				ParamType:         fmt.Sprintf("Builder[%s]", inner),
				IsVariadicBuilder: true,
				InnerType:         inner,
			}, true
		}
		// *scalar → value setter (we take v T and store &v)
		if param := typeStr(t.Elem()); param != "" {
			return setterDef{Method: method, Field: f.Name(), ParamType: param}, true
		}

	case *types.Map:
		// map[K]V → value setter (stored directly, no address-of)
		if param := typeStr(t); param != "" {
			return setterDef{Method: method, Field: f.Name(), ParamType: param, IsMap: true}, true
		}

	case *types.Slice:
		// []T → variadic setter (v ...T, stored directly)
		if elem := typeStr(t.Elem()); elem != "" {
			return setterDef{Method: method, Field: f.Name(), IsSlice: true, ElemType: elem}, true
		}
	}

	// Interfaces (e.g. WarningHandler), funcs — skip and let the author
	// write these by hand.
	return setterDef{}, false
}

// typeStr returns a Go source string for t, or "" for types we don't handle.
func typeStr(t types.Type) string {
	t = types.Unalias(t)
	switch t := t.(type) {
	case *types.Basic:
		return t.Name()
	case *types.Named:
		obj := t.Obj()
		// Types from outside the package need a "pkg.Name" qualifier.
		if obj.Pkg() != nil && obj.Pkg().Name() != "options" {
			return obj.Pkg().Name() + "." + obj.Name()
		}
		return obj.Name()
	case *types.Map:
		k, v := typeStr(t.Key()), typeStr(t.Elem())
		if k == "" || v == "" {
			return ""
		}
		return fmt.Sprintf("map[%s]%s", k, v)
	case *types.Slice:
		if elem := typeStr(t.Elem()); elem != "" {
			return "[]" + elem
		}
	case *types.Interface:
		if t.Empty() {
			return "any"
		}
	}
	return ""
}

// ----- Shared helpers -----

func namedStruct(obj types.Object) (*types.Struct, bool) {
	if _, ok := obj.(*types.TypeName); !ok {
		return nil, false
	}
	s, ok := obj.Type().Underlying().(*types.Struct)
	return s, ok
}

// ----- Rendering -----

type renderJob struct {
	PkgName string
	Tmpl    *template.Template
	Data    any
}

func writeFile(dir, filename string, job renderJob) error {
	src, err := render(job)
	if err != nil {
		return fmt.Errorf("render %s: %w", filename, err)
	}
	path := filepath.Join(dir, filename)
	if err := os.WriteFile(path, src, 0644); err != nil {
		return fmt.Errorf("write %s: %w", path, err)
	}
	fmt.Printf("wrote %s\n", path)
	return nil
}

func render(job renderJob) ([]byte, error) {
	var buf bytes.Buffer
	if err := job.Tmpl.Execute(&buf, job); err != nil {
		return nil, err
	}
	src, err := imports.Process("", buf.Bytes(), nil)
	if err != nil {
		// Include the unformatted source in the error for easier debugging.
		return buf.Bytes(), fmt.Errorf("goimports failed: %w\n\n%s", err, buf.String())
	}
	return src, nil
}

// ----- Templates -----

const boilerplate = `// Copyright DataStax, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
//
// Code generated by gen-options; DO NOT EDIT.`

var childrenTmpl = template.Must(template.New("children").Parse(boilerplate + `

package {{ .PkgName }}
{{ range .Data }}
// Children implements ChildValidator for {{ .Name }}.
// Returns all non-nil Validator fields for recursive validation in MergeOptions.
func (o *{{ .Name }}) Children() []Validator {
	var children []Validator
	{{- range .Fields }}
	if o.{{ . }} != nil {
		children = append(children, o.{{ . }})
	}
	{{- end }}
	return children
}
{{ end }}`))

var buildersTmpl = template.Must(template.New("builders").Parse(boilerplate + `

package {{ .PkgName }}
{{ range .Data }}{{ $o := . }}
// List implements Builder[{{ .OptsType }}] allowing the raw struct to be
// passed directly to methods that accept ...Builder[{{$o.OptsType}}].
func (o *{{ .OptsType }}) List() []func(*{{ .OptsType }}) {
	return NoopBuilder(o)
}
{{ if .GenValidate }}
// Validate implements Validator for {{ .OptsType }}.
func (o {{ .OptsType }}) Validate() error { return nil }
{{ end }}
{{- if .HasBuilder }}
// {{ .BuilderType }} is a builder for {{ .OptsType }}.
type {{ .BuilderType }} struct {
	Opts []func(*{{ .OptsType }})
}

// {{ .Constructor }} creates a new {{ .BuilderType }}.
func {{ .Constructor }}() *{{ .BuilderType }} {
	return &{{ .BuilderType }}{}
}

// List implements Builder[{{ .OptsType }}].
func (b *{{ .BuilderType }}) List() []func(*{{ .OptsType }}) {
	return b.Opts
}
{{ range .Setters }}
{{- if .IsVariadicBuilder }}
// {{ .Method }} sets the {{ .Field }} option.
func (b *{{ $o.BuilderType }}) {{ .Method }}(v ...{{ .ParamType }}) *{{ $o.BuilderType }} {
	b.Opts = append(b.Opts, func(o *{{ $o.OptsType }}) {
		merged, _ := MergeOptions(v...)
		o.{{ .Field }} = merged
	})
	return b
}
{{ else if .IsSlice }}
// {{ .Method }} sets the {{ .Field }} option.
func (b *{{ $o.BuilderType }}) {{ .Method }}(v ...{{ .ElemType }}) *{{ $o.BuilderType }} {
	b.Opts = append(b.Opts, func(o *{{ $o.OptsType }}) { o.{{ .Field }} = v })
	return b
}
{{ else if .IsMap }}
// {{ .Method }} sets the {{ .Field }} option.
func (b *{{ $o.BuilderType }}) {{ .Method }}(v {{ .ParamType }}) *{{ $o.BuilderType }} {
	b.Opts = append(b.Opts, func(o *{{ $o.OptsType }}) { o.{{ .Field }} = v })
	return b
}
{{ else }}
// {{ .Method }} sets the {{ .Field }} option.
func (b *{{ $o.BuilderType }}) {{ .Method }}(v {{ .ParamType }}) *{{ $o.BuilderType }} {
	b.Opts = append(b.Opts, func(o *{{ $o.OptsType }}) { o.{{ .Field }} = &v })
	return b
}
{{ end -}}
{{ end }}
{{ end -}}
{{ end }}`))
