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
        let response = query::query_custom(b"move_hook_query_transfer_funds", b"");
        let res = json::unmarshal<Option<TransferFunds>>(response);
        assert!(option::is_some(&res), 1000);

        let res = option::borrow(&res);
        let coin_metadata = coin::denom_to_metadata(res.amount_in_packet.denom);
        coin::transfer(sender, receiver, coin_metadata, res.amount_in_packet.amount);
    }
}
