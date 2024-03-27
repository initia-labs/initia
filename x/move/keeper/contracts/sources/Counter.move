module std::Counter {
    struct Count has key {
        value: u64
    }

    fun init_module(chain: &signer) {
        move_to(chain, Count {
            value: 0,
        });
    } 

    public entry fun increase() acquires Count {
        let count = borrow_global_mut<Count>(@initia_std);
        count.value = count.value + 1;
    }

    public entry fun ibc_ack(
        callback_id: u64,
        success:     bool,
    ) acquires Count {
        let num = if (success) { callback_id } else { 1 };
        let count = borrow_global_mut<Count>(@initia_std);
        count.value = count.value + num;
    }

    public entry fun ibc_timeout(
        callback_id: u64,
    ) acquires Count {
        let count = borrow_global_mut<Count>(@initia_std);
        count.value = count.value + callback_id;
    }

    #[view]
    public fun get(): u64 acquires Count {
        let c = borrow_global<Count>(@initia_std);
        c.value
    }
}