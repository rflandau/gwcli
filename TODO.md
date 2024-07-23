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

- [ ] search around for a BubbleTea rendering fix for window size messages in non-altmode.
    - currently, get artefacting above most recent draw

- BUG: DS's results lose the alternating color if the start of the entry is cut off (aka: the termainl escape characters get cut off at the start)

- support X-Y notation in records downloading via DS

- support RecordsPerPage flag/option in datascope

- implement macro edit action

- add debouncer to DS to reduce lag when holding a key


- utilize DataScope's table's native filtering
    - provide keybind and external ("API") filter TI
        - place in footer? hide/show header?
    - will require utilizing the table's update method, which is currently not called
        - somewhat conflicts with the viewport wrapper
        - remember to disable the table's keybinds, other than filtering

- support more FieldTypes in scaffold create
    - radio button
    - checkbox

- add aliases to the dynamic search generation at Mother's prompt

- `extractor create`: figure out how to support dynamic module suggestion based on current tags
    - `ExploreGenerate()` returns a map where the keys are extraction modules, but it appears to be a heavier operation if we only want the module names.
        - There must be a better way to filter the list of module names.
    - Suggestions would need to automatically update whenever a new, valid tag is punched into the tags TI
        - this TI must be aware of the other TI, meaning the function signature of this customTI feature likely needs references to other parts of the createModel.
