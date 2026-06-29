// Copyright IBM Corp.
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

// gen-options generates boilerplate for the options package:
//
//  1. Children() methods — for each struct with nested option fields, emits a
//     Children() []any method so MergeAndValidate can recursively validate them.
//
//  2. Builder implementations — for each XxxOptions struct that has a corresponding
//     XxxOptionsBuilder struct, emits the builder struct definition, constructor,
//     Setters(), and all Set* methods. Also emits the options struct's Setters() method
//     and trivial Validate() stubs (when no hand-written Validate exists).
//     Hand-written convenience methods (e.g. UpdateIndexingAllow) are left alone in their
//     original files and simply layer on top of the generated setters.
//
// Usage (via go:generate in options/options.go):
//
//	//go:generate go run -modfile=../../tools/gen-options/go.mod ../../tools/gen-options -pkg .
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
	"reflect"
	"strings"
	"text/template"
	"unicode"
	"unicode/utf8"

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
	name        string
	types       *types.Package
	shouldMerge *types.Interface // the ShouldMerge interface defined in this package
	syntax      []*ast.File
	fset        *token.FileSet
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

	smIface := shouldMergeInterface(p.Types)
	return &loadedPkg{name: p.Name, types: p.Types, shouldMerge: smIface, syntax: p.Syntax, fset: fset}, nil
}

func shouldMergeInterface(pkg *types.Package) *types.Interface {
	obj := pkg.Scope().Lookup("shouldMerge")
	if obj == nil {
		log.Fatalf("shouldMerge not found in package %q", pkg.Name())
	}
	iface, ok := obj.Type().Underlying().(*types.Interface)
	if !ok {
		log.Fatalf("shouldMerge is not an interface")
	}
	return iface
}

// handWrittenTypes scans non-generated source files and returns a set of
// type names that are manually defined (not generated).
func handWrittenTypes(pkg *loadedPkg) map[string]bool {
	types := make(map[string]bool)
	for _, file := range pkg.syntax {
		filename := filepath.Base(pkg.fset.File(file.Pos()).Name())
		if strings.Contains(filename, "_gen") {
			continue
		}
		for _, decl := range file.Decls {
			gen, ok := decl.(*ast.GenDecl)
			if !ok || gen.Tok != token.TYPE {
				continue
			}
			for _, spec := range gen.Specs {
				ts, ok := spec.(*ast.TypeSpec)
				if !ok {
					continue
				}
				types[ts.Name.Name] = true
			}
		}
	}
	return types
}

// ----- Children generation -----

// childStruct is an options struct that has at least one nested options field.
type childStruct struct {
	Name   string   // e.g. "CreateCollectionOptions"
	Fields []string // names of nested options pointer fields
}

func childrenSrc(pkg *loadedPkg) renderJob {
	var structs []childStruct
	for _, name := range pkg.types.Scope().Names() {
		s, ok := namedStruct(pkg.types.Scope().Lookup(name))
		if !ok {
			continue
		}
		fields := optionsPtrFields(s)
		if len(fields) > 0 {
			structs = append(structs, childStruct{Name: name, Fields: fields})
		}
	}
	return renderJob{PkgName: pkg.name, Tmpl: childrenTmpl, Data: structs}
}

// optionsPtrFields returns field names whose type is *T where T is an Options struct.
func optionsPtrFields(s *types.Struct) []string {
	var names []string
	for i := range s.NumFields() {
		f := s.Field(i)
		if ptr, ok := f.Type().(*types.Pointer); ok {
			if named, ok := ptr.Elem().(*types.Named); ok && strings.HasSuffix(named.Obj().Name(), "Options") {
				names = append(names, f.Name())
			}
		}
	}
	return names
}

// ----- Builder generation -----

// optsDef describes an options type and its optional builder.
type optsDef struct {
	OptsType    string // e.g. "CreateCollectionOptions"
	GenAlias    bool   // true → generate type alias (e.g. CreateCollectionOption)
	HasBuilder  bool   // true → generate builder struct, constructor, setters
	BuilderType string // e.g. "createCollectionOptionsBuilder"
	Constructor string // e.g. "CreateCollection"
	Setters     []setterDef
	Alias       aliasDef
}

