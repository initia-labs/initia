package genesis_markets

const GenesisMarkets = `[
	{
	  "ticker": {
		"currency_pair": {
		  "Base": "ATOM",
		  "Quote": "USD"
		},
		"decimals": 9,
		"min_provider_count": 3
	  },
	  "provider_configs": [
		{
		  "name": "binance_api",
		  "off_chain_ticker": "ATOMUSDT",
		  "normalize_by_pair": {
			"Base": "USDT",
			"Quote": "USD"
		  }
		},
		{
		  "name": "bybit_ws",
		  "off_chain_ticker": "ATOMUSDT",
		  "normalize_by_pair": {
			"Base": "USDT",
			"Quote": "USD"
		  }
		},
		{
		  "name": "coinbase_api",
		  "off_chain_ticker": "ATOM-USD"
		},
		{
		  "name": "gate_ws",
		  "off_chain_ticker": "ATOM_USDT",
		  "normalize_by_pair": {
			"Base": "USDT",
			"Quote": "USD"
		  }
		},
		{
		  "name": "kraken_api",
		  "off_chain_ticker": "ATOMUSD"
		},
		{
		  "name": "kucoin_ws",
		  "off_chain_ticker": "ATOM-USDT",
		  "normalize_by_pair": {
			"Base": "USDT",
			"Quote": "USD"
		  }
		},
		{
		  "name": "mexc_ws",
		  "off_chain_ticker": "ATOMUSDT",
		  "normalize_by_pair": {
			"Base": "USDT",
			"Quote": "USD"
		  }
		},
		{
		  "name": "okx_ws",
		  "off_chain_ticker": "ATOM-USDT",
		  "normalize_by_pair": {
			"Base": "USDT",
			"Quote": "USD"
		  }
		}
	  ]
	},
	{
	  "ticker": {
		"currency_pair": {
		  "Base": "OSMO",
		  "Quote": "USD"
		},
		"decimals": 9,
		"min_provider_count": 3
	  },
	  "provider_configs": [
		{
		  "name": "binance_api",
		  "off_chain_ticker": "OSMOUSDT",
		  "normalize_by_pair": {
			"Base": "USDT",
			"Quote": "USD"
		  }
		},
		{
		  "name": "coinbase_api",
		  "off_chain_ticker": "OSMO-USD"
		},
		{
		  "name": "gate_ws",
		  "off_chain_ticker": "OSMO_USDT",
		  "normalize_by_pair": {
			"Base": "USDT",
			"Quote": "USD"
		  }
		},
		{
		  "name": "kraken_api",
		  "off_chain_ticker": "OSMOUSD"
		},
		{
		  "name": "kucoin_ws",
		  "off_chain_ticker": "OSMO-USDT",
		  "normalize_by_pair": {
			"Base": "USDT",
			"Quote": "USD"
		  }
		},
		{
		  "name": "mexc_ws",
		  "off_chain_ticker": "OSMOUSDT",
		  "normalize_by_pair": {
			"Base": "USDT",
			"Quote": "USD"
		  }
		},
		{
		  "name": "okx_ws",
		  "off_chain_ticker": "OSMO-USDT",
		  "normalize_by_pair": {
			"Base": "USDT",
			"Quote": "USD"
		  }
		}
	  ]
	},
	{
	  "ticker": {
		"currency_pair": {
		  "Base": "BNB",
		  "Quote": "USD"
		},
		"decimals": 7,
		"min_provider_count": 3
	  },
	  "provider_configs": [
		{
		  "name": "okx_ws",
		  "off_chain_ticker": "BNB-USDT",
		  "normalize_by_pair": {
			"Base": "USDT",
			"Quote": "USD"
		  }
		},
		{
		  "name": "kucoin_ws",
		  "off_chain_ticker": "BNB-USDT",
		  "normalize_by_pair": {
			"Base": "USDT",
			"Quote": "USD"
		  }
		},
		{
		  "name": "mexc_ws",
		  "off_chain_ticker": "BNBUSDT",
		  "normalize_by_pair": {
			"Base": "USDT",
			"Quote": "USD"
		  }
		},
		{
		  "name": "binance_api",
		  "off_chain_ticker": "BNBUSDT",
		  "normalize_by_pair": {
			"Base": "USDT",
			"Quote": "USD"
		  }
		},
		{
		  "name": "bybit_ws",
		  "off_chain_ticker": "BNBUSDT",
		  "normalize_by_pair": {
			"Base": "USDT",
			"Quote": "USD"
		  }
		},
		{
		  "name": "gate_ws",
		  "off_chain_ticker": "BNB_USDT",
		  "normalize_by_pair": {
			"Base": "USDT",
			"Quote": "USD"
		  }
		}
	  ]
	},
	{
	  "ticker": {
		"currency_pair": {
		  "Base": "BTC",
		  "Quote": "USD"
		},
		"decimals": 5,
		"min_provider_count": 3
	  },
	  "provider_configs": [
		{
		  "name": "binance_api",
		  "off_chain_ticker": "BTCUSDT",
		  "normalize_by_pair": {
			"Base": "USDT",
			"Quote": "USD"
		  }
		},
		{
		  "name": "bybit_ws",
		  "off_chain_ticker": "BTCUSDT",
		  "normalize_by_pair": {
			"Base": "USDT",
			"Quote": "USD"
		  }
		},
		{
		  "name": "coinbase_api",
		  "off_chain_ticker": "BTC-USD"
		},
		{
		  "name": "huobi_ws",
		  "off_chain_ticker": "btcusdt",
		  "normalize_by_pair": {
			"Base": "USDT",
			"Quote": "USD"
		  }
		},
		{
		  "name": "kraken_api",
		  "off_chain_ticker": "XXBTZUSD"
		},
		{
		  "name": "kucoin_ws",
		  "off_chain_ticker": "BTC-USDT",
		  "normalize_by_pair": {
			"Base": "USDT",
			"Quote": "USD"
		  }
		},
		{
		  "name": "mexc_ws",
		  "off_chain_ticker": "BTCUSDT",
		  "normalize_by_pair": {
			"Base": "USDT",
			"Quote": "USD"
		  }
		},
		{
		  "name": "okx_ws",
		  "off_chain_ticker": "BTC-USDT",
		  "normalize_by_pair": {
			"Base": "USDT",
			"Quote": "USD"
		  }
		}
	  ]
	},
	{
	  "ticker": {
		"currency_pair": {
		  "Base": "NTRN",
		  "Quote": "USD"
		},
		"decimals": 8,
		"min_provider_count": 2
	  },
	  "provider_configs": [
		{
		  "name": "binance_api",
		  "off_chain_ticker": "NTRNUSDT",
		  "normalize_by_pair": {
			"Base": "USDT",
			"Quote": "USD"
		  }
		},
		{
		  "name": "gate_ws",
		  "off_chain_ticker": "NTRN_USDT",
		  "normalize_by_pair": {
			"Base": "USDT",
			"Quote": "USD"
		  }
		},
		{
		  "name": "kucoin_ws",
		  "off_chain_ticker": "NTRN-USDT",
		  "normalize_by_pair": {
			"Base": "USDT",
			"Quote": "USD"
		  }
		},
		{
		  "name": "okx_ws",
		  "off_chain_ticker": "NTRN-USDT",
		  "normalize_by_pair": {
			"Base": "USDT",
			"Quote": "USD"
		  }
		}
	  ]
	},
	{
	  "ticker": {
		"currency_pair": {
		  "Base": "USDT",
		  "Quote": "USD"
		},
		"decimals": 9,
		"min_provider_count": 1
	  },
	  "provider_configs": [
		{
		  "name": "binance_api",
		  "off_chain_ticker": "USDCUSDT",
		  "invert": true
		},
		{
		  "name": "bybit_ws",
		  "off_chain_ticker": "USDCUSDT",
		  "invert": true
		},
		{
		  "name": "coinbase_api",
		  "off_chain_ticker": "USDT-USD"
		},
		{
		  "name": "huobi_ws",
		  "off_chain_ticker": "ethusdt",
		  "normalize_by_pair": {
			"Base": "ETH",
			"Quote": "USD"
		  },
		  "invert": true
		},
		{
		  "name": "kraken_api",
		  "off_chain_ticker": "USDTZUSD"
		},
		{
		  "name": "kucoin_ws",
		  "off_chain_ticker": "BTC-USDT",
		  "normalize_by_pair": {
			"Base": "BTC",
			"Quote": "USD"
		  },
		  "invert": true
		},
		{
		  "name": "okx_ws",
		  "off_chain_ticker": "USDC-USDT",
		  "invert": true
		}
	  ]
	},
	{
	  "ticker": {
		"currency_pair": {
		  "Base": "USDC",
		  "Quote": "USD"
		},
		"decimals": 9,
		"min_provider_count": 1
	  },
	  "provider_configs": [
		{
		  "name": "binance_api",
		  "off_chain_ticker": "USDCUSDT"
		},
		{
		  "name": "bybit_ws",
		  "off_chain_ticker": "USDCUSDT"
		},
		{
		  "name": "coinbase_api",
		  "off_chain_ticker": "USDC-USD"
		},
		{
		  "name": "huobi_ws",
		  "off_chain_ticker": "ethusdc",
		  "normalize_by_pair": {
			"Base": "ETH",
			"Quote": "USD"
		  },
		  "invert": true
		},
		{
		  "name": "kraken_api",
		  "off_chain_ticker": "USDCZUSD"
		},
		{
		  "name": "kucoin_ws",
		  "off_chain_ticker": "BTC-USDC",
		  "normalize_by_pair": {
			"Base": "BTC",
			"Quote": "USD"
		  },
		  "invert": true
		},
		{
		  "name": "okx_ws",
		  "off_chain_ticker": "USDC-USDT"
		}
	  ]
	},
	{
	  "ticker": {
		"currency_pair": {
		  "Base": "SUI",
		  "Quote": "USD"
		},
		"decimals": 10,
		"min_provider_count": 3
	  },
	  "provider_configs": [
		{
		  "name": "binance_api",
		  "off_chain_ticker": "SUIUSDT",
		  "normalize_by_pair": {
			"Base": "USDT",
			"Quote": "USD"
		  }
		},
		{
		  "name": "bybit_ws",
		  "off_chain_ticker": "SUIUSDT",
		  "normalize_by_pair": {
			"Base": "USDT",
			"Quote": "USD"
		  }
		},
		{
		  "name": "coinbase_api",
		  "off_chain_ticker": "SUI-USD"
		},
		{
		  "name": "gate_ws",
		  "off_chain_ticker": "SUI_USDT",
		  "normalize_by_pair": {
			"Base": "USDT",
			"Quote": "USD"
		  }
		},
		{
		  "name": "huobi_ws",
		  "off_chain_ticker": "suiusdt",
		  "normalize_by_pair": {
			"Base": "USDT",
			"Quote": "USD"
		  }
		},
		{
		  "name": "kucoin_ws",
		  "off_chain_ticker": "SUI-USDT",
		  "normalize_by_pair": {
			"Base": "USDT",
			"Quote": "USD"
		  }
		},
		{
		  "name": "mexc_ws",
		  "off_chain_ticker": "SUIUSDT",
		  "normalize_by_pair": {
			"Base": "USDT",
			"Quote": "USD"
		  }
		},
		{
		  "name": "okx_ws",
		  "off_chain_ticker": "SUI-USDT",
		  "normalize_by_pair": {
			"Base": "USDT",
			"Quote": "USD"
		  }
		}
	  ]
	},
	{
	  "ticker": {
		"currency_pair": {
		  "Base": "SOL",
		  "Quote": "USD"
		},
		"decimals": 8,
		"min_provider_count": 3
	  },
	  "provider_configs": [
		{
		  "name": "binance_api",
		  "off_chain_ticker": "SOLUSDT",
		  "normalize_by_pair": {
			"Base": "USDT",
			"Quote": "USD"
		  }
		},
		{
		  "name": "bybit_ws",
		  "off_chain_ticker": "SOLUSDT",
		  "normalize_by_pair": {
			"Base": "USDT",
			"Quote": "USD"
		  }
		},
		{
		  "name": "coinbase_api",
		  "off_chain_ticker": "SOL-USD"
		},
		{
		  "name": "huobi_ws",
		  "off_chain_ticker": "solusdt",
		  "normalize_by_pair": {
			"Base": "USDT",
			"Quote": "USD"
		  }
		},
		{
		  "name": "kraken_api",
		  "off_chain_ticker": "SOLUSD"
		},
		{
		  "name": "kucoin_ws",
		  "off_chain_ticker": "SOL-USDT",
		  "normalize_by_pair": {
			"Base": "USDT",
			"Quote": "USD"
		  }
		},
		{
		  "name": "mexc_ws",
		  "off_chain_ticker": "SOLUSDT",
		  "normalize_by_pair": {
			"Base": "USDT",
			"Quote": "USD"
		  }
		},
		{
		  "name": "okx_ws",
		  "off_chain_ticker": "SOL-USDT",
		  "normalize_by_pair": {
			"Base": "USDT",
			"Quote": "USD"
		  }
		}
	  ]
	},
	{
	  "ticker": {
		"currency_pair": {
		  "Base": "TIA",
		  "Quote": "USD"
		},
		"decimals": 8,
		"min_provider_count": 3
	  },
	  "provider_configs": [
		{
		  "name": "binance_api",
		  "off_chain_ticker": "TIAUSDT",
		  "normalize_by_pair": {
			"Base": "USDT",
			"Quote": "USD"
		  }
		},
		{
		  "name": "bybit_ws",
		  "off_chain_ticker": "TIAUSDT",
		  "normalize_by_pair": {
			"Base": "USDT",
			"Quote": "USD"
		  }
		},
		{
		  "name": "coinbase_api",
		  "off_chain_ticker": "TIA-USD"
		},
		{
		  "name": "huobi_ws",
		  "off_chain_ticker": "tiausdt",
		  "normalize_by_pair": {
			"Base": "USDT",
			"Quote": "USD"
		  }
		},
		{
		  "name": "kraken_api",
		  "off_chain_ticker": "TIAUSD"
		},
		{
		  "name": "kucoin_ws",
		  "off_chain_ticker": "TIA-USDT",
		  "normalize_by_pair": {
			"Base": "USDT",
			"Quote": "USD"
		  }
		},
		{
		  "name": "mexc_ws",
		  "off_chain_ticker": "TIAUSDT",
		  "normalize_by_pair": {
			"Base": "USDT",
			"Quote": "USD"
		  }
		},
		{
		  "name": "okx_ws",
		  "off_chain_ticker": "TIA-USDT",
		  "normalize_by_pair": {
			"Base": "USDT",
			"Quote": "USD"
		  }
		}
	  ]
	},
	{
	  "ticker": {
		"currency_pair": {
		  "Base": "BERA",
		  "Quote": "USD"
		},
		"decimals": 8,
		"min_provider_count": 3
	  },
	  "provider_configs": [
		{
		  "name": "binance_api",
		  "off_chain_ticker": "BERAUSDT",
		  "normalize_by_pair": {
			"Base": "USDT",
			"Quote": "USD"
		  }
		},
		{
		  "name": "bybit_ws",
		  "off_chain_ticker": "BERAUSDT",
		  "normalize_by_pair": {
			"Base": "USDT",
			"Quote": "USD"
		  }
		},
		{
		  "name": "coinbase_api",
		  "off_chain_ticker": "BERA-USD"
		},
		{
		  "name": "huobi_ws",
		  "off_chain_ticker": "berausdt",
		  "normalize_by_pair": {
			"Base": "USDT",
			"Quote": "USD"
		  }
		},
		{
		  "name": "kraken_api",
		  "off_chain_ticker": "BERAUSD"
		},
		{
		  "name": "kucoin_ws",
		  "off_chain_ticker": "BERA-USDT",
		  "normalize_by_pair": {
			"Base": "USDT",
			"Quote": "USD"
		  }
		},
		{
		  "name": "mexc_ws",
		  "off_chain_ticker": "BERAUSDT",
		  "normalize_by_pair": {
			"Base": "USDT",
			"Quote": "USD"
		  }
		},
		{
		  "name": "okx_ws",
		  "off_chain_ticker": "BERA-USDT",
		  "normalize_by_pair": {
			"Base": "USDT",
			"Quote": "USD"
		  }
		}
	  ]
	},
	{
	  "ticker": {
		"currency_pair": {
		  "Base": "ENA",
		  "Quote": "USD"
		},
		"decimals": 8,
		"min_provider_count": 3
	  },
	  "provider_configs": [
		{
		  "name": "binance_api",
		  "off_chain_ticker": "ENAUSDT",
		  "normalize_by_pair": {
			"Base": "USDT",
			"Quote": "USD"
		  }
		},
		{
		  "name": "bybit_ws",
		  "off_chain_ticker": "ENAUSDT",
		  "normalize_by_pair": {
			"Base": "USDT",
			"Quote": "USD"
		  }
		},
		{
		  "name": "coinbase_api",
		  "off_chain_ticker": "ENA-USD"
		},
		{
		  "name": "huobi_ws",
		  "off_chain_ticker": "enausdt",
		  "normalize_by_pair": {
			"Base": "USDT",
			"Quote": "USD"
		  }
		},
		{
		  "name": "kraken_api",
		  "off_chain_ticker": "ENAUSD"
		},
		{
		  "name": "kucoin_ws",
		  "off_chain_ticker": "ENA-USDT",
		  "normalize_by_pair": {
			"Base": "USDT",
			"Quote": "USD"
		  }
		},
		{
		  "name": "mexc_ws",
		  "off_chain_ticker": "ENAUSDT",
		  "normalize_by_pair": {
			"Base": "USDT",
			"Quote": "USD"
		  }
		},
		{
		  "name": "okx_ws",
		  "off_chain_ticker": "ENA-USDT",
		  "normalize_by_pair": {
			"Base": "USDT",
			"Quote": "USD"
		  }
		}
	  ]
	},
	{
	  "ticker": {
		"currency_pair": {
		  "Base": "APT",
		  "Quote": "USD"
		},
		"decimals": 9,
		"min_provider_count": 3
	  },
	  "provider_configs": [
		{
		  "name": "binance_api",
		  "off_chain_ticker": "APTUSDT",
		  "normalize_by_pair": {
			"Base": "USDT",
			"Quote": "USD"
		  }
		},
		{
		  "name": "bybit_ws",
		  "off_chain_ticker": "APTUSDT",
		  "normalize_by_pair": {
			"Base": "USDT",
			"Quote": "USD"
		  }
		},
		{
		  "name": "coinbase_api",
		  "off_chain_ticker": "APT-USD"
		},
		{
		  "name": "gate_ws",
		  "off_chain_ticker": "APT_USDT",
		  "normalize_by_pair": {
			"Base": "USDT",
			"Quote": "USD"
		  }
		},
		{
		  "name": "huobi_ws",
		  "off_chain_ticker": "aptusdt",
		  "normalize_by_pair": {
			"Base": "USDT",
			"Quote": "USD"
		  }
		},
		{
		  "name": "kucoin_ws",
		  "off_chain_ticker": "APT-USDT",
		  "normalize_by_pair": {
			"Base": "USDT",
			"Quote": "USD"
		  }
		},
		{
		  "name": "mexc_ws",
		  "off_chain_ticker": "APTUSDT",
		  "normalize_by_pair": {
			"Base": "USDT",
			"Quote": "USD"
		  }
		},
		{
		  "name": "okx_ws",
		  "off_chain_ticker": "APT-USDT",
		  "normalize_by_pair": {
			"Base": "USDT",
			"Quote": "USD"
		  }
		}
	  ]
	},
	{
	  "ticker": {
		"currency_pair": {
		  "Base": "ARB",
		  "Quote": "USD"
		},
		"decimals": 9,
		"min_provider_count": 3
	  },
	  "provider_configs": [
		{
		  "name": "binance_api",
		  "off_chain_ticker": "ARBUSDT",
		  "normalize_by_pair": {
			"Base": "USDT",
			"Quote": "USD"
		  }
		},
		{
		  "name": "bybit_ws",
		  "off_chain_ticker": "ARBUSDT",
		  "normalize_by_pair": {
			"Base": "USDT",
			"Quote": "USD"
		  }
		},
		{
		  "name": "coinbase_api",
		  "off_chain_ticker": "ARB-USD"
		},
		{
		  "name": "gate_ws",
		  "off_chain_ticker": "ARB_USDT",
		  "normalize_by_pair": {
			"Base": "USDT",
			"Quote": "USD"
		  }
		},
		{
		  "name": "huobi_ws",
		  "off_chain_ticker": "arbusdt",
		  "normalize_by_pair": {
			"Base": "USDT",
			"Quote": "USD"
		  }
		},
		{
		  "name": "kucoin_ws",
		  "off_chain_ticker": "ARB-USDT",
		  "normalize_by_pair": {
			"Base": "USDT",
			"Quote": "USD"
		  }
		},
		{
		  "name": "mexc_ws",
		  "off_chain_ticker": "ARBUSDT",
		  "normalize_by_pair": {
			"Base": "USDT",
			"Quote": "USD"
		  }
		},
		{
		  "name": "okx_ws",
		  "off_chain_ticker": "ARB-USDT",
		  "normalize_by_pair": {
			"Base": "USDT",
			"Quote": "USD"
		  }
		}
	  ]
	},
	{
	  "ticker": {
		"currency_pair": {
		  "Base": "ETH",
		  "Quote": "USD"
		},
		"decimals": 6,
		"min_provider_count": 3
	  },
	  "provider_configs": [
		{
		  "name": "binance_api",
		  "off_chain_ticker": "ETHUSDT",
		  "normalize_by_pair": {
			"Base": "USDT",
			"Quote": "USD"
		  }
		},
		{
		  "name": "bybit_ws",
		  "off_chain_ticker": "ETHUSDT",
		  "normalize_by_pair": {
			"Base": "USDT",
			"Quote": "USD"
		  }
		},
		{
		  "name": "coinbase_api",
		  "off_chain_ticker": "ETH-USD"
		},
		{
		  "name": "huobi_ws",
		  "off_chain_ticker": "ethusdt",
		  "normalize_by_pair": {
			"Base": "USDT",
			"Quote": "USD"
		  }
		},
		{
		  "name": "kraken_api",
		  "off_chain_ticker": "XETHZUSD"
		},
		{
		  "name": "kucoin_ws",
		  "off_chain_ticker": "ETH-USDT",
		  "normalize_by_pair": {
			"Base": "USDT",
			"Quote": "USD"
		  }
		},
		{
		  "name": "mexc_ws",
		  "off_chain_ticker": "ETHUSDT",
		  "normalize_by_pair": {
			"Base": "USDT",
			"Quote": "USD"
		  }
		},
		{
		  "name": "okx_ws",
		  "off_chain_ticker": "ETH-USDT",
		  "normalize_by_pair": {
			"Base": "USDT",
			"Quote": "USD"
		  }
		}
	  ]
	}
]`
