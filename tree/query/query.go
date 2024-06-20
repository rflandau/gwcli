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

const (
	// the Gravwell client can only consume time formatted as follows
	timeFormat = "2006-01-02T15:04:05.999999999Z07:00"

	defaultDuration = 1 * time.Hour

	pageSize = 500 // fetch results page by page
)

var (
	ErrSuperfluousQuery = "query is empty and therefore ineffectual"
)

var localFS pflag.FlagSet

//#region command/action set up

func GenerateAction() action.Pair {
	cmd := treeutils.NewActionCommand("query", "submit a query",
		"Generate and send a query to the remote server. Results can be received via this cli or later on the web GUI.\n"+
			"All bare arguments after `query` will be passed to the instance as the query string.", []string{"q", "search"}, run)

	localFS = initialLocalFlagSet()

	cmd.Example = "./gwcli -u USERNAME -p PASSWORD query tag=gravwell"

	cmd.Flags().AddFlagSet(&localFS)

	cmd.MarkFlagsRequiredTogether("name", "description", "schedule")

	return treeutils.GenerateAction(cmd, Query)
}

func initialLocalFlagSet() pflag.FlagSet {
	fs := pflag.FlagSet{}

	fs.DurationP("duration", "t", time.Hour*1, "the historical timeframe (now minus duration) the query should pour over.\nEx: the past hour")
	fs.StringP("reference", "r", "", "a reference to a query library item to execute instead of a provided query.")
	fs.StringP("output", "o", "", "file to write results to. Truncates file unless --append is also given.")
	fs.Bool("append", false, "append to the given output file instead of truncating.")

	// scheduled searches
	fs.StringP("name", "n", "", "name for a scheduled search.")
	fs.StringP("description", "d", "", "(flavour) description.")
	fs.StringP("schedule", "s", "", "schedule this search to be run at a later date, over the given duration.")

	return fs
}

//#endregion

//#region cobra command

func run(cmd *cobra.Command, args []string) {
	var err error

	var (
		duration time.Duration
		qry      string
		s        grav.Search // ongoing search
		script   bool        // script mode
	)

	// fetch required flags
	duration, err = cmd.Flags().GetDuration("duration")
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
	if script, err = cmd.Flags().GetBool("no-interactive"); err != nil {
		clilog.TeeError(cmd.ErrOrStderr(), err.Error())
		return
	}

	qry, err = FetchQueryString(cmd.Flags(), args)
	if err != nil {
		clilog.TeeError(cmd.ErrOrStderr(), err.Error())
		return
	} else if qry == "" { // superfluous query, don't bother
		clilog.TeeError(cmd.ErrOrStderr(), ErrSuperfluousQuery)
		return
	}

	// submit the query
	s, err = tryQuery(qry, duration)
	if err != nil {
		clilog.TeeError(cmd.ErrOrStderr(), err.Error())
		return
	}

	if script {
		// in script mode, wait syncronously
		if err := connection.Client.WaitForSearch(s); err != nil {
			clilog.TeeError(cmd.ErrOrStderr(), err.Error())
			return
		}
	} else {
		// outside of script mode wait via goroutine so we can display a spinner
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
	}

	clilog.Writer.Infof("Search succeeded. Fetching results (renderer %v)...", s.RenderMod)

	// if we are outputting to a terminal AND not in script mode, spin up the displayer module
	// TODO

	// prepare output file
	var of *os.File
	if outfile, err := cmd.Flags().GetString("output"); err != nil {
		clilog.TeeError(cmd.ErrOrStderr(), err.Error())
		return
	} else if outfile = strings.TrimSpace(outfile); outfile != "" {
		append, err := cmd.Flags().GetBool("append")
		if err != nil {
			clilog.TeeError(cmd.ErrOrStderr(), err.Error())
			return
		}
		if of, err = openFile(outfile, append); err != nil {
			clilog.TeeError(cmd.ErrOrStderr(), err.Error())
			return
		}
	}
	// only write to output file if it was given/not null
	if of != nil {
		defer of.Close()
		// if we are outputting to a file, use the provided Download functionality
		// TODO unclear if an empty TR will use the search's timeframe, as desired
		// TODO switch format on rendermod?
		rc, err := connection.Client.DownloadSearch(s.ID, types.TimeRange{}, "text")
		if err != nil {
			clilog.TeeError(cmd.ErrOrStderr(), err.Error())
			return
		}

		b, err := of.ReadFrom(rc)
		if err != nil {
			clilog.TeeError(cmd.ErrOrStderr(), err.Error())
			return
		}
		clilog.Writer.Infof("Streamed %d bytes into %s", b, of.Name())
	} else {
		// batch results until we have the last of them
		var (
			results []types.SearchEntry = make([]types.SearchEntry, 0, pageSize)
			low     uint64              = 0
			high    uint64              = 0
		)
		for {
			r, err := connection.Client.GetTextResults(s, low, high)
			if err != nil {
				clilog.TeeError(cmd.ErrOrStderr(), err.Error())
				return
			}
			results = append(results, r.Entries...)
			if !r.AdditionalEntries { // all records obtained
				break
			}
			low = high + 1 // ! this assumes Get*Results is inclusive
			high = high + pageSize + 1

		}

		clilog.Writer.Infof("%d results obtained", len(results))

		for _, r := range results {
			fmt.Printf("%s\n", r.Data)
		}
	}

}

//#endregion

// Pulls a query from args or a reference uuid, depending on if the latter is given.
// Does not consider an empty query to be an error.
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

// Opens and returns a file handle, configured by the state of append.
//
// Errors are logged to clilogger internally
func openFile(path string, append bool) (*os.File, error) {
	var flags int = os.O_WRONLY | os.O_CREATE
	if append { // check append
		flags |= os.O_APPEND
	} else {
		flags |= os.O_TRUNC
	}

	f, err := os.OpenFile(path, flags, 0644)
	if err != nil {
		clilog.Writer.Errorf("Failed to open file %s (flags %d, mode %d): %v", path, flags, 0644, err)
		return nil, err
	}

	if s, err := f.Stat(); err != nil {
		clilog.Writer.Warnf("Failed to stat file %s: %v", f.Name(), err)
	} else {
		clilog.Writer.Debugf("Opened file %s of size %v", f.Name(), s.Size())
	}

	return f, nil
}
