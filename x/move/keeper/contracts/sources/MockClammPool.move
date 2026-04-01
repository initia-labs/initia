/// Mock CLAMM pool module used in unit tests.
///
/// The Pool struct mirrors the real CLAMM pool BCS layout so that
/// the Go connector (ReadCLAMMPool / CLAMMBaseSpotPrice) can be verified
/// against actual Move-VM serialization.
///
/// BCS field offsets (each Object<T>/address = 32 bytes, u64 = 8 bytes, u128 = 16 bytes):
///   [0,   32): metadata_0            - Object<Metadata>
///   [32,  64): metadata_1            - Object<Metadata>
///   [64,  96): collection_obj_inner  - address (Object<T>.inner placeholder)
///   [96, 128): mutator_ref_self      - address (MutatorRef.self placeholder)
///   [128,160): oracle_obj_inner      - address (Object<T>.inner placeholder)
///   [160,168): swap_fee_bps          - u64
///   [168,176): tick_spacing          - u64
///   [176,184): max_liquidity_per_tick- u64
///   [184,216): extend_ref_self       - address (ExtendRef.self placeholder)
///   [216,224): extend_ref_version    - u64     (ExtendRef.version placeholder)
///   [224,232): position_id           - u64
///   [232,248): sqrt_price            - u128  (Q64.64 fixed-point)
module cafe::pool {
    use std::signer;
    use std::string::String;
    use std::error;
    use initia_std::object::{Self, Object};
    use initia_std::fungible_asset::Metadata;
    use initia_std::primary_fungible_store;

    const ERR_POOL_NOT_EXISTS: u64 = 1;
    const ERR_INSUFFICIENT_INPUT: u64 = 2;
    const ERR_PRICE_LIMIT_GT_SQRT_PRICE: u64 = 3;
    const ERR_PRICE_LIMIT_LT_MIN_SQRT_RATIO: u64 = 4;
    const ERR_PRICE_LIMIT_LT_SQRT_PRICE: u64 = 5;
    const ERR_PRICE_LIMIT_GT_MAX_SQRT_RATIO: u64 = 6;
    const ERR_INSUFFICIENT_OUTPUT: u64 = 7;

    // tick_math constants from dex_clamm_math::tick_math
    const MIN_SQRT_RATIO: u128 = 4295048017;
    const MAX_SQRT_RATIO: u128 = 79226673515401279992447579062;

    struct Pool has key {
        /// Metadata of asset 0 (address-sorted smaller)
        metadata_0: Object<Metadata>,
        /// Metadata of asset 1 (address-sorted larger)
        metadata_1: Object<Metadata>,
        /// Placeholder: Object<Collection>.inner  (32 bytes)
        collection_obj_inner: address,
        /// Placeholder: MutatorRef.self            (32 bytes)
        mutator_ref_self: address,
        /// Placeholder: Object<Oracle>.inner       (32 bytes)
        oracle_obj_inner: address,
        /// Swap fee in basis points
        swap_fee_bps: u64,
        /// Pool tick spacing
        tick_spacing: u64,
        /// Maximum liquidity per tick
        max_liquidity_per_tick: u64,
        /// Placeholder: ExtendRef.self             (32 bytes)
        extend_ref_self: address,
        /// Placeholder: ExtendRef.version          (8 bytes)
        extend_ref_version: u64,
        /// Last utilized position number
        position_id: u64,
        /// Current sqrt price in Q64.64 fixed-point: sqrt(token1/token0) * 2^64
        sqrt_price: u128,
    }

    /// Test-only configuration to control simulated swap output.
    struct TestSwapConfig has key {
        zero_for_one_amount_out: u64,
        one_for_zero_amount_out: u64,
    }

    /// Stores object::ExtendRef so swap can generate a signer for pool-owned balances.
    struct PoolSignerCap has key {
        extend_ref: object::ExtendRef,
    }

    /// Create a mock CLAMM pool at the signer's address.
    /// metadata_0_addr and metadata_1_addr must be address-sorted (0 <= 1).
    /// sqrt_price is the Q64.64 value: sqrt(token1/token0) * 2^64.
    public entry fun create_pool(
        account: &signer,
        metadata_0_addr: address,
        metadata_1_addr: address,
        sqrt_price: u128,
    ) {
        let constructor_ref = &object::create_named_object(account, b"mock_pool");
        let pool_signer = &object::generate_signer(constructor_ref);

        let metadata_0 = object::address_to_object<Metadata>(metadata_0_addr);
        let metadata_1 = object::address_to_object<Metadata>(metadata_1_addr);
        move_to(pool_signer, Pool {
            metadata_0,
            metadata_1,
            collection_obj_inner: @0x0,
            mutator_ref_self: @0x0,
            oracle_obj_inner: @0x0,
            swap_fee_bps: 0,
            tick_spacing: 1,
            max_liquidity_per_tick: 0,
            extend_ref_self: @0x0,
            extend_ref_version: 0,
            position_id: 0,
            sqrt_price,
        });

        move_to(pool_signer, TestSwapConfig {
            zero_for_one_amount_out: 0,
            one_for_zero_amount_out: 0,
        });
        move_to(pool_signer, PoolSignerCap {
            extend_ref: object::generate_extend_ref(constructor_ref),
        });
    }

