package types

import (
	"slices"
)

func NewChannelState(portID, channelID string) ChannelState {
	return ChannelState{
		ChannelId: channelID,
		PortId:    portID,
		HaltState: HaltState{
			Halted:   false,
			HaltedBy: "",
		},
		Relayers: []string{},
	}
}

func (rl *ChannelState) HasRelayer(addrStr string) bool {
	return slices.Contains(rl.Relayers, addrStr)
}

func (rl *ChannelState) AddRelayer(addrStr string) {
	rl.Relayers = append(rl.Relayers, addrStr)
}
