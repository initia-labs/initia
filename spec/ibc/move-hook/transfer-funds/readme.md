# Transfer Funds Custom Query

## Overview

The Transfer Funds Custom Query is a feature within the IBC Hooks module that provides Move smart contracts with access to IBC transfer packet information through a custom query interface. This enables contracts to retrieve details about the actual amount transferred and balance changes that occurred during IBC transfers.

## Purpose

The custom query system provides Move contracts with real-time access to IBC transfer data by:

1. Capturing transfer information during IBC packet processing
2. Exposing this data through a dedicated custom query interface
3. Enabling contracts to make informed decisions based on actual transfer amounts
4. Automatically managing data lifecycle (set, query, clear)

## Architecture

### Data Flow

```plaintext
IBC Transfer Packet Received
    ↓
Record Balance Before Transfer
    ↓
Execute Underlying Transfer Logic
    ↓
Record Balance After Transfer
    ↓
Calculate Balance Change
    ↓
Store TransferFunds Data
    ↓
Execute Move Contract Hook
    ↓
Contract Queries TransferFunds via Custom Query
    ↓
Contract Performs Custom Logic
    ↓
Clear TransferFunds Data
```

### Data Structure

The `TransferFunds` struct contains:

```go
type TransferFunds struct {
    BalanceChange  types.Coin `json:"balance_change"`   // Actual balance change
    AmountInPacket types.Coin `json:"amount_in_packet"` // Amount specified in packet
}
```

- **BalanceChange**: The actual change in balance that occurred during the transfer
- **AmountInPacket**: The amount that was specified in the original IBC packet

## Implementation Details

### Storage

The transfer funds data is stored in a transient collection, meaning it's only available for the duration of the current transaction and is automatically cleared afterward. This ensures data is only accessible during hook execution.

```go
transferFunds collections.Item[types.TransferFunds]
```

## Query Interface

### Custom Query Name

```plantext
move_hook_get_transfer_funds
```

### Query Parameters

- **Input**: Empty byte array (`[]byte{}`)
- **Output**: JSON-encoded `TransferFunds` or `null` (`0x1::option::Option<TransferFunds>`)

### Response Format

#### When data is available

```json
{
  "balance_change": {
    "denom": "uinit",
    "amount": "1000000"
  },
  "amount_in_packet": {
    "denom": "uinit", 
    "amount": "1000000"
  }
}
```

#### When no data is available

```json
null
```

## Move Contract Integration

### Example Contract Usage

```move
module std::hook_sender {
    use initia_std::coin;
    use initia_std::query;
    use initia_std::string::String;
    use initia_std::json;
    use initia_std::option::{Self, Option};

    struct TransferFunds has copy, drop {
        balance_change: Coin,
        amount_in_packet: Coin,
    }

    struct Coin has copy, drop {
        denom: String,
        amount: u64,
    }

    public entry fun send_funds(sender: &signer, receiver: address) {
        // Execute custom query to get transfer funds data
        let response = query::query_custom(b"move_hook_get_transfer_funds", b"");
        let res = json::unmarshal<Option<TransferFunds>>(response);
        
        // Ensure data is available
        assert!(option::is_some(&res), 1000);

        // Extract the transfer funds data
        let res = option::borrow(&res);
        
        // Get coin metadata for the balance change denom
        let coin_metadata = coin::denom_to_metadata(res.balance_change.denom);
        
        // Transfer the actual balance change amount
        coin::transfer(sender, receiver, coin_metadata, res.balance_change.amount);
    }
}
```

### Key Points for Move Contracts

1. **Query Name**: Use `b"move_hook_get_transfer_funds"` as the custom query name
2. **Empty Parameters**: Pass empty byte array `b""` as parameters
3. **Optional Response**: The response is wrapped in an `Option<TransferFunds>`
4. **Null Handling**: Check if the option contains data before using it
5. **Data Availability**: Data is only available during hook execution
