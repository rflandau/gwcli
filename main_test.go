// Tests from a complete-program perspective, confirming consistent input begets
// reliable output

package main

import (
	"bytes"
	"gwcli/tree"
	"io"
	"os"
	"strings"
	"testing"
)

var realStderr, mockStderr, realStdout, mockStdout *os.File

//#region non-interactive

func TestNonInteractive(t *testing.T) {
	defer restoreIO() // each test should result before checking results, but ensure a deferred restore

	t.Run("bad usage: no credentials", func(t *testing.T) {
		stdoutData, stderrData, err := mockIO()
		if err != nil {
			restoreIO()
			panic(err)
		}

		errCode := tree.Execute([]string{"--no-interactive"})
		t.Logf("error code: %d\n", errCode)
		restoreIO()
		results := <-stdoutData
		resultsErr := <-stderrData

		expFirstLine := "Error: username (-u) and password (-p) required"

		// don't really care if usage was printed; just check to first newline
		if resultsErr == "" || results != "" || strings.Split(resultsErr, "\n")[0] != expFirstLine {
			t.Fatalf("Expected stderr to begin with '%s', got:\n(stdout:\n%s)\n(stderr:\n%s)", expFirstLine, results, resultsErr)
		}
	})

}

//#endregion

func mockIO() (stdoutData chan string, stderrData chan string, err error){
	// capture stdout
	realStdout = os.Stdout
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
	realStderr = os.Stderr
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
