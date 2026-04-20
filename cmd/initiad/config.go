package main

import (
	tmcfg "github.com/cometbft/cometbft/config"

	"github.com/initia-labs/initia/cmd/initiad/appconfig"
)

// initAppConfig helps to override default appConfig template and configs.
// return "", nil if no custom configuration is required for the application.
func initAppConfig() (string, any) {
	template, cfg := appconfig.InitAppConfig()
	return template, cfg
}

// initTendermintConfig helps to override default Tendermint Config values.
// return tmcfg.DefaultConfig if no custom configuration is required for the application.
func initTendermintConfig() *tmcfg.Config {
	return appconfig.InitTendermintConfig()
}
