/**
 * Singleton instantiation of the gravwell client library.
 * The client is instantiated and exported in this package so it can be shared
 * by the rest of the application without trying to pass pointers through cobra
 * subroutine signatures.
 */

package connection

import (
	"errors"
	"fmt"
	"gwcli/clilog"
	"os"
	"path"
	"strings"

	grav "github.com/gravwell/gravwell/v3/client"
	"github.com/gravwell/gravwell/v3/client/objlog"
	"github.com/gravwell/gravwell/v3/client/types"
	"github.com/gravwell/gravwell/v3/ingest/log"
)

// files within the config directory
const (
	tokenName   = "token"
	restLogName = "rest.log"
)

// all data persistent data is store in $os.UserConfigDir/gwcli/
// or local to the instantiation, if that fails
var cfgDir string // set on init

// on startup, identify and cache the config directory
func init() {
	const cfgSubFolder = "gwcli"
	cd, err := os.UserConfigDir()
	if err != nil {
		cd = "."
	}
	cfgDir = path.Join(cd, cfgSubFolder)
}

var Client *grav.Client

var myInfo types.UserDetails
var myInfoCached bool

// Initializes Client using the given connection string of the form <host>:<port>.
// Destroys a pre-existing connection (but does not log out), if there was one.
// restLogPath should be left empty outside of test packages
func Initialize(conn string, UseHttps, InsecureNoEnforceCerts bool, restLogPath string) (err error) {
	if Client != nil {
		Client.Close()
		// TODO should probably close the logger, if possible externally
		Client = nil
	}

	var l objlog.ObjLog = nil
	if restLogPath != "" { // used for testing, not intended for production modes
		l, err = objlog.NewJSONLogger(restLogPath)
		if err != nil {
			return err
		}
	} else if clilog.Writer != nil && clilog.Writer.GetLevel() >= log.Level(clilog.INFO) {
		// spin up the rest logger if in INFO+
		l, err = objlog.NewJSONLogger(path.Join(cfgDir, restLogName))
		if err != nil {
			return err
		}
	}

	if Client, err = grav.NewOpts(
		grav.Opts{
			Server:                 conn,
			UseHttps:               UseHttps,
			InsecureNoEnforceCerts: InsecureNoEnforceCerts,
			ObjLogger:              l,
		}); err != nil {
		return err
	}
	return nil
}

type Credentials struct {
	Username     string
	Password     string
	PassfilePath string
}

// Login the initialized Client. Attempts to use a JWT token first, then falls back to supplied
// credentials.
//
// Ineffectual if Client is already logged in.
func Login(cred Credentials, scriptMode bool) (err error) {
	if Client.LoggedIn() {
		return nil
	}

	// login is attempted via JWT token first
	// If any stage in the process fails
	// the error is logged and we fall back to flags and prompting
	if err := LoginViaToken(); err != nil {
		// jwt token failure; log and move on
		clilog.Writer.Warnf("Failed to login via JWT token: %v", err)

		if err = loginViaCredentials(cred, scriptMode); err != nil {
			clilog.Writer.Errorf("Failed to login via credentials: %v", err)
			return err
		}
		clilog.Writer.Infof("Logged in via credentials")

		if err := CreateToken(); err != nil {
			clilog.Writer.Warnf(err.Error())
			// failing to create the token is not fatal
		}
	} else {
		clilog.Writer.Infof("Logged in via JWT")
	}

	return nil
}

// Attempts to login via JWT token in the user's config directory.
// Returns an error on failures. This error should be considered nonfatal and the user logged in via
// an alternative method instead.
func LoginViaToken() (err error) {
	var tknbytes []byte
	// NOTE the reversal of standard error checking (`err == nil`)
	if tknbytes, err = os.ReadFile(path.Join(cfgDir, tokenName)); err == nil {
		if err = Client.ImportLoginToken(string(tknbytes)); err == nil {
			if err = Client.TestLogin(); err == nil {
				return nil
			}
		}
	}
	return
}

// Attempts to login via the given credentials struct.
// A given password takes presidence over a passfile.
func loginViaCredentials(cred Credentials, scriptMode bool) error {
	// check for password in file
	if strings.TrimSpace(cred.Password) == "" {
		if cred.PassfilePath != "" {
			b, err := os.ReadFile(cred.PassfilePath)
			if err != nil {
				return fmt.Errorf("failed to read password from %v: %v", cred.PassfilePath, err)
			}
			cred.Password = strings.TrimSpace(string(b))
		}
	}

	if cred.Username == "" || cred.Password == "" {
		// if script mode, do not prompt
		if scriptMode {
			return fmt.Errorf("no valid token found.\n" +
				"Please login via --username and {--password | --passfile}")
		}

		// prompt for credentials
		credM, err := CredPrompt(cred.Username, cred.Password)
		if err != nil {
			return err
		}
		// pull input results
		if finalCredM, ok := credM.(credModel); !ok {
			return err
		} else if finalCredM.killed {
			return errors.New("you must authenticate to use gwcli")
		} else {
			cred.Username = finalCredM.UserTI.Value()
			cred.Password = finalCredM.PassTI.Value()
		}
	}

	return Client.Login(cred.Username, cred.Password)
}

// Creates a login token for future use.
// The token's path is saved to an environment variable to be looked up on future runs
func CreateToken() error {
	var (
		err   error
		token string
	)
	if token, err = Client.ExportLoginToken(); err != nil {
		return fmt.Errorf("failed to export login token: %v", err)
	}
	if err = os.MkdirAll(cfgDir, 0700); err != nil {
		// check for exists error
		clilog.Writer.Debugf("mkdir error: %v", err)
		pe := err.(*os.PathError)
		if pe.Err != os.ErrExist {
			return fmt.Errorf("failed to ensure existance of directory %v: %v",
				cfgDir, err)
		}
	}
	tokenPath := path.Join(cfgDir, tokenName)

	// write out the token
	fd, err := os.OpenFile(tokenPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0600)
	if err != nil {
		return fmt.Errorf("failed to create token: %v", err)
	}
	if _, err := fd.WriteString(token); err != nil {
		return fmt.Errorf("failed to write token: %v", err)
	}

	if err = fd.Close(); err != nil {
		return fmt.Errorf("failed to close token file: %v", err)
	}

	clilog.Writer.Infof("Created token file @ %v", tokenPath)
	return nil
}

// Closes the connection to the server.
// Does not logout the user as to not invalidate existing JWTs.
func End() error {
	if Client == nil {
		return nil
	}

	Client.Close()
	return nil
}

func MyInfo() (types.UserDetails, error) {
	if myInfoCached {
		return myInfo, nil
	}
	return Client.MyInfo()

}
