- [ ] Implement `dry-run` flag for shell access
    - Validate syntax and exit

- [ ] create CSV exclude variant
- [ ] create JSON exclude variant
    - [ ] blocked by Gabs
- [ ] create Table exclude variant

- [ ] Display suggestions beneath current prompt

- [ ] displaying help should not establish a connection
    - especially important for nav-runs executed with --no-interactive mode 

- [x] Colourize Cobra help

- [ ] Allow list commands to pass in additional flags to treeutils.NewListCmd()

- [ ] Add support for home (~) or root (/) navigation

- [x] Do not display an unknown command error on empty input

- [ ] If an action that required arguments (ex: query) is called bare and --script is not supplied, invoke Mother

- [ ] Handle CTRL+left/right word jumping

- [x] BUG: Spinner only spins on the first submitted query; it is frozen on future queries

- [x] add append flag to query to append results to the file instead of overwriting

- [ ] Provide debouncer utility for children to use while loading

- Search Improvements
    - [ ] non-interactive scheduling via flag
    - [x] second viewport: search settings
        - [x] switch between search editor and search settings via tab cycling
        - support flag setting
            - [x] duration
            - [x] output
            - [ ] scheduling
                - [ ] name
                - [ ] description
                - [ ] schedule
        - [x] unique helpkey sets based on current focus
    - [ ] interactive mode syntax highlighting
    - [x] result pagination and/or scrollable viewport
        - [?] pass in pagination increment

- [ ] Move List_generic to boiler pkg

- [ ] rename basic_generic to just basic