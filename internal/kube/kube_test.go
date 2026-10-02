package kube

import (
	"errors"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/clientcmd/api"
)

const devAndProd = `apiVersion: v1
kind: Config
current-context: dev
clusters:
- name: dev-cluster
  cluster: {server: "https://dev.example:6443"}
- name: prod-cluster
  cluster: {server: "https://prod.example:6443"}
users:
- name: me
  user: {token: secret}
contexts:
- name: dev
  context: {cluster: dev-cluster, user: me, namespace: web}
- name: prod
  context: {cluster: prod-cluster, user: me}
`

// devFile and prodFile split devAndProd so tests can check which file of a
// merged KUBECONFIG each change lands in.
const devFile = `apiVersion: v1
kind: Config
current-context: dev
clusters:
- name: dev-cluster
  cluster: {server: "https://dev.example:6443"}
users:
- name: me
  user: {token: secret}
contexts:
- name: dev
  context: {cluster: dev-cluster, user: me, namespace: web}
`

const prodFile = `apiVersion: v1
kind: Config
clusters:
- name: prod-cluster
  cluster: {server: "https://prod.example:6443"}
contexts:
- name: prod
  context: {cluster: prod-cluster, user: me}
`

func writeKubeconfigs(t *testing.T, contents ...string) []string {
	t.Helper()
	dir := t.TempDir()
	paths := make([]string, len(contents))
	for i, c := range contents {
		paths[i] = filepath.Join(dir, "config"+string(rune('a'+i)))
		if err := os.WriteFile(paths[i], []byte(c), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	t.Setenv("KUBECONFIG", strings.Join(paths, string(filepath.ListSeparator)))
	return paths
}

func readFile(t *testing.T, path string) *api.Config {
	t.Helper()
	cfg, err := clientcmd.LoadFromFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return cfg
}

func TestLoadMissingKubeconfig(t *testing.T) {
	t.Setenv("KUBECONFIG", filepath.Join(t.TempDir(), "absent"))
	cfg, err := Load(clientcmd.NewDefaultPathOptions())
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if _, err := CurrentContext(cfg); !errors.Is(err, ErrNoCurrentContext) {
		t.Errorf("CurrentContext error = %v, want ErrNoCurrentContext", err)
	}
}

func TestCurrentContext(t *testing.T) {
	contexts := map[string]*api.Context{
		"dev":  {Cluster: "dev-cluster", Namespace: "web"},
		"prod": {Cluster: "prod-cluster"},
	}
	tests := []struct {
		name    string
		current string
		want    Current
		wantErr bool
	}{
		{"namespace set", "dev", Current{"dev", "dev-cluster", "web"}, false},
		{"namespace defaults", "prod", Current{"prod", "prod-cluster", "default"}, false},
		{"unset", "", Current{}, true},
		{"dangling", "gone", Current{}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := CurrentContext(&api.Config{CurrentContext: tt.current, Contexts: contexts})
			if tt.wantErr {
				if !errors.Is(err, ErrNoCurrentContext) {
					t.Fatalf("error = %v, want ErrNoCurrentContext", err)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if got != tt.want {
				t.Errorf("got %+v, want %+v", got, tt.want)
			}
		})
	}
}

func TestListing(t *testing.T) {
	cfg := &api.Config{Contexts: map[string]*api.Context{
		"b": {Namespace: "web"},
		"a": {Namespace: "default"},
		"c": {},
		"d": {Namespace: "api"},
		"e": {Namespace: "web"},
	}}
	if got, want := ContextNames(cfg), []string{"a", "b", "c", "d", "e"}; !slices.Equal(got, want) {
		t.Errorf("ContextNames = %v, want %v", got, want)
	}
	if got, want := Namespaces(cfg), []string{"api", "default", "web"}; !slices.Equal(got, want) {
		t.Errorf("Namespaces = %v, want %v", got, want)
	}
}

func TestUseContext(t *testing.T) {
	paths := writeKubeconfigs(t, devAndProd)
	access := clientcmd.NewDefaultPathOptions()

	if err := UseContext(access, "prod"); err != nil {
		t.Fatalf("UseContext: %v", err)
	}
	if got := readFile(t, paths[0]).CurrentContext; got != "prod" {
		t.Errorf("current-context = %q, want prod", got)
	}
	if err := UseContext(access, "missing"); err == nil {
		t.Error("UseContext(missing) succeeded, want error")
	}
}

func TestUseContextMerged(t *testing.T) {
	paths := writeKubeconfigs(t, devFile, prodFile)

	if err := UseContext(clientcmd.NewDefaultPathOptions(), "prod"); err != nil {
		t.Fatalf("UseContext: %v", err)
	}
	if got := readFile(t, paths[0]).CurrentContext; got != "prod" {
		t.Errorf("first file current-context = %q, want prod", got)
	}
	assertUnchanged(t, paths[1], prodFile)
}

func TestSetNamespace(t *testing.T) {
	paths := writeKubeconfigs(t, devAndProd)
	access := clientcmd.NewDefaultPathOptions()

	if err := SetNamespace(access, "payments"); err != nil {
		t.Fatalf("SetNamespace: %v", err)
	}
	cfg := readFile(t, paths[0])
	if got := cfg.Contexts["dev"].Namespace; got != "payments" {
		t.Errorf("dev namespace = %q, want payments", got)
	}
	if got := cfg.Contexts["prod"].Namespace; got != "" {
		t.Errorf("prod namespace = %q, want unchanged", got)
	}

	for _, bad := range []string{"Payments", "-web", "a_b", strings.Repeat("a", 64)} {
		if err := SetNamespace(access, bad); err == nil {
			t.Errorf("SetNamespace(%q) succeeded, want error", bad)
		}
	}
}

func TestSetNamespaceMerged(t *testing.T) {
	paths := writeKubeconfigs(t, devFile, prodFile)
	access := clientcmd.NewDefaultPathOptions()
	if err := UseContext(access, "prod"); err != nil {
		t.Fatal(err)
	}
	before, err := os.ReadFile(paths[0])
	if err != nil {
		t.Fatal(err)
	}

	if err := SetNamespace(access, "payments"); err != nil {
		t.Fatalf("SetNamespace: %v", err)
	}
	if got := readFile(t, paths[1]).Contexts["prod"].Namespace; got != "payments" {
		t.Errorf("prod namespace in second file = %q, want payments", got)
	}
	assertUnchanged(t, paths[0], string(before))
}

func TestSetNamespaceNoCurrentContext(t *testing.T) {
	writeKubeconfigs(t, prodFile)
	err := SetNamespace(clientcmd.NewDefaultPathOptions(), "web")
	if !errors.Is(err, ErrNoCurrentContext) {
		t.Errorf("error = %v, want ErrNoCurrentContext", err)
	}
}

func assertUnchanged(t *testing.T, path, want string) {
	t.Helper()
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != want {
		t.Errorf("%s was modified:\n%s", filepath.Base(path), got)
	}
}
