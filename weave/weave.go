/**
 * The output module.
 * Weave consumes arbitrary structs, orchestrating them into a specified format
 * and returning the formatted string.
 */
package weave

import (
	"errors"
	"fmt"
	"gwcli/clilog"
	"gwcli/stylesheet"
	"reflect"
	"strings"
)

//#region errors

const (
	ErrNotAStruct  string = "given value is not a struct or pointer to a struct"
	ErrStructIsNil string = "given value is nil"
)

//#endregion

// Takes an array of arbitrary struct `st` and the *ordered* columns to
// include/exclude and returns a string containing the csv representation of the
// data contained therein.
//
// Uses qualified names to access nested structs/fields of arbitrary depth.
// Promoted names can still be accessed unqualified, but all other nested structs/fields are accessed via dot separators.
// Ex: structField.fieldA
//
// # See FindQualifiedField for more on qualification.
//
// ! column names are case sensitive
// ! Returns the empty string if columns or st are empty
// ! the array of interfaces are expected to be structs with identical structure
// TODO incorporate exclude boolean to blacklist columns instead of assuming whitelist
// TODO allow column names to be case-insensitive
func ToCSV[Any any](st []Any, columns []string) string {
	// DESIGN:
	// We have a list of column, ordered.
	// We have a map of column names -> field index.
	// For each struct s in the list of structs:
	//	iterate through the list of columns and use the map to fetch the
	//	column/field's values by index, building the csv token by token

	if columns == nil || st == nil || len(st) < 1 || len(columns) < 1 { // superfluous request
		clilog.Writer.Warnf("superfluous request (columns: %v, st: %v)", columns, st)
		return ""
	}

	var hdrBldr strings.Builder

	// deconstruct the first struct to validate requested columns
	// coordinate columns
	columnMap := make(map[string][]int, len(columns)) // column name -> recursive field indices
	for i := range columns {
		hdrBldr.WriteString(columns[i] + ",") // generate header
		// map column names to their field indices
		// if a name is not found, nil it so it can be skipped later
		_, fo, index, err := FindQualifiedField[any](columns[i], st[0])
		if err != nil {
			panic(err)
		}
		if !fo {
			columnMap[columns[i]] = nil
			continue
		}
		columnMap[columns[i]] = index
	}
	var hdr string = chomp(hdrBldr.String())

	var csv strings.Builder // stores the actual data

	for _, s := range st { // operate on each struct
		csv.WriteString(stringifyStructCSV(s, columns, columnMap) + "\n")
	}

	return strings.TrimSpace(hdr + "\n" + csv.String())
}

// helper function for ToCSVHash
// returns a string of a CSV row populated by the data in the struct that corresponds to the columns
func stringifyStructCSV(s interface{}, columns []string, columnMap map[string][]int) string {
	var row strings.Builder

	// deconstruct the struct
	structVals := reflect.ValueOf(s)

	// search for each column
	for _, col := range columns {
		findices := columnMap[col]
		if findices == nil {
			// no matching field
			// do nothing
		} else {
			// use field index to retrieve value
			data := structVals.FieldByIndex(findices)
			if data.Kind() == reflect.Pointer {
				data = data.Elem()
			}
			row.WriteString(fmt.Sprintf("%v", data))
		}
		row.WriteString(",") // append comma to token
	}

	return chomp(row.String())
}

func chomp(s string) string {
	return strings.TrimSuffix(s, ",")
}

// Given a fully qualified column name (ex: "outerstruct.innerstruct.field"),
// finds the associated field, if it exists.
//
// Returns the field, whether or not it was found, the index path (for
// FieldByIndex) to the field (more on this below), and any errors.
//
// index path is returned here becaue field.Index is NOT reliable for some
// nested structures. Fields do not necessarily know their complete index path
// for the given parent struct and therefore using field.Index in FieldByIndex
// can cause unexpected, erroneous reults.
// The returned index path is composed of the known indices of every field
// touched during traversal, returning a complete path.
//
// ! st must be a struct
func FindQualifiedField[Any any](qualCol string, st any) (field reflect.StructField, found bool, index []int, err error) {
	// pre checks
	if qualCol == "" {
		return reflect.StructField{}, false, nil, nil
	}
	if st == nil {
		return reflect.StructField{}, false, nil, errors.New(ErrStructIsNil)
	}
	t := reflect.TypeOf(st)
	if t.Kind() != reflect.Struct {
		return reflect.StructField{}, false, nil, errors.New(ErrNotAStruct)
	}

	index = make([]int, 0)

	exploded := strings.Split(qualCol, ".")
	field.Type = t
	// iterate down the field tree until we run out of qualifications or cannot
	// location the next qualification
	for i, e := range exploded {
		field, found = field.Type.FieldByName(e)
		if !found { // no value found
			fmt.Printf("DEBUG: found no value for qualifier '%s' at depth %d\n", e, i)
			return reflect.StructField{}, false, nil, nil
		}
		// build path
		index = append(index, field.Index...)
	}
	// if we reached the end of the loop, we have our final field
	return field, true, index, nil

}

func ToTable[Any any](st []Any, columns []string) string {

	if columns == nil || st == nil || len(st) < 1 || len(columns) < 1 { // superfluous request
		return ""
	}

	var data [][]string = make([][]string, len(st))
	// TODO import stylesheet.Table, instead calling base styling from stylesheet.NewTable

	// TODO
	return stylesheet.Table(columns, data)
}
