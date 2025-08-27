package ibctesting

import (
	"time"

	connectiontypes "github.com/cosmos/ibc-go/v8/modules/core/03-connection/types"
	channeltypes "github.com/cosmos/ibc-go/v8/modules/core/04-channel/types"
	"github.com/cosmos/ibc-go/v8/modules/core/exported"
	ibctm "github.com/cosmos/ibc-go/v8/modules/light-clients/07-tendermint"
	"github.com/cosmos/ibc-go/v8/testing/mock"
	ibctmattestor "github.com/initia-labs/initia/x/ibc/light-clients/07-tendermint-attestor"

	sdksecp256k1 "github.com/cosmos/cosmos-sdk/crypto/keys/secp256k1"
	cryptotypes "github.com/cosmos/cosmos-sdk/crypto/types"
)

type ClientConfig interface {
	GetClientType() string
}

type TendermintConfig struct {
	TrustLevel      ibctm.Fraction
	TrustingPeriod  time.Duration
	UnbondingPeriod time.Duration
	MaxClockDrift   time.Duration
}

func NewTendermintConfig() *TendermintConfig {
	return &TendermintConfig{
		TrustLevel:      DefaultTrustLevel,
		TrustingPeriod:  TrustingPeriod,
		UnbondingPeriod: UnbondingPeriod,
		MaxClockDrift:   MaxClockDrift,
	}
}

func (tmcfg *TendermintConfig) GetClientType() string {
	return exported.Tendermint
}

type TendermintAttestorConfig struct {
	TendermintConfig
	AttestorPrivkeys []cryptotypes.PrivKey
	Threshold        uint32
}

func (tmcfg *TendermintAttestorConfig) GetClientType() string {
	return ibctmattestor.TendermintAttestor
}

func NewTendermintAttestorConfig(numAttestors, threshold int) *TendermintAttestorConfig {
	privKeys := make([]cryptotypes.PrivKey, 0, numAttestors)
	for range numAttestors {
		privKeys = append(privKeys, sdksecp256k1.GenPrivKey())
	}

	return &TendermintAttestorConfig{
		TendermintConfig: TendermintConfig{
			TrustLevel:      DefaultTrustLevel,
			TrustingPeriod:  TrustingPeriod,
			UnbondingPeriod: UnbondingPeriod,
			MaxClockDrift:   MaxClockDrift,
		},
		AttestorPrivkeys: privKeys,
		Threshold:        uint32(threshold),
	}
}

type ConnectionConfig struct {
	DelayPeriod uint64
	Version     *connectiontypes.Version
}

func NewConnectionConfig() *ConnectionConfig {
	return &ConnectionConfig{
		DelayPeriod: DefaultDelayPeriod,
		Version:     ConnectionVersion,
	}
}

type ChannelConfig struct {
	PortID  string
	Version string
	Order   channeltypes.Order
}

func NewChannelConfig() *ChannelConfig {
	return &ChannelConfig{
		PortID:  mock.PortID,
		Version: DefaultChannelVersion,
		Order:   channeltypes.UNORDERED,
	}
}
