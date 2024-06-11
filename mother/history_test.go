package mother

import (
	"fmt"
	"math/rand"
	"testing"
)

func TestHistoryLimits(t *testing.T) {
	t.Run("unset clash", func(t *testing.T) {
		// overlap between unset macro will cause indexing issues
		if arrayEnd >= unset {
			t.Errorf("unset macro (%d) is in the set of array indices (%v)", unset, arrayEnd)
		}
	})
}

// A somewhat redudant test to ensure new sets all parameters correctly
func TestNewHistory(t *testing.T) {
	t.Run("New", func(t *testing.T) {
		h := NewHistory()
		if h.fetchIndex != unset {
			t.Errorf("fetch index did not start unset")
		}
		if h.insertionIndex != 0 {
			t.Errorf("insertion index is not at 0th index")
		}
		for i, c := range h.commands {
			if c != "" {
				t.Errorf("non empty (%v) @ index %v", c, i)
			}
		}
	})
}

func Test_history_Insert(t *testing.T) {
	t.Parallel()
	t.Run("first record", func(t *testing.T) {
		h := NewHistory()
		record := "first"
		h.Insert(record)
		if r := h.commands[0]; r != record {
			t.Errorf("record mismatch: expected %s, got %s", record, h.commands[0])
		}
		if h.insertionIndex != 1 {
			t.Errorf("insertion index not incremeneted")
		}
		if h.fetchIndex != unset {
			t.Errorf("fetch index altered during insert")
		}
	})
	t.Run("empty second record", func(t *testing.T) {
		h := NewHistory()
		h.Insert("first")
		record := ""
		h.Insert(record)
		if h.insertionIndex != 1 {
			t.Errorf("insertion index was incremeneted")
		}
		if h.fetchIndex != unset {
			t.Errorf("fetch index altered during insert")
		}
	})
	t.Run("interspersed empty records", func(t *testing.T) {
		// no number of empty insertions should alter history at all
		h := NewHistory()
		h.Insert("first")
		h.Insert("")
		h.Insert("second")
		insertCount := rand.Intn(500)
		for i := 0; i < insertCount; i++ {
			h.Insert("")
		}
		h.Insert("third")
		insertCount = rand.Intn(500)
		for i := 0; i < insertCount; i++ {
			h.Insert("")
		}
		h.Insert("fourth")

		if h.insertionIndex != 4 {
			t.Errorf("insertion index mismatch: expected %v, got %v", 4, h.insertionIndex)
		}
		if h.fetchIndex != unset {
			t.Errorf("fetch index altered during insert")
		}
	})
}

