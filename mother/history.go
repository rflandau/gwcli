/**
 * The history struct is used for managing the historical record of user input
 * and facillitating their retrieval.
 * It stores a list of commands and handles efficiently retreiving local
 * commands for quick reuse.
 *
 * NOTE: The array is self-destructive; once the cap is reached, the array will
 * begin to overwrite its oldest commands.
 *
 * Newer commands have higher indices.
 */
package mother

import (
	"math"
	"strings"
)

const ( // readability "macros"
	unset            = math.MaxUint16
	arrayEnd  uint16 = 999          // last valid index
	arraySize uint16 = arrayEnd + 1 // actual array size
)

type history struct {
	commands       []string // previous commands; lower indices are newer
	fetchedIndex   uint16   // last index used to retrieve a record
	insertionIndex uint16   // index to insert next history record at
}

func newHistory() *history {
	h := history{}
	h.commands = make([]string, arraySize)
	h.fetchedIndex = unset
	h.insertionIndex = 0

	return &h
}

// Inserts a new record at the current end of the list
func (h *history) insert(record string) {
	record = strings.TrimSpace(record)
	if record == "" { // do not insert empty records
		return
	}
	h.commands[h.insertionIndex] = record
	h.insertionIndex = increment(h.insertionIndex)
}

// Starting at the newest record, returns progressively older records for each successive call.
// Stops progressing successive calls after returning an empty record.
// Call `.UnsetFetch` to restart at the newest record.
func (h *history) getOlderRecord() string {
	if h.fetchedIndex == unset {
		h.fetchedIndex = decrement(h.insertionIndex)
		return h.commands[h.fetchedIndex]
	}

	// do not move past boundary empty record
	if h.commands[h.fetchedIndex] == "" && h.commands[decrement(h.fetchedIndex)] == "" {
		// do nothing
		return ""
	}

	h.fetchedIndex = decrement(h.fetchedIndex)

	return h.commands[h.fetchedIndex]
}

// Flipside to primary command GetOlderRecord.
func (h *history) getNewerRecord() string {
	if h.fetchedIndex == unset {
		h.fetchedIndex = increment(h.insertionIndex)
		return h.commands[h.fetchedIndex]
	}

	// do not move past boundary empty record
	if h.commands[h.fetchedIndex] == "" && h.commands[increment(h.fetchedIndex)] == "" {
		// do nothing
		return ""
	}

	h.fetchedIndex = increment(h.fetchedIndex)

	return h.commands[h.fetchedIndex]

}

// Resets the fetch index, causing the next call to GetRecord to begin at the newest record.
func (h *history) unsetFetch() {
	h.fetchedIndex = unset
}

// Returns all history records, ordered from [0]newest to [len-1]oldest
// NOTE: this is a destructive call in that it will reset current GetRecord index
func (h *history) getAllRecords() (records []string) {
	records = make([]string, arraySize)
	var i uint16
	h.fetchedIndex = unset
	for i = 0; i < arraySize; i++ {
		r := h.getOlderRecord()
		if r == "" { // all records given
			break
		}
		records[i] = r
	}

	h.fetchedIndex = unset

	return records[:i] // clip length
}

// Decrements the given number, underflowing around arraysize
func decrement(i uint16) uint16 {
	if i == 0 {
		i = arrayEnd
	} else {
		i -= 1
	}
	return i
}

// Sister function to decrement; overflows around arraysize
func increment(i uint16) uint16 {
	if i == arrayEnd {
		i = 0
	} else {
		i += 1
	}
	return i
}
