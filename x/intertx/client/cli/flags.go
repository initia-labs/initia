package cli

import (
	flag "github.com/spf13/pflag"
)

const (
	// The connection end identifier on the controller chain
	FlagConnectionID = "connection-id"
	// The controller chain channel version
	FlagVersion = "version"
	// Set the ordering of the channel to ordered
	FlagOrdered = "ordered"
)

// common flagsets to add to various functions
var (
	fsConnectionID = flag.NewFlagSet("", flag.ContinueOnError)
	fsVersion      = flag.NewFlagSet("", flag.ContinueOnError)
	fsOrdered      = flag.NewFlagSet("", flag.ContinueOnError)
)

func init() {
	fsConnectionID.String(FlagConnectionID, "", "Connection ID")
	fsVersion.String(FlagVersion, "", "Version")
	fsOrdered.Bool(FlagOrdered, false, "Set the ordering of the channel to ordered")
}
