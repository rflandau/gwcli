- [ ] create CSV exclude variant
- [ ] create JSON exclude variant
    - [ ] blocked by Gabs
- [ ] create Table exclude variant

- [ ] query editor syntax highlighting

- [x] Add datascope support and keys for downloading all data or just the current page of data

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

- BUG: DS's results lose the alternating color if the start of the entry is cut off (aka: the termainl escape characters get cut off at the start)

- support X-Y notation in records downloading via DS

- support RecordsPerPage flag/option in datascope

- BUG: suggestions strangely overlap when history is keyed through on Mother's prompt

- add auto-schedule when a schedule is given to DS

- implement scheduled search management
    - list
    - delete

- implement macro edit action