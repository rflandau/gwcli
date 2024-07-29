/*
ft (flagtext) provides a repository of strings used for flags across gwcli.
While all are constant and *should not be modified at runtime*, it is organized as a struct for
clearer access.
*/
package ft

// Common flag names used across a variety of actions
var Name = struct {
	Dryrun    string
	Name      string
	Desc      string
	ID        string
	Query     string
	Frequency string
}{
	Dryrun:    "dryrun",
	Name:      "name",
	Desc:      "description",
	ID:        "id",
	Query:     "query",
	Frequency: "frequency",
}

// Common flag usage description used across a variety of actions
// The compiler should inline all of these functions so they are overhead-less.
var Usage = struct {
	Name      func(singular string) string
	Desc      func(singular string) string
	Frequency string
}{
	Name: func(singular string) string {
		return "name of the " + singular
	},
	Desc: func(singular string) string {
		return "flavour description of the " + singular
	},
	Frequency: "cron-style execution frequency",
}
