module Publisher::BenchHeavyState {
    use initia_std::table::{Self, Table};
    use initia_std::signer;

    struct State has key {
        data: Table<u64, u64>,
        counter: u64,
    }

    struct SharedState has key {
        data: Table<u64, u64>,
        counter: u64,
    }

    fun init_module(account: &signer) {
        move_to(account, SharedState {
            data: table::new<u64, u64>(),
            counter: 0,
        });
    }

    /// writes `shared_count` entries to a shared global table (contended)
    /// and `local_count` entries to the caller's own table (non-contended).
    public entry fun write_mixed(
        account: &signer,
        shared_count: u64,
        local_count: u64,
    ) acquires SharedState, State {
        let shared = borrow_global_mut<SharedState>(@Publisher);
        let i = 0;
        while (i < shared_count) {
            let key = shared.counter + i;
            shared.data.upsert(key, key);
            i += 1;
        };
        shared.counter += shared_count;

        let addr = signer::address_of(account);
        if (!exists<State>(addr)) {
            move_to(account, State {
                data: table::new<u64, u64>(),
                counter: 0,
            });
        };
        let state = borrow_global_mut<State>(addr);
        i = 0;
        while (i < local_count) {
            let key = state.counter + i;
            state.data.add(key, key);
            i += 1;
        };
        state.counter += local_count;
    }
}
