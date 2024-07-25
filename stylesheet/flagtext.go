package stylesheet

const ( // standardized flag description text for local flags "shared" across actions
	FlagOutputDesc   = "file to write results to.\nTruncates file unless --append is also given."
	FlagAppendDesc   = "append to the given output file instead of truncating it."
	FlagCSVDesc      = "display results as CSV.\nOnly effectual with --output.\nMutually exclusive with JSON."
	FlagJSONDesc     = "display results as JSON.\nOnly effectual with --output.\nMutually exclusive with CSV."
	FlagDurationDesc = "the historical timeframe (now minus duration) the query should pour over.\n" +
		"Ex: the past hour"
	FlagDryrunDesc = "feigns action, descibing what result would have taken place"

	// Macros
	FlagDescMacroName      = "name of the macro"
	FlagDescMacroDesc      = "flavour description of the macro"
	FlagDescMacroExpansion = "value for the macro to expand to"
)

const ( // description that require further formatting
	// would include "Ignored if you are not admin" suffixed, except I cannot guarentee all Client
	// library GetAll* functions actually do this rather than failing outright.
	FlagListAllDescFormat = "ADMIN-ONLY. Lists all %s on the system."
)

const ( // flag name uniformity, a la descriptions
	FlagNameDryrun         = "dryrun"
	FlagNameMacroName      = "name"
	FlagNameMacroDesc      = "description"
	FlagNameMacroExpansion = "expansion"
)
