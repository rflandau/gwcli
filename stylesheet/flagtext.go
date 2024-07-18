package stylesheet

/**
 * Standardized flag description text for local flags "shared" across actions
 */

const (
	FlagOutputDesc   = "file to write results to.\nTruncates file unless --append is also given."
	FlagAppendDesc   = "append to the given output file instead of truncating it."
	FlagCSVDesc      = "display results as CSV.\nOnly effectual with --output.\nMutually exclusive with JSON."
	FlagJSONDesc     = "display results as JSON.\nOnly effectual with --output.\nMutually exclusive with CSV."
	FlagDurationDesc = "the historical timeframe (now minus duration) the query should pour over.\n" +
		"Ex: the past hour"
)
