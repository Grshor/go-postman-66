// Command go-postman-66 translates Go route registrations into a
// Postman Collection v2.1 JSON file.
package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/Grshor/go-postman-66/internal/postman"
	"github.com/Grshor/go-postman-66/internal/scan"
)

func main() {
	os.Exit(run())
}

func run() int {
	dir := flag.String("dir", ".", "directory to scan recursively")
	out := flag.String("out", "postman_collection.json", "output file ('-' for stdout)")
	name := flag.String("name", "API", "collection name")
	baseURL := flag.String("base-url", "http://localhost:8080", "value for the {{base_url}} variable")
	flag.Usage = func() {
		fmt.Fprintln(os.Stderr, "go-postman-66 — translate Go route registrations into a Postman collection")
		fmt.Fprintln(os.Stderr, "\nUsage:")
		fmt.Fprintln(os.Stderr, "  go-postman-66 [-dir .] [-out postman_collection.json] [-name API] [-base-url http://localhost:8080]")
		fmt.Fprintln(os.Stderr, "\nFlags:")
		flag.PrintDefaults()
	}
	flag.Parse()

	routes, warnings, err := scan.Scan(*dir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "go-postman-66: scan: %v\n", err)
		return 2
	}
	for _, w := range warnings {
		fmt.Fprintln(os.Stderr, "warning:", w)
	}

	coll, err := postman.Build(routes, postman.Options{Name: *name, BaseURL: *baseURL})
	if err != nil {
		fmt.Fprintf(os.Stderr, "go-postman-66: build: %v\n", err)
		return 2
	}
	data, err := postman.Render(coll)
	if err != nil {
		fmt.Fprintf(os.Stderr, "go-postman-66: render: %v\n", err)
		return 2
	}

	if *out == "-" {
		os.Stdout.Write(data)
	} else {
		if err := os.WriteFile(*out, data, 0o644); err != nil {
			fmt.Fprintf(os.Stderr, "go-postman-66: write %s: %v\n", *out, err)
			return 2
		}
	}

	pkgs := map[string]bool{}
	for _, r := range routes {
		pkgs[r.Package] = true
	}
	fmt.Fprintf(os.Stderr, "go-postman-66: %d routes in %d package(s) → %s\n", len(routes), len(pkgs), *out)
	return 0
}
