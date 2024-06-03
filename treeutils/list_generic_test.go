package treeutils

import (
	"gwcli/weave"
	"reflect"
	"testing"
)

func TestStructFields(t *testing.T) {
	type dblmbd struct {
		y string
	}
	type mbd struct {
		dblmbd
		z string
	}
	type triple struct {
		mbd
		ins mbd
		dbl dblmbd
		a   int
		b   uint
	}

	type inner2 struct {
		z    *string
		none string
	}

	type ptr struct {
		a        *int
		b        *int
		innerptr *inner2
		inner    inner2
		non      string
	}

	type args struct {
		st any
	}

	triple_want := []string{"mbd.dblmbd.y", "mbd.z", "ins.dblmbd.y", "ins.z", "dbl.y", "a", "b"}

	tests := []struct {
		name        string
		args        args
		wantColumns []string
	}{
		{"single level", args{st: dblmbd{y: "y string"}}, []string{"y"}},
		{"second level", args{st: mbd{z: "z string", dblmbd: dblmbd{y: "y sting"}}}, []string{"dblmbd.y", "z"}},
		{"third level", args{
			st: triple{
				a:   -780,
				b:   1,
				dbl: dblmbd{y: "y string"},
				ins: mbd{z: "z string", dblmbd: dblmbd{y: "y string 2"}},
				mbd: mbd{dblmbd: dblmbd{y: "y string 3"},
					z: "z string 2"},
			}}, triple_want},
		{"third level valueless", args{st: triple{}}, triple_want},
		{"third level pointer", args{st: &triple{}}, triple_want},
		{"pointers", args{ptr{}}, []string{"a", "b", "innerptr.z", "innerptr.none", "inner.z", "inner.none", "non"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotColumns, err := StructFields(tt.args.st)
			if err != nil {
				t.Error(err)
			}
			if !reflect.DeepEqual(gotColumns, tt.wantColumns) {
				t.Errorf("StructFields() = %v, want %v", gotColumns, tt.wantColumns)
			}
		})
	}
	// validate errors
	t.Run("struct is nil", func(t *testing.T) {
		c, err := StructFields(nil)
		if err.Error() != weave.ErrStructIsNil || c != nil {
			t.Errorf("Error value mismatch: err: %v c: %v", err, c)
		}
	})
	t.Run("not a struct", func(t *testing.T) {
		m := make(map[string]int)
		c, err := StructFields(m)
		if err.Error() != weave.ErrNotAStruct || c != nil {
			t.Errorf("Error value mismatch: err: %v c: %v", err, c)
		}
	})
}

func Test_format_String(t *testing.T) {
	tests := []struct {
		name string
		f    format
		want string
	}{
		{"JSON", json, "JSON"},
		{"CSV", csv, "CSV"},
		{"table", table, "table"},
		{"unknown", 5, "unknown format (5)"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.f.String(); got != tt.want {
				t.Errorf("format.String() = %v, want %v", got, tt.want)
			}
		})
	}
}
