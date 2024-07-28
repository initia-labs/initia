package movecmd

import (
	"encoding/hex"
	"fmt"
	"os"
	"path"
	"strings"

	"cosmossdk.io/core/address"
	errorsmod "cosmossdk.io/errors"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/client/tx"
	"github.com/cosmos/cosmos-sdk/version"

	movecli "github.com/initia-labs/initia/x/move/client/cli"
	movetypes "github.com/initia-labs/initia/x/move/types"

	"github.com/initia-labs/movevm/api"
	"github.com/initia-labs/movevm/types/compiler"
	buildtypes "github.com/initia-labs/movevm/types/compiler/build"
	coveragetypes "github.com/initia-labs/movevm/types/compiler/coverage"
	docgentypes "github.com/initia-labs/movevm/types/compiler/docgen"
	provetypes "github.com/initia-labs/movevm/types/compiler/prove"
	testtypes "github.com/initia-labs/movevm/types/compiler/test"
	"github.com/pelletier/go-toml"
	"github.com/spf13/cobra"
)

const (
	/* common */
	flagVerbose          = "verbose"
	flagVerboseShorthand = "v"
	flagFilter           = "filter"
	flagFilterShorthand  = "f"
	/* build options */
	flagDevMode                = "dev"
	flagDevModeShorthand       = "d"
	flagTestMode               = "test"
	flagGenerateDocs           = "doc"
	flagGenerateABI            = "abi"
	flagBuild                  = "build"
	flagPackagePath            = "path" // also used by moveDeployCommand()
	flagPackagePathShorthand   = "p"
	flagInstallDir             = "install-dir"
	flagForceRecompiliation    = "force"
	flagFetchDepsOnly          = "fetch-deps-only"
	flagSkipFetchLatestGitDeps = "skip-fetch-latest-git-deps"
	flagBytecodeVersion        = "bytecode-version"
	/* test options */
	flagGasLimit                  = "gas-limit"
	flagGasLimitShorthand         = "g"
	flagList                      = "list"
	flagListShorthand             = "l"
	flagNumThreads                = "threads"
	flagNumThreadsShorthand       = "t"
	flagReportStatistics          = "statistics"
	flagReportStatisticsShorthand = "s"
	flagReportStorageOnError      = "state-on-error"          // original move cli uses snake case, not kebab.
	flagIgnoreCompileWarnings     = "ignore-compile-warnings" // original move cli uses snake case, noe kebab.
	fiagCheckStacklessVM          = "stackless"
	flagComputeCoverage           = "coverage"
	// clean options
	flagCleanCache = "clean-cache"
	// prove options
	flagProcCores           = "proc-cores"
	flagTrace               = "trace"
	flagTraceShorthand      = "t"
	flagCVC5                = "cvc5"
	flagStratificationDepth = "stratification-depth"
	flagRandomSeed          = "random-seed"
	flagVcTimeout           = "vc-timeout"
	flagCheckInconsistency  = "check-inconsistency"
	flagKeepLoops           = "keep-loops"
	flagLoopUnroll          = "loop-unroll"
	flagStableTestOutput    = "stable-test-output"
	flagDump                = "dump"
	flagVerbosity           = "verbosity"
	// verify options
	flagVerify = "verify"
	// coverage options
	flagFunctions = "functions"
	flagOutputCSV = "output-csv"
	// docgen
	flagIncludeImpl         = "include-impl"
	flagIncludeSpecs        = "include-specs"
	flagSpecsInlined        = "specs-inlined"
	flagIncludeDepDiagram   = "include-dep-diagram"
	flagCollapsedSections   = "collapsed-sections"
	flagLandingPageTemplate = "landing-page-template"
	flagReferencesFile      = "references-file"
)

const (
	defaultPackagePath = "."
	defaultInstallDir  = "."
)