// setterDef describes a single Set* method to generate.
type setterDef struct {
	Comment           string // doc comment from the struct field, if any
	Method            string // e.g. "SetLimit"
	Field             string // e.g. "Limit"
	ParamType         string // e.g. "int", "Builder[VectorOptions]", "map[string]any"
	IsVariadicBuilder bool   // true → takes ...Builder[T], calls MergeInto
	IsDirectAssign    bool   // true → stored directly, no pointer wrapping
	IsSlice           bool   // true → variadic element setter, stored directly
	ElemType          string // element type when IsSlice is true
	ContainerField    string // e.g. "APIOptions"
	ContainerType     string // e.g. "APIOptions"
}

func getUnderlyingStruct(t types.Type) (*types.Struct, bool) {
	if ptr, ok := t.Underlying().(*types.Pointer); ok {
		t = ptr.Elem()
	}
	s, ok := t.Underlying().(*types.Struct)
	return s, ok
}

// aliasDef holds info for generating a doc comment and type alias for Options builders.
type aliasDef struct {
	Constructor    string // e.g. "CreateKeyspace"
	Alias          string // e.g. "CreateKeyspaceOption"
	OptsType       string // e.g. "CreateKeyspaceOptions"
	HasSimpleField bool   // Whether the example has a simple field
	Method         string // e.g. "SetBlocking"
	Field          string // e.g. "Blocking"
	BuilderArg     string // e.g. "false"
	StructVal      string // e.g. "ptr.To(false)"
}

// aliasSimpleTmpl is used when a simple scalar field is available for a concrete example.
var aliasSimpleTmpl = template.Must(template.New("aliasSimple").Parse(
	`// {{ .Alias }} configures a {{ .Constructor }} operation.
// You can use the fluent-style builder or a pointer to [{{ .OptsType }}] interchangeably.
// 
// Example using the fluent builder ([{{ .Constructor}}]):
//
//	// No need to use pointer for builder; the builder handles that for you.
//	opts := options.{{ .Constructor }}().{{ .Method }}({{ .BuilderArg }})
//
// Example using a pointer to [{{ .OptsType }}] without the fluent builder:
//
//	opts := &options.{{ .OptsType }}{ {{- .Field}}: {{ .StructVal }}}
type {{ .Alias }} = Builder[{{ .OptsType }}]`))

// aliasFallbackTmpl is used when there are setters but no simple scalar field.
var aliasFallbackTmpl = template.Must(template.New("aliasFallback").Parse(
	`// {{ .Alias }} configures a {{ .Constructor }} operation.
// You can use the fluent-style builder or a pointer to [{{ .OptsType }}] interchangeably.
// 
// Example using the fluent builder ([{{ .Constructor}}]):
//
//	opts := options.{{ .Constructor }}().{{ .Method }}(...)
//
// Example using a pointer to [{{ .OptsType }}] without the fluent builder:
//
//	opts := &options.{{ .OptsType }}{...}
type {{ .Alias }} = Builder[{{ .OptsType }}]`))

// aliasMinimalTmpl is used when there are no setters at all.
var aliasMinimalTmpl = template.Must(template.New("aliasMinimal").Parse(
	`// {{ .Alias }} configures a {{ .Constructor }} operation.
type {{ .Alias }} = Builder[{{ .OptsType }}]`))

// String returns the full doc comment block for the type alias.
func (e aliasDef) String() string {
	var t *template.Template
	switch {
	case e.HasSimpleField:
		t = aliasSimpleTmpl
	case e.Method != "":
		t = aliasFallbackTmpl
	default:
		t = aliasMinimalTmpl
	}
	var buf strings.Builder
	t.Execute(&buf, e)
	return buf.String()
}

// pickAliasExample selects a simple field from the setters to use as a
// concrete example in the alias doc comment.
func pickAliasExample(setters []setterDef) aliasDef {
	simpleTypes := map[string][2]string{
		"bool":          {"false", "ptr.To(false)"},
		"int":           {"42", "ptr.To(42)"},
		"string":        {`"value"`, `ptr.To("value")`},
		"time.Duration": {"10 * time.Second", "ptr.To(10 * time.Second)"},
	}
	for _, s := range setters {
		if s.IsVariadicBuilder || s.IsDirectAssign || s.IsSlice {
			continue
		}
		if vals, ok := simpleTypes[s.ParamType]; ok {
			return aliasDef{
				HasSimpleField: true,
				Method:         s.Method,
				Field:          s.Field,
				BuilderArg:     vals[0],
				StructVal:      vals[1],
			}
		}
	}
	// Fallback: use the first setter with a generic form.
	if len(setters) > 0 {
		return aliasDef{Method: setters[0].Method}
	}
	// Empty options struct with no setters at all. Don't know if/when this is going to happen in the real world.
	return aliasDef{}
}

