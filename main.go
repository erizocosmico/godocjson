package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"go/doc"
	"go/printer"
	"go/token"
	"log"
	"os"
	"strings"

	parseutil "gopkg.in/src-d/go-parse-utils.v1"
)

type Pkg struct {
	Doc        string
	Name       string
	ImportPath string
	Imports    []string
	Filenames  []string
	Notes      map[string][]*doc.Note

	Bugs []string

	Consts []*Value
	Types  []*Type
	Vars   []*Value
	Funcs  []*Func
}

func NewPkg(pkg *doc.Package, fset *token.FileSet) *Pkg {
	var consts = make([]*Value, len(pkg.Consts))
	for i, c := range pkg.Consts {
		consts[i] = NewValue(c, fset)
	}

	var vars = make([]*Value, len(pkg.Vars))
	for i, v := range pkg.Vars {
		vars[i] = NewValue(v, fset)
	}

	var funcs = make([]*Func, len(pkg.Funcs))
	for i, f := range pkg.Funcs {
		funcs[i] = NewFunc(f, fset)
	}

	var types = make([]*Type, len(pkg.Types))
	for i, t := range pkg.Types {
		types[i] = NewType(t, fset)
	}
	return &Pkg{
		Doc:        pkg.Doc,
		Name:       pkg.Name,
		ImportPath: pkg.ImportPath,
		Imports:    pkg.Imports,
		Filenames:  pkg.Filenames,
		Notes:      pkg.Notes,
		Bugs:       pkg.Bugs,
		Consts:     consts,
		Types:      types,
		Vars:       vars,
		Funcs:      funcs,
	}
}

type Type struct {
	Doc  string
	Name string
	Decl string

	Consts  []*Value
	Vars    []*Value
	Funcs   []*Func
	Methods []*Func
}

func NewType(typ *doc.Type, fset *token.FileSet) *Type {
	var buf bytes.Buffer
	printer.Fprint(&buf, fset, typ.Decl)

	var consts = make([]*Value, len(typ.Consts))
	for i, c := range typ.Consts {
		consts[i] = NewValue(c, fset)
	}

	var vars = make([]*Value, len(typ.Vars))
	for i, v := range typ.Vars {
		vars[i] = NewValue(v, fset)
	}

	var funcs = make([]*Func, len(typ.Funcs))
	for i, f := range typ.Funcs {
		funcs[i] = NewFunc(f, fset)
	}

	var methods = make([]*Func, len(typ.Methods))
	for i, m := range typ.Methods {
		methods[i] = NewFunc(m, fset)
	}

	return &Type{
		Doc:     typ.Doc,
		Name:    typ.Name,
		Decl:    buf.String(),
		Consts:  consts,
		Vars:    vars,
		Funcs:   funcs,
		Methods: methods,
	}
}

type Value struct {
	Doc   string
	Names []string
	Decl  string
}

func NewValue(val *doc.Value, fset *token.FileSet) *Value {
	var buf bytes.Buffer
	printer.Fprint(&buf, fset, val.Decl)
	return &Value{
		Doc:   val.Doc,
		Names: val.Names,
		Decl:  buf.String(),
	}
}

type Func struct {
	Doc  string
	Name string
	Decl string

	Recv  string
	Orig  string
	Level int
}

func NewFunc(fn *doc.Func, fset *token.FileSet) *Func {
	var buf bytes.Buffer
	printer.Fprint(&buf, fset, fn.Decl)
	return &Func{
		Doc:   fn.Doc,
		Name:  fn.Name,
		Recv:  fn.Recv,
		Orig:  fn.Orig,
		Level: fn.Level,
		Decl:  buf.String(),
	}
}

func main() {
	if len(os.Args) != 2 {
		log.Fatal("unexpected number of arguments: expecting one argument with a package name")
	}

	pkgName := os.Args[1]
	if pkgName == "" {
		log.Fatal("-pkg cannot be empty")
	}

	pkg, err := parseutil.PackageAST(pkgName)
	if err != nil {
		log.Fatal(err)
	}

	docPkg := doc.New(pkg, pkgName, 0)
	docPkg.Filter(func(name string) bool {
		return !strings.HasPrefix(name, "Test")
	})

	bytes, err := json.MarshalIndent(NewPkg(docPkg, token.NewFileSet()), "", "\t")
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println(string(bytes))
}
