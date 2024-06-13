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

	t.Run("bad usage: no credentials", func(t *testing.T) {
		stdoutData, stderrData, err := mockIO()
		if err != nil {
			restoreIO()
			panic(err)
		}

		tree.Execute([]string{"--no-interactive"})
		restoreIO()
		results := <-stdoutData
		resultsErr := <-stderrData

		expFirstLine := "Error: username (-u) and password (-p) required"

		// don't really care if usage was printed; just check to first newline
		if resultsErr == "" || results != "" || strings.Split(resultsErr, "\n")[0] != expFirstLine {
			t.Fatalf("Expected stderr to begin with '%s', got:\n(stdout:\n%s)\n(stderr:\n%s)", expFirstLine, results, resultsErr)
		}
	})

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

		args := strings.Split("-u admin -p changeme --insecure --no-interactive tools macros list --csv --columns=UID,Global,Name", " ")

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

	// TODO add a test for script-form query

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
