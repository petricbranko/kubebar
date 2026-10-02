// Package waybar formats output for a Waybar custom module with
// "return-type": "json".
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
)

type Status struct {
	Text    string `json:"text"`
	Tooltip string `json:"tooltip"`
	Class   string `json:"class"`
}

func Context(context, cluster, namespace string, prod bool) Status {
	class := ClassOK
	if prod {
		class = ClassProd
	}
	return Status{
		Text:    context + "/" + namespace,
		Tooltip: fmt.Sprintf("Context: %s\nCluster: %s\nNamespace: %s", context, cluster, namespace),
		Class:   class,
	}
}

func None(text, tooltip string) Status {
	return Status{Text: text, Tooltip: tooltip, Class: ClassNone}
}

// Write prints s as a single line, which is what Waybar reads per update.
func Write(w io.Writer, s Status) error {
	if err := json.NewEncoder(w).Encode(s); err != nil {
		return fmt.Errorf("write waybar status: %w", err)
	}
	return nil
}