func MoveCommand(ac address.Codec) *cobra.Command {
	cmd := &cobra.Command{
		Use:                        "move",
		Short:                      "move subcommands",
		DisableFlagParsing:         true,
		SuggestionsMinimumDistance: 2,
		RunE:                       client.ValidateCmd,
	}

	cmd.AddCommand(
		moveBuildCmd(),
		moveTestCmd(),
		moveEncodeCmd(ac),
		moveNewCmd(),
		moveCleanCmd(),
		moveDeployCmd(ac),
		moveProveCmd(),
		moveVerifyCmd(),
		moveDocgenCmd(),
	)

	coverageCmd := &cobra.Command{
		Use:                        "coverage",
		Short:                      "coverage subcommands",
		DisableFlagParsing:         true,
		SuggestionsMinimumDistance: 2,
		RunE:                       client.ValidateCmd,
	}
	coverageCmd.AddCommand(
		moveCoverageSummaryCmd(),
		moveCoverageSourceCmd(),
		moveCoverageBytecodeCmd(),
	)

	cmd.AddCommand(coverageCmd)

	//initiaapp.ModuleBasics.AddQueryCommands(cmd)
	cmd.PersistentFlags().String(flags.FlagChainID, "", "The network chain ID")

	return cmd
}

func moveBuildCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "build [flags]",
		Short: "build a move package",
		Long:  "Build a move package. The provided path must specify the path of move package to build",
		Args:  cobra.ExactArgs(0),
		RunE: func(cmd *cobra.Command, args []string) error {
			arg, err := getCompilerArgument(cmd)
			if err != nil {
				return err
			}

			_, err = api.BuildContract(*arg)
			if err != nil {
				return err
			}

			return nil
		},
	}

	addMoveBuildFlags(cmd)
	return cmd
}

func moveTestCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "test [flags]",
		Short: "run tests in a move package",
		Long:  "Run tests in a move package. The provided path must specify the path of move package to test",
		Args:  cobra.ExactArgs(0),
		RunE: func(cmd *cobra.Command, args []string) error {
			arg, err := getCompilerArgument(cmd)
			if err != nil {
				return err
			}
			tc, err := getTestConfig(cmd)
			if err != nil {
				return err
			}

			_, err = api.TestContract(*arg, *tc)
			if err != nil {
				return err
			}

			return nil
		},
	}

	addMoveBuildFlags(cmd)
	addMoveTestFlags(cmd)

	return cmd
}

func moveEncodeCmd(ac address.Codec) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "encode [flags]",
		Short: "encode a move arguments in BCS format",
		Long: fmt.Sprintf(`
		Provide BCS encoding for move arguments.

		Supported types : u8, u16, u32, u64, u128, u256, bool, string, address, raw_hex, raw_base64,
			vector<inner_type>, option<inner_type>, decimal128, decimal256, fixed_point32, fixed_point64
		Example of args: address:0x1 bool:true u8:0 string:hello vector<u32>:a,b,c,d
		
		Example:
		$ %s move encode --args '["address:0x1", "bool:true", "u8:0x01", "u128:1234", "vector<u32>:a,b,c,d", "string:hello world"]'
`, version.AppName,
		),
		Args: cobra.ExactArgs(0),
		RunE: func(cmd *cobra.Command, args []string) error {
			flagArgs, err := movecli.ReadAndDecodeJSONStringArray[string](cmd, movecli.FlagArgs)
			if err != nil {
				return errorsmod.Wrap(err, "failed to read move args")
			}

			bcsArgs, err := movecli.BCSEncode(ac, flagArgs)
			if err != nil {
				return errorsmod.Wrap(err, "failed to encode move args")
			}

			for _, bcsArg := range bcsArgs {
				fmt.Println("0x" + hex.EncodeToString(bcsArg))
			}

			return nil
		},
	}

	cmd.Flags().AddFlagSet(movecli.FlagSetArgs())

	return cmd
}

func moveCoverageSummaryCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "summary [flags]",
		Short: "Display a coverage summary for all modules in this package",
		Args:  cobra.ExactArgs(0),
		RunE: func(cmd *cobra.Command, args []string) error {
			arg, err := getCompilerArgument(cmd)
			if err != nil {
				return err
			}

			config, err := getCoverageSummaryConfig(cmd)
			if err != nil {
				return err
			}

			_, err = api.CoverageSummary(*arg, *config)
			if err != nil {
				return err
			}

			return nil
		},
	}

	addMoveBuildFlags(cmd)
	addMoveCoverageSummaryFlags(cmd)

	return cmd
}

func moveCoverageSourceCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "source [module-name] [flags]",
		Short: "Display coverage information about the module against source code",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			arg, err := getCompilerArgument(cmd)
			if err != nil {
				return err
			}

			_, err = api.CoverageSource(*arg, coveragetypes.CoverageSourceConfig{
				ModuleName: args[0],
			})
			if err != nil {
				return err
			}

			return nil
		},
	}

	addMoveBuildFlags(cmd)

	return cmd
}

func moveCoverageBytecodeCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "bytecode [module-name] [flags]",
		Short: "Display coverage information about the module against disassembled bytecode",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			arg, err := getCompilerArgument(cmd)
			if err != nil {
				return err
			}

			_, err = api.CoverageBytecode(*arg, coveragetypes.CoverageBytecodeConfig{
				ModuleName: args[0],
			})
			if err != nil {
				return err
			}

			return nil
		},
	}

	addMoveBuildFlags(cmd)

	return cmd
}

func moveNewCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "new <NAME>",
		Short: "create a new move package",
		Long:  "Create a new Move package with name `name` at `path`. If `path` is not provided the package will be created in the directory `name`",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			arg, err := getCompilerArgument(cmd)
			if err != nil {
				return err
			}

			_, err = api.CreateContractPackage(*arg, args[0])
			if err != nil {
				return err
			}

			return nil
		},
	}

	addMoveBuildFlags(cmd)
	return cmd
}

func moveCleanCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "clean [flags]",
		Short: "remove build and its cache",
		Long:  "Remove previously built data and its cache",
		Args:  cobra.ExactArgs(0),
		RunE: func(cmd *cobra.Command, args []string) error {
			arg := compiler.CompilerArgument{}

			cleanCache, err := cmd.Flags().GetBool(flagCleanCache)
			if err != nil {
				return err
			}

			_, err = api.CleanContractPackage(arg, cleanCache, false, false)
			if err != nil {
				return err
			}

			return nil
		},
	}

	addMoveCleanFlags(cmd)
	return cmd
}

func getModuleBundle(packagePath string) ([][]byte, error) {
	moduleBundle := [][]byte{}

	manifest, err := toml.LoadFile(path.Join(packagePath, "Move.toml"))
	if err != nil {
		return nil, err
	}
	packageName, ok := manifest.Get("package.name").(string)
	if !ok {
		return nil, fmt.Errorf("failed to parse Move Manifest: %+v", packageName)
	}

	modulePath := path.Join(packagePath, "build", packageName, "bytecode_modules")
	fis, err := os.ReadDir(modulePath)
	if err != nil {
		return nil, fmt.Errorf("failed to find mv binaries: %v", err)
	}

	for _, fi := range fis {
		if fi.IsDir() || !strings.HasSuffix(fi.Name(), ".mv") {
			continue
		}
		moduleBytes, err := os.ReadFile(path.Join(modulePath, fi.Name()))
		if err != nil {
			return nil, fmt.Errorf("failed to find a mv binary: %v", err)
		}
		moduleBundle = append(moduleBundle, moduleBytes)
	}

	return moduleBundle, nil
}

