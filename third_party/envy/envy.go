// Package envy automatically exposes environment
// variables for all of your flags.
package envy

import (
	"flag"
	"fmt"
	"os"
	"strings"
)

// Parse takes a prefix string and exposes environment variables
// for all flags in the default FlagSet (flag.CommandLine) in the
// form of PREFIX_FLAGNAME.
func Parse(p string) {
	update(p, flag.CommandLine)
}

// update takes a prefix string p and *flag.FlagSet. Each flag
// in the FlagSet is exposed as an upper case environment variable
// prefixed with p. Any flag that was not explicitly set by a user
// is updated to the environment variable, if set.
func update(p string, fs *flag.FlagSet) {
	// Build a map of explicitly set flags.
	set := map[string]interface{}{}
	fs.Visit(func(f *flag.Flag) {
		set[f.Name] = nil
	})

	fs.VisitAll(func(f *flag.Flag) {
		// Create an env var name
		// based on the supplied prefix.
		envVar := fmt.Sprintf("%s_%s", p, strings.ToUpper(f.Name))
		envVar = strings.Replace(envVar, "-", "_", -1)

		// Update the Flag.Value if the
		// env var is non "".
		if val := os.Getenv(envVar); val != "" {
			// Update the value if it hasn't
			// already been set.
			if _, defined := set[f.Name]; !defined {
				fs.Set(f.Name, val)
			}
		}

		// Append the env var to the
		// Flag.Usage field.
		f.Usage = fmt.Sprintf("%s [%s]", f.Usage, envVar)
	})
}
