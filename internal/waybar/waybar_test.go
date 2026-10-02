package waybar

import (
	"strings"
	"testing"
)

func TestWrite(t *testing.T) {
	tests := []struct {
		name   string
		status Status
		want   string
	}{
		{
			name:   "ok",
			status: Context("dev", "dev-cluster", "web", false),
			want:   `{"text":"dev/web","tooltip":"Context: dev\nCluster: dev-cluster\nNamespace: web","class":["ok"]}` + "\n",
		},
		{
			name:   "prod",
			status: Context("prod", "prod-cluster", "default", true),
			want:   `{"text":"prod/default","tooltip":"Context: prod\nCluster: prod-cluster\nNamespace: default","class":["prod","active"]}` + "\n",
		},
		{
			name:   "none",
			status: None("no context", "no current context"),
			want:   `{"text":"no context","tooltip":"no current context","class":["none"]}` + "\n",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var b strings.Builder
			if err := Write(&b, tt.status); err != nil {
				t.Fatal(err)
			}
			if b.String() != tt.want {
				t.Errorf("got  %s\nwant %s", b.String(), tt.want)
			}
		})
	}
}
