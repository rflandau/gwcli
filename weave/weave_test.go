package weave

import (
	"fmt"
	"reflect"
	"strings"
	"testing"
)

const longCSVLineCount = 17000

type inner struct {
	foo string
}

type outer struct {
	inner
	a        int
	b        uint
	c        float32
	d        string
	Exported float64
}

func TestToCSVHash(t *testing.T) {
	type args struct {
		st      []interface{}
		columns []string
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{"∃!c∃!r",
			args{
				st: []interface{}{
					outer{
						a:     10,
						b:     0,
						c:     5.0123,
						d:     "D",
						inner: inner{foo: "FOO"}}},
				columns: []string{"a"}},
			"a\n" + "10",
		},
		{"∀c∃!r, ordered as struct",
			args{
				st: []interface{}{
					outer{
						inner:    inner{foo: "FOO"},
						a:        10,
						b:        0,
						c:        5.0123,
						d:        "D",
						Exported: 3.145}},
				columns: []string{
					"a", "b", "c", "d", "foo", "Exported",
				}},
			"a,b,c,d,foo,Exported\n" + "10,0,5.0123,D,FOO,3.145",
		},
		{"∀c∃!r, ordered randomly",
			args{
				st: []interface{}{
					outer{
						inner:    inner{foo: "FOO"},
						a:        10,
						b:        0,
						c:        5.0123,
						d:        "D",
						Exported: 3.145}},
				columns: []string{
					"c", "foo", "Exported", "d", "a", "b",
				}},
			"c,foo,Exported,d,a,b\n" + "5.0123,FOO,3.145,D,10,0",
		},
		{"∀c5r, ordered randomly",
			args{
				st: []interface{}{
					outer{
						inner:    inner{foo: "FOO"},
						a:        10,
						b:        0,
						c:        5.0123,
						d:        "D",
						Exported: 3.145},
					outer{
						inner:    inner{foo: "FOO"},
						a:        57,
						b:        0,
						c:        5.0123,
						d:        "D",
						Exported: 3.145},
					outer{
						inner:    inner{foo: "FOO"},
						a:        10,
						b:        256,
						c:        5.0123,
						d:        "D",
						Exported: 3.145},
					outer{
						inner:    inner{foo: "FOO"},
						a:        10,
						b:        0,
						c:        5.0123,
						d:        "D",
						Exported: 3.145},
					outer{
						inner:    inner{foo: "FOO"},
						a:        10,
						b:        0,
						c:        5.0123,
						d:        "D!",
						Exported: 3.145}},
				columns: []string{
					"c", "foo", "Exported", "d", "a", "b",
				}},
			"c,foo,Exported,d,a,b\n" +
				"5.0123,FOO,3.145,D,10,0\n" +
				"5.0123,FOO,3.145,D,57,0\n" +
				"5.0123,FOO,3.145,D,10,256\n" +
				"5.0123,FOO,3.145,D,10,0\n" +
				"5.0123,FOO,3.145,D!,10,0",
		},
		{"∃c2r, non-existant column 'missing' and 'foobar'",
			args{
				st: []interface{}{
					outer{
						inner:    inner{foo: "FOO"},
						a:        10,
						b:        0,
						c:        5.0123,
						d:        "D",
						Exported: 3.145},
					outer{
						inner:    inner{foo: "FOO"},
						a:        10,
						b:        0,
						c:        5.0123,
						d:        "D",
						Exported: 3.145}},
				columns: []string{
					"c", "foo", "Exported", "missing", "d", "a", "b", "foobar",
				}},
			"c,foo,Exported,missing,d,a,b,foobar\n" + "5.0123,FOO,3.145,,D,10,0,\n" + "5.0123,FOO,3.145,,D,10,0,",
		},
		{"superfluous, no columns",
			args{
				st: []interface{}{
					outer{
						inner:    inner{foo: "FOO"},
						a:        10,
						b:        0,
						c:        5.0123,
						d:        "D",
						Exported: 3.145},
					outer{
						inner:    inner{foo: "FOO"},
						a:        10,
						b:        0,
						c:        5.0123,
						d:        "D",
						Exported: 3.145}},
				columns: []string{}},
			"",
		},
		{"superfluous, no data",
			args{
				st:      []interface{}{},
				columns: []string{"c", "foo", "Exported", "missing", "d", "a", "b", "foobar"}},
			"",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ToCSV(tt.args.st, tt.args.columns); got != tt.want {
				t.Errorf("\n---ToCSVHash()---\n'%v'\n---want---\n'%v'", got, tt.want)
			}
		})
	}

	// test against significant amounts of data
	t.Run("long data", func(t *testing.T) {
		// set up the data and structures
		type innerInnerInnerNest struct {
			iiin string
		}
		type innerInnerNest struct {
			innerInnerInnerNest
			iin string
		}
		type innerNest struct {
			innerInnerNest
			in string
		}
		type nest struct {
			innerNest
			n string
		}

		var data []nest = make([]nest, longCSVLineCount)
		for i := 0; i < longCSVLineCount; i++ {
			data[i] = nest{
				n: fmt.Sprintf("%dN", i), innerNest: innerNest{
					in: "IN", innerInnerNest: innerInnerNest{
						iin: "IIN", innerInnerInnerNest: innerInnerInnerNest{iiin: "IIIN"},
					},
				},
			}
		}

		var expectedBldr strings.Builder
		expectedBldr.Grow(longCSVLineCount * 16)    // roughly 14-16B per line; better overallocate
		expectedBldr.WriteString("n,in,iin,iiin\n") // header
		for i := 0; i < longCSVLineCount; i++ {
			expectedBldr.WriteString(
				fmt.Sprintf("%dN,IN,IIN,IIIN\n", i),
			)
		}

		actual := ToCSV(data, []string{"n", "in", "iin", "iiin"})
		expected := strings.TrimSpace(expectedBldr.String()) // chomp newline
		if actual != expected {
			// count newlines in parallel
			actualCountDone := make(chan int)
			var actualCount int
			// check line length
			go func() {
				actualCountDone <- strings.Count(actual, "\n")
				close(actualCountDone)
			}()

			expectedCountDone := make(chan int)
			var expectedCount int
			go func() {
				expectedCountDone <- strings.Count(expected, "\n")
				close(expectedCountDone)
			}()

			// wait for children
			actualCount = <-actualCountDone
			expectedCount = <-expectedCountDone

			if actualCount != expectedCount {
				t.Errorf("# of lines in actual (%d) <> # of lines in expected (%d)", actualCount, expectedCount)
			}

			t.Errorf("actual does not match expected!\n---actual---\n%s\n---expected---\n%s\n", actual, expected)
		}
	})

	t.Run("ptr", func(t *testing.T) {
		// define struct with pointer
		type ptrstruct struct {
			a   int
			ptr *int
		}

		ptrval := 5
		st := ptrstruct{
			a:   1,
			ptr: &ptrval,
		}

		want := "a,ptr\n" +
			"1,5"

		actual := ToCSV([]ptrstruct{st}, []string{"a", "ptr"})

		if actual != want {
			t.Errorf("\n---ToCSVHash()---\n'%v'\n---want---\n'%v'", actual, want)
		}

	})

	// nested pointers
	type ptrstruct struct {
		a int
		b string
	}
	type inner struct {
		inptr *int
		p     *ptrstruct
	}
	type outer struct {
		inner
		z uint
	}

	t.Run("embedded pointers, all initialized", func(t *testing.T) {
		inptrVal := -9
		ptrStructVal := ptrstruct{a: 0, b: "B"}
		v := outer{z: 10, inner: inner{inptr: &inptrVal, p: &ptrStructVal}}
		actual := ToCSV([]outer{v}, []string{"z", "inptr", "p", "a", "b"})
		expected := "z,inptr,p,a,b\n" +
			"10,-9,{0 B},,"
		if actual != expected {
			t.Errorf("\n---ToCSVHash()---\n'%v'\n---want---\n'%v'", actual, expected)
		}
	})
}

func TestFindQualifiedField(t *testing.T) {
	type lvl1 struct {
		a string
	}

	t.Run("depth 0", func(t *testing.T) {

		_, wantFound := reflect.TypeOf(lvl1{}).FieldByName("a")
		_, actualFound, actualErr := FindQualifiedField[lvl1]("a", lvl1{})
		if actualErr != nil {
			t.Error(actualErr)
		}
		if actualFound != wantFound {
			t.Errorf("found mismatch: actual (%v) != want (%v)", actualFound, wantFound)
		}
		// TODO compare fields
	})

	/*type args struct {
		qualCol string
		st      any
	}
	tests := []struct {
		name    string
		args    args
		want    reflect.StructField
		want1   bool
		wantErr bool
	}{

	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, got1, err := FindQualifiedField(tt.args.qualCol, tt.args.st)
			if (err != nil) != tt.wantErr {
				t.Errorf("FindQualifiedField() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("FindQualifiedField() got = %v, want %v", got, tt.want)
			}
			if got1 != tt.want1 {
				t.Errorf("FindQualifiedField() got1 = %v, want %v", got1, tt.want1)
			}
		})
	} */
}
