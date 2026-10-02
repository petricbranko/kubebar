package config

import (
	"os"
	"path/filepath"
	"slices"
	"testing"
)

func TestLoad(t *testing.T) {
	defaults := loaded{[]string{"prod"}, "walker --dmenu -p", 8}
	tests := []struct {
		name    string
		content *string
		want    loaded
		wantErr bool
	}{
		{name: "missing file", content: nil, want: defaults},
		{name: "empty file", content: ptr(""), want: defaults},
		{
			name: "all keys",
			content: ptr(`prod_patterns = ["^prd-", "live$"]
menu_command = "fuzzel --dmenu -p"
waybar_signal = 12
`),
			want: loaded{[]string{"^prd-", "live$"}, "fuzzel --dmenu -p", 12},
		},
		{name: "partial keeps defaults", content: ptr(`waybar_signal = 3`), want: loaded{[]string{"prod"}, "walker --dmenu -p", 3}},
		{name: "no prod patterns", content: ptr(`prod_patterns = []`), want: loaded{[]string{}, "walker --dmenu -p", 8}},
		{name: "invalid regex", content: ptr(`prod_patterns = ["("]`), wantErr: true},
		{name: "unknown key", content: ptr(`menu = "rofi"`), wantErr: true},
		{name: "signal out of range", content: ptr(`waybar_signal = 0`), wantErr: true},
		{name: "empty menu", content: ptr(`menu_command = " "`), wantErr: true},
		{name: "malformed", content: ptr(`prod_patterns = [`), wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "config.toml")
			if tt.content != nil {
				if err := os.WriteFile(path, []byte(*tt.content), 0o600); err != nil {
					t.Fatal(err)
				}
			}
			cfg, err := Load(path)
			if tt.wantErr {
				if err == nil {
					t.Fatal("Load succeeded, want error")
				}
				return
			}
			if err != nil {
				t.Fatalf("Load: %v", err)
			}
			got := loaded{patternStrings(cfg), cfg.MenuCommand, cfg.WaybarSignal}
			if !slices.Equal(got.patterns, tt.want.patterns) || got.menu != tt.want.menu || got.signal != tt.want.signal {
				t.Errorf("got %+v, want %+v", got, tt.want)
			}
		})
	}
}

type loaded struct {
	patterns []string
	menu     string
	signal   int
}

func TestPath(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	tests := []struct {
		xdg  string
		want string
	}{
		{"/xdg", "/xdg/kubebar/config.toml"},
		{"", filepath.Join(home, ".config/kubebar/config.toml")},
		{"relative", filepath.Join(home, ".config/kubebar/config.toml")},
	}
	for _, tt := range tests {
		t.Setenv("XDG_CONFIG_HOME", tt.xdg)
		got, err := Path()
		if err != nil {
			t.Fatal(err)
		}
		if got != tt.want {
			t.Errorf("XDG_CONFIG_HOME=%q: Path = %q, want %q", tt.xdg, got, tt.want)
		}
	}
}

func TestIsProd(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.toml")
	if err := os.WriteFile(path, []byte(`prod_patterns = ["prod", "^(?i)live-"]`), 0o600); err != nil {
		t.Fatal(err)
	}
	cfg, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	tests := []struct {
		context string
		want    bool
	}{
		{"prod", true},
		{"kind-prod-test", true},
		{"arn:aws:eks:eu-west-1:123:cluster/production", true},
		{"LIVE-eu", true},
		{"dev", false},
		{"staging-live-mirror", false},
		{"", false},
	}
	for _, tt := range tests {
		if got := cfg.IsProd(tt.context); got != tt.want {
			t.Errorf("IsProd(%q) = %v, want %v", tt.context, got, tt.want)
		}
	}
	if (Config{}).IsProd("prod") {
		t.Error("config without patterns matched prod")
	}
}

func patternStrings(c Config) []string {
	out := make([]string, len(c.ProdPatterns))
	for i, re := range c.ProdPatterns {
		out[i] = re.String()
	}
	return out
}

func ptr(s string) *string { return &s }
