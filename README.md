# gwcli

A redesigned Gravwell client for terminal, supporting both TUI-served interactivity and non-interactive script calls. 

# Usage

`./gwcli -u USERNAME -p PASSWORD`

The CLI can be used interactively or as a script tool.

Calling an action directly (ex: `./gwcli -u USERNAME -p PASSWORD query tag=gravwell`) will invoke the action and return the results.

Calling gwcli bare or from a menu (ex: `./gwlic -u USERNAME -p PASSWORD tools macros`) will start an interactive prompt at that directory (unless `--no-interactive` is given, in which case it will fail out).

Check `-h` for full details on flags and commands. 

## Special Keys

In interactive mode, certain keys have special functionality.

CTRL+C (SIGINT) is caught and will return the user to the main prompt (if an action is running) or exit the prompt.

ESC will do the former and nothing otherwise.

# Design

## Terminology

Bubble Tea has the `tea.Model` interface that must be implemented by a model struct of our own. Bubbles.TextInput is a tea.Model under the hood. Cobra is composed of `cobra.Commands` and Bubble Tea drives its I/O via `tea.Cmds`. CLI invocation is composed of commands, arguments, and flags.

So we are using our own terminology to avoid further homonyms. 

Our Bubble Tea model implementation, our controller, is *Mother*.

Tree leaves (commands that can be invoked interactively or from a script), such as `search`, are *Actions*.

Tree nodes, commands that require further input, such as `admin`, are *Navs*.

## See [Contributing](CONTRIBUTING.md) for more design information