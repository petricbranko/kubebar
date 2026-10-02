# kubebar

Show the current Kubernetes context and namespace in the Omarchy bar, and switch them from the Omarchy menu.

Built for Omarchy 4. Waybar setups, such as Omarchy 3, are covered [below](#waybar-and-omarchy-3).

## Install

```sh
sudo pacman -S --needed go
GOBIN=~/.local/bin go install github.com/petricbranko/kubebar/cmd/kubebar@latest
kubebar status
```

`~/.local/bin` is on the `PATH` that the Omarchy shell and Hyprland use, so the bar and keybindings can find `kubebar`.

## Bar

Add kubebar to the right side of the bar:

```sh
f=~/.config/omarchy/shell.json
[ -f "$f" ] || cp "$OMARCHY_PATH/config/omarchy/shell.json" "$f"
jq '.bar.layout.right |= [{"id":"kubebar","type":"command","exec":"kubebar status","interval":2,"onClick":"kubebar switch","onRightClick":"kubebar ns"}] + map(select(.id != "kubebar"))' "$f" > "$f.tmp" && mv "$f.tmp" "$f"
```

The shell reloads `shell.json` on change. Running the command again replaces the entry instead of adding a second one. The entry is also in [contrib/omarchy/bar-module.json](contrib/omarchy/bar-module.json) if you prefer to paste it into `bar.layout` yourself, and you can drag it to another spot on the bar.

Left click switches the context. Right click switches the namespace. Production contexts are drawn in the active highlight color.

## Keybindings

Append to `~/.config/hypr/bindings.lua`:

```lua
o.bind("SUPER + SHIFT + K", "Kube context", "kubebar switch")
o.bind("SUPER + SHIFT + ALT + K", "Kube namespace", "kubebar ns")
```

## Usage

```
kubebar status   # one line of JSON for the bar
kubebar switch   # pick a context
kubebar ns       # pick a namespace, or choose "Other namespace..." to type one
```

Switching to a context that matches a prod pattern sends a critical notification.

## Config

Optional, at `$XDG_CONFIG_HOME/kubebar/config.toml` (default `~/.config/kubebar/config.toml`). All keys are optional:

```toml
# Regexes matched against the context name. Matching contexts are
# highlighted and trigger a critical notification on switch.
prod_patterns = ["prod"]

# Run with sh -c. Items arrive on stdin, the prompt is appended as the last
# argument, and the first line of output is the selection.
menu_command = "omarchy-menu-select"

# Prompts for a namespace that is not in the list. Set to "" when the menu
# returns typed text itself, as Walker and rofi do.
input_command = "omarchy-menu-input"

# Only used with Waybar: refreshed with pkill -RTMIN+<signal> waybar.
waybar_signal = 8
```

A non-zero exit from the menu counts as cancelled.

## Waybar and Omarchy 3

`kubebar status` prints Waybar custom module JSON. Use the files in [contrib/waybar/](contrib/waybar/):

- `config.toml` to `~/.config/kubebar/config.toml`, so kubebar opens Walker
- `waybar.jsonc` into `~/.config/waybar/config.jsonc`
- `style.css` appended to `~/.config/waybar/style.css`
- `bindings.conf` appended to `~/.config/hypr/bindings.conf`

## How it works

kubebar loads the kubeconfig with client-go, so `KUBECONFIG`, merged files and defaults behave exactly like kubectl.
Changes are written with the same code kubectl uses, so `current-context` and namespaces land in the right file.
It never talks to a cluster: namespaces come from what you type or what is already set on your contexts.

## License

MIT
