module std::hook_sender {
    use initia_std::coin;
    use initia_std::query;
    use initia_std::string::String;
    use initia_std::json;

    struct HookTransferFunds has copy, drop {
        denom: String,
        amount: u64
    }

    public entry fun send_funds(sender: &signer, receiver: address) {
        let response = query::query_custom(b"move_hook_get_transfer_funds", b"");
        let res = json::unmarshal<HookTransferFunds>(response);

        let coin_metadata = coin::denom_to_metadata(res.denom);
        coin::transfer(sender, receiver, coin_metadata, res.amount);
    }
}
