package types

import (
	transfertypes "github.com/cosmos/ibc-go/v8/modules/apps/transfer/types"

	nfttransfertypes "github.com/initia-labs/initia/x/ibc/nft-transfer/types"
)

type ICSData struct {
	ICS20Data  *transfertypes.FungibleTokenPacketData
	ICS721Data *nfttransfertypes.NonFungibleTokenPacketData
}

func (d ICSData) GetBytes() []byte {
	if d.ICS20Data != nil {
		return d.ICS20Data.GetBytes()
	}
	if d.ICS721Data != nil {
		return d.ICS721Data.GetBytes()
	}
	return nil
}

func (d ICSData) SetMemo(memo string) {
	switch {
	case d.ICS20Data != nil:
		d.ICS20Data.Memo = memo
	case d.ICS721Data != nil:
		d.ICS721Data.Memo = memo
	}
}

func (d ICSData) GetMemo() string {
	switch {
	case d.ICS20Data != nil:
		return d.ICS20Data.Memo
	case d.ICS721Data != nil:
		return d.ICS721Data.Memo
	default:
		return ""
	}
}

func (d ICSData) SetReceiver(receiver string) {
	switch {
	case d.ICS20Data != nil:
		d.ICS20Data.Receiver = receiver
	case d.ICS721Data != nil:
		d.ICS721Data.Receiver = receiver
	}
}

func (d ICSData) GetReceiver() string {
	switch {
	case d.ICS20Data != nil:
		return d.ICS20Data.Receiver
	case d.ICS721Data != nil:
		return d.ICS721Data.Receiver
	default:
		return ""
	}
}

func (d ICSData) SetSender(sender string) {
	switch {
	case d.ICS20Data != nil:
		d.ICS20Data.Sender = sender
	case d.ICS721Data != nil:
		d.ICS721Data.Sender = sender
	}
}

func (d ICSData) GetSender() string {
	switch {
	case d.ICS20Data != nil:
		return d.ICS20Data.Sender
	case d.ICS721Data != nil:
		return d.ICS721Data.Sender
	default:
		return ""
	}
}
