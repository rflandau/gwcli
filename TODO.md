- [ ] Implement `dry-run` flag for shell access
    - Validate syntax and exit

- [ ] Implement `weave` for taking arbitrary structs and returning a JSON/CSV/etc representation of the data
    - [x] CSV output
        - [x] take list of columns to include from output
        - [ ] create exclude variant
            - [ ] default to all columns if no columns are given
    - [ ] JSON output
        - [ ] take list of columns to include/exclude from output
        - [ ] create exclude variant
            - [ ] default to all columns if no columns are given
    - [x] table output
        - [x] take list of columns to include/exclude from output
        - [ ] create exclude variant
            - [ ] default to all columns if no columns are given

- [ ] Display suggestions beneath current prompt

- [ ] Implement arrow-key-scrollable history

- [ ] Implement --no-color functionality

- [ ] displaying help should not establish a connection
    - especially important for nav-runs executed with --no-interactive mode 

- [ ] Colourize Cobra help

- [ ] Disable Cobra displaying help on failed PersistentPreRun

- [ ] Replace the panics floating around mother with appropriate error handling

- [?] Spin `weave` off into its own package

- [x] Eliminate the nil at the end of table/csv output when invoked non-interactively

- [ ] Allow list commands to pass in additional flags to treeutils.NewListCmd()

- [ ] Add support for home (~) or root (/) navigation

- [x] BUG: interactive list use prints nothing the first time each command is called