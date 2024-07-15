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
	"gwcli/mother"
	"gwcli/stylesheet"
	"gwcli/tree/query/datascope"
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
	defaultDuration = 1 * time.Hour

	pageSize = 500 // fetch results page by page

	NoResultsText = "No results found for given query"
)

var (
	ErrSuperfluousQuery = "query is empty and therefore ineffectual"
)

var localFS pflag.FlagSet

//#region command/action set up

func NewQueryAction() action.Pair {
	cmd := treeutils.NewActionCommand("query", "submit a query",
		"Generate and send a query to the remote server either by arguments or the interactive query builder.\n"+
			"All bare arguments after `query` will be passed to the instance as the query string.", []string{"q", "search"}, run)

	localFS = initialLocalFlagSet()

	cmd.Example = "./gwcli query tag=gravwell"

	cmd.Flags().AddFlagSet(&localFS)

	//cmd.MarkFlagsRequiredTogether("name", "description", "schedule")

	return treeutils.GenerateAction(cmd, Query)
}

func initialLocalFlagSet() pflag.FlagSet {
	fs := pflag.FlagSet{}

	fs.DurationP("duration", "t", time.Hour*1, "the historical timeframe (now minus duration) the query should pour over.\nEx: the past hour")
	//fs.StringP("reference", "r", "", "a reference to a query library item to execute instead of a provided query.")
	fs.StringP("output", "o", "", stylesheet.FlagOutputDesc)
	fs.Bool("append", false, stylesheet.FlagAppendDesc)
	fs.Bool("json", false, stylesheet.FlagJSONDesc)
	fs.Bool("csv", false, stylesheet.FlagCSVDesc)

	// scheduled searches
	fs.StringP("name", "n", "", "SCHEDULED. a title for the scheduled search")
	fs.StringP("description", "d", "", "SCHEDULED. a description of the search")
	fs.StringP("schedule", "s", "", "SCHEDULED. 5-cron-time schedule for execution")

	return fs
}

//#endregion

//#region cobra command

func run(cmd *cobra.Command, args []string) {
	var err error

	// fetch flags
	flags, err := transmogrifyFlags(cmd.Flags())
	if err != nil {
		clilog.Tee(clilog.ERROR, cmd.ErrOrStderr(), err.Error())
		return
	}

	// TODO pull qry from referenceID, if given

	qry := strings.TrimSpace(strings.Join(args, " "))

	if qry == "" { // superfluous query
		if flags.script { // fail out
			clilog.Tee(clilog.INFO, cmd.OutOrStdout(), "query is empty. Exitting...")
			return
		}

		// spawn mother
		if err := mother.Spawn(cmd.Root(), cmd, args); err != nil {
			clilog.Tee(clilog.CRITICAL, cmd.ErrOrStderr(),
				"failed to spawn a mother instance: "+err.Error())
		}
		return
	}

	// check if it is a scheduled query
	if flags.schedule.cronfreq != "" {
		id, invalid, err := connection.CreateScheduledSearch(flags.schedule.name, flags.schedule.desc,
			flags.schedule.cronfreq, qry, flags.duration)
		if invalid != "" { // bad parameters
			clilog.Tee(clilog.INFO, cmd.OutOrStdout(), invalid)
			return
		} else if err != nil {
			clilog.Tee(clilog.ERROR, cmd.ErrOrStderr(), err.Error())
		}
		clilog.Tee(clilog.INFO, cmd.OutOrStdout(),
			fmt.Sprintf("Successfully scheduled query (ID: %v)", id))
		return
	}
	// submit the immediate query
	var search grav.Search
	if s, err := connection.StartQuery(qry, -flags.duration); err != nil {
		clilog.Tee(clilog.ERROR, cmd.ErrOrStderr(), err.Error())
		return
	} else {
		search = s
	}

	// wait for query to complete
	if err := waitForSearch(search, flags.script); err != nil {
		clilog.Tee(clilog.ERROR, cmd.ErrOrStderr(), err.Error())
		return
	}

	// if an output file was given and we are in script mode, stream the results into it
	// if we are not in script mode, DS will automatically download results for us
	if flags.outfn != "" && flags.script {
		// open the file
		var of *os.File
		if of, err = openFile(flags.outfn, flags.append); err != nil {
			clilog.Tee(clilog.ERROR, cmd.ErrOrStderr(), err.Error())
			return
		}
		defer of.Close()
		if err := connection.DownloadResults(&search, of, flags.json, flags.csv); err != nil {
			clilog.Tee(clilog.ERROR, cmd.ErrOrStderr(), err.Error())
		} else {
			fmt.Fprintln(cmd.OutOrStdout(),
				connection.DownloadQuerySuccessfulString(of.Name(), flags.append))
		}
		return
	}

	if results, err := fetchTextResults(search); err != nil {
		clilog.Tee(clilog.ERROR, cmd.ErrOrStderr(), err.Error())
		return
	} else if len(results) > 0 {
		// if script mode, spew the result to stdout and quit
		if flags.script {
			for _, r := range results {
				fmt.Fprintf(cmd.OutOrStdout(), "%s\n", r.Data)
			}
			return
		}

		// if interactive mode, feed the results to datascope for user control
		var strs []string = make([]string, len(results))
		for i, r := range results {
			strs[i] = string(r.Data)
		}

		// spin up a scrolling pager to display
		if p, err := datascope.CobraNew(
			strs, &search, flags.outfn, flags.append, flags.json, flags.csv,
		); err != nil {
			clilog.Tee(clilog.ERROR, cmd.ErrOrStderr(), err.Error())
			return
		} else {
			if _, err := p.Run(); err != nil {
				clilog.Tee(clilog.ERROR, cmd.ErrOrStderr(), err.Error())
				return
			}
		}
	} else { // no results to display
		fmt.Fprintln(cmd.OutOrStdout(), NoResultsText)
	}

}

// Stops execution and waits for the given search to complete.
// Adds a spinner if not in script mode.
func waitForSearch(s grav.Search, scriptMode bool) error {
	// in script mode, wait syncronously
	if scriptMode {
		if err := connection.Client.WaitForSearch(s); err != nil {
			return err
		}
	} else {
		// outside of script mode wait via goroutine so we can display a spinner
		spnrP := busywait.CobraNew()
		go func() {
			if err := connection.Client.WaitForSearch(s); err != nil {
				clilog.Writer.Error(err.Error())
			}
			spnrP.Quit()
		}()

		if _, err := spnrP.Run(); err != nil {
			return err
		}
	}
	return nil
}

//#endregion

// Pulls a query from args or a reference uuid, depending on if the latter is given.
// Does not consider an empty query to be an error.
func fetchQueryString(fs *pflag.FlagSet, args []string) (query string, err error) {
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

// just enough information to schedule a given query
type schedule struct {
	name     string
	desc     string
	cronfreq string // run frequency in cron format
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

// Fetches all text results related to the given search by continually re-fetching until no more
// results remain
func fetchTextResults(s grav.Search) ([]types.SearchEntry, error) {
	// return results for output to terminal
	// batch results until we have the last of them
	var (
		results []types.SearchEntry = make([]types.SearchEntry, 0, pageSize)
		low     uint64              = 0
		high    uint64              = pageSize
	)
	for { // accumulate the results
		r, err := connection.Client.GetTextResults(s, low, high)
		if err != nil {
			return nil, err
		}
		results = append(results, r.Entries...)
		if !r.AdditionalEntries { // all records obtained
			break
		}
		// ! Get*Results is half-open [)
		low = high
		high = high + pageSize
	}

	clilog.Writer.Infof("%d results obtained", len(results))

	return results, nil
}
