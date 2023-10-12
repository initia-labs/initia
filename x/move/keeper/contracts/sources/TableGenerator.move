module TestAccount::TableGenerator {
    use initia_std::table as T;

    struct S<phantom K: copy + drop,phantom V> has key {
        t: T::Table<K, V>
    }

    public entry fun generate_table(s: &signer, a: u64){
        let t = T::new<u64, u64>();
        let index_a = 0;
        while(index_a < a) {
            T::add(&mut t, index_a, index_a);
            index_a = index_a + 1;
        };
        move_to(s, S { t });
    }

}