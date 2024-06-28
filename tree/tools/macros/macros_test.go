package macros_test

import (
	"gwcli/clilog"
	"gwcli/connection"
	"gwcli/tree/tools/macros/create"
	"gwcli/tree/tools/macros/delete"
	"io"
	"os"
	"path"
	"regexp"
	"strconv"
	"testing"

	"github.com/gravwell/gravwell/v3/client"
)

const (
	server = "localhost:80"
	user   = "admin"
	pass   = "changeme"
)

var (
	logFile     = path.Join(os.TempDir(), "gwcli.Test_NewMacroCreateAndDestroy.log")
	restLogFile = path.Join(os.TempDir(), "gwcli.Test_NewMacroCreateAndDestroy.rest.log")
)

var rgxCreatedID = regexp.MustCompile(`\(ID: \d*`)

// Using the normal generation method, tests creating a macro via flags and then
// destorying it via flags in two sepeate tests
// Assumes the macro does not exist prior to this function
func TestNewMacroCreateAndDestroy(t *testing.T) {
	// set up the logger
	clilog.Init(logFile, "DEBUG")

	// connect to the server
	if err := connection.Initialize(server,
		restLogFile,
		false,
		true); err != nil {
		panic(err)
	}
	if err := connection.Login(user, pass); err != nil {
		panic(err)
	}
	t.Cleanup(
		func() {
			connection.Client.Logout()
			connection.End()
		})

	var macroID string
	t.Run("create from long flags", func(t *testing.T) {
		// set up output
		outFile, err := os.CreateTemp(os.TempDir(), "gwcli_TestNewMacroCreate.*.out")
		// junk the outfile
		t.Cleanup(func() {
			fn := outFile.Name()
			outFile.Close()
			if !t.Failed() {
				os.Remove(path.Join(os.TempDir(), fn))
			}
		})
		if err != nil {
			t.Fail()
		}
		// generate the command
		cmd := create.NewMacroCreateAction().Action
		cmd.SetOut(outFile)

		cmd.ParseFlags([]string{
			"--name=TestNewMacroCreate",
			"--description=some desc",
			"--expansion=\"to expand to\"",
		})

		cmd.Run(cmd, []string{})

		if err := outFile.Sync(); err != nil {
			t.Log("failed to sync file " + outFile.Name())
			t.Fail()
		}

		if _, err := outFile.Seek(0, 0); err != nil {
			t.Log("failed to reset file " + outFile.Name() + " via seek")
			t.Fail()
		}

		// check the outfile file
		bytes, err := io.ReadAll(outFile)
		if err != nil {
			t.Log("failed to read from file " + outFile.Name())
			t.FailNow()
		}
		if len(bytes) < 1 {
			t.Log("no data in file " + outFile.Name())
			t.Fail()
		} else {
			t.Log(string(bytes))
			// fetch macro id for future deletion from
			prefixedID := string(rgxCreatedID.Find(bytes))[5:]
			//prefixedID = [:] // strip out the first couple characters
			_, err := strconv.ParseUint(prefixedID, 10, 64)
			if err != nil {
				t.Logf("failed to parse id from %v (%v): %v", prefixedID, prefixedID, err)
				t.Fail()
			}
			macroID = prefixedID
		}
	})

	t.Run("delete previous macro", func(t *testing.T) {

		// set up output
		outFile, err := os.CreateTemp(os.TempDir(), "gwcli_TestNewMacroDelete.*.out")
		// junk the outfile
		t.Cleanup(func() {
			fn := outFile.Name()
			outFile.Close()
			if !t.Failed() {
				os.Remove(path.Join(os.TempDir(), fn))
			}
		})
		if err != nil {
			t.Fail()
		}
		// generate the command
		cmd := delete.NewMacroDeleteAction().Action
		cmd.SetOut(outFile)

		cmd.ParseFlags([]string{
			"--id=" + macroID,
		})

		cmd.Run(cmd, []string{})

		// confirm the macro was actually deleted
		macroIDUint, err := strconv.ParseUint(macroID, 10, 64)
		if err != nil {
			t.Logf("failed to parse id from %v: %v", macroID, err)
			t.Fail()
		}
		_, err = connection.Client.GetMacro(macroIDUint)
		if err == nil {
			t.Logf("found macro with ID %v post delete; deletion did not occur", macroIDUint)
			t.Fail()
		}
		if err, ok := err.(*client.ClientError); !ok || err.StatusCode != 404 {
			t.Logf("expected 404 error seaching deleted macro; got %v", err)
		}
	})

}
