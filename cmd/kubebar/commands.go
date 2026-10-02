package main

import (
	"errors"
	"fmt"
	"io"

	"k8s.io/client-go/tools/clientcmd"

	"github.com/petricbranko/kubebar/internal/config"
	"github.com/petricbranko/kubebar/internal/kube"
	"github.com/petricbranko/kubebar/internal/waybar"
)

// otherNamespace can never collide with a real namespace, which must be a
// lowercase RFC 1123 label.
const otherNamespace = "Other namespace..."

// menuFunc shows items and returns the chosen line, or "" when the menu was
// dismissed.
type menuFunc func(prompt string, items []string) (string, error)

type app struct {
	cfg    config.Config
	access clientcmd.ConfigAccess
	menu   menuFunc
	// input is nil when the menu accepts typed text on its own.
	input   menuFunc
	refresh func() error
	notify  func(summary, body string) error
	stderr  io.Writer
}

func newApp(cfg config.Config, access clientcmd.ConfigAccess, stderr io.Writer) *app {
	a := &app{
		cfg:     cfg,
		access:  access,
		menu:    shellMenu(cfg.MenuCommand, stderr),
		refresh: func() error { return signalWaybar(cfg.WaybarSignal) },
		notify:  notifyCritical,
		stderr:  stderr,
	}
	if cfg.InputCommand != "" {
		a.input = shellMenu(cfg.InputCommand, stderr)
	}
	return a
}

func status(cfg config.Config, access clientcmd.ConfigAccess) waybar.Status {
	kc, err := kube.Load(access)
	if err != nil {
		return waybar.None("kubeconfig error", err.Error())
	}
	cur, err := kube.CurrentContext(kc)
	if err != nil {
		return waybar.None("no context", err.Error())
	}
	return waybar.Context(cur.Context, cur.Cluster, cur.Namespace, cfg.IsProd(cur.Context))
}

func (a *app) switchContext() error {
	kc, err := kube.Load(a.access)
	if err != nil {
		return err
	}
	names := kube.ContextNames(kc)
	if len(names) == 0 {
		return errors.New("no contexts in kubeconfig")
	}
	choice, err := a.menu("Kube context", names)
	if err != nil || choice == "" {
		return err
	}
	if err := kube.UseContext(a.access, choice); err != nil {
		return err
	}
	a.warn(a.refresh())
	if a.cfg.IsProd(choice) {
		a.warn(a.notify("Kubernetes context", "Switched to production context "+choice))
	}
	return nil
}

func (a *app) switchNamespace() error {
	kc, err := kube.Load(a.access)
	if err != nil {
		return err
	}
	cur, err := kube.CurrentContext(kc)
	if err != nil {
		return err
	}
	prompt := "Namespace for " + cur.Context
	items := kube.Namespaces(kc)
	if a.input != nil {
		items = append(items, otherNamespace)
	}
	choice, err := a.menu(prompt, items)
	if err == nil && choice == otherNamespace {
		choice, err = a.input(prompt, nil)
	}
	if err != nil || choice == "" {
		return err
	}
	if err := kube.SetNamespace(a.access, choice); err != nil {
		return err
	}
	a.warn(a.refresh())
	return nil
}

// warn reports side-effect failures without failing a switch that has
// already been written to the kubeconfig.
func (a *app) warn(err error) {
	if err != nil {
		fmt.Fprintf(a.stderr, "kubebar: warning: %v\n", err)
	}
}
