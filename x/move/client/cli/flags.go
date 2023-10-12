package cli

import (
	"fmt"

	flag "github.com/spf13/pflag"

	"github.com/initia-labs/initia/x/move/types"
)

const (
	FlagUpgradePolicy = "upgrade-policy"
	FlagTypeArgs      = "type-args"
	FlagArgs          = "args"
)

// FlagSetUpgradePolicy Returns the FlagSet for upgrade policy related operations.
func FlagSetUpgradePolicy() *flag.FlagSet {
	fs := flag.NewFlagSet("", flag.ContinueOnError)
	fs.String(FlagUpgradePolicy, types.UpgradePolicy_name[int32(types.UpgradePolicy_ARBITRARY)],
		fmt.Sprintf(`The module upgrade policy, which should be one of "%s", "%s" and "%s")`,
			types.UpgradePolicy_name[int32(types.UpgradePolicy_ARBITRARY)],
			types.UpgradePolicy_name[int32(types.UpgradePolicy_COMPATIBLE)],
			types.UpgradePolicy_name[int32(types.UpgradePolicy_IMMUTABLE)],
		))
	return fs
}

// FlagSetArgs Returns the FlagSet for args related operations.
func FlagSetArgs() *flag.FlagSet {
	fs := flag.NewFlagSet("", flag.ContinueOnError)
	fs.String(FlagArgs, "", `The arguments for move functions.
ex) address:0x1 bool:true u8:0x01 u128:1234 'vector<u32>:a,b,c,d'`)
	return fs
}

// FlagSetTypeArgs Returns the FlagSet for type args related operations.
func FlagSetTypeArgs() *flag.FlagSet {
	fs := flag.NewFlagSet("", flag.ContinueOnError)
	fs.String(FlagTypeArgs, "", `The type arguments for move functions.
ex) 0x1::BasicCoin::getBalance<u8>`)
	return fs
}
