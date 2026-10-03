package postman

import (
	"encoding/json"
	"os"
	"strings"
	"testing"

	"github.com/Grshor/go-postman-66/internal/scan"
)

func TestPathSegments(t *testing.T) {
	cases := []struct {
		in   string
		want []string
	}{
		{"/users/{id}", []string{"users", ":id"}},
		{"/files/{path...}", []string{"files", ":path"}},
		{"/fiber/tasks/:id", []string{"fiber", "tasks", ":id"}},
		{"/", []string{}},
		{"/a/b", []string{"a", "b"}},
	}
	for _, c := range cases {
		got := pathSegments(c.in)
		if len(got) != len(c.want) {
			t.Errorf("pathSegments(%q) = %v, want %v", c.in, got, c.want)
			continue
		}
		for i := range got {
			if got[i] != c.want[i] {
				t.Errorf("pathSegments(%q)[%d] = %q, want %q", c.in, i, got[i], c.want[i])
			}
		}
	}
}

func TestBuildDeterministic(t *testing.T) {
	routes := []scan.Route{
		{Method: "POST", Path: "/b", Handler: "hB", Package: "p2", File: "b.go", Line: 10},
		{Method: "GET", Path: "/a", Handler: "hA", Package: "p2", File: "a.go", Line: 5},
		{Method: "GET", Path: "/z", Handler: "hZ", Package: "p1", File: "z.go", Line: 1},
	}
	c1, err := Build(routes, Options{Name: "T", BaseURL: "http://x"})
	if err != nil {
		t.Fatal(err)
	}
	c2, err := Build(routes, Options{Name: "T", BaseURL: "http://x"})
	if err != nil {
		t.Fatal(err)
	}
	b1, _ := Render(c1)
	b2, _ := Render(c2)
	if string(b1) != string(b2) {
		t.Fatal("Build/Render is not deterministic")
	}

	// packages sorted, requests sorted inside folders
	if c1.Item[0].Name != "p1" || c1.Item[1].Name != "p2" {
		t.Errorf("folders not sorted: %q, %q", c1.Item[0].Name, c1.Item[1].Name)
	}
	if c1.Item[1].Item[0].Name != "GET /a" || c1.Item[1].Item[1].Name != "POST /b" {
		t.Errorf("requests not sorted inside folder")
	}
}

func TestGolden(t *testing.T) {
	routes, _, err := scan.Scan("../../testdata/testapp")
	if err != nil {
		t.Fatal(err)
	}
	coll, err := Build(routes, Options{Name: "testapp", BaseURL: "http://localhost:8080"})
	if err != nil {
		t.Fatal(err)
	}
	got, err := Render(coll)
	if err != nil {
		t.Fatal(err)
	}

	const golden = "../../testdata/testapp.golden.json"
	if os.Getenv("UPDATE_GOLDEN") != "" {
		if err := os.WriteFile(golden, got, 0o644); err != nil {
			t.Fatal(err)
		}
		return
	}
	want, err := os.ReadFile(golden)
	if err != nil {
		t.Fatalf("golden file missing (run with UPDATE_GOLDEN=1): %v", err)
	}
	if strings.TrimSpace(string(want)) != strings.TrimSpace(string(got)) {
		t.Fatalf("output differs from golden file; run with UPDATE_GOLDEN=1 to refresh")
	}
}

func TestRenderIsV21JSON(t *testing.T) {
	coll, err := Build(nil, Options{Name: "T"})
	if err != nil {
		t.Fatal(err)
	}
	b, err := Render(coll)
	if err != nil {
		t.Fatal(err)
	}
	var m map[string]any
	if err := json.Unmarshal(b, &m); err != nil {
		t.Fatalf("not valid JSON: %v", err)
	}
	info, _ := m["info"].(map[string]any)
	if info["schema"] != schema {
		t.Errorf("schema = %v, want %v", info["schema"], schema)
	}
}