func moveDeployCmd(ac address.Codec) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "deploy [flags]",
		Short: "deploy a whole move package",
		Long:  "deploy a whole move package. This command occurs a tx to publish module bundle.",
		Args:  cobra.ExactArgs(0),
		RunE: func(cmd *cobra.Command, args []string) error {

			// build package
			flagBuild, err := cmd.Flags().GetBool(flagBuild)
			if err != nil {
				return err
			}

			if flagBuild {
				arg, err := getCompilerArgument(cmd)
				if err != nil {
					return err
				}

				_, err = api.BuildContract(*arg)
				if err != nil {
					return err
				}
			}

			// verify package
			flagVerify, err := cmd.Flags().GetBool(flagVerify)
			if err != nil {
				return err
			}

			var vc *verifyConfig
			if flagVerify { // load verify config here to check flags validation before publishing
				vc, err = getVerifyConfig(cmd)
				if err != nil {
					return err
				}
			}

			// deploy package
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			packagePath, err := cmd.Flags().GetString(flagPackagePath)
			if err != nil {
				return err
			}

			module_bundle, err := getModuleBundle(packagePath)
			if err != nil {
				return err
			}

			upgradePolicyStr, err := cmd.Flags().GetString(movecli.FlagUpgradePolicy)
			if err != nil {
				return err
			}

			upgradePolicy, found := movetypes.UpgradePolicy_value[upgradePolicyStr]
			if !found {
				return fmt.Errorf("invalid upgrade-policy `%s`", upgradePolicyStr)
			}

			msg := movetypes.MsgPublish{
				Sender:        clientCtx.FromAddress.String(),
				CodeBytes:     module_bundle,
				UpgradePolicy: movetypes.UpgradePolicy(upgradePolicy),
			}

			if err = msg.Validate(ac); err != nil {
				return err
			}

			err = tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), &msg)
			if err != nil {
				return err
			}

			// request contract verify
			if flagVerify {
				if err := verifyContract(*vc); err != nil {
					return errorsmod.Wrap(err, "failed to verify published package")
				}
			}

			return nil
		},
	}

	// add flat set for upgrade policy
	cmd.Flags().AddFlagSet(movecli.FlagSetUpgradePolicy())

	addMoveDeployFlags(cmd)
	addMoveBuildFlags(cmd)
	addMoveVerifyFlags(cmd, false)
	flags.AddTxFlagsToCmd(cmd)
	return cmd
}

func moveProveCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "prove [flags]",
		Short: "prove a move package",
		Long:  "run formal verification of a Move package using the Move prover",
		Args:  cobra.ExactArgs(0),
		RunE: func(cmd *cobra.Command, args []string) error {
			arg, err := getCompilerArgument(cmd)
			if err != nil {
				return err
			}

			pc, err := getProveConfig(cmd)
			if err != nil {
				return err
			}

			_, err = api.ProveContract(*arg, *pc)
			if err != nil {
				return err
			}

			fmt.Println("Prove success")
			return nil
		},
	}

	addMoveBuildFlags(cmd)
	addMoveProveFlags(cmd)
	return cmd
}

func moveVerifyCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "verify [flags]",
		Short: "verify a move package",
		Long:  `verify a move package to reveal the source code of the onchain contract`,
		Args:  cobra.ExactArgs(0),
		RunE: func(cmd *cobra.Command, args []string) error {
			uc, err := getVerifyConfig(cmd)
			if err != nil {
				return err
			}

			err = verifyContract(*uc)
			if err != nil {
				return errorsmod.Wrap(err, "failed to verify")
			}

			fmt.Println("Verification done.")
			return nil
		},
	}

	addMoveVerifyFlags(cmd, true)
	return cmd
}

func moveDocgenCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "document [flags]",
		Short: "Generate documents of a move package",
		Long:  "Generate documents of a move package. The provided path must specify the path of move package to generate docs",
		Args:  cobra.ExactArgs(0),
		RunE: func(cmd *cobra.Command, args []string) error {
			arg, err := getCompilerArgument(cmd)
			if err != nil {
				return err
			}
			dgc, err := getDocgenConfig(cmd)
			if err != nil {
				return err
			}

			_, err = api.Docgen(*arg, *dgc)
			if err != nil {
				return err
			}

			return nil
		},
	}

	addMoveBuildFlags(cmd)
	addMoveDocgenFlags(cmd)

	return cmd
}

