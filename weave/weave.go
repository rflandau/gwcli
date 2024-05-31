/**
 * The output module.
 * Weave consumes arbitrary structs, orchestrating them into a specified format
 * and returning the formatted string.
 */
package weave

import (
	"fmt"
	"gwcli/stylesheet"
	"reflect"
	"strings"
)

/**
 * Takes an array of arbitrary struct `st` and the *ordered* columns to include/exclude
 * and returns a string containing the csv representation of the data contained
 * therein.
 * ! Returns the empty string if columns or st are empty
 * ! the array of interfaces are expected to be structs with identical structure
 * TODO incorporate exclude boolean to blacklist columns instead of assuming whitelist
 * TODO allow column names to be case-insensitive
 */
func ToCSV[Any any](st []Any, columns []string) string {
	// DESIGN:
	// We have a list of column, ordered.
	// We have a map of column names -> field index.
	// For each struct s in the list of structs:
	//	iterate through the list of columns and use the map to fetch the
	//	column/field's values by index, building the csv token by token

	if columns == nil || st == nil || len(st) < 1 || len(columns) < 1 { // superfluous request
		return ""
	}

	// deconstruct the first struct to validate requested columns
	vals := reflect.ValueOf(st[0])
	types := vals.Type()

	var hdrBldr strings.Builder

	// coordinate columns
	// TODO keying the map on columns index would bring faster lookups
	columnMap := make(map[string][]int, len(columns)) // column name -> recursive field indices
	for i := range columns {
		hdrBldr.WriteString(columns[i] + ",") // generate header
		// map column names to their field indices
		// if a name is not found, nil it so it can be skipped later
		field, found := types.FieldByName(columns[i])
		if !found {
			columnMap[columns[i]] = nil
			continue
		}
		columnMap[columns[i]] = field.Index
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

func ToTable[Any any](st []Any, columns []string) string {

	if columns == nil || st == nil || len(st) < 1 || len(columns) < 1 { // superfluous request
		return ""
	}

	var data [][]string = make([][]string, len(st))
	// TODO import stylesheet.Table, instead calling base styling from stylesheet.NewTable

	// TODO
	return stylesheet.Table(columns, data)
}
