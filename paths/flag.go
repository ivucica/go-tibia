package paths

import (
	"flag"
)

// SetupFilePathFlag creates a new string flag with the passed name with a sane
// default for the path to the file, if found using the Find function. If not,
// the flag defaults to an empty string.
func SetupFilePathFlag(fileName, flagName string, flagPtr *string) {
	flag.StringVar(flagPtr, flagName, Find(fileName), "Path to "+fileName)
}
