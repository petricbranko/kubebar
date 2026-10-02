// Package kube reads and edits the kubeconfig with the same loading and
// writing rules as kubectl. It never contacts a cluster.
package kube

import (
	"errors"
	"fmt"
	"maps"
	"regexp"
	"slices"

	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/clientcmd/api"
)

// DefaultNamespace is what kubectl uses when a context sets no namespace.
const DefaultNamespace = "default"

// ErrNoCurrentContext is returned when current-context is unset or names a
// context that does not exist.
var ErrNoCurrentContext = errors.New("no current context")

var namespacePattern = regexp.MustCompile(`^[a-z0-9]([-a-z0-9]{0,61}[a-z0-9])?$`)

// Current is the active context. Namespace is never empty.
type Current struct {
	Context   string
	Cluster   string
	Namespace string
}

// Load returns the merged kubeconfig. A missing kubeconfig yields an empty
// config, not an error.
func Load(access clientcmd.ConfigAccess) (*api.Config, error) {
	cfg, err := access.GetStartingConfig()
	if err != nil {
		return nil, fmt.Errorf("load kubeconfig: %w", err)
	}
	return cfg, nil
}

func CurrentContext(cfg *api.Config) (Current, error) {
	if cfg.CurrentContext == "" {
		return Current{}, ErrNoCurrentContext
	}
	ctx, ok := cfg.Contexts[cfg.CurrentContext]
	if !ok {
		return Current{}, fmt.Errorf("%w: context %q not found", ErrNoCurrentContext, cfg.CurrentContext)
	}
	ns := ctx.Namespace
	if ns == "" {
		ns = DefaultNamespace
	}
	return Current{Context: cfg.CurrentContext, Cluster: ctx.Cluster, Namespace: ns}, nil
}

func ContextNames(cfg *api.Config) []string {
	return slices.Sorted(maps.Keys(cfg.Contexts))
}

// Namespaces lists the namespaces known without asking a cluster: the
// default one plus every namespace already set on a context.
func Namespaces(cfg *api.Config) []string {
	names := []string{DefaultNamespace}
	for _, ctx := range cfg.Contexts {
		if ctx.Namespace != "" {
			names = append(names, ctx.Namespace)
		}
	}
	slices.Sort(names)
	return slices.Compact(names)
}

func UseContext(access clientcmd.ConfigAccess, name string) error {
	cfg, err := Load(access)
	if err != nil {
		return err
	}
	if _, ok := cfg.Contexts[name]; !ok {
		return fmt.Errorf("context %q not found", name)
	}
	next := cfg.DeepCopy()
	next.CurrentContext = name
	return save(access, next)
}

// SetNamespace sets the namespace on the current context, in the file that
// context was loaded from.
func SetNamespace(access clientcmd.ConfigAccess, namespace string) error {
	if !namespacePattern.MatchString(namespace) {
		return fmt.Errorf("invalid namespace %q: must be a lowercase RFC 1123 label", namespace)
	}
	cfg, err := Load(access)
	if err != nil {
		return err
	}
	cur, err := CurrentContext(cfg)
	if err != nil {
		return err
	}
	next := cfg.DeepCopy()
	next.Contexts[cur.Context].Namespace = namespace
	return save(access, next)
}

func save(access clientcmd.ConfigAccess, cfg *api.Config) error {
	if err := clientcmd.ModifyConfig(access, *cfg, true); err != nil {
		return fmt.Errorf("write kubeconfig: %w", err)
	}
	return nil
}
