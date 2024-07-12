package datascope

import (
	"gwcli/utilities/uniques"
	"testing"
	"time"

	grav "github.com/gravwell/gravwell/v3/client"
	"github.com/gravwell/gravwell/v3/client/types"
)

const ( // mock credentials
	user     = "admin"
	password = "changeme"
	server   = "localhost:80"
)

func TestKeepAlive(t *testing.T) {
	// connect and login
	// connect to the server for manual calls
	testclient, err := grav.NewOpts(grav.Opts{Server: server, UseHttps: false, InsecureNoEnforceCerts: true})
	if err != nil {
		panic(err)
	}
	if err = testclient.Login(user, password); err != nil {
		panic(err)
	}

	const minDuration = 3 * time.Minute
	// skip this test if we do not have a long enough timeout
	if deadline, unset := t.Deadline(); !unset || !deadline.After(time.Now().Add(minDuration)) {
		t.Skip("this test requires a -timeout of at least ", minDuration)
	}

	// submit a query
	s, err := testclient.StartSearch("tag=gravwell", time.Now().Add(-30*time.Second), time.Now(), false)
	if err != nil {
		t.Fatal("failed to start query:", err)
	}

	// spawn keepalive on it
	go keepAlive(&s)

	// pull results from the query every so often
	for i := 0; i < 3; i++ { // run for 3 minutes
		time.Sleep(time.Minute)

		if _, err := testclient.DownloadSearch(s.ID,
			types.TimeRange{},
			uniques.SearchTimeFormat); err != nil {
			t.Fatalf("failed to download search after %v minutes: %v", i, err)
		}
	}

	// change the sid
	// TODO

	// confirm that keepalive is dead by repulling results and expecting a 404
	// TODO

}
