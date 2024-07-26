package scaffoldedit

import "github.com/gravwell/gravwell/v3/client/types"

// Pulls the specific, edit-able struct when skipping list/selecting mode.
type SelectSubroutine = func(id uint64) (
	item types.SearchMacro, err error,
)

// Fetches all edit-able structs. Not used in script mode.
type FetchAllSubroutine = func() (
	items []types.SearchMacro, err error,
)

// Function to retrieve the struct value associated to the field key without reflection.
// This is probably a switch statement that maps (key -> item.X).
//
// Sister to setFieldFunction.
type GetFieldSubroutine = func(item types.SearchMacro, fieldKey string) (
	value string, err error,
)

// Function to set the struct value associated to the field key without reflection.
// This is probably a switch statement that maps (key -> item.X).
// Returns invalid if the value is invalid for the keyed field and err on an unrecoverable error.
//
// Sister to getFieldFunction.
type SetFieldSubroutine = func(item *types.SearchMacro, fieldKey, val string) (
	invalid string, err error,
)

// Performs the actual update of the data on the GW instance
type UpdateStructSubroutine = func(data *types.SearchMacro) (
	identifier string, err error,
)

// Set of all subroutines required by an edit implementation.
//
// ! AddEditAction will panic if any subroutine is nil
type SubroutineSet struct {
	SelectSub   SelectSubroutine       // fetch a specific editable struct
	FetchSub    FetchAllSubroutine     // used in interactive mode to fetch all editable structs
	GetFieldSub GetFieldSubroutine     // get a value within the struct
	SetFieldSub SetFieldSubroutine     // set a value within the struct
	UpdateSub   UpdateStructSubroutine // submit the struct as updated
}

// Validates that all functions were set.
// Panics if any are missing.
func (funcs *SubroutineSet) guarantee() {
	if funcs.SelectSub == nil {
		panic("select function is required")
	}
	if funcs.FetchSub == nil {
		panic("fetch all function is required")
	}
	if funcs.GetFieldSub == nil {
		panic("get field function is required")
	}
	if funcs.SetFieldSub == nil {
		panic("set field function is required")
	}
	if funcs.UpdateSub == nil {
		panic("update struct function is required")
	}
}
