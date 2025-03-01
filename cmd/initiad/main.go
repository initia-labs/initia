package main

import (
    "fmt"
    "os"
    "runtime/debug"

    svrcmd "github.com/cosmos/cosmos-sdk/server/cmd"
    "github.com/pkg/errors"

    initiaapp "github.com/initia-labs/initia/app"
)

func main() {
    defer func() {
        if r := recover(); r != nil {
            fmt.Fprintf(
                os.Stderr, 
                "Critical error: %v\nStack: %s", 
                r, 
                debug.Stack()
            )
            os.Exit(1)
        }
    }()

    rootCmd, err := NewRootCmd()
    if err != nil {
        fmt.Fprintf(os.Stderr, "Failed to create root command: %v\n", err)
        os.Exit(1)
    }

    if err := svrcmd.Execute(rootCmd, initiaapp.EnvPrefix, initiaapp.DefaultNodeHome); err != nil {
        fmt.Fprintf(
            rootCmd.OutOrStderr(), 
            "Execution error: %v\n", 
            err
        )
        os.Exit(1)
    }
}
