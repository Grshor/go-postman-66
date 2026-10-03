// Package scan walks Go source trees and extracts HTTP route registrations:
// net/http-style HandleFunc/Handle calls (including Go 1.22 method patterns),
// and method-style router calls (gin, chi, echo: r.GET; fiber: app.Get).
package scan

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"path/filepath"
	"sort"
	"strings"
)

// Route is one detected endpoint registration.
type Route struct {
	Method    string // "GET", "POST", ... or "" when the pattern carries no method
	Path      string // raw path pattern as written in the source
	Handler   string // handler expression text ("getUser", "admin.Handler", "<anonymous>")
	Package   string // Go package name the registration lives in
	File      string // path relative to the scan root
	Line      int    // 1-based line of the call
	Desc      string // doc comment of the handler function, first paragraph
	Framework string // "handlefunc" | "router" | "fiber"
}

// Scan walks root, parses every non-test Go file and returns the detected
// routes plus non-fatal warnings (e.g. non-literal path patterns).
func Scan(root string) ([]Route, []string, error) {
	var files []string
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			name := d.Name()
			if name == ".git" || name == "vendor" || name == "testdata" || name == "node_modules" {
				return filepath.SkipDir
			}
			return nil
		}
		if strings.HasSuffix(path, ".go") && !strings.HasSuffix(path, "_test.go") {
			files = append(files, path)
		}
		return nil
	})
	if err != nil {
		return nil, nil, err
	}
	sort.Strings(files)

	fset := token.NewFileSet()
	var routes []Route
	var warnings []string

	for _, file := range files {
		af, err := parser.ParseFile(fset, file, nil, parser.ParseComments)
		if err != nil {
			warnings = append(warnings, fmt.Sprintf("%s: parse: %v", file, err))
			continue
		}
		rel, relErr := filepath.Rel(root, file)
		if relErr != nil {
			rel = file
		}

		// handler function name -> doc comment (first paragraph)
		docs := map[string]string{}
		for _, decl := range af.Decls {
			fd, ok := decl.(*ast.FuncDecl)
			if !ok || fd.Doc == nil {
				continue
			}
			text := strings.TrimSpace(fd.Doc.Text())
			if i := strings.Index(text, "\n"); i > 0 {
				text = strings.TrimSpace(text[:i])
			}
			docs[fd.Name.Name] = text
		}

		pkg := af.Name.Name
		rel = filepath.ToSlash(rel)

		ast.Inspect(af, func(n ast.Node) bool {
			call, ok := n.(*ast.CallExpr)
			if !ok || len(call.Args) < 2 {
				return true
			}

			var method string
			framework := ""
			switch fun := call.Fun.(type) {
			case *ast.SelectorExpr:
				m := fun.Sel.Name
				switch {
				case m == "HandleFunc" || m == "Handle":
					framework = "handlefunc"
				case isUpperVerb(m):
					framework = "router"
					if m == "Any" || m == "All" {
						method = "" // router catch-all: no single verb
					} else {
						method = strings.ToUpper(m)
					}
				default:
					return true
				}
			default:
				return true
			}

			patLit, ok := call.Args[0].(*ast.BasicLit)
			if !ok || patLit.Kind != token.STRING {
				warnings = append(warnings, fmt.Sprintf("%s:%d: non-literal path pattern skipped (%s)",
					rel, fset.Position(call.Pos()).Line, exprText(call.Args[0])))
				return true
			}
			pattern := strings.Trim(patLit.Value, "`\"")

			patMethod, path := splitMethodPattern(pattern)
			if patMethod != "" {
				method = patMethod
			}
			if framework == "handlefunc" {
				method = patMethod // no verb in the selector itself
			}

			handler := exprText(call.Args[1])
			desc := ""
			if id, ok := call.Args[1].(*ast.Ident); ok {
				desc = docs[id.Name]
			}
			if method == "" {
				method = "GET"
				if desc == "" {
					desc = "Method not detected — the route may accept any method."
				} else {
					desc += " (method not detected — may accept any method)"
				}
			}

			routes = append(routes, Route{
				Method:    strings.ToUpper(method),
				Path:      path,
				Handler:   handler,
				Package:   pkg,
				File:      rel,
				Line:      fset.Position(call.Pos()).Line,
				Desc:      desc,
				Framework: framework,
			})
			return true
		})
	}

	sort.Slice(routes, func(i, j int) bool {
		a, b := routes[i], routes[j]
		if a.Package != b.Package {
			return a.Package < b.Package
		}
		if a.File != b.File {
			return a.File < b.File
		}
		return a.Line < b.Line
	})
	return routes, warnings, nil
}

// splitMethodPattern handles Go 1.22 ServeMux patterns: "GET /users/{id}".
func splitMethodPattern(pattern string) (method, path string) {
	fields := strings.Fields(pattern)
	if len(fields) >= 2 && isUpperVerb(fields[0]) {
		return fields[0], strings.TrimSpace(pattern[len(fields[0]):])
	}
	return "", strings.TrimSpace(pattern)
}

// isUpperVerb reports whether a selector name looks like a router
// registration method: HTTP verbs in either case, plus Any/All catch-alls.
func isUpperVerb(s string) bool {
	switch s {
	case "GET", "POST", "PUT", "DELETE", "PATCH", "HEAD", "OPTIONS",
		"Get", "Post", "Put", "Delete", "Patch", "Head", "Options",
		"ANY", "Any", "ALL", "All":
		return true
	}
	return false
}

func exprText(e ast.Expr) string {
	switch v := e.(type) {
	case *ast.Ident:
		return v.Name
	case *ast.SelectorExpr:
		return exprText(v.X) + "." + v.Sel.Name
	case *ast.FuncLit:
		return "<anonymous>"
	default:
		return "<expr>"
	}
}
