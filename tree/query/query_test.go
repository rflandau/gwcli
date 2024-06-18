// Assumes a gravwell instance is running at `server` endpoint with credentials `user`, `pass`.
// UUIDs are not seeded, so make sure the uuid1str const actually exists on the gravwell server.
// Unsetting the constant skips tests taht require it
package query

import (
	"gwcli/connection"
	"testing"

	"github.com/spf13/cobra"
)

const (
	server   = "localhost:80"
	user     = "admin"
	pass     = "changeme"
	uuid1str = "" // ex: 52985695-ae81-4e82-ba1d-bce54f96def7
)

func TestGenerateQueryString(t *testing.T) {
	if err := connection.Initialize(server, false, true); err != nil {
		panic(err)
	}
	if err := connection.Login(user, pass); err != nil {
		panic(err)
	}

	type args struct {
		// args managed by the cobra.Command, such as flags
		// global flags are assumed to be managed (per the constant above)
		flagArgs []string
		args     []string // leftover, positional arguments cobra would pass here after parsing
	}
	tests := []struct {
		name      string
		args      args
		wantQuery string
		wantErr   bool
		skip      bool
	}{
		{
			name:      "basic argument query",
			args:      args{[]string{}, []string{"tag=gravwell"}},
			wantQuery: "tag=gravwell", wantErr: false,
		},
		{
			name:      "basic multiwork argument query",
			args:      args{[]string{}, []string{"tag=dpkg words status"}},
			wantQuery: "tag=dpkg words status", wantErr: false,
		},
		{
			name:      "uuid " + uuid1str,
			args:      args{[]string{"-r", uuid1str}, []string{}},
			wantQuery: "tag=gravwell", wantErr: false,
			skip: uuid1str == "", // skip if constant is unset
		},
		{
			name:      "invalid uuid 'all-hail-the-gopher'",
			args:      args{[]string{"-r", "all-hail-the-gopher"}, []string{}},
			wantQuery: "", wantErr: true,
			skip: uuid1str == "", // skip if constant is unset
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.skip {
				t.SkipNow()
			}
			cmd := generateCobraCommand(tt.args.flagArgs)

			gotQuery, err := FetchQueryString(cmd.Flags(), tt.args.args)
			if (err != nil) != tt.wantErr {
				t.Errorf("GenerateQueryString() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if gotQuery != tt.wantQuery {
				t.Errorf("GenerateQueryString() = %v, want %v", gotQuery, tt.wantQuery)
			}
		})
	}
}

func generateCobraCommand(args []string) *cobra.Command {
	cmd := cobra.Command{Use: "test"}

	fs := initialLocalFlagSet()
	cmd.Flags().AddFlagSet(&fs)
	cmd.MarkFlagsRequiredTogether("name", "description", "schedule")

	// this cmd isn't being executed, so we have to call parse manually

	cmd.ParseFlags(args)

	return &cmd
}

/*
func TestOpenOutFile(t *testing.T) {
	fn := "testOpenOutFile_test_data.tmp"
	defer os.Remove(fn)
	expFileContents := []byte("Hello World\nHi Program\n")

	t.Run("create new file", func(t *testing.T) {
		args := []string{"-o", fn}
		fs := initialLocalFlagSet()
		fs.Parse(args)
		f, err := openOutFile(&fs)
		if err != nil {
			t.Fatal(err)
		} else if f == nil {
			t.Fatal("f is nil")
		}
		defer f.Close()

		if _, err := f.Write(expFileContents); err != nil {
			t.Fatal(err)
		}

		// check expected
		by, err := os.ReadFile(fn)
		if err != nil {
			t.Fatal(err)
		}
		if len(by) != len(expFileContents) {
			t.Fatalf("len mismatch: expected (len %d) %s, got (len %d) %s",
				len(expFileContents), expFileContents,
				len(by), by)
		}
		lenBy := len(by)
		for i := 0; i < lenBy; i++ {
			if by[i] != expFileContents[i] {
				t.Errorf("content mismatch @ byte %d: expected %v, got %v", i,
					expFileContents[i], by[i])
			}
		}
	})

	t.Run("append to existing file", func(t *testing.T) {
		appendedContents := []byte("I was appended")
		newExp := append(expFileContents, appendedContents...)
		args := []string{"-o", fn, "--append"}
		fs := initialLocalFlagSet()
		fs.Parse(args)
		f, err := openOutFile(&fs)
		if err != nil {
			t.Fatal(err)
		} else if f == nil {
			t.Fatal("f is nil")
		}
		defer f.Close()

		if _, err := f.Write(appendedContents); err != nil {
			t.Fatal(err)
		}

		// check expected
		by, err := os.ReadFile(fn)
		if err != nil {
			t.Fatal(err)
		}
		if len(by) != len(newExp) {
			t.Fatalf("len mismatch: expected (len %d) %s, got (len %d) %s",
				len(newExp), newExp,
				len(by), by)
		}
		lenBy := len(by)
		for i := 0; i < lenBy; i++ {
			if by[i] != newExp[i] {
				t.Errorf("content mismatch @ byte %d: expected %v, got %v", i,
					newExp[i], by[i])
			}
		}
	})

}
*/
