- [~] Implement `dry-run` flag for shell access
    - Validate syntax and exit
    - this may be case-by-case functionality?
        - unclear if it should be supported by a global flag 

- [ ] create CSV exclude variant
- [ ] create JSON exclude variant
    - [ ] blocked by Gabs
- [ ] create Table exclude variant

- [ ] Display suggestions beneath current prompt

- [ ] Add support for home (~) or root (/) navigation

- [ ] If an action that required arguments (ex: query) is called bare and --script is not supplied, invoke Mother

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

- [ ] rework tools/macros/delete interactive styling in items.go
    - Could be cleaner
    - [ ] provide a struct in stylesheet for list-like (but not list-action-like!) displays such as this  