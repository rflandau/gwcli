# gwcli

A redesigned Gravwell client for the terminal, supporting both TUI-served interactivity and non-interactive script calls. 

# Usage

`./gwcli`

The CLI can be used interactively or as a script tool.

Calling an action directly (ex: `./gwcli query tag=gravwell`) will invoke the action and return the results.

Calling gwcli bare or from a menu (ex: `./gwcli tools macros`) will start an interactive prompt at that directory (unless `--no-interactive` is given, in which case it will display help).

Attach `-h` to any command for full details on flags and commands.

## Login

gwcli automatically logs in via token once one has been created. Use `-u USER -p PASS` the first call to generate the token automatically, then `./gwcli` can be invoked without.

# Design

## Terminology

Bubble Tea has the `tea.Model` interface that must be implemented by a model struct of our own. Bubbles.TextInput is a tea.Model under the hood. Cobra is composed of `cobra.Commands` and Bubble Tea drives its I/O via `tea.Cmds`. CLI invocation is composed of commands, arguments, and flags.

So we are using our own terminology to avoid further homonyms. 

Our Bubble Tea model implementation, our controller, is *Mother*.

Tree leaves (commands that can be invoked interactively or from a script), such as `search`, are *Actions*.

Tree nodes, commands that require further input, such as `admin`, are *Navs*.

## See [Contributing](CONTRIBUTING.md) for more design information