func addMoveBuildFlags(cmd *cobra.Command) {
	cmd.Flags().StringP(flagPackagePath, flagPackagePathShorthand, defaultPackagePath, "Path to a package which the command should be run with respect to")
	cmd.Flags().Bool(flagGenerateABI, false, "Generate ABIs for packages")
	cmd.Flags().BoolP(flagDevMode, flagDevModeShorthand, false, `Compile in 'dev' mode. The 'dev-addresses' and
'dev-dependencies' fields will be used if this flag is set. 
This flag is useful for development of packages that expose
named addresses that are not set to a specific value`)
	cmd.Flags().Bool(flagGenerateDocs, false, "Generate documentation for packages")
	cmd.Flags().Bool(flagFetchDepsOnly, false, "Only fetch dependency repos to MOVE_HOME")
	cmd.Flags().Bool(flagForceRecompiliation, false, "Force recompilation of all packages")
	cmd.Flags().String(flagInstallDir, defaultInstallDir, "Installation directory for compiled artifacts.")
	cmd.Flags().Bool(flagTestMode, false, `Compile in 'test' mode. The 'dev-addresses' and
'dev-dependencies' fields will be used along with any code in
the 'tests' directory`)
	cmd.Flags().Bool(flagVerbose, false, "Print additional diagnostics if available")
	cmd.Flags().Bool(flagSkipFetchLatestGitDeps, false, "Skip fetching latest git dependencies")
	cmd.Flags().Uint32(flagBytecodeVersion, 0, "Specify the version of the bytecode the compiler is going to emit")
}

func addMoveTestFlags(cmd *cobra.Command) {
	cmd.Flags().Bool(flagComputeCoverage, false, `Collect coverage information for later use with the various \'package coverage\' subcommands`)
	cmd.Flags().StringP(flagFilter, flagFilterShorthand, "", `A filter string to determine which unit tests to run. A unit test will be run only if it
contains this string in its fully qualified (<addr>::<module_name>::<fn_name>) name`)
	cmd.Flags().Bool(flagReportStorageOnError, false, "Show the storage state at the end of execution of a failing test")
	cmd.Flags().Bool(flagIgnoreCompileWarnings, false, "Ignore compiler's warning, and continue run tests")
	cmd.Flags().BoolP(flagReportStatistics, flagReportStatisticsShorthand, false, "Report test statistics at the end of testing")
}

func addMoveCoverageSummaryFlags(cmd *cobra.Command) {
	cmd.Flags().Bool(flagFunctions, true, "Whether function coverage summaries should be displayed")
	cmd.Flags().Bool(flagOutputCSV, true, "Output CSV data of coverage")
}

func addMoveCleanFlags(cmd *cobra.Command) {
	cmd.Flags().Bool(flagCleanCache, false, "Flush cache directory")
}

func addMoveDeployFlags(cmd *cobra.Command) {
	cmd.Flags().Bool(flagBuild, false, "Build package before deployment")
	cmd.Flags().Bool(flagVerify, false, "Verify the contract compared to the onchain package")
}

func addMoveProveFlags(cmd *cobra.Command) {
	cmd.Flags().Uint(flagProcCores, provetypes.DefaultProcCores, "The number of cores to use for parallel processing of verification conditions")
	cmd.Flags().BoolP(flagTrace, flagTraceShorthand, false, "Whether to display additional information in error reports. This may help debugging but also can make verification slower")
	cmd.Flags().Bool(flagCVC5, false, "Whether to use cvc5 as the smt solver backend. The environment variable `CVC5_EXE` should point to the binary")
	cmd.Flags().Uint(flagStratificationDepth, provetypes.DefaultStratificationDepth, "The depth until which stratified functions are expanded")
	cmd.Flags().Uint(flagRandomSeed, 0, "A seed to the prover")
	cmd.Flags().Uint(flagVcTimeout, provetypes.DefaultVcTimeout, "A (soft) timeout for the solver, per verification condition, in seconds")
	cmd.Flags().Bool(flagCheckInconsistency, false, "Whether to check consistency of specs by injecting impossible assertions")
	cmd.Flags().Bool(flagKeepLoops, false, "Whether to keep loops as they are and pass them on to the underlying solver")
	cmd.Flags().Uint(flagLoopUnroll, 0, "Number of iterations to unroll loops")
	cmd.Flags().Bool(flagStableTestOutput, false, "Whether output for e.g. diagnosis shall be stable/redacted so it can be used in test output")
	cmd.Flags().Bool(flagDump, false, "Whether to dump intermediate step results to files")
	cmd.Flags().StringP(flagVerbosity, flagVerboseShorthand, "", "Verbosity level")
}

