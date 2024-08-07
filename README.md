# This has been merged into the Gravwell monorepo; use that instead. (PR#[1142](https://github.com/gravwell/gravwell/pull/1142))

# gwcli

A redesigned Gravwell client for the terminal, supporting both TUI-served interactivity and non-interactive script calls. 

![demo](demo.gif)

# Features

- interactive *and* scriptable

- full query editor (syntax highlighting NYI)

- dynamic viewport for interacting with query results

- mutliple query result download formats (CSV, JSON, raw)

- shell-style navigation

- `tree` command to view entire structure

- command history

- context-aware help for every command

- automatic login via token (for subsequent logins)

- completions for zsh, fish, bash, and powershell

- limited tab completion in interactive mode

- pluggable framework for easily adding new capabilities (complete with genericized boilerplate and generator functions)


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

# Known Issues

- Suggestions do not populate when navigation includes `..` (upward) or `~`/`/` (from home/root)

    - This is just based on how bubbletea manages its suggestions. It operates off a list of strings, where each string is a *complete command*, returning closest-match. The suggestion engine is not intelligent enough to "walk" the current prompt (like the walk() subroutine is). We simple recur downward and supply these strings on every move. We do not supply suggestions with upward or rooted navigation because it would add a boatload more iterations and recusive logic when building the suggestions each hop. Relately, Bubble Tea does not handle .SetSuggestions coming in asyncronously, meaning each hop has to determine its suggestions immediately so keeping the lists down improves responsiveness.
