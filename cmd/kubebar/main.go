// Command kubebar shows the current Kubernetes context in Waybar and switches
// contexts and namespaces through a dmenu-style launcher.
package main

import (
	"fmt"
	"io"
	"os"

	"k8s.io/client-go/tools/clientcmd"

	"github.com/petricbranko/kubebar/internal/config"
	"github.com/petricbranko/kubebar/internal/waybar"
)

const usage = `Usage: kubebar <command>

Commands:
  status  print the current context and namespace as Waybar JSON
  switch  pick a context from the menu and make it current
  ns      pick or type a namespace for the current context
`

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

func run(args []string, stdout, stderr io.Writer) int {
	if len(args) != 1 {
		fmt.Fprint(stderr, usage)
		return 2
	}

	access := clientcmd.NewDefaultPathOptions()
	cfg, cfgErr := loadConfig()

	switch args[0] {
	case "status":
		st := waybar.None("kubebar error", fmt.Sprint(cfgErr))
		if cfgErr == nil {
			st = status(cfg, access)
		}
		if err := waybar.Write(stdout, st); err != nil {
			fmt.Fprintf(stderr, "kubebar: %v\n", err)
		}
		// Waybar hides or breaks the module on a non-zero exit.
		return 0
	case "switch", "ns":
		if cfgErr != nil {
			fmt.Fprintf(stderr, "kubebar: %v\n", cfgErr)
			return 1
		}
		a := newApp(cfg, access, stderr)
		cmd := a.switchContext
		if args[0] == "ns" {
			cmd = a.switchNamespace
		}
		if err := cmd(); err != nil {
			fmt.Fprintf(stderr, "kubebar: %v\n", err)
			return 1
		}
		return 0
	case "help", "-h", "--help":
		fmt.Fprint(stdout, usage)
		return 0
	default:
		fmt.Fprintf(stderr, "kubebar: unknown command %q\n\n%s", args[0], usage)
		return 2
	}
}

func loadConfig() (config.Config, error) {
	path, err := config.Path()
	if err != nil {
		return config.Config{}, err
	}
	return config.Load(path)
}
