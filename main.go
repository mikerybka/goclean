package main

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/printer"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"unicode"
)

func main() {
	fset := token.NewFileSet()
	pkgs, err := parser.ParseDir(fset, ".", filter, parser.ParseComments)
	if err != nil {
		fmt.Println("Error:", err)
		return
	}
	if len(pkgs) != 1 {
		fmt.Println("Error: expected exactly one package to be defined in the current directory: found", len(pkgs))
		return
	}
	tmpDir, err := os.MkdirTemp("", "")
	if err != nil {
		fmt.Println("Error:", err)
		return
	}
	defer os.RemoveAll(tmpDir)
	for _, pkg := range pkgs {
		// Process each file and write new files leaving out type methods.
		for _, file := range pkg.Files {
			headerBuffer := &strings.Builder{}
			headerBuffer.WriteString("package " + pkg.Name + "\n\n")
			for _, decl := range file.Decls {
				switch decl := decl.(type) {
				case *ast.GenDecl:
					if decl.Tok == token.IMPORT {
						headerBuffer.WriteString("import (\n")
						for _, spec := range decl.Specs {
							switch spec := spec.(type) {
							case *ast.ImportSpec:
								headerBuffer.WriteString("\t")
								printer.Fprint(headerBuffer, fset, spec)
								headerBuffer.WriteString("\n")
							}
						}
						headerBuffer.WriteString(")\n\n")
					}
				}
			}
			header := headerBuffer.String()
			for _, decl := range file.Decls {
				switch decl := decl.(type) {
				case *ast.FuncDecl:
				case *ast.GenDecl:
					for _, spec := range decl.Specs {
						switch spec := spec.(type) {
						case *ast.ValueSpec:
							for i, name := range spec.Names {
								filename := filepath.Join(tmpDir, goFilename(name.Name))
								f, err := os.Create(filename)
								if err != nil {
									fmt.Println("Error:", err)
									return
								}
								defer f.Close()
								fmt.Fprintf(f, header)
								switch decl.Tok {
								case token.CONST:
									fmt.Fprintf(f, "const ")
								case token.VAR:
									fmt.Fprintf(f, "var ")
								}
								printer.Fprint(f, fset, name)
								fmt.Fprintf(f, " ")
								printer.Fprint(f, fset, spec.Type)
								if len(spec.Values) > i {
									fmt.Fprintf(f, " = ")
									printer.Fprint(f, fset, spec.Values[i])
								}
								fmt.Fprintf(f, "\n")
							}
						case *ast.TypeSpec:
						}
					}
				}
			}
		}
		// Process the files again just for the methods.
		// for _, file := range pkg.Files {
		// 	for _, decl := range file.Decls {
		// 		switch decl := decl.(type) {
		// 		case *ast.FuncDecl:
		// 		}
		// 	}
		// }
	}

	// TODO: copy the files from tmpDir to the current directory.
	entry, err := os.ReadDir(tmpDir)
	if err != nil {
		panic(err)
	}
	for _, info := range entry {
		b, err := os.ReadFile(filepath.Join(tmpDir, info.Name()))
		if err != nil {
			panic(err)
		}
		err = os.WriteFile(filepath.Join("..", "test2", info.Name()), b, 0644)
		if err != nil {
			panic(err)
		}
	}

	// TODO: run gofmt on the result.
}

func filter(info fs.FileInfo) bool {
	return !strings.HasSuffix(info.Name(), "_test.go")
}

func goFilename(name string) string {
	var buf strings.Builder
	for i, r := range name {
		if i > 0 && unicode.IsUpper(r) {
			buf.WriteRune('_')
		}
		buf.WriteRune(unicode.ToLower(r))
	}
	return buf.String() + ".go"
}
