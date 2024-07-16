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
	"gwcli/utilities/uniques"
	"io"
	"os"
	"strings"
	"time"

	grav "github.com/gravwell/gravwell/v3/client"
	"github.com/gravwell/gravwell/v3/client/types"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

const (
	defaultDuration = 1 * time.Hour

	pageSize = 500 // fetch results page by page

	NoResultsText = "No results found for given query"

	helpDesc = "Generate and send a query to the remote server either by arguments or " +
		"the interactive query builder.\n" +
		"All bare arguments after `query` will be passed to the instance as the query string.\n" +
		"\n" +
		"Omitting --script will open the results in an interactive viewing pane with additional" +
		"functionality for downloading the results to a file or scheduling this query to run in " +
		"the future" +
		"\n" +
		"If --json or --csv is not given when outputting to a file (`-o`), the results will be " +
		"text (if able) or an archive binary blob (if unable), depending on the query's render " +
		"module.\n" +
		"gwcli will not dump binary to terminal; you must supply -o if the results are a binary " +
		"blob (aka: your query uses a chart-style renderer)."
)

var (
	ErrSuperfluousQuery = "query is empty and therefore ineffectual"
)

var localFS pflag.FlagSet

//#region command/action set up

func NewQueryAction() action.Pair {
	cmd := treeutils.NewActionCommand("query", "submit a query",
		helpDesc,
		[]string{"q", "search"}, run)

	localFS = initialLocalFlagSet()

	cmd.Example = "./gwcli query \"tag=gravwell\""

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

	// branch on script mode
	if flags.script {
		runNonInteractive(cmd, flags, qry)
		return
	}
	runInteractive(cmd, flags, qry)
}