func buildersSrc(pkg *loadedPkg) renderJob {
	hwTypes := handWrittenTypes(pkg)
	comments := fieldComments(pkg)
	scope := pkg.types.Scope()

	// Pre-scan: collect all Options struct names.
	// This set lets setterForField generate builder-style setters
	// for fields that embed other options structs.
	futureOptions := make(map[string]bool)
	for _, name := range scope.Names() {
		if !strings.HasSuffix(name, "Options") || strings.HasSuffix(name, "OptionsBuilder") {
			continue
		}
		if _, ok := namedStruct(scope.Lookup(name)); ok {
			futureOptions[name] = true
		}
	}

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
		constructor := strings.TrimSuffix(name, "Options")
		aliasName := constructor + "Option"
		def := optsDef{
			OptsType:    name,
			GenAlias:    !hwTypes[aliasName],
			HasBuilder:  true,
			BuilderType: unexportedName(builderName),
			Constructor: constructor,
			Setters:     settersFor(name, optsStruct, pkg.shouldMerge, futureOptions, comments),
		}

		def.Alias = pickAliasExample(def.Setters)
		def.Alias.Constructor = constructor
		def.Alias.OptsType = name
		def.Alias.Alias = aliasName
		defs = append(defs, def)
	}

	return renderJob{PkgName: pkg.name, Tmpl: buildersTmpl, Data: defs}
}

func typeName(t types.Type) string {
	if ptr, ok := t.Underlying().(*types.Pointer); ok {
		t = ptr.Elem()
	}
	if named, ok := t.(*types.Named); ok {
		return named.Obj().Name()
	}
	return ""
}

// settersFor inspects every field of s and returns a setterDef for each one we
// know how to generate. Unrecognised field kinds are silently skipped — they can
// be written by hand as convenience methods that delegate to the generated ones.
func settersFor(structName string, s *types.Struct, shouldMerge *types.Interface, futureOptions map[string]bool, comments map[string]map[string]string) []setterDef {
	var setters []setterDef
	for i := 0; i < s.NumFields(); i++ {
		f := s.Field(i)

		// Check for optlift tag
		tag := s.Tag(i)
		if lift := reflect.StructTag(tag).Get("optlift"); lift != "" {
			nestedStruct, ok := getUnderlyingStruct(f.Type())
			if !ok {
				continue
			}

			nestedStructName := typeName(f.Type())
			for _, item := range strings.Split(lift, ",") {
				lfName := strings.TrimSpace(item)
				alias := ""
				if parts := strings.Split(lfName, ":"); len(parts) == 2 {
					lfName = strings.TrimSpace(parts[0])
					alias = strings.TrimSpace(parts[1])
				}

				// Find field in nestedStruct
				found := false
				for j := 0; j < nestedStruct.NumFields(); j++ {
					nf := nestedStruct.Field(j)
					if nf.Name() != lfName {
						continue
					}

					found = true
					if sd, ok := setterForField(nestedStructName, nf, shouldMerge, futureOptions, comments); ok {
						if alias != "" {
							sd.Method = "Set" + alias
						}
						sd.Comment = fmt.Sprintf("%s (lifted from %s)", sd.Comment, nestedStructName)
						sd.ContainerField = f.Name()
						sd.ContainerType = nestedStructName
						setters = append(setters, sd)
					}
				}

				if !found {
					log.Fatalf("optlift: field %s not found in %s (lifted by %s.%s)", lfName, nestedStructName, structName, f.Name())
				}
			}
		}

		if sd, ok := setterForField(structName, f, shouldMerge, futureOptions, comments); ok {
			setters = append(setters, sd)
		}
	}
	return setters
}