func Test_history_GetRecord(t *testing.T) {
	t.Parallel()
	t.Run("empty fetches", func(t *testing.T) {
		h := NewHistory()
		// fetchIndex should be altered even if the newest record is empty
		t.Run("first fetch, setter", func(t *testing.T) {
			r := h.GetRecord()
			if r != "" {
				t.Errorf("empty history did not fetch empty string (got: %s)", r)
			}

			// fetching immediately on creation should wrap fetchIndex to arrayEnd,
			//	as history does not know if we just started or overflowed
			if h.fetchIndex != arrayEnd {
				t.Errorf("empty history did not unflow on decrement (fetchIndex: %d)", h.fetchIndex)
			}
		})
		// fetchIndex should not be altered on the second empty record
		t.Run("second fetch", func(t *testing.T) {
			r := h.GetRecord()
			if r != "" {
				t.Errorf("empty history did not fetch empty string (got: %s)", r)
			}
			if h.fetchIndex != arrayEnd {
				t.Errorf("second decrement occured despite empty history (fetchIndex: %d, expected: %d)", h.fetchIndex, arrayEnd)
			}
		})
		t.Run("unset, repeat first fetch", func(t *testing.T) {
			h.UnsetFetch()
			r := h.GetRecord()
			if r != "" {
				t.Errorf("empty history did not fetch empty string (got: %s)", r)
			}

			// fetching immediately on creation should wrap fetchIndex to arrayEnd,
			//	as history does not know if we just started or overflowed
			if h.fetchIndex != arrayEnd {
				t.Errorf("empty history did not unflow on decrement (fetchIndex: %d)", h.fetchIndex)
			}
		})
	})
	t.Run("at limit", func(t *testing.T) {
		h := NewHistory()
		var i uint16
		for i = 0; i < arraySize; i++ {
			h.Insert(fmt.Sprintf("%v", i))
		}
		t.Run("first GetRecord", func(t *testing.T) {
			if r := h.GetRecord(); r != fmt.Sprintf("%v", arrayEnd) ||
				r != h.commands[arrayEnd] {
			t.Errorf("GetRecord did not return last record. Expected %s, got %s. Commands: %v",
				fmt.Sprintf("%v", arrayEnd), r, h.commands)
			}
		})
		t.Run("second GetRecord", func(t *testing.T) {
			want := fmt.Sprintf("%v", arrayEnd-1)
			if r := h.GetRecord(); r != want || r != h.commands[arrayEnd-1] {
			t.Errorf("GetRecord did not return second-to-last record. Expected %s, got %s. Commands: %v",
				want, r, h.commands)
			}
		})
		t.Run("edge of underflow", func(t *testing.T){
			want := fmt.Sprintf("%v", 0)
			for i := arrayEnd-1; i > 1; i--{
				_ = h.GetRecord()
			}
			r := h.GetRecord();
			if h.fetchIndex != 0{
				t.Fatalf("fetch index error. r: %s, h: %+v",r, h)
			}
			if  r != want {
				t.Errorf("GetRecord did not return oldest (first) record. Expected %s, got %s. Commands: %v",
				want, r, h.commands)
			}
		})
		t.Run("underflow", func(t *testing.T){
			want := fmt.Sprintf("%v", 999)
			r := h.GetRecord();
			if h.fetchIndex != 999{
				t.Fatalf("fetch index error. r: %s, h: %+v",r, h)
			}
			if  r != want {
				t.Errorf("GetRecord did not return oldest (first) record. Expected %s, got %s. Commands: %v",
				want, r, h.commands)
			}
		})
	})
}

func Test_history_GetAllRecords(t *testing.T){
	t.Run("Clipped", func(t *testing.T) {
		cap := 50
		h := NewHistory()
		want := make([]string, cap)
		for i := 0; i < cap; i++{
			h.Insert("command")
			want[i] = "command"
		}
		rs := h.GetAllRecords()
		if len(rs) != cap {
			t.Errorf("GetAllRecords did not clip return. Expected %v (len %d). Got %v (len %d).",
			want, len(want), rs, len(rs))
		}
	})


	h := NewHistory()
	t.Run("no overflow", func(t *testing.T) {
		for i := int(arrayEnd); i >= 0; i--{
			h.Insert(fmt.Sprintf("%d", -i))
		}
		rs := h.GetAllRecords()
		for i := 0; i < int(arrayEnd); i++{
			if rs[i] != fmt.Sprintf("%d", -i){
				t.Fatalf("value mismatch: (index: %d) (want: %s, got (rs[i]): %s)", i, fmt.Sprintf("%d", -i), rs[i])
			}
		}
	})
	t.Run("single overflow", func(t *testing.T){
		h.Insert("A")
		rs := h.GetAllRecords()
		if rs[0] != "A" || rs[1] != "0"{
			t.Errorf("GetAllRecords did not sort newest first."+
			"Expected first record 'A' (got: %v), second record '0' (got: %v)", rs[0], 0)
		}
		if h.commands[0] != "A" || h.commands[1] != fmt.Sprintf("-%d", arrayEnd-1){
			t.Errorf("Command list corrupt on overflow."+
			"Expected [A, -998, -997, ...]. Got: [%s, %s, %s, ...]", 
			h.commands[0], h.commands[1], h.commands[2])
		}
	})
	
	
}