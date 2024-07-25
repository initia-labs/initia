module TestAccount::Vesting {
    use initia_std::fungible_asset::{Self, Metadata, FungibleAsset};
    use initia_std::primary_fungible_store;
    use initia_std::object::{Self, Object, ExtendRef};
    use initia_std::table::{Self, Table};
    use initia_std::signer;
    use initia_std::error;
    use initia_std::block;
    use initia_std::event;
    use initia_std::option::{Self, Option};

    /// A struct that represents the vesting storage.
    struct VestingStore has key {
        /// A admin address.
        admin: address,

        /// The token metadata to vest.
        token_metadata: Object<Metadata>,

        /// Object extend reference.
        extend_ref: ExtendRef,

        /// A map of vesting schedules.
        vestings: Table<address, Vesting>,
    }

    /// A struct that represents a vesting schedule.
    struct Vesting has store, copy, drop {
        /// Amount of tokens to be vested
        allocation: u64,

        /// The total number of tokens that a given address will
        /// be able to ever claim.
        claimed_amount: u64,

        /// The time in which the vesting period starts.
        start_time: u64,

        /// The total period over which the tokens will vest.
        vesting_period: u64,

        /// The period of time in which tokens are vesting but
        /// cannot claimed. After the cliff period, tokens vested
        /// during the cliff period after immediately claimable.
        cliff_period: u64,

        /// The frequency in which an address can claim tokens post-cliff period
        claim_frequency: u64,
    }

    // Events

    #[event]
    struct VestingClaimed has drop, store {
        recipient: address,
        amount: u64,
    }

    // Errors

    const EALREADY_CREATED: u64 = 1;
    const EADMIN_PERMISSION: u64 = 2;
    const EVESTING_NOT_FOUND: u64 = 3;
    const EVESTING_ALREADY_EXISTS: u64 = 4;
    const EVESTING_NOT_EXISTS: u64 = 5;

    // Admin functions

    /// Creates a new vesting store.
    public entry fun create_vesting_store(
        creator: &signer,
        token_metadata: Object<Metadata>
    ) {
        let creator_addr = signer::address_of(creator);
        assert!(
            !exists<VestingStore>(creator_addr),
            error::already_exists(EALREADY_CREATED)
        );

        let constructor_ref = object::create_object(creator_addr, false);
        let extend_ref = object::generate_extend_ref(&constructor_ref);

        move_to(
            creator,
            VestingStore {
                admin: signer::address_of(creator),
                token_metadata,
                extend_ref,
                vestings: table::new<address, Vesting>(),
            }
        );
    }

    public entry fun fund_vesting(
        depositor: &signer, 
        creator: address,
        amount: u64,
    ) acquires VestingStore {
        let store = borrow_global<VestingStore>(creator);
        let token = primary_fungible_store::withdraw(depositor, store.token_metadata, amount);
        let store_addr = object::address_from_extend_ref(&store.extend_ref);
        primary_fungible_store::deposit(store_addr, token);
    }

    /// Withdraw vesting funds from the store.
    public entry fun withdraw_vesting_funds(
        admin: &signer,
        amount: u64
    ) acquires VestingStore {
        let store = borrow_global_mut<VestingStore>(signer::address_of(admin));
        assert!(
            store.admin == signer::address_of(admin),
            error::permission_denied(EADMIN_PERMISSION),
        );

        let store_signer = object::generate_signer_for_extending(&store.extend_ref);
        let token = primary_fungible_store::withdraw(
            &store_signer, store.token_metadata,
            amount
        );

        primary_fungible_store::deposit(store.admin, token);
    }

    /// Append vesting schedule for a recipient.
    public entry fun add_vesting(
        admin: &signer,
        recipient: address,
        allocation: u64,
        start_time: Option<u64>,
        vesting_period: u64,
        cliff_period: u64,
        claim_frequency: u64
    ) acquires VestingStore {
        let store = borrow_global_mut<VestingStore>(signer::address_of(admin));
        assert!(
            store.admin == signer::address_of(admin),
            error::permission_denied(EADMIN_PERMISSION),
        );
        assert!(
            !table::contains(&store.vestings, recipient),
            error::already_exists(EVESTING_ALREADY_EXISTS)
        );

        let start_time = if (option::is_some(&start_time)) {
            *option::borrow(&start_time)
        } else {
            let (_, cur_time) = block::get_block_info();
            cur_time
        };

        table::add(
            &mut store.vestings,
            recipient,
            Vesting {
                allocation,
                claimed_amount: 0,
                start_time,
                vesting_period,
                cliff_period,
                claim_frequency,
            }
        );
    }

    // User functions

    public entry fun claim_script(account: &signer, creator: address) acquires VestingStore {
        let tokens = claim(account, creator);

        // deposit to the account
        primary_fungible_store::deposit(signer::address_of(account), tokens);
    }

    /// Claims the vested tokens.
    public fun claim(account: &signer, creator: address): FungibleAsset acquires VestingStore {
        let account_addr = signer::address_of(account);
        let store = borrow_global_mut<VestingStore>(creator);
        assert!(
            table::contains(&store.vestings, account_addr),
            error::invalid_state(EVESTING_NOT_FOUND)
        );

        let vesting = table::borrow_mut(&mut store.vestings, account_addr);
        let claimable_amount = calc_claimable_amount(vesting);
        if (claimable_amount == 0) {
            return fungible_asset::zero(store.token_metadata)
        };

        // increase the claimed amount
        vesting.claimed_amount = vesting.claimed_amount + claimable_amount;

        // remove the vesting if all tokens are claimed
        if (vesting.claimed_amount == vesting.allocation) {
            table::remove(&mut store.vestings, account_addr);
        };

        // emit the event
        event::emit(
            VestingClaimed {
                recipient: account_addr,
                amount: claimable_amount,
            }
        );

        let store_signer = object::generate_signer_for_extending(&store.extend_ref);
        primary_fungible_store::withdraw(
            &store_signer, store.token_metadata,
            claimable_amount
        )
    }

    // View functions

    #[view]
    public fun vesting_token_metadata(creator: address): Object<Metadata> acquires VestingStore {
        let store = borrow_global<VestingStore>(creator);
        store.token_metadata
    }

    #[view]
    /// Returns the address of the vesting store. This address can be used
    /// to deposit vesting funds.
    public fun store_addr(creator: address): address acquires VestingStore {
        let store = borrow_global<VestingStore>(creator);
        object::address_from_extend_ref(&store.extend_ref)
    }

    #[view]
    public fun vesting_funds(creator: address): u64 acquires VestingStore {
        let store = borrow_global<VestingStore>(creator);
        let store_addr = object::address_from_extend_ref(&store.extend_ref);
        primary_fungible_store::balance(store_addr, store.token_metadata)
    }

    #[view]
    public fun has_vesting(creator: address, recipient: address): bool acquires VestingStore {
        let store = borrow_global<VestingStore>(creator);

        table::contains(&store.vestings, recipient)
    }

    #[view]
    public fun vesting_info(creator: address, recipient: address): Vesting acquires VestingStore {
        let store = borrow_global<VestingStore>(creator);
        assert!(
            table::contains(&store.vestings, recipient),
            error::not_found(EVESTING_NOT_EXISTS)
        );

        *table::borrow(&store.vestings, recipient)
    }

    #[view]
    public fun claimable_amount(creator: address, recipient: address): u64 acquires VestingStore {
        let store = borrow_global<VestingStore>(creator);
        assert!(
            table::contains(&store.vestings, recipient),
            error::not_found(EVESTING_NOT_EXISTS)
        );

        let vesting = table::borrow(&store.vestings, recipient);
        calc_claimable_amount(vesting)
    }

    #[view]
    public fun vesting_amount(creator: address, recipient: address): u64 acquires VestingStore {
        let store = borrow_global<VestingStore>(creator);
        assert!(
            table::contains(&store.vestings, recipient),
            error::not_found(EVESTING_NOT_EXISTS)
        );

        let vesting = table::borrow(&store.vestings, recipient);
        vesting.allocation - calc_claimable_amount(vesting)
    }

    #[view]
    public fun vesting_table_handle(creator: address): address acquires VestingStore {
        let store = borrow_global<VestingStore>(creator);
        table::handle(&store.vestings)
    }

    // Internal functions

    fun calc_claimable_amount(vesting: &Vesting): u64 {
        let (_, cur_time) = block::get_block_info();
        let cliff_time = vesting.start_time + vesting.cliff_period;

        // check if the vesting is still in the cliff period
        if (cur_time < cliff_time) {
            return 0
        };

        // calculate elapsed claim frequencies
        let elapsed_claim_frequencies = (cur_time - cliff_time) / vesting.claim_frequency;
        let elapsed_period = vesting.cliff_period + elapsed_claim_frequencies * vesting.claim_frequency;
        let vested_amount = (
            (vesting.allocation as u128) * (elapsed_period as u128) / (
                vesting.vesting_period as u128
            ) as u64
        );

        if (vested_amount > vesting.claimed_amount) {
            vested_amount - vesting.claimed_amount
        } else {
            0
        }
    }
}
