- [ ] create CSV exclude variant
- [ ] create JSON exclude variant
    - [ ] blocked by Gabs
- [ ] create Table exclude variant

- [ ] query editor syntax highlighting

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

- query's modifView.go has a lot of repeated code for each bool and each textinput
    - [ ] genericize and consoliate the repeated code

- [ ] require name, desc, and schedule in query, if scheduled is checked
    - should error if any are unset