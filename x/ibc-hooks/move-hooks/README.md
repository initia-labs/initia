# IBC-hooks

This module is copied from [osmosis](https://github.com/osmosis-labs/osmosis) and changed to execute move contract with ICS-20 token transfer calls.

## Move Hooks

The move hook is an IBC middleware which is used to allow ICS-20 token transfers to initiate contract calls.
This allows cross-chain contract calls, that involve token movement.
This is useful for a variety of use cases.
One of primary importance is cross-chain swaps, which is an extremely powerful primitive.

The mechanism enabling this is a `memo` field on every ICS20 and ICS721 transfer packet as of [IBC v3.4.0](https://medium.com/the-interchain-foundation/moving-beyond-simple-token-transfers-d42b2b1dc29b).
Move hooks is an IBC middleware that parses an ICS20 transfer, and if the `memo` field is of a particular form, executes a move contract call. We now detail the `memo` format for `move` contract calls, and the execution guarantees provided.

### Move Contract Execution Format

Before we dive into the IBC metadata format, we show the hook data format, so the reader has a sense of what are the fields we need to be setting in.
The move `MsgExecute` is defined [here](../../move/types/tx.pb.go) and other types are defined [here](./message.go) as the following type:

```go
// HookData defines a wrapper for move execute message
// and async callback.
type HookData struct {
 // Message is a move execute message which will be executed
 // at `OnRecvPacket` of receiver chain.
 Message movetypes.MsgExecute `json:"message"`

 // AsyncCallback is a callback message which will be executed
 // at `OnTimeoutPacket` and `OnAcknowledgementPacket` of
 // sender chain.
 AsyncCallback *AsyncCallback `json:"async_callback,omitempty"`
}

// AsyncCallback is data wrapper which is required
// when we implement async callback.
type AsyncCallback struct {
 // callback id should be issued form the executor contract
 Id            uint64 `json:"id"`
 ModuleAddress string `json:"module_address"`
 ModuleName    string `json:"module_name"`
}

type MsgExecute struct {
 // Sender is the that actor that signed the messages
 Sender string `protobuf:"bytes,1,opt,name=sender,proto3" json:"sender,omitempty"`
 // ModuleAddress is the address of the module deployer
 ModuleAddress string `protobuf:"bytes,2,opt,name=module_address,json=moduleAddress,proto3" json:"module_address,omitempty"`
 // ModuleName is the name of module to execute
 ModuleName string `protobuf:"bytes,3,opt,name=module_name,json=moduleName,proto3" json:"module_name,omitempty"`
 // FunctionName is the name of a function to execute
 FunctionName string `protobuf:"bytes,4,opt,name=function_name,json=functionName,proto3" json:"function_name,omitempty"`
 // TypeArgs is the type arguments of a function to execute
 // ex) "0x1::BasicCoin::Initia", "bool", "u8", "u64"
 TypeArgs []string `protobuf:"bytes,5,rep,name=type_args,json=typeArgs,proto3" json:"type_args,omitempty"`
 // Args is the arguments of a function to execute
 // - number: little endian
 // - string: base64 bytes
 Args [][]byte `protobuf:"bytes,6,rep,name=args,proto3" json:"args,omitempty"`
}
```

So we detail where we want to get each of these fields from:

- Sender: We cannot trust the sender of an IBC packet, the counter-party chain has full ability to lie about it.
  We cannot risk this sender being confused for a particular user or module address on Initia.
  So we replace the sender with an account to represent the sender prefixed by the channel and a move module prefix.
  This is done by setting the sender to `Bech32(Hash(Hash("ibc-move-hook-intermediary") + channelID/sender))`, where the channelId is the channel id on the local chain.
- ModuleAddress: This field should be directly obtained from the ICS-20 packet metadata
- ModuleName: This field should be directly obtained from the ICS-20 packet metadata
- FunctionName: This field should be directly obtained from the ICS-20 packet metadata
- TypeArgs: This field should be directly obtained from the ICS-20 packet metadata
- Args: This field should be directly obtained from the ICS-20 packet metadata.

So our constructed move message that we execute will look like:

```go
msg := MsgExecuteContract{
 // Sender is the that actor that signed the messages
 Sender: "init1-hash-of-channel-and-sender",
 // ModuleAddress is the address of the module deployer
 ModuleAddress: packet.data.memo["move"]["message"]["module_address"],
    // ModuleName is the name of module to execute
 ModuleName: packet.data.memo["move"]["message"]["module_name"],
    // FunctionName is the name of a function to execute
 FunctionName: packet.data.memo["move"]["message"]["function_name"],
 // TypeArgs is the type arguments of a function to execute
 // ex) "0x1::BasicCoin::Initia", "bool", "u8", "u64"
 TypeArgs: packet.data.memo["move"]["message"]["type_args"],
 // Args is the arguments of a function to execute
 // - number: little endian
 // - string: base64 bytes
 Args: packet.data.memo["move"]["message"]["args"]}
```

### ICS20 packet structure

So given the details above, we propagate the implied ICS20 packet data structure.
ICS20 is JSON native, so we use JSON for the memo format.

```json
{
  //... other ibc fields that we don't care about
  "data": {
    "denom": "denom on counterparty chain (e.g. uatom)", // will be transformed to the local denom (ibc/...)
    "amount": "1000",
    "sender": "addr on counterparty chain", // will be transformed
    "receiver": "ModuleAddr::ModuleName::FunctionName",
    "memo": {
      "move": {
          // execute message on receive packet
          "message": {
            "module_address": "0x1",
            "module_name": "dex",
            "function_name": "swap",
            "type_args": ["0x1::native_uinit::Coin", "0x1::native_uusdc::Coin"],
            "args": ["base64 encoded bytes array"]
          },
          // optional field to get async callback (ack and timeout)
          "async_callback": {
            "id": 1,
            "module_address": "0x1",
            "module_name": "dex"
          }
        }
      }
    }
  }
}
```

An ICS20 packet is formatted correctly for movehooks iff the following all hold:

- `memo` is not blank
- `memo` is valid JSON
- `memo` has at least one key, with value `"move"`
- `memo["move"]["message"]` has exactly five entries, `"module_address"`, `"module_name"`, `"function_name"`, `"type_args"` and `"args"`
- `receiver` == "" || `receiver` == "module_address::module_name::function_name"

We consider an ICS20 packet as directed towards movehooks iff all of the following hold:

- `memo` is not blank
- `memo` is valid JSON
- `memo` has at least one key, with name `"move"`

If an ICS20 packet is not directed towards movehooks, movehooks doesn't do anything.
If an ICS20 packet is directed towards movehooks, and is formatted incorrectly, then movehooks returns an error.

### Execution flow

Pre move hooks:

- Ensure the incoming IBC packet is cryptogaphically valid
- Ensure the incoming IBC packet is not timed out.

In move hooks, pre packet execution:

- Ensure the packet is correctly formatted (as defined above)
- Edit the receiver to be the hardcoded IBC module account

In move hooks, post packet execution:

- Construct move message as defined before
- Execute move message
- if move message has error, return ErrAck
- otherwise continue through middleware

### Async Callback

A contract that sends an IBC transfer, may need to listen for the ACK from that packet.
To allow contracts to listen on the ack of specific packets, we provide Ack callbacks.
The contract, which wants to receive ack callback, have to implement two functions.

- ibc_ack
- ibc_timeout

```move
public entry fun ibc_ack(
  callback_id: u64,
  success:     bool,
)

public entry fun ibc_timeout(
  callback_id: u64,
)
```

Also when a contract make IBC transfer request, it should provide async callback data through memo field.

- `memo['move']['async_callback']['id']`:  the async callback id is assigned from the contract. so later it will be passed as argument of `ibc_ack` and `ibc_timeout`.
- `memo['move']['async_callback']['module_address']`: The address of module which defines the callback function.
- `memo['move']['async_callback']['module_name']`: The name of module which defines the callback function.