func addMoveDocgenFlags(cmd *cobra.Command) {
	cmd.Flags().Bool(flagIncludeImpl, false, "Whether to include private declarations and implementations into the generated documentation.")
	cmd.Flags().Bool(flagIncludeSpecs, false, "Whether to include specifications in the generated documentation.")
	cmd.Flags().Bool(flagSpecsInlined, false, "Whether specifications should be put side-by-side with declarations or into a separate section.")
	cmd.Flags().Bool(flagIncludeDepDiagram, false, "Whether to include a dependency diagram.")
	cmd.Flags().Bool(flagCollapsedSections, false, "Whether details should be put into collapsed sections. This is not supported by all markdown, but the github dialect.")
	cmd.Flags().String(flagLandingPageTemplate, "", "Package-relative path to an optional markdown template which is a used to create a landing page.")
	cmd.Flags().String(flagReferencesFile, "", "Package-relative path to a file whose content is added to each generated markdown file.")
}

func getCompilerArgument(cmd *cobra.Command) (*compiler.CompilerArgument, error) {
	bc, err := getBuildConfig(cmd)
	if err != nil {
		return nil, err
	}

	packagePath, err := cmd.Flags().GetString(flagPackagePath)
	if err != nil {
		return nil, err
	}

	verbose, err := cmd.Flags().GetBool(flagVerbose)
	if err != nil {
		return nil, err
	}

	return &compiler.CompilerArgument{
		PackagePath: packagePath,
		Verbose:     verbose,
		BuildConfig: *bc,
	}, nil
}

func getBuildConfig(cmd *cobra.Command) (*buildtypes.BuildConfig, error) {

	options := []func(*buildtypes.BuildConfig){}

	boolFlags := map[string]func(*buildtypes.BuildConfig){}
	boolFlags[flagDevMode] = buildtypes.WithDevMode()
	boolFlags[flagTestMode] = buildtypes.WithTestMode()
	boolFlags[flagGenerateDocs] = buildtypes.WithGenerateDocs()
	boolFlags[flagGenerateABI] = buildtypes.WithGenerateABIs()
	boolFlags[flagForceRecompiliation] = buildtypes.WithForceRecompiliation()
	boolFlags[flagFetchDepsOnly] = buildtypes.WithFetchDepsOnly()
	boolFlags[flagSkipFetchLatestGitDeps] = buildtypes.WithSkipFetchLatestGitDeps()

	for fn, opt := range boolFlags {
		flag, err := cmd.Flags().GetBool(fn)
		if err != nil {
			return nil, err
		}
		if flag {
			options = append(options, opt)
		}
	}
	installDir, err := cmd.Flags().GetString(flagInstallDir)
	if err != nil {
		return nil, err
	}
	options = append(options, buildtypes.WithInstallDir(installDir))

	bytecodeVersion, err := cmd.Flags().GetUint32(flagBytecodeVersion)
	if err != nil {
		return nil, err
	}
	options = append(options, buildtypes.WithBytecodeVersion(bytecodeVersion))

	bc := buildtypes.NewBuildConfig(options...)

	return &bc, nil
}

func getTestConfig(cmd *cobra.Command) (*testtypes.TestConfig, error) {
	options := []func(*testtypes.TestConfig){}

	boolFlags := map[string]func(*testtypes.TestConfig){}
	boolFlags[flagComputeCoverage] = testtypes.WithComputeCoverage()
	boolFlags[flagReportStatistics] = testtypes.WithReportStatistics()
	boolFlags[flagReportStorageOnError] = testtypes.WithReportStorageOnError()
	boolFlags[flagIgnoreCompileWarnings] = testtypes.WithIgnoreCompileWarnings()

	for fn, opt := range boolFlags {
		flag, err := cmd.Flags().GetBool(fn)
		if err != nil {
			return nil, err
		}
		if flag {
			options = append(options, opt)
		}
	}

	filter, err := cmd.Flags().GetString(flagFilter)
	if err != nil {
		return nil, err
	}
	if filter != "" {
		options = append(options, testtypes.WithFilter(filter))
	}

	tc := testtypes.NewTestConfig(options...)
	return &tc, nil
}

