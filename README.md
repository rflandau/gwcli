# gwcli

A redesigned Gravwell client for the terminal, supporting both TUI-served interactivity and non-interactive script calls. 

# Usage

`./gwcli`

The CLI can be used interactively or as a script tool.

Calling an action directly (ex: `./gwcli query tag=gravwell`) will invoke the action and return the results.

Calling gwcli bare or from a menu (ex: `./gwcli tools macros`) will start an interactive prompt at that directory (unless `--script` is given, in which case it will display help).

Attach `-h` to any command for full details on flags and commands.

## Login

gwcli automatically logs in via token once one has been created. Use `-u USER -p PASS` the first call to generate the token automatically, then `./gwcli` can be invoked without.

# Troubleshooting

## Client Not Ready For Login

Does your gravwell instance have a valid cert? If not, make sure you are using `--insecure`.

# Design

gwcli is built on the fabulous BubbleTea and Cobra libraries. In the simplest of terms, gwcli is a cobra.Command tree with a bubbletea.Model crawling around it, interacting with Gravwell via their batteries-included client library.

## See [Contributing](CONTRIBUTING.md) for a deep dive on the design philosophy and practical implementation.