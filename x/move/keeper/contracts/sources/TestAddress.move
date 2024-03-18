module TestAccount::TestAddress {
    use initia_std::string::String;
    use initia_std::address;

    #[view]
    public fun to_sdk(addr: address): String {
        address::to_sdk(addr)    
    }

    #[view]
    public fun from_sdk(addr: String): address {
        address::from_sdk(addr)
    }
}