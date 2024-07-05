package stylesheet

/**
 * Standardized flag description text for local flags "shared" across actions
 */

const (
	FlagOutputDesc = "file to write results to. Truncates file unless --append is also given."
	FlagAppendDesc = "append to the given output file instead of truncating."
	FlagCSVDesc    = "display results as CSV. Only effectual with --output. Mutually exclusive with JSON."
	FlagJSONDesc   = "display results as JSON. Only effectual with --output. Mutually exclusive with CSV."
)
