package delete

import "github.com/gravwell/gravwell/v3/client/types"

type item types.SearchMacro

func (i item) FilterValue() string { return "" }