func getCoverageSummaryConfig(cmd *cobra.Command) (*coveragetypes.CoverageSummaryConfig, error) {
	functions, err := cmd.Flags().GetBool(flagFunctions)
	if err != nil {
		return nil, err
	}
	outputCSV, err := cmd.Flags().GetBool(flagOutputCSV)
	if err != nil {
		return nil, err
	}
	return &coveragetypes.CoverageSummaryConfig{
		Functions: functions,
		OutputCSV: outputCSV,
	}, nil
}

func getProveConfig(cmd *cobra.Command) (*provetypes.ProveConfig, error) {
	options := []func(*provetypes.ProveConfig){}

	boolFlags := map[string]func(*provetypes.ProveConfig){}
	boolFlags[flagTrace] = provetypes.WithTrace()
	boolFlags[flagCVC5] = provetypes.WithCVC5()
	boolFlags[flagCheckInconsistency] = provetypes.WithCVC5()
	boolFlags[flagKeepLoops] = provetypes.WithKeepLoops()
	boolFlags[flagStableTestOutput] = provetypes.WithStableTestOutput()
	boolFlags[flagDump] = provetypes.WithDump()

	for fn, opt := range boolFlags {
		flag, err := cmd.Flags().GetBool(fn)
		if err != nil {
			return nil, err
		}
		if flag {
			options = append(options, opt)
		}
	}

	procCores, err := cmd.Flags().GetUint(flagProcCores)
	if err != nil {
		return nil, err
	}
	options = append(options, provetypes.WithProcCores(procCores))

	stratificationDepth, err := cmd.Flags().GetUint(flagStratificationDepth)
	if err != nil {
		return nil, err
	}
	options = append(options, provetypes.WithStratificationDepth(stratificationDepth))

	randomSeed, err := cmd.Flags().GetUint(flagRandomSeed)
	if err != nil {
		return nil, err
	}
	options = append(options, provetypes.WithRandomSeed(randomSeed))

	vcTimeout, err := cmd.Flags().GetUint(flagVcTimeout)
	if err != nil {
		return nil, err
	}
	options = append(options, provetypes.WithVcTimeout(vcTimeout))

	loopUnroll, err := cmd.Flags().GetUint(flagLoopUnroll)
	if err != nil {
		return nil, err
	}
	options = append(options, provetypes.WithLoopUnroll(loopUnroll))

	verbosity, err := cmd.Flags().GetString(flagVerbosity)
	if err != nil {
		return nil, err

	}
	options = append(options, provetypes.WithVerbosity(verbosity))

	pc := provetypes.NewProveConfig(options...)
	return &pc, nil
}

func getDocgenConfig(cmd *cobra.Command) (*docgentypes.DocgenConfig, error) {
	options := []func(*docgentypes.DocgenConfig){}

	boolFlags := map[string]func(*docgentypes.DocgenConfig){}
	boolFlags[flagIncludeImpl] = docgentypes.WithIncludeImpl()
	boolFlags[flagIncludeSpecs] = docgentypes.WithIncludeSpecs()
	boolFlags[flagSpecsInlined] = docgentypes.WithSpecsInlined()
	boolFlags[flagIncludeDepDiagram] = docgentypes.WithIncludeDepDiagram()
	boolFlags[flagCollapsedSections] = docgentypes.WithCollapsedSections()

	for fn, opt := range boolFlags {
		flag, err := cmd.Flags().GetBool(fn)
		if err != nil {
			return nil, err
		}
		if flag {
			options = append(options, opt)
		}
	}

	if landingPageTemplate, err := cmd.Flags().GetString(flagLandingPageTemplate); err != nil {
		return nil, err
	} else if landingPageTemplate != "" {
		options = append(options, docgentypes.WithLandingPageTemplate(landingPageTemplate))
	}

	if referencesFile, err := cmd.Flags().GetString(flagReferencesFile); err != nil {
		return nil, err
	} else if referencesFile != "" {
		options = append(options, docgentypes.WithReferencesFile(referencesFile))
	}

	dgc := docgentypes.NewDocgenConfig(options...)
	return &dgc, nil
}
