package main

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/printer"
	"go/token"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"unicode"
)

var goimports string

func init() {
	var err error
	goimports, err = exec.LookPath("goimports")
	if err != nil {
		fmt.Println("Error:", err)
		os.Exit(1)
	}
}

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
					// Skip type methods.
					if decl.Recv != nil {
						continue
					}
					filename := filepath.Join(tmpDir, goFilename(decl.Name.Name))
					f, err := os.Create(filename)
					if err != nil {
						fmt.Println("Error:", err)
						return
					}
					defer f.Close()
					fmt.Fprintf(f, header)
					printer.Fprint(f, fset, decl)
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
								writeCommentGroup(f, decl.Doc)
								writeCommentGroup(f, spec.Doc)
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
								writeCommentGroup(f, spec.Comment)
							}
						case *ast.TypeSpec:
							filename := filepath.Join(tmpDir, goFilename(spec.Name.Name))
							f, err := os.Create(filename)
							if err != nil {
								fmt.Println("Error:", err)
								return
							}
							defer f.Close()
							fmt.Fprintf(f, header)
							writeCommentGroup(f, decl.Doc)
							writeCommentGroup(f, spec.Doc)
							fmt.Fprintf(f, "type ")
							printer.Fprint(f, fset, spec.Name)
							fmt.Fprintf(f, " ")
							printer.Fprint(f, fset, spec.Type)
							writeCommentGroup(f, spec.Comment)
						}
					}
				}
			}
		}
		// Process the files again just for the methods.
		for _, file := range pkg.Files {
			for _, decl := range file.Decls {
				switch decl := decl.(type) {
				case *ast.FuncDecl:
					// Skip non-methods.
					if decl.Recv == nil {
						continue
					}
					filename := filepath.Join(tmpDir, goFilename(getTypeName(decl.Recv.List[0].Type)))
					f, err := os.OpenFile(filename, os.O_APPEND|os.O_WRONLY, 0644)
					if err != nil {
						fmt.Println("Error:", err)
						return
					}
					defer f.Close()
					fmt.Fprintf(f, "\n")
					printer.Fprint(f, fset, decl)
				}
			}
		}
	}

	err = exec.Command("goimports", "-w", tmpDir).Run()
	if err != nil {
		fmt.Println("Error:", err)
		return
	}

	renameFiles()

	entry, err := os.ReadDir(tmpDir)
	if err != nil {
		panic(err)
	}
	for _, info := range entry {
		b, err := os.ReadFile(filepath.Join(tmpDir, info.Name()))
		if err != nil {
			panic(err)
		}
		err = os.WriteFile(info.Name(), b, 0644)
		if err != nil {
			panic(err)
		}
	}
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

func writeCommentGroup(w io.Writer, comments *ast.CommentGroup) {
	if comments != nil {
		for _, comment := range comments.List {
			fmt.Fprintf(w, "%s\n", comment.Text)
		}
	}
}

func getTypeName(expr ast.Expr) string {
	switch expr := expr.(type) {
	case *ast.Ident:
		return expr.Name
	case *ast.StarExpr:
		return getTypeName(expr.X)
	default:
		panic(fmt.Sprintf("unhandled type: %T", expr))
	}
}

func renameFiles() {
	entry, err := os.ReadDir(".")
	if err != nil {
		panic(err)
	}
	for _, info := range entry {
		name := info.Name()
		if strings.HasSuffix(name, "_test.go") {
			continue
		}
		if strings.HasSuffix(name, ".go") {
			err = os.Rename(name, name+".bak")
			if err != nil {
				panic(err)
			}
		}
	}
}
