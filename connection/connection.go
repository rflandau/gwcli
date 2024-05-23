/**
 * Wrapper and instantiation of the gravwell client library.
 * The client is instantiated and exported in this package so it can be shared
 * by the rest of the application without trying to pass pointers through cobra
 * subroutine signatures.
 */

package connection

import (
	tea "github.com/charmbracelet/bubbletea"
	grav "github.com/gravwell/gravwell/v3/client"
	"github.com/gravwell/gravwell/v3/client/objlog"
)

var Client *grav.Client

/**
 * Initializes Client using the given connection string of the form <host>:<port>
 */
func Initialize(conn string) (err error) { // TODO accept more arguments to fill out opt struct
	l, err := objlog.NewJSONLogger("rest_client.json")
	if err != nil {
		return err
	}
	opts := grav.Opts{Server: conn, UseHttps: false, InsecureNoEnforceCerts: true, ObjLogger: l}
	Client, err = grav.NewOpts(opts)
	if err != nil {
		return err
	}
	return nil
}

func Login(user, pass string) (err error) {
	return Client.Login(user, pass)
}

/* Logs out the current user and closes the connection to the server. */
func End() tea.Msg {
	var errString string
	errString = Client.Logout().Error()
	if errString != "" {
		errString += "|"
	}
	errString += Client.Close().Error()

	return errString
}
