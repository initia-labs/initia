module std::Counter {
    struct Count has key {
        value: u64
    }

    enum Shape has copy, drop{
        Circle { radius: u64 },
        Square { side: u64 },
        Triangle { base: u64, height: u64 },
    }

    #[event]
    struct TestEvent has drop {
        shape: Shape,
    }

    fun init_module(chain: &signer) {
        move_to(chain, Count {
            value: 0,
        });
    } 

    public entry fun increase() acquires Count {
        let count = borrow_global_mut<Count>(@initia_std);
        count.value = count.value + 1;

        std::event::emit(TestEvent{shape: Shape::Circle{radius: count.value} });
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