// fieldComments extracts doc comments from struct fields across all source files.
// Map layout is map[structname]map[fieldname]comment. So can use like:
//
//	comments["CreateTableOptions"]["IfNotExists"] // get comment for CreateTableOptions.IfNotExists
func fieldComments(pkg *loadedPkg) map[string]map[string]string {
	result := make(map[string]map[string]string)

	for _, file := range pkg.syntax {
		// Skip generated files
		filename := filepath.Base(pkg.fset.File(file.Pos()).Name())
		if strings.Contains(filename, "_gen") {
			continue
		}

		for _, decl := range file.Decls {
			gen, ok := decl.(*ast.GenDecl)
			if !ok || gen.Tok != token.TYPE {
				continue
			}
			for _, spec := range gen.Specs {
				ts, ok := spec.(*ast.TypeSpec)
				if !ok {
					continue
				}
				st, ok := ts.Type.(*ast.StructType)
				if !ok {
					continue
				}

				fields := make(map[string]string)
				for _, field := range st.Fields.List {
					if field.Doc == nil || len(field.Names) == 0 {
						continue
					}
					// field.Doc is the comment group directly above the field.
					// field.Comment is text comment on the same line (not what we want).
					comment := strings.TrimSpace(field.Doc.Text())
					if comment != "" {
						fields[field.Names[0].Name] = comment
					}
				}
				if len(fields) > 0 {
					result[ts.Name.Name] = fields
				}
			}
		}
	}
	return result
}

func setterForField(structName string, f *types.Var, shouldMerge *types.Interface, futureOptions map[string]bool, comments map[string]map[string]string) (setterDef, bool) {
	method := "Set" + f.Name()
	comment := ""
	if structComments, ok := comments[structName]; ok {
		if c, ok := structComments[f.Name()]; ok {
			comment = strings.ReplaceAll(c, "\n", "\n// ")
		}
	}

	switch t := f.Type().(type) {

	case *types.Pointer:
		named, isNamed := t.Elem().(*types.Named)

		// Nested Options child → variadic builder setter using Merge.
		isOption := false
		if isNamed {
			isOption = futureOptions[named.Obj().Name()]
		}
		if isOption || (types.Implements(t, shouldMerge) && isNamed) {
			paramType := fmt.Sprintf("Builder[%s]", named.Obj().Name())
			if strings.HasSuffix(named.Obj().Name(), "Options") {
				paramType = strings.TrimSuffix(named.Obj().Name(), "Options") + "Option"
			}

			return setterDef{
				Comment:           comment,
				Method:            "Update" + f.Name(),
				Field:             f.Name(),
				ParamType:         paramType,
				IsVariadicBuilder: true,
			}, true
		}
		// *scalar → value setter (we take v T and store &v)
		if param := typeStr(t.Elem()); param != "" {
			isDirect := false
			if _, ok := t.Elem().Underlying().(*types.Basic); !ok {
				// Non-basic types (structs, etc.) should be passed by pointer
				param = "*" + param
				isDirect = true
			}
			return setterDef{Comment: comment, Method: method, Field: f.Name(), ParamType: param, IsDirectAssign: isDirect}, true
		}

	case *types.Map:
		// map[K]V → value setter (stored directly, no address-of)
		if param := typeStr(t); param != "" {
			return setterDef{Comment: comment, Method: method, Field: f.Name(), ParamType: param, IsDirectAssign: true}, true
		}

	case *types.Slice:
		// []T → variadic setter (v ...T, stored directly)
		if elem := typeStr(t.Elem()); elem != "" {
			return setterDef{Comment: comment, Method: method, Field: f.Name(), IsSlice: true, ElemType: elem}, true
		}
	case *types.Named:
		// Named interface (e.g. sort.Sortable). Value setter (stored directly)
		if _, ok := t.Underlying().(*types.Interface); ok {
			obj := t.Obj()
			if obj.Pkg() != nil && obj.Pkg().Name() != "options" {
				param := obj.Pkg().Name() + "." + obj.Name()
				return setterDef{Comment: comment, Method: method, Field: f.Name(), ParamType: param, IsDirectAssign: true}, true
			}
		}
	}

	// Unsupported interfaces (e.g. WarningHandler), funcs — skip and let the author
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

// unexportedName returns s with the first word lowercased (so it will not be exported).
func unexportedName(s string) string {
	i, lastRuneSize := 0, 0
	for i < len(s) {
		char, size := utf8.DecodeRuneInString(s[i:])
		// Advance until we hit a lower-case, non-digit char.
		if !unicode.IsUpper(char) && !unicode.IsDigit(char) {
			break
		}
		i, lastRuneSize = i+size, size
	}
	// goes back a rune if it's not the end of the string so something like
	// XYZOptionsBuilder becomes xyzOptionsBuilder and not xyzoptionsBuilder.
	if i > 1 && i > lastRuneSize && i < len(s) {
		i -= lastRuneSize
	}
	return strings.ToLower(s[:i]) + s[i:]
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

const boilerplate = `// Copyright IBM Corp.
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
// Returns all non-nil option fields.
func (o *{{ .Name }}) Children() []any {
	var children []any
	{{- range .Fields }}
	if o.{{ . }} != nil {
		children = append(children, o.{{ . }})
	}
	{{- end }}
	return children
}
{{ end }}`))