// run function with --script given, making it entirely independent of user input.
// Results will be output to a file (if given) or dumped into stdout.
func runNonInteractive(cmd *cobra.Command, flags queryflags, qry string) {
	var err error

	if flags.schedule.cronfreq != "" { // check if it is a scheduled query
		// warn about ignored flags
		if clilog.Active(clilog.WARN) { // only warn if WARN level is enabled
			if flags.outfn != "" {
				fmt.Fprint(cmd.ErrOrStderr(), uniques.WarnFlagIgnore("output", "schedule")+"\n")
			}
			if flags.append {
				fmt.Fprint(cmd.ErrOrStderr(), uniques.WarnFlagIgnore("append", "schedule")+"\n")
			}
			if flags.json {
				fmt.Fprint(cmd.ErrOrStderr(), uniques.WarnFlagIgnore("json", "schedule")+"\n")
			}
			if flags.csv {
				fmt.Fprint(cmd.ErrOrStderr(), uniques.WarnFlagIgnore("csv", "schedule")+"\n")
			}
		}

		// if a name was not given, populate a default name
		if flags.schedule.name == "" {
			flags.schedule.name = "cli_" + time.Now().Format(uniques.SearchTimeFormat)
		}
		// if a description was not given, populate a default description
		if flags.schedule.desc == "" {
			flags.schedule.desc = "generated in gwcli @" + time.Now().Format(uniques.SearchTimeFormat)
		}

		id, invalid, err := connection.CreateScheduledSearch(
			flags.schedule.name, flags.schedule.desc,
			flags.schedule.cronfreq, qry,
			flags.duration,
		)
		if invalid != "" { // bad parameters
			clilog.Tee(clilog.INFO, cmd.ErrOrStderr(), invalid)
			return
		} else if err != nil {
			clilog.Tee(clilog.ERROR, cmd.ErrOrStderr(), err.Error())
		}
		clilog.Tee(clilog.INFO, cmd.OutOrStdout(),
			fmt.Sprintf("Successfully scheduled query '%v' (ID: %v)", flags.schedule.name, id))
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
	if err := waitForSearch(search, true); err != nil {
		clilog.Tee(clilog.ERROR, cmd.ErrOrStderr(), err.Error())
		return
	}

	// fetch the data from the search
	var (
		results io.ReadCloser
		format  string
	)
	if results, format, err = connection.DownloadSearch(
		&search, types.TimeRange{}, flags.csv, flags.json,
	); err != nil {
		clilog.Tee(clilog.ERROR, cmd.ErrOrStderr(),
			fmt.Sprintf("failed to retrieve results from search %s (format %v): %v",
				search.ID, format, err.Error()))
		return
	}
	defer results.Close()

	// if an output file was given, write results into it
	if flags.outfn != "" {
		// open the file
		var of *os.File
		if of, err = openFile(flags.outfn, flags.append); err != nil {
			clilog.Tee(clilog.ERROR, cmd.ErrOrStderr(), err.Error())
			return
		}
		defer of.Close()

		// consumes the results and spit them into the open file
		if b, err := of.ReadFrom(results); err != nil {
			clilog.Tee(clilog.ERROR, cmd.ErrOrStderr(), err.Error())
			return
		} else {
			clilog.Writer.Infof("Streamed %d bytes (format %v) into %s", b, format, of.Name())
		}
		// stdout output is acceptible as the user is redirecting actual results to a file.
		fmt.Fprintln(cmd.OutOrStdout(),
			connection.DownloadQuerySuccessfulString(of.Name(), flags.append, format))
		return
	} else if format == types.DownloadArchive { // check for binary output
		fmt.Fprintf(cmd.OutOrStdout(), "refusing to dump binary blob (format %v) to stdout.\n"+
			"If this is intentional, re-run with -o <FILENAME>.\n", format)
	} else { // text results, stdout
		if r, err := io.ReadAll(results); err != nil {
			clilog.Tee(clilog.ERROR, cmd.ErrOrStderr(), err.Error())
			return
		} else {
			if len(r) == 0 {
				fmt.Fprintln(cmd.OutOrStdout(), "no results to display")
			} else {
				fmt.Fprintf(cmd.OutOrStdout(), "%s", r)
			}
		}
	}

}

// run function without --script given, making it acceptable to rely on user input
// NOTE: download and schedule flags are handled inside of datascope
func runInteractive(cmd *cobra.Command, flags queryflags, qry string) {
	// submit the immediate query
	var search grav.Search
	if s, err := connection.StartQuery(qry, -flags.duration); err != nil {
		clilog.Tee(clilog.ERROR, cmd.ErrOrStderr(), err.Error())
		return
	} else {
		search = s
	}

	// wait for query to complete
	if err := waitForSearch(search, false); err != nil {
		clilog.Tee(clilog.ERROR, cmd.ErrOrStderr(), err.Error())
		return
	}

	// get results to pass to data scope
	var results []string
	switch search.RenderMod {
	case types.RenderNameTable:
		if columns, rows, err := fetchTableResults(search); err != nil {
			clilog.Tee(clilog.ERROR, cmd.ErrOrStderr(), err.Error())
			return
		} else if len(rows) != 0 {
			// format the table datascope
			results = make([]string, len(rows)+1)
			results[0] = strings.Join(columns, ",")
			for i, row := range rows {
				results[i+1] = strings.Join(row.Row, ",")
			}
		}
	case types.RenderNameRaw, types.RenderNameText, types.RenderNameHex:
		if rawResults, err := fetchTextResults(search); err != nil {
			clilog.Tee(clilog.ERROR, cmd.ErrOrStderr(), err.Error())
			return
		} else if len(rawResults) != 0 {
			// format the data for datascope
			results = make([]string, len(rawResults))
			for i, r := range rawResults {
				results[i] = string(r.Data)
			}
		}
	default:
		fmt.Fprintf(cmd.OutOrStdout(), "Unable to display results of type %v.\n",
			search.RenderMod)
		return
	}
	if results == nil {
		fmt.Fprintln(cmd.OutOrStdout(), NoResultsText)
		return
	}

	// pass results into datascope
	// spin up a scrolling pager to display
	if p, err := datascope.CobraNew(
		results, &search,
		flags.outfn, flags.append, flags.json, flags.csv,
		flags.schedule.cronfreq, flags.schedule.name, flags.schedule.desc,
	); err != nil {
		clilog.Tee(clilog.ERROR, cmd.ErrOrStderr(), err.Error())
		return
	} else {
		if _, err := p.Run(); err != nil {
			clilog.Tee(clilog.ERROR, cmd.ErrOrStderr(), err.Error())
			return
		}
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

// Sister subroutine to fetchTextResults()
func fetchTableResults(s grav.Search) (
	columns []string, rows []types.TableRow, err error,
) {
	// return results for output to terminal
	// batch results until we have the last of them
	var (
		low  uint64 = 0
		high uint64 = pageSize
		r    types.TableResponse
	)
	rows = make([]types.TableRow, 0, pageSize)
	for { // accumulate the row results
		r, err = connection.Client.GetTableResults(s, low, high)
		if err != nil {
			return nil, nil, err
		}
		rows = append(rows, r.Entries.Rows...)
		if !r.AdditionalEntries { // all records obtained
			break
		}
		// ! Get*Results is half-open [)
		low = high
		high = high + pageSize
	}

	// save off columns
	columns = r.Entries.Columns

	clilog.Writer.Infof("%d results obtained", len(rows))

	return
}
