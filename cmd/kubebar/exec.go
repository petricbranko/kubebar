package main

import (
	"errors"
	"fmt"
	"io"
	"os/exec"
	"strings"
)

func shellMenu(command string, stderr io.Writer) menuFunc {
	return func(prompt string, items []string) (string, error) {
		// sh -c allows shell quoting in menu_command. The prompt is appended
		// as the last argument, where omarchy-menu-select expects it and where
		// "walker --dmenu -p" reads it as the value of -p.
		cmd := exec.Command("sh", "-c", command+` "$@"`, "sh", prompt)
		if len(items) > 0 {
			cmd.Stdin = strings.NewReader(strings.Join(items, "\n") + "\n")
		}
		cmd.Stderr = stderr
		out, err := cmd.Output()
		var exitErr *exec.ExitError
		switch {
		// Menus exit non-zero when dismissed. 126 and 127 come from sh and
		// mean the command itself could not run.
		case errors.As(err, &exitErr) && exitErr.ExitCode() < 126:
			return "", nil
		case err != nil:
			return "", fmt.Errorf("run menu %q: %w", command, err)
		}
		line, _, _ := strings.Cut(string(out), "\n")
		return strings.TrimSpace(line), nil
	}
}

func signalWaybar(signal int) error {
	err := exec.Command("pkill", fmt.Sprintf("-RTMIN+%d", signal), "waybar").Run()
	var exitErr *exec.ExitError
	// pkill exits 1 when nothing matched, which just means Waybar is not running.
	if errors.As(err, &exitErr) && exitErr.ExitCode() == 1 {
		return nil
	}
	if err != nil {
		return fmt.Errorf("refresh waybar: %w", err)
	}
	return nil
}

func notifyCritical(summary, body string) error {
	if err := exec.Command("notify-send", "-u", "critical", summary, body).Run(); err != nil {
		return fmt.Errorf("notify-send: %w", err)
	}
	return nil
}
