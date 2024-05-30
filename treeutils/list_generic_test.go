package treeutils

import (
	"testing"
)

func TestStructFields(t *testing.T) {
	type mbd struct {
		z string
	}
	type st struct {
		mbd
		a int
		b uint
	}

	t.Run("base", func(t *testing.T) {
		t.Error(StructFields(st{}))
	})

	/*type args struct {
		st any
	}
	tests := []struct {
		name        string
		args        args
		wantColumns []string
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if gotColumns := StructFields(tt.args.st); !reflect.DeepEqual(gotColumns, tt.wantColumns) {
				t.Errorf("StructFields() = %v, want %v", gotColumns, tt.wantColumns)
			}
		})
	}*/
}
