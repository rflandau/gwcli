- [~] Implement `dry-run` flag for shell access
    - Validate syntax and exit
    - this may be case-by-case functionality?
        - unclear if it should be supported by a global flag 

- [ ] create CSV exclude variant
- [ ] create JSON exclude variant
    - [ ] blocked by Gabs
- [ ] create Table exclude variant

- [ ] Upgrade suggestions to dynamically map the entire tree
    - Currently only works on immediate children and builtins

- [x] Add support for home (~) or root (/) navigation

- [ ] If an action that required arguments (ex: query) is called bare and --script is not supplied, invoke Mother
    - [ ] spin up standalone deletion picker if delete is called from cli and !script
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

- [ ] Add datascope support and keys for downloading all data or just the current page of data

- [ ] Store help strings within mother somewhere so we can lazy-compile them rather than regenerating each call

- [ ] confirm the status of the no-color flag
    - likely not respected, but should be implementable via lipgloss.NoColor
    - [ ] check for respect of the NoColor env, per https://no-color.org/

- [ ] actor.go's BurnFirstView...

- [ ] expand RenderToDownload to encompass the remaining types.Download* constants
    - Renders -> Download
        - Hex,Raw,Text -> Text
        - Pcap -> PCAP
        - Table -> ?
        - Guage -> ?
        - Numbercard -> ?
        - Char -> ?
        - Fdg -> ?
        - Stackgraph -> ?
        - Pointmap -> ?
        - Heatmap -> ?
        - P2P -> ?
        - ? -> LookupData
        - ? -> IPExists
        - ? -> Archive

- [ ] check for sentinel errors in createMacro

- [ ] search around for a BubbleTea rendering fix for window size messages in non-altmode.
    - currently, get artefacting above most recent draw

- [ ] allow --schedule flag in query to pass in absolute datetimes (as it is now) AND now+ values (ex: +5h, +20m, ...)