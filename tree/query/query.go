// Core query module
// Query is important and complex enough to be broken into multiple files; this is the shared and
// central module entrypoint
package query

import (
	"fmt"
	"gwcli/action"
	"gwcli/busywait"
	"gwcli/clilog"
	"gwcli/connection"
	"gwcli/treeutils"
	"os"
	"strings"
	"time"

	"github.com/google/uuid"
	grav "github.com/gravwell/gravwell/v3/client"
	"github.com/gravwell/gravwell/v3/client/types"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

// the Gravwell client can only consume time formatted as follows
const timeFormat = "2006-01-02T15:04:05.999999999Z07:00"

const ( // defaults
	defaultDuration = 1 * time.Hour
)

var (
	ErrSuperfluousQuery = "query is empty and therefore ineffectual"
)

var localFS pflag.FlagSet

//#region command/action set up

func GenerateAction() action.Pair {
	cmd := treeutils.NewActionCommand("query", "submit a query",
		"Generate and send a query to the remote server. Results can be received via this cli or later on the web GUI.\n"+
			"All arguments after `query` will be passed to the instance as the string to query.", []string{}, run)

	localFS = initialLocalFlagSet()

	cmd.Example = "./gwcli -u USERNAME -p PASSWORD query tag=gravwell"

	cmd.Flags().AddFlagSet(&localFS)

	cmd.MarkFlagsRequiredTogether("name", "description", "schedule")

	return treeutils.GenerateAction(cmd, Query)
}

func initialLocalFlagSet() pflag.FlagSet {
	fs := pflag.FlagSet{}

	fs.DurationP("duration", "t", time.Hour*1, "the historical timeframe (now minus duration) the query should pour over.")
	fs.StringP("reference", "r", "", "a reference to a query library item to execute instead of a provided query.")
	fs.StringP("output", "o", "", "file to write results to. Truncates file unless --append is also given.")
	fs.Bool("append", false, "append to the given output file.")

	// scheduled searches
	fs.StringP("name", "n", "", "name for a scheduled search")
	fs.StringP("description", "d", "", "(flavour) description")
	fs.StringP("schedule", "s", "", "schedule this search to be run at a later date, over the given duration.")

	return fs
}

//#endregion

//#region cobra command

func run(cmd *cobra.Command, args []string) {
	var err error

	// fetch required flags
	duration, err := cmd.Flags().GetDuration("duration")
	if err != nil {
		clilog.TeeError(cmd.ErrOrStderr(), err.Error())
		return
	}

	if schedule, err := cmd.Flags().GetString("schedule"); err != nil {
		clilog.TeeError(cmd.ErrOrStderr(), err.Error())
		return
	} else if schedule != "" {
		var name, description, schedule string
		clilog.Writer.Infof("Scheduling search %v, %v, %v... (NYI)",
			name, description, schedule)
		// TODO implement scheduled searches
		return
	}

	q, err := FetchQueryString(cmd.Flags(), args)
	if err != nil {
		clilog.TeeError(cmd.ErrOrStderr(), err.Error())
		return
	} else if q == "" { // superfluous query, don't bother
		clilog.TeeError(cmd.ErrOrStderr(), ErrSuperfluousQuery)
		return
	}

	s, err := tryQuery(q, duration)
	if err != nil {
		clilog.TeeError(cmd.ErrOrStderr(), err.Error())
		return
	}

	// spin up a spinner
	spnrP := busywait.CobraNew()
	go func() {
		if err := connection.Client.WaitForSearch(s); err != nil {
			clilog.TeeError(cmd.ErrOrStderr(), err.Error())
			return
		}
		spnrP.Quit()
	}()

	if _, err := spnrP.Run(); err != nil {
		clilog.TeeError(cmd.ErrOrStderr(), err.Error())
		return
	}

	// TODO allow user to provide row count via --head to set last
	results, err := connection.Client.GetTextResults(s, 0, 500)
	if err != nil {
		clilog.TeeError(cmd.ErrOrStderr(), err.Error())
		return
	}

	of, err := openOutFile(cmd.Flags())
	if err != nil {
		clilog.TeeError(cmd.ErrOrStderr(), err.Error())
		return
	}
	defer of.Close()

	of.WriteString(fmt.Sprintf("%v", results))

	for _, e := range results.Entries {
		fmt.Printf("%s\n", e.Data)
	}
	//fmt.Printf("%#v\n", results)

}

// Pulls a query from args or a reference uuid, depending on if the latter is given
func FetchQueryString(fs *pflag.FlagSet, args []string) (query string, err error) {
	var ref string // query library uuid
	if ref, err = fs.GetString("reference"); err != nil {
		return "", err
	} else if strings.TrimSpace(ref) != "" {
		if err := uuid.Validate(ref); err != nil {
			return "", err
		}
		uuid, err := uuid.Parse(ref)
		if err != nil {
			return "", err
		}
		sl, err := connection.Client.GetSearchLibrary(uuid)
		if err != nil {
			return "", err
		}
		return sl.Query, nil
	}

	return strings.TrimSpace(strings.Join(args, " ")), nil
}

//#endregion

// Validates and (if valid) submits the given query to the connected server instance
func tryQuery(qry string, duration time.Duration) (grav.Search, error) {
	var err error
	// validate search query
	if err = connection.Client.ParseSearch(qry); err != nil {
		return grav.Search{}, fmt.Errorf("'%s' is not a valid query: %s", qry, err.Error())
	}

	end := time.Now()
	sreq := types.StartSearchRequest{
		SearchStart:  end.Add(-duration).Format(timeFormat),
		SearchEnd:    end.Format(timeFormat),
		Background:   false,
		SearchString: qry, // pull query from the commandline
		NoHistory:    false,
		Preview:      false,
	}
	go func() {
		clilog.Writer.Infof("Executing foreground search '%v' from %v -> %v",
			sreq.SearchString, sreq.SearchStart, sreq.SearchEnd)
	}()
	return connection.Client.StartSearchEx(sreq)
}

// Checks --output for a file path. If found, creates a file at that path and returns its handle.
// If --output is not set, returned file will be nil.
func openOutFile(fs *pflag.FlagSet) (*os.File, error) {
	var f *os.File
	if outfile, err := fs.GetString("output"); err != nil {
		return nil, err
	} else if outfile != "" {
		if append, err := fs.GetBool("append"); err != nil {
			return nil, err
		} else if append {
			if f, err = os.OpenFile(outfile, os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0644); err != nil {
				return nil, err
			}
			return f, nil
		}
		if f, err = os.Create(outfile); err != nil {
			return nil, err
		}
	}
	return f, nil
}
