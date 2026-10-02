# kubebar

Show the current Kubernetes context and namespace in Waybar, and switch them from a dmenu-style launcher (Walker on Omarchy).

## Install

```sh
go install github.com/petricbranko/kubebar/cmd/kubebar@latest
```

Waybar and Hyprland need to find `kubebar` on their `PATH`. If `~/go/bin` is not on it, use the full path in the snippets below.

## Usage

```
kubebar status   # one line of Waybar JSON
kubebar switch   # pick a context
kubebar ns       # pick or type a namespace for the current context
```

## Waybar

Add `"custom/kube"` to a `modules-*` list in `~/.config/waybar/config.jsonc` and add the module:

```jsonc
"custom/kube": {
  "exec": "kubebar status",
  "return-type": "json",
  "interval": 5,
  "signal": 8,
  "escape": true,
  "max-length": 40,
  "format": "k8s {}",
  "on-click": "kubebar switch",
  "on-click-right": "kubebar ns"
}
```

Append to `~/.config/waybar/style.css`:

```css
#custom-kube.ok   { color: @foreground; }
#custom-kube.prod { color: @background; background-color: @foreground; padding: 0 6px; font-weight: bold; }
#custom-kube.none { color: @foreground; opacity: 0.4; }
```

The status has class `prod`, `ok` or `none` (no kubeconfig, no current context, or an error, shown in the tooltip).

## Keybinding

Append to `~/.config/hypr/bindings.conf`:

```
bindd = SUPER SHIFT, K, Kube context, exec, kubebar switch
bindd = SUPER SHIFT ALT, K, Kube namespace, exec, kubebar ns
```

Ready-to-copy files are in [contrib/](contrib/).

## Config

Optional, at `$XDG_CONFIG_HOME/kubebar/config.toml` (default `~/.config/kubebar/config.toml`). All keys are optional:

```toml
# Regexes matched against the context name. Matching contexts get the
# "prod" class and a critical notification on switch.
prod_patterns = ["prod"]

# Run with sh -c. Items arrive on stdin, the prompt is appended as the last
# argument, and the first line of output is the selection.
menu_command = "walker --dmenu -p"

# Waybar is refreshed with pkill -RTMIN+<signal> waybar. Must match "signal".
waybar_signal = 8
```

A non-zero exit from the menu counts as cancelled. To ignore the appended prompt, end the command with `#`, for example `menu_command = "head -n1 #"`.

## How it works

kubebar loads the kubeconfig with client-go, so `KUBECONFIG`, merged files and defaults behave exactly like kubectl.
Changes are written with the same code kubectl uses, so `current-context` and namespaces land in the right file.
It never talks to a cluster: namespaces come from what you type or what is already set on your contexts.

## License

MIT
