/**
 * Singleton instantiation of the gravwell client library.
 * The client is instantiated and exported in this package so it can be shared
 * by the rest of the application without trying to pass pointers through cobra
 * subroutine signatures.
 */

package connection

import (
	grav "github.com/gravwell/gravwell/v3/client"
	"github.com/gravwell/gravwell/v3/client/objlog"
)

var Client *grav.Client

// Initializes Client using the given connection string of the form <host>:<port>
func Initialize(conn string, UseHttps, InsecureNoEnforceCerts bool) (err error) {
	l, err := objlog.NewJSONLogger("rest_client.json")
	if err != nil {
		return err
	}
	opts := grav.Opts{Server: conn,
		UseHttps:               UseHttps,
		InsecureNoEnforceCerts: InsecureNoEnforceCerts,
		ObjLogger:              l}
	Client, err = grav.NewOpts(opts)
	if err != nil {
		return err
	}
	return nil
}

func Login(user, pass string) (err error) {
	return Client.Login(user, pass)
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
