module cafe::test_dispatchable_token {
    use initia_std::fungible_asset::{Self, MintRef, BurnRef, TransferRef};
    use initia_std::dispatchable_fungible_asset;
    use initia_std::primary_fungible_store;
    use initia_std::object::{Self, Object};
    use initia_std::function_info;

    use std::option;
    use std::option::Option;
    use std::signer;
    use std::string;

    struct ManagingRefs has key {
        mint_ref: MintRef,
        burn_ref: BurnRef,
        transfer_ref: TransferRef,
    }

    public entry fun initialize(deployer: &signer) {
        let constructor_ref =
            &object::create_named_object(deployer, b"test_token");

        primary_fungible_store::create_primary_store_enabled_fungible_asset(
            constructor_ref,
            option::some(1000000000000000000),
            string::utf8(b"Test Token Name"),
            string::utf8(b"test_token"),
            6,
            string::utf8(b"https://example.com/icon.png"),
            string::utf8(b"https://example.com/project")
        );

        let balance_value =
            function_info::new_function_info(
                deployer,
                string::utf8(b"test_dispatchable_token"),
                string::utf8(b"derived_balance")
            );
        let supply_value =
            function_info::new_function_info(
                deployer,
                string::utf8(b"test_dispatchable_token"),
                string::utf8(b"derived_supply")
            );
        dispatchable_fungible_asset::register_dispatch_functions(
            constructor_ref,
            option::none(),
            option::none(),
            option::some(balance_value)
        );
        dispatchable_fungible_asset::register_derive_supply_dispatch_function(
            constructor_ref, option::some(supply_value)
        );

        let mint_ref = fungible_asset::generate_mint_ref(constructor_ref);
        let burn_ref = fungible_asset::generate_burn_ref(constructor_ref);
        let transfer_ref = fungible_asset::generate_transfer_ref(constructor_ref);

        move_to(deployer, ManagingRefs {
            mint_ref,
            burn_ref,
            transfer_ref,
        });
    }

    public fun derived_balance<T: key>(store: Object<T>): u64 {
        // Derived value is always 10x!
        fungible_asset::balance_without_sanity_check(store) * 10
    }

    public fun derived_supply<T: key>(metadata: Object<T>): Option<u128> {
        // Derived supply is 10x.
        if (option::is_some(&fungible_asset::supply_without_sanity_check(metadata))) {
            return option::some(
                option::extract(&mut fungible_asset::supply_without_sanity_check(metadata)) * 10
            )
        };
        option::none()
    }

    public entry fun mint(account: &signer, to: address, amount: u64) acquires ManagingRefs {
        let mint_ref = &borrow_global<ManagingRefs>(signer::address_of(account)).mint_ref;
        let fa = fungible_asset::mint(mint_ref, amount);
        primary_fungible_store::deposit(to, fa);
    }
}
