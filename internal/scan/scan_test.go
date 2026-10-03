package scan

import (
	"testing"
)

func TestScanTestApp(t *testing.T) {
	routes, warnings, err := Scan("../../testdata/testapp")
	if err != nil {
		t.Fatalf("scan: %v", err)
	}
	for _, w := range warnings {
		t.Logf("warning: %s", w)
	}

	type key struct{ method, path string }
	got := map[key]Route{}
	for _, r := range routes {
		got[key{r.Method, r.Path}] = r
	}

	want := []struct {
		method, path, handler, framework string
		hasDesc                          bool
	}{
		{"GET", "/health", "<anonymous>", "handlefunc", false},
		{"POST", "/users", "getUsers", "handlefunc", true},
		{"DELETE", "/users/{id}", "deleteUser", "handlefunc", true},
		{"GET", "/assets/", "<expr>", "handlefunc", false},
		{"GET", "/gorilla/orders", "createOrder", "handlefunc", false}, // .Methods() chain not resolved in v0.1
		{"GET", "/gin/items", "listItems", "router", false},
		{"POST", "/gin/items", "ginCreateItem", "router", false},
		{"GET", "/chi/reports", "reportsIndex", "router", false},
		{"GET", "/echo/ping", "pong", "router", false},
		{"GET", "/fiber/tasks/:id", "taskPage", "router", false},
	}

	if len(routes) != len(want) {
		var gotList []string
		for _, r := range routes {
			gotList = append(gotList, r.Method+" "+r.Path)
		}
		t.Fatalf("got %d routes, want %d: %v", len(routes), len(want), gotList)
	}
	for _, w := range want {
		r, ok := got[key{w.method, w.path}]
		if !ok {
			t.Errorf("route %s %s not found", w.method, w.path)
			continue
		}
		if r.Handler != w.handler {
			t.Errorf("%s %s handler = %q, want %q", w.method, w.path, r.Handler, w.handler)
		}
		if r.Framework != w.framework {
			t.Errorf("%s %s framework = %q, want %q", w.method, w.path, r.Framework, w.framework)
		}
		if w.hasDesc && r.Desc == "" {
			t.Errorf("%s %s: expected a doc-comment description", w.method, w.path)
		}
	}
}

func TestSplitMethodPattern(t *testing.T) {
	cases := []struct{ in, method, path string }{
		{"GET /users/{id}", "GET", "/users/{id}"},
		{"POST /users", "POST", "/users"},
		{"/plain", "", "/plain"},
		{"GET /", "GET", "/"},
	}
	for _, c := range cases {
		m, p := splitMethodPattern(c.in)
		if m != c.method || p != c.path {
			t.Errorf("splitMethodPattern(%q) = (%q, %q), want (%q, %q)", c.in, m, p, c.method, c.path)
		}
	}
}
