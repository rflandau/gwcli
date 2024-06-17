- [ ] Implement `dry-run` flag for shell access
    - Validate syntax and exit

- [ ] create CSV exclude variant
- [ ] create JSON exclude variant
    - [ ] blocked by Gabs
- [ ] create Table exclude variant

- [ ] Display suggestions beneath current prompt

- [ ] displaying help should not establish a connection
    - especially important for nav-runs executed with --no-interactive mode 

- [ ] Colourize Cobra help

- [ ] Allow list commands to pass in additional flags to treeutils.NewListCmd()

- [ ] Add support for home (~) or root (/) navigation

- [ ] Do not display an unknown command error on empty input

- [ ] If an action that required arguments (ex: query) is called bare and --script is not supplied, invoke Mother

- [ ] BUG (may be an issue with my terminal, specifically): Invoking -h works correctly, but puts garbage ("11;rgb:0000/0000/00003;1R") on the next shell prompt.

- [ ] Handle CTRL+left/right word jumping

- [x] BUG: Spinner only spins on the first submitted query; it is frozen on future queries

- [x] add append flag to query to append results to the file instead of overwriting