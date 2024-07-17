- [ ] create CSV exclude variant
- [ ] create JSON exclude variant
    - [ ] blocked by Gabs
- [ ] create Table exclude variant

- [ ] query editor syntax highlighting

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

- implement macro edit action

- add debouncer to DS to reduce lag when holding a key

- create scheduled search `create` function

- create generic `create` scaffold

- utilize DataScope's table's native filtering
    - provide keybind and external ("API") filter TI
        - place in footer? hide/show header?
    - will require utilizing the table's update method, which is currently not called
        - somewhat conflicts with the viewport wrapper
        - remember to disable the table's keybinds, other than filtering