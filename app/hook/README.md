# Bridge Hook

A bridge hook is designed to intercept the events of bridge creation or bridge updating. Its primary role is to establish a permissioned Inter-Blockchain Communication (IBC) relayer for the connections. The input for this hook should be a UTF-8 encoded bytes JSON in the metadata field of `MsgCreateBridge`, as shown in the example below:

```json
{
  "perm_channels": [
    {
      "port_id": "transfer",
      "channel_id": "channel-0"
    },
    {
      "port_id": "icqhost",
      "channel_id": "channel-1"
    }
  ]
}
```

In this case, two channels are defined with a permissioned IBC relayer. The first channel has the port_id "transfer" and the channel_id "channel-0". The second channel has the port_id "icqhost" and the channel_id "channel-1".
