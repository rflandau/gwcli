// Tests from a complete-program perspective, confirming consistent input begets
// reliable output

package main

import (
	"bytes"
	"gwcli/connection"
	"gwcli/tree"
	"io"
	"os"
	"strings"
	"testing"

	grav "github.com/gravwell/gravwell/v3/client"
	"github.com/gravwell/gravwell/v3/utils/weave"
)

const ( // mock credentials
	user     = "admin"
	password = "changeme"
	server   = "localhost:80"
)

var realStderr, mockStderr, realStdout, mockStdout *os.File

//#region non-interactive

func TestNonInteractive(t *testing.T) {
	defer restoreIO() // each test should result before checking results, but ensure a deferred restore

	realStdout = os.Stdout
	realStderr = os.Stderr

	// connect to the server for manually calls
	client, err := grav.NewOpts(grav.Opts{Server: server, UseHttps: false, InsecureNoEnforceCerts: true})
	if err != nil {
		panic(err)
	}
	if err = client.Login(user, password); err != nil {
		panic(err)
	}

	// need to reset the client used by gwcli between runs
	connection.End()
	connection.Client = nil

	t.Run("tools macros list --csv", func(t *testing.T) {
		// generate results manually, for comparison
		myInfo, err := client.MyInfo()
		if err != nil {
			panic(err)
		}
		macros, err := client.GetUserMacros(myInfo.UID)
		if err != nil {
			panic(err)
		}
		columns := []string{"UID", "Global", "Name"}
		want := weave.ToCSV(macros, columns)

		// prepare IO
		stdoutData, stderrData, err := mockIO()
		if err != nil {
			restoreIO()
			panic(err)
		}

		args := strings.Split("-u admin -p changeme --insecure --script tools macros list --csv --columns=UID,Global,Name", " ")

		// run the test body
		errCode := tree.Execute(args)
		restoreIO()
		if errCode != 0 {
			t.Errorf("non-zero error code: %v", errCode)
		}
		results := <-stdoutData
		resultsErr := <-stderrData
		if resultsErr != "" {
			t.Errorf("non-empty stderr:\n(%v)", resultsErr)
		}

		// compare against expected
		if strings.TrimSpace(results) != strings.TrimSpace(want) {
			t.Errorf("output mismatch\nwant:\n(%v)\ngot:\n(%v)\n", want, results)
		}
	})

	connection.End()
	connection.Client = nil

	t.Run("query 'tags=gravwell'", func(t *testing.T) {
		//prepare IO
		stdoutData, stderrData, err := mockIO()
		if err != nil {
			restoreIO()
			panic(err)
		}

		// run the test body
		outfn := "testnoninteractive.query.json"
		qry := "query tag=gravwell"
		args := strings.Split("--insecure --script "+qry+
			" -o "+outfn+" --json", " ")

		errCode := tree.Execute(args)
		restoreIO()
		if errCode != 0 {
			t.Errorf("non-zero error code: %v", errCode)
		}

		results := <-stdoutData
		resultsErr := <-stderrData
		if resultsErr != "" {
			t.Errorf("non-empty stderr:\n(%v)", resultsErr)
		}
		// check that no data was output to stdout in script and -o mode
		if results != "" {
			t.Errorf("output mismatch\n expected no data output to stdout. got:\n(%v)\n", results)
		}

		// slurp the file, check for valid JSON
		output, err := os.ReadFile(outfn)
		t.Logf("slurping %v...", outfn)
		if err != nil {
			t.Fatal(err)
		} else if strings.TrimSpace(string(output)) == "" {
			t.Fatal("empty output file")
		}
		// we cannot check json validity because the grav client lib outputs individual JSON
		// records, not a single blob
		/*if !json.Valid(output) {
			t.Errorf("json is not valid")
		}*/
		// fetch the search and check that record counts line up
		searches, err := client.GetSearchHistoryRange(0, 5)
		if err != nil {
			t.Fatal(err)
		} else if len(searches) < 1 {
			t.Fatalf("found no previous searches")
		}
		//var search types.SearchLog
		for _, s := range searches {
			if s.UserQuery == qry {
				//search = s
				// get SearchHistory* does not pull back the searchID, meaning I
				// cannnot pull more details about the search
				// TODO
				break
			}
		}

		// clean up
		if !t.Failed() {
			os.Remove(outfn)
		}
	})

}

//#endregion

func mockIO() (stdoutData chan string, stderrData chan string, err error) {
	// capture stdout
	var readMockStdout *os.File
	readMockStdout, mockStdout, err = os.Pipe()
	if err != nil {
		return nil, nil, err
	}
	stdoutData = make(chan string) // pass data from read to write
	go func() {
		var buf bytes.Buffer
		io.Copy(&buf, readMockStdout)
		stdoutData <- buf.String()
	}()
	os.Stdout = mockStdout

	// capture stderr
	var readMockStderr *os.File
	readMockStderr, mockStderr, err = os.Pipe()
	if err != nil {
		return nil, nil, err
	}
	stderrData = make(chan string)
	go func() {
		var buf bytes.Buffer
		io.Copy(&buf, readMockStderr)
		stderrData <- buf.String()
	}()
	os.Stderr = mockStderr

	return stdoutData, stderrData, nil
}

func restoreIO() {
	// stdout
	if mockStdout != nil {
		_ = mockStdout.Close()
	}
	if realStdout == nil {
		panic("failed to restore stdout; no saved handle")
	}
	os.Stdout = realStdout

	// stderr
	if mockStderr != nil {
		_ = mockStderr.Close()
	}
	if realStderr == nil {
		panic("failed to restore stderr; no saved handle")
	}
	os.Stderr = realStderr
}
