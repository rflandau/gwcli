package macros_test

import (
	"gwcli/clilog"
	"gwcli/connection"
	"gwcli/tree/tools/macros/create"
	"io"
	"os"
	"path"
	"testing"
)

const (
	server = "localhost:80"
	user   = "admin"
	pass   = "changeme"
)

// Using the normal generation method, tests creating a macro via flags and then
// destorying it via flags in two sepeate tests
// Assumes the macro does not exist prior to this function
func TestNewMacroCreateAndDestroy(t *testing.T) {
	// generate the command
	cmd := create.NewMacroCreateAction().Action

	// set up the logger
	clilog.Init("gwcli.Test_NewMacroCreateAction.log", "DEBUG")

	// connect to the server
	if err := connection.Initialize(server, false, true); err != nil {
		panic(err)
	}
	if err := connection.Login(user, pass); err != nil {
		panic(err)
	}

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
		cmd.SetOut(outFile)

		cmd.ParseFlags([]string{
			"--name=TestNewMacroCreate",
			"--description=some desc",
			"--expansion=\"to expand to\"",
		})

		cmd.Run(cmd, []string{})

		// check the outfile file
		bytes, err := io.ReadAll(outFile)
		if err != nil {
			t.FailNow()
		}
		if len(bytes) < 1 {
			t.Log("no data in file " + outFile.Name())
			t.Fail()
		} else {
			t.Log(string(bytes))
		}

	})

}
