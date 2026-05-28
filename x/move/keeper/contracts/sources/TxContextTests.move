module TestAccount::TxContextTests {
    use std::option::Option;
    use initia_std::transaction_context;

    struct FeePayerStore has key {
        value: Option<address>,
    }

    public entry fun store_fee_payer(sender: &signer) {
        let fee_payer = transaction_context::fee_payer();
        move_to(sender, FeePayerStore { value: fee_payer });
    }

    #[view]
    public fun read_stored_fee_payer(addr: address): Option<address> acquires FeePayerStore {
        borrow_global<FeePayerStore>(addr).value
    }
}