    /// Test-only function to configure mock swap outputs.
    public entry fun set_test_swap_amounts(
        account: &signer,
        pool_obj: address,
        zero_for_one_amount_out: u64,
        one_for_zero_amount_out: u64
    ) acquires TestSwapConfig {
        let _ = signer::address_of(account);
        assert!(exists<Pool>(pool_obj), error::not_found(ERR_POOL_NOT_EXISTS));
        assert!(exists<TestSwapConfig>(pool_obj), error::not_found(ERR_POOL_NOT_EXISTS));

        let cfg = borrow_global_mut<TestSwapConfig>(pool_obj);
        cfg.zero_for_one_amount_out = zero_for_one_amount_out;
        cfg.one_for_zero_amount_out = one_for_zero_amount_out;
    }

    public fun swap(
        account: &signer,
        pool_obj: address,
        amount_in: u64,
        amount_out: u64,
        sqrt_price_limit: u128,
        exact_in: bool,
        zero_for_one: bool,
        integrator: String
    ) acquires Pool, TestSwapConfig, PoolSignerCap {
        let _ = signer::address_of(account);
        let _ = integrator;

        assert!(exists<Pool>(pool_obj), error::not_found(ERR_POOL_NOT_EXISTS));
        assert!(exists<TestSwapConfig>(pool_obj), error::not_found(ERR_POOL_NOT_EXISTS));
        assert!(exists<PoolSignerCap>(pool_obj), error::not_found(ERR_POOL_NOT_EXISTS));
        assert!(amount_in > 0, error::invalid_argument(ERR_INSUFFICIENT_INPUT));

        let pool = borrow_global<Pool>(pool_obj);
        if (zero_for_one) {
            assert!(
                sqrt_price_limit < pool.sqrt_price,
                error::invalid_argument(ERR_PRICE_LIMIT_GT_SQRT_PRICE)
            );
            assert!(
                sqrt_price_limit > MIN_SQRT_RATIO,
                error::invalid_argument(ERR_PRICE_LIMIT_LT_MIN_SQRT_RATIO)
            );
        } else {
            assert!(
                sqrt_price_limit > pool.sqrt_price,
                error::invalid_argument(ERR_PRICE_LIMIT_LT_SQRT_PRICE)
            );
            assert!(
                sqrt_price_limit < MAX_SQRT_RATIO,
                error::invalid_argument(ERR_PRICE_LIMIT_GT_MAX_SQRT_RATIO)
            );
        };

        let cfg = borrow_global<TestSwapConfig>(pool_obj);
        let simulated_amount_out =
            if (zero_for_one) cfg.zero_for_one_amount_out
            else cfg.one_for_zero_amount_out;

        if (exact_in) {
            assert!(
                simulated_amount_out >= amount_out,
                error::invalid_argument(ERR_INSUFFICIENT_OUTPUT)
            );
        } else {
            assert!(
                simulated_amount_out == amount_out,
                error::invalid_argument(ERR_INSUFFICIENT_OUTPUT)
            );
        };

        // Real input transfer: executor -> pool address.
        let (metadata_in, metadata_out) =
            if (zero_for_one) (pool.metadata_0, pool.metadata_1)
            else (pool.metadata_1, pool.metadata_0);
        let asset_in =
            primary_fungible_store::withdraw(account, metadata_in, amount_in);
        primary_fungible_store::deposit(pool_obj, asset_in);

        // Real output transfer: pool address -> executor.
        let signer_cap = borrow_global<PoolSignerCap>(pool_obj);
        let pool_signer = &object::generate_signer_for_extending(&signer_cap.extend_ref);
        let asset_out =
            primary_fungible_store::withdraw(pool_signer, metadata_out, simulated_amount_out);
        primary_fungible_store::deposit(signer::address_of(account), asset_out);
    }
}

module cafe::scripts {
    use std::string::String;

    use cafe::pool;

    public entry fun swap(
        account: &signer,
        pool_obj: address,
        amount_in: u64,
        amount_out: u64,
        sqrt_price_limit: u128,
        exact_in: bool,
        zero_for_one: bool,
        integrator: String
    ) {
        pool::swap(
            account,
            pool_obj,
            amount_in,
            amount_out,
            sqrt_price_limit,
            exact_in,
            zero_for_one,
            integrator
        );
    }
}
