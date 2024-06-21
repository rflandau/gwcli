// Assumes a gravwell instance is running at `server` endpoint with credentials `user`, `pass`.
package query

import (
	"gwcli/clilog"
	"gwcli/connection"
	"testing"
	"time"

	"github.com/spf13/cobra"
)

const (
	server = "localhost:80"
	user   = "admin"
	pass   = "changeme"
)

// UUIDs are not seeded, so make sure the uuid1str const actually exists on the gravwell server.
// Unsetting the constant skips tests that require it
func TestGenerateQueryString(t *testing.T) {
	const uuid1str = "" // ex: 52985695-ae81-4e82-ba1d-bce54f96def7

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
			name:      "basic multiword argument query",
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

	// this cmd isn't being executed, so we have to call parse manually

	cmd.ParseFlags(args)

	return &cmd
}

func Test_tryQuery(t *testing.T) {
	// establish connection
	if err := connection.Initialize(server, false, true); err != nil {
		panic(err)
	}
	if err := connection.Login(user, pass); err != nil {
		panic(err)
	}
	// establish cli writer
	clilog.Init("gwcli.Test_tryQuery.log", "DEBUG")

	type args struct {
		qry      string
		duration time.Duration
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name:    "invalid query",
			args:    args{qry: "tags=gravwelll", duration: -2 * time.Hour},
			wantErr: true,
		},
		{
			name:    "whitespace query",
			args:    args{qry: " ", duration: -2 * time.Hour},
			wantErr: true,
		},
		{
			name:    "positive duration",
			args:    args{qry: " ", duration: 2 * time.Hour},
			wantErr: true,
		},
		{
			name:    "valid",
			args:    args{qry: "tag=gravwell", duration: -30 * time.Minute},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tryQuery(tt.args.qry, tt.args.duration)
			if err != nil {
				if tt.wantErr {
					return
				}
				t.Errorf("tryQuery() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			// unable to really compare search structs returned,
			// just check they were created as expected
			if got.ID == "" || got.SearchString != tt.args.qry {
				t.Errorf("tryQuery() invalid search struct: got struct %v", got)
				return
			}
		})
	}
}
