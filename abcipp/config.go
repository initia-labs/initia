package abcipp

import (
	"time"

	"github.com/spf13/cast"

	servertypes "github.com/cosmos/cosmos-sdk/server/types"
)

const (
	FlagMaxQueuedPerSender = "abcipp.max-queued-per-sender"
	FlagMaxQueuedTotal     = "abcipp.max-queued-total"
	FlagQueuedGapTTL       = "abcipp.queued-gap-ttl"
)

const (
	// DefaultMaxQueuedPerSender is the default per-sender queued tx limit.
	DefaultMaxQueuedPerSender = 64
	// DefaultMaxQueuedTotal is the default total queued tx limit.
	DefaultMaxQueuedTotal = 1024
	// DefaultQueuedGapTTL is the default max time to keep sender queued txs when
	// the sender has no active tx and is missing the head nonce.
	DefaultQueuedGapTTL = 60 * time.Second
)

// AppConfig is the node-local configuration for abcipp mempool behavior.
type AppConfig struct {
	MaxQueuedPerSender int           `mapstructure:"max-queued-per-sender"`
	MaxQueuedTotal     int           `mapstructure:"max-queued-total"`
	QueuedGapTTL       time.Duration `mapstructure:"queued-gap-ttl"`
}

// DefaultAppConfig returns default abcipp app config values.
func DefaultAppConfig() AppConfig {
	return AppConfig{
		MaxQueuedPerSender: DefaultMaxQueuedPerSender,
		MaxQueuedTotal:     DefaultMaxQueuedTotal,
		QueuedGapTTL:       DefaultQueuedGapTTL,
	}
}

// GetConfig loads abcipp config from app options.
func GetConfig(appOpts servertypes.AppOptions) AppConfig {
	return AppConfig{
		MaxQueuedPerSender: cast.ToInt(appOpts.Get(FlagMaxQueuedPerSender)),
		MaxQueuedTotal:     cast.ToInt(appOpts.Get(FlagMaxQueuedTotal)),
		QueuedGapTTL:       cast.ToDuration(appOpts.Get(FlagQueuedGapTTL)),
	}
}

// DefaultConfigTemplate is the app.toml section for abcipp config.
const DefaultConfigTemplate = `
###############################################################################
###                         ABCIPP                                           ###
###############################################################################

[abcipp]
# Maximum queued transactions kept per sender.
max-queued-per-sender = {{ .ABCIPP.MaxQueuedPerSender }}
# Maximum queued transactions kept globally.
max-queued-total = {{ .ABCIPP.MaxQueuedTotal }}
# How long to keep queued txs for a stalled sender missing its head nonce.
queued-gap-ttl = "{{ .ABCIPP.QueuedGapTTL }}"
`
