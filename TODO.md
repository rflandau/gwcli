- [ ] Implement multi-token parsing on the interactive prompt
    - [ ] navigate multiple levels, matching against each context's children
    - [ ] allow contextual help via `<command> help`
        - Auxillary to already-implemented F1 prompt reading
    - [ ] pass tokens past an action to the action as arguments
        - [ ] allow actions to define a .Validate to ensure argument status

- [ ] Implement `dry-run` flag for shell access
    - Validate syntax and exit

- [ ] Implement `weaver` for taking arbitrary structs and returning a JSON/CSV/etc representation of the data
    - [ ] take list of columns to include/exclude from output
    - iterate over struct reflection as key/values

- [ ] Display suggestions beneath current prompt

- [ ] Implement arrow-key-scrollable history

- [ ] Implement --no-color functionality

- [ ] displaying help should not establish a connection
    - especially important for nav-runs executed with --no-interactive mode 

- [ ] Colourize Cobra help

- [ ] Disable Cobra displaying help on failed PersistentPreRun