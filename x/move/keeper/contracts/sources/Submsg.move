module TestAccount::submsg {
    use std::cosmos;
    use std::string::String;
    use std::event;
    use std::signer;

    public entry fun stargate(
        account: &signer,
        data: vector<u8>,
        allow_failure: bool,
        id: u64, // callback id
        fid: String, // function id
    ) {
        if (allow_failure) {
            cosmos::stargate_with_options(account, data, cosmos::allow_failure_with_callback(id, fid));
        } else {
            cosmos::stargate_with_options(account, data, cosmos::disallow_failure_with_callback(id, fid));
        }
    }

    #[event]
    struct ResultEvent has drop {
        id: u64,
        success: bool,
    }

    #[event]
    struct ResultEventWithSigner has drop {
        account: address,
        id: u64,
        success: bool,
    }

    public entry fun callback_with_signer(
        account: &signer,
        id: u64,
        success: bool,
    ) {
        event::emit(ResultEventWithSigner { account: signer::address_of(account), id, success });
        
    }

    public entry fun callback_without_signer(
        id: u64,
        success: bool,
    ) {
        event::emit(ResultEvent { id, success });
    }
}