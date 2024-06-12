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
			name:      "empty query",
			args:      args{[]string{}, []string{}},
			wantQuery: "", wantErr: true,
		},
		{
			name:      "uuid " + uuid1str,
			args:      args{[]string{"-r", uuid1str}, []string{}},
			wantQuery: "tag=gravwell", wantErr: false,
			skip: uuid1str == "", // skip if constant is unset
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.skip {
				t.SkipNow()
			}
			cmd := generateCobraCommand(tt.args.flagArgs)

			gotQuery, err := GenerateQueryString(cmd, tt.args.args)
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
