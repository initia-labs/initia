module std::BasicCoin {
    use std::signer;
    use std::event;
    use std::string;
    use initia_std::type_info;

    struct Initia {}

    struct Coin<phantom CoinType> has key, copy {
        value: u64,
        test: bool,
    }

    #[event]
    /// Event emitted when some amount of coins are withdrawn from an Collateral.
    struct MintEvent has drop, store {
        account: address,
        amount: u64,
        coin_type: string::String,
    }

    public entry fun mint<CoinType>(account: signer, value: u64) acquires Coin {
        let account_addr = signer::address_of(&account);
        if (!exists<Coin<CoinType>>(account_addr)) {
            move_to(&account, Coin<CoinType> { value, test: true });
        } else {
            let coin = borrow_global_mut<Coin<CoinType>>(account_addr);
            coin.value = coin.value + value;
        };

        // emit event
        event::emit(
            MintEvent {
                account: account_addr,
                amount: value,
                coin_type: type_info::type_name<CoinType>(),
            }
        );
    }

    #[view]
    public fun get<CoinType>(account: address): u64 acquires Coin{
        let c = borrow_global<Coin<CoinType>>(account);
        c.value
    }

    #[view]
    public fun number():u64 {
        123
    }

    #[view]
    public fun get_coin<CoinType>(addr: address): Coin<CoinType> acquires Coin {
        *borrow_global<Coin<CoinType>>(addr)
    }
}