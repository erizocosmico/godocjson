package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"go/ast"
	"go/doc"
	"go/parser"
	"go/printer"
	"go/token"
	"log"
	"os"
	"path/filepath"
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

	var files = make([]string, len(pkg.Filenames))
	for i, f := range pkg.Filenames {
		files[i] = removeGoPath(f)
	}
	return &Pkg{
		Doc:        pkg.Doc,
		Name:       pkg.Name,
		ImportPath: pkg.ImportPath,
		Imports:    pkg.Imports,
		Filenames:  files,
		Notes:      pkg.Notes,
		Bugs:       pkg.Bugs,
		Consts:     consts,
		Types:      types,
		Vars:       vars,
		Funcs:      funcs,
	}
}

type Pos struct {
	Start *FilePos
	End   *FilePos
}

func NewPos(node ast.Node, fset *token.FileSet) *Pos {
	return &Pos{
		Start: NewFilePos(node.Pos(), fset),
		End:   NewFilePos(node.End(), fset),
	}
}

type FilePos struct {
	Line   int
	Column int
	File   string
}

func NewFilePos(pos token.Pos, fset *token.FileSet) *FilePos {
	p := fset.Position(pos)
	return &FilePos{
		Line:   p.Line,
		Column: p.Column,
		File:   removeGoPath(p.Filename),
	}
}

type Type struct {
	Kind string
	Doc  string
	Name string
	Decl string
	Pos  *Pos

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
		Kind:    "type",
		Doc:     typ.Doc,
		Name:    typ.Name,
		Decl:    buf.String(),
		Consts:  consts,
		Vars:    vars,
		Funcs:   funcs,
		Methods: methods,
		Pos:     NewPos(typ.Decl, fset),
	}
}

type Value struct {
	Kind  string
	Doc   string
	Names []string
	Decl  string
	Pos   *Pos
}

func NewValue(val *doc.Value, fset *token.FileSet) *Value {
	var buf bytes.Buffer
	printer.Fprint(&buf, fset, val.Decl)
	return &Value{
		Kind:  "value",
		Doc:   val.Doc,
		Names: val.Names,
		Decl:  buf.String(),
		Pos:   NewPos(val.Decl, fset),
	}
}

type Func struct {
	Kind string
	Doc  string
	Name string
	Decl string

	Recv  string
	Orig  string
	Level int

	Pos *Pos
}

func NewFunc(fn *doc.Func, fset *token.FileSet) *Func {
	var buf bytes.Buffer
	printer.Fprint(&buf, fset, fn.Decl)
	return &Func{
		Kind:  "func",
		Doc:   fn.Doc,
		Name:  fn.Name,
		Recv:  fn.Recv,
		Orig:  fn.Orig,
		Level: fn.Level,
		Decl:  buf.String(),
		Pos:   NewPos(fn.Decl, fset),
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

	fset := token.NewFileSet()
	pkg, err := parsePackage(pkgName, fset)
	if err != nil {
		log.Fatal(err)
	}

	docPkg := doc.New(pkg, pkgName, 0)
	docPkg.Filter(func(name string) bool {
		return !strings.HasPrefix(name, "Test")
	})

	bytes, err := json.MarshalIndent(NewPkg(docPkg, fset), "", "\t")
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println(string(bytes))
}

func parsePackage(pkgName string, fset *token.FileSet) (*ast.Package, error) {
	srcDir, err := parseutil.DefaultGoPath.Abs(pkgName)
	if err != nil {
		return nil, err
	}

	pkgs, err := parser.ParseDir(fset, srcDir, func(fi os.FileInfo) bool {
		return !strings.HasSuffix(fi.Name(), "_test.go")
	}, parser.ParseComments)
	if err != nil {
		return nil, err
	}

	var pkg *ast.Package
	for name, p := range pkgs {
		if !strings.HasSuffix(name, "_test") {
			pkg = p
		}
	}

	if pkg == nil {
		return nil, errors.New("no package found at given package name")
	}

	return pkg, nil
}

func removeGoPath(path string) string {
	for _, p := range parseutil.DefaultGoPath {
		p = filepath.Join(p, "src")
		if strings.HasPrefix(path, p) {
			return strings.TrimLeft(path[len(p):], "/\\")
		}
	}
	return path
}
