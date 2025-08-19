module TestAccount::dex_migration {
    use initia_std::object::{Self, Object, ExtendRef};
    use initia_std::fungible_asset::{Self, Metadata, FungibleStore};
    use initia_std::primary_fungible_store;
    use initia_std::coin;
    use initia_std::string::String;
    use initia_std::signer;

    struct Info has key {
        extend_ref: ExtendRef,
        coin_in_store: Object<FungibleStore>,
        coin_out_store: Object<FungibleStore>
    }

    public entry fun initialize(
        account: &signer, coin_in: Object<Metadata>, coin_out: Object<Metadata>
    ) {
        assert!(signer::address_of(account) == @TestAccount, 0);

        let extend_ref =
            object::generate_extend_ref(&object::create_object(@TestAccount, false));
        let pool_signer = object::generate_signer_for_extending(&extend_ref);
        let pool_address = signer::address_of(&pool_signer);

        move_to(
            account,
            Info {
                extend_ref: extend_ref,
                coin_in_store: primary_fungible_store::ensure_primary_store_exists(
                    pool_address, coin_in
                ),
                coin_out_store: primary_fungible_store::ensure_primary_store_exists(
                    pool_address, coin_out
                )
            }
        );
    }

    public entry fun provide_liquidity(
        account: &signer, coin: Object<Metadata>, amount: u64
    ) acquires Info {
        let info = borrow_global<Info>(@TestAccount);
        let pool_signer = object::generate_signer_for_extending(&info.extend_ref);
        let pool_address = signer::address_of(&pool_signer);

        coin::transfer(account, pool_address, coin, amount);
    }

    public entry fun convert(
        account: &signer,
        coin_in: Object<Metadata>,
        coin_out: Object<Metadata>,
        amount: u64
    ) acquires Info {
        let info = borrow_global<Info>(@TestAccount);
        let pool_signer = object::generate_signer_for_extending(&info.extend_ref);
        let pool_address = signer::address_of(&pool_signer);

        coin::transfer(account, pool_address, coin_in, amount);
        coin::transfer(
            &pool_signer,
            signer::address_of(account),
            coin_out,
            amount
        );
    }

    #[view]
    public fun denom_out(denom_in: String): String acquires Info {
        let info = borrow_global<Info>(@TestAccount);

        assert!(
            coin::denom_to_metadata(denom_in)
                == fungible_asset::store_metadata(info.coin_in_store),
            0
        );

        coin::metadata_to_denom(fungible_asset::store_metadata(info.coin_out_store))
    }
}