// Note on // {{ .Comment }} where .Comment can be empty string:
// The resulting comment won't have a trailing blank line because
// we are running this through gofmt.
var buildersTmpl = template.Must(template.New("builders").Parse(boilerplate + `

package {{ .PkgName }}
{{ range .Data }}{{ $o := . }}
{{ if .GenAlias }}{{ .Alias }}
{{ end }}
// Setters implements Builder[{{ .OptsType }}] allowing the raw struct to be
// passed directly to methods that accept ...Builder[{{$o.OptsType}}].
func (o *{{ .OptsType }}) Setters() []func(*{{ .OptsType }}) {
	return NoopBuilder(o)
}
{{- if .HasBuilder }}
// {{ .BuilderType }} is a builder for {{ .OptsType }}.
type {{ .BuilderType }} struct {
	setters []func(*{{ .OptsType }})
}

// {{ .Constructor }} creates a new builder for [{{ .OptsType }}].
func {{ .Constructor }}() *{{ .BuilderType }} {
	return &{{ .BuilderType }}{}
}

// Setters implements Builder[{{ .OptsType }}].
func (b *{{ .BuilderType }}) Setters() []func(*{{ .OptsType }}) {
	return b.setters
}
{{ range .Setters }}
{{- if .IsVariadicBuilder }}
// {{ .Method }} sets the {{ .Field }} option.
// {{ .Comment }}
func (b *{{ $o.BuilderType }}) {{ .Method }}(v ...{{ .ParamType }}) *{{ $o.BuilderType }} {
	b.setters = append(b.setters, func(o *{{ $o.OptsType }}) {
		{{- if .ContainerField }}
		if o.{{ .ContainerField }} == nil {
			o.{{ .ContainerField }} = &{{ .ContainerType }}{}
		}
		{{- end }}
		MergeInto(&o.{{ if .ContainerField }}{{ .ContainerField }}.{{ end }}{{ .Field }}, v...)
	})
	return b
}
{{ else if .IsSlice }}
// {{ .Method }} sets the {{ .Field }} option.
// {{ .Comment }}
func (b *{{ $o.BuilderType }}) {{ .Method }}(v ...{{ .ElemType }}) *{{ $o.BuilderType }} {
	b.setters = append(b.setters, func(o *{{ $o.OptsType }}) {
		{{- if .ContainerField }}
		if o.{{ .ContainerField }} == nil {
			o.{{ .ContainerField }} = &{{ .ContainerType }}{}
		}
		{{- end }}
		o.{{ if .ContainerField }}{{ .ContainerField }}.{{ end }}{{ .Field }} = v
	})
	return b
}
{{ else if .IsDirectAssign }}
// {{ .Method }} sets the {{ .Field }} option.
// {{ .Comment }}
func (b *{{ $o.BuilderType }}) {{ .Method }}(v {{ .ParamType }}) *{{ $o.BuilderType }} {
	b.setters = append(b.setters, func(o *{{ $o.OptsType }}) {
		{{- if .ContainerField }}
		if o.{{ .ContainerField }} == nil {
			o.{{ .ContainerField }} = &{{ .ContainerType }}{}
		}
		{{- end }}
		o.{{ if .ContainerField }}{{ .ContainerField }}.{{ end }}{{ .Field }} = v
	})
	return b
}
{{ else }}
// {{ .Method }} sets the {{ .Field }} option.
// {{ .Comment }}
func (b *{{ $o.BuilderType }}) {{ .Method }}(v {{ .ParamType }}) *{{ $o.BuilderType }} {
	b.setters = append(b.setters, func(o *{{ $o.OptsType }}) {
		{{- if .ContainerField }}
		if o.{{ .ContainerField }} == nil {
			o.{{ .ContainerField }} = &{{ .ContainerType }}{}
		}
		{{- end }}
		o.{{ if .ContainerField }}{{ .ContainerField }}.{{ end }}{{ .Field }} = &v
	})
	return b
}
{{ end -}}
{{ end }}
{{ end -}}
{{ end }}`))
