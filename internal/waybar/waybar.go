// Package waybar formats status in the Waybar custom module JSON format,
// which the Omarchy shell's command modules also read.
package waybar

import (
	"encoding/json"
	"fmt"
	"io"
)

const (
	ClassOK   = "ok"
	ClassProd = "prod"
	ClassNone = "none"
	// ClassActive is the only class the Omarchy shell's command modules
	// style, so prod carries it to stand out there.
	ClassActive = "active"
)

type Status struct {
	Text    string   `json:"text"`
	Tooltip string   `json:"tooltip"`
	Class   []string `json:"class"`
}

func Context(context, cluster, namespace string, prod bool) Status {
	class := []string{ClassOK}
	if prod {
		class = []string{ClassProd, ClassActive}
	}
	return Status{
		Text:    context + "/" + namespace,
		Tooltip: fmt.Sprintf("Context: %s\nCluster: %s\nNamespace: %s", context, cluster, namespace),
		Class:   class,
	}
}

func None(text, tooltip string) Status {
	return Status{Text: text, Tooltip: tooltip, Class: []string{ClassNone}}
}

// Write prints s as a single line, which is what Waybar reads per update.
func Write(w io.Writer, s Status) error {
	if err := json.NewEncoder(w).Encode(s); err != nil {
		return fmt.Errorf("write waybar status: %w", err)
	}
	return nil
}
