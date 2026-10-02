package main

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"testing"

	"k8s.io/client-go/tools/clientcmd"

	"github.com/petricbranko/kubebar/internal/config"
	"github.com/petricbranko/kubebar/internal/waybar"
)

const kubeconfig = `apiVersion: v1
kind: Config
current-context: %s
clusters:
- name: dev-cluster
  cluster: {server: "https://dev.example:6443"}
- name: prod-cluster
  cluster: {server: "https://prod.example:6443"}
contexts:
- name: dev
  context: {cluster: dev-cluster, namespace: web}
- name: prod-eu
  context: {cluster: prod-cluster}
`

var testConfig = config.Config{
	ProdPatterns: []*regexp.Regexp{regexp.MustCompile("prod")},
	MenuCommand:  "unused",
	WaybarSignal: 8,
}

func setupKubeconfig(t *testing.T, current string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "config")
	content := fmt.Sprintf(kubeconfig, current)
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("KUBECONFIG", path)
	return path
}

func TestStatus(t *testing.T) {
	tests := []struct {
		name    string
		current string
		noFile  bool
		want    waybar.Status
	}{
		{
			name:    "ok",
			current: "dev",
			want:    waybar.Status{Text: "dev/web", Tooltip: "Context: dev\nCluster: dev-cluster\nNamespace: web", Class: "ok"},
		},
		{
			name:    "prod",
			current: "prod-eu",
			want:    waybar.Status{Text: "prod-eu/default", Tooltip: "Context: prod-eu\nCluster: prod-cluster\nNamespace: default", Class: "prod"},
		},
		{
			name:    "no current context",
			current: `""`,
			want:    waybar.Status{Text: "no context", Tooltip: "no current context", Class: "none"},
		},
		{
			name:    "dangling current context",
			current: "gone",
			want:    waybar.Status{Text: "no context", Tooltip: `no current context: context "gone" not found`, Class: "none"},
		},
		{
			name:   "no kubeconfig",
			noFile: true,
			want:   waybar.Status{Text: "no context", Tooltip: "no current context", Class: "none"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.noFile {
				t.Setenv("KUBECONFIG", filepath.Join(t.TempDir(), "absent"))
			} else {
				setupKubeconfig(t, tt.current)
			}
			if got := status(testConfig, clientcmd.NewDefaultPathOptions()); got != tt.want {
				t.Errorf("got  %+v\nwant %+v", got, tt.want)
			}
		})
	}
}

func TestRunStatusAlwaysSucceeds(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("KUBECONFIG", filepath.Join(t.TempDir(), "absent"))
	var out strings.Builder
	if code := run([]string{"status"}, &out, io.Discard); code != 0 {
		t.Errorf("exit code = %d, want 0", code)
	}
	if want := `{"text":"no context","tooltip":"no current context","class":"none"}` + "\n"; out.String() != want {
		t.Errorf("output = %s, want %s", out.String(), want)
	}
}

type recorder struct {
	prompts   []string
	items     [][]string
	refreshes int
	notices   []string
}

func newTestApp(rec *recorder, choice string, menuErr error) *app {
	return &app{
		cfg:    testConfig,
		access: clientcmd.NewDefaultPathOptions(),
		menu: func(prompt string, items []string) (string, error) {
			rec.prompts = append(rec.prompts, prompt)
			rec.items = append(rec.items, items)
			return choice, menuErr
		},
		refresh: func() error { rec.refreshes++; return nil },
		notify: func(summary, body string) error {
			rec.notices = append(rec.notices, body)
			return nil
		},
		stderr: io.Discard,
	}
}

func TestSwitchContext(t *testing.T) {
	tests := []struct {
		name        string
		choice      string
		menuErr     error
		wantCurrent string
		wantRefresh int
		wantNotices int
		wantErr     bool
	}{
		{name: "dev", choice: "dev", wantCurrent: "dev", wantRefresh: 1},
		{name: "prod notifies", choice: "prod-eu", wantCurrent: "prod-eu", wantRefresh: 1, wantNotices: 1},
		{name: "cancelled", choice: "", wantCurrent: "dev"},
		{name: "unknown", choice: "nope", wantCurrent: "dev", wantErr: true},
		{name: "menu fails", menuErr: errors.New("boom"), wantCurrent: "dev", wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := setupKubeconfig(t, "dev")
			rec := &recorder{}
			err := newTestApp(rec, tt.choice, tt.menuErr).switchContext()
			if (err != nil) != tt.wantErr {
				t.Fatalf("error = %v, wantErr %v", err, tt.wantErr)
			}
			if got := currentContext(t, path); got != tt.wantCurrent {
				t.Errorf("current-context = %q, want %q", got, tt.wantCurrent)
			}
			if !slices.Equal(rec.items[0], []string{"dev", "prod-eu"}) {
				t.Errorf("menu items = %v", rec.items[0])
			}
			if rec.refreshes != tt.wantRefresh || len(rec.notices) != tt.wantNotices {
				t.Errorf("refreshes = %d, notices = %d, want %d, %d", rec.refreshes, len(rec.notices), tt.wantRefresh, tt.wantNotices)
			}
		})
	}
}

func TestSwitchNamespace(t *testing.T) {
	tests := []struct {
		name    string
		choice  string
		wantNS  string
		wantErr bool
	}{
		{name: "listed", choice: "default", wantNS: "default"},
		{name: "typed", choice: "payments", wantNS: "payments"},
		{name: "cancelled", choice: "", wantNS: "web"},
		{name: "invalid", choice: "Not Valid", wantNS: "web", wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := setupKubeconfig(t, "dev")
			rec := &recorder{}
			err := newTestApp(rec, tt.choice, nil).switchNamespace()
			if (err != nil) != tt.wantErr {
				t.Fatalf("error = %v, wantErr %v", err, tt.wantErr)
			}
			cfg, err := clientcmd.LoadFromFile(path)
			if err != nil {
				t.Fatal(err)
			}
			if got := cfg.Contexts["dev"].Namespace; got != tt.wantNS {
				t.Errorf("namespace = %q, want %q", got, tt.wantNS)
			}
			if !slices.Equal(rec.items[0], []string{"default", "web"}) {
				t.Errorf("menu items = %v", rec.items[0])
			}
		})
	}
}

func TestShellMenu(t *testing.T) {
	tests := []struct {
		name    string
		command string
		want    string
		wantErr bool
	}{
		{name: "picks first", command: "head -n1 #", want: "dev"},
		{name: "receives prompt", command: `printf '%s\n'`, want: "Pick one"},
		{name: "dismissed", command: "exit 1 #", want: ""},
		{name: "missing command", command: "kubebar-no-such-menu", wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := shellMenu(tt.command, io.Discard)("Pick one", []string{"dev", "prod"})
			if (err != nil) != tt.wantErr {
				t.Fatalf("error = %v, wantErr %v", err, tt.wantErr)
			}
			if got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}

func currentContext(t *testing.T, path string) string {
	t.Helper()
	cfg, err := clientcmd.LoadFromFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return cfg.CurrentContext
}
