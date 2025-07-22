module 0xcafe::public_key_authenticator {
    use std::signer;
    use initia_std::table::{Self, Table};
    use initia_std::ed25519;
    use initia_std::bcs_stream::{Self, deserialize_u8};
    use initia_std::auth_data::{Self, AbstractionAuthData};

    // ====== Error Codes ====== //

    const EINVALID_PUBLIC_KEY: u64 = 0x20000;
    const EPUBLIC_KEY_NOT_PERMITTED: u64 = 0x20001;
    const EENTRY_ALREADY_EXISTS: u64 = 0x20002;
    const ENO_PERMISSIONS: u64 = 0x20003;
    const EINVALID_SIGNATURE: u64 = 0x20004;

    // ====== Data Structures ====== //

    struct PublicKeyPermissions has key {
        public_key_table: Table<vector<u8>, bool>
    }

    // ====== Authenticator ====== //

    public fun authenticate(
        account: signer, auth_data: AbstractionAuthData
    ): signer acquires PublicKeyPermissions {
        let account_addr = signer::address_of(&account);
        assert!(exists<PublicKeyPermissions>(account_addr), ENO_PERMISSIONS);
        let permissions = borrow_global<PublicKeyPermissions>(account_addr);

        // Extract the public key and signature from the authenticator
        let authenticator = *auth_data::authenticator(&auth_data);
        let stream = bcs_stream::new(authenticator);
        let public_key =
            ed25519::public_key_from_bytes(
                bcs_stream::deserialize_vector<u8>(&mut stream, |x| deserialize_u8(x))
            );
        let signature =
            ed25519::signature_from_bytes(
                bcs_stream::deserialize_vector<u8>(&mut stream, |x| deserialize_u8(x))
            );

        // Check if the public key is permitted
        assert!(
            table::contains(
                &permissions.public_key_table,
                ed25519::public_key_to_bytes(&public_key)
            ),
            EPUBLIC_KEY_NOT_PERMITTED
        );

        // Verify the signature
        let digest = *auth_data::digest(&auth_data);
        assert!(
            ed25519::verify(digest, &public_key, &signature),
            EINVALID_SIGNATURE
        );

        account
    }

    // ====== Core Functionality ====== //

    public entry fun permit_public_key(
        signer: &signer, public_key: vector<u8>
    ) acquires PublicKeyPermissions {
        let account_addr = signer::address_of(signer);
        assert!(std::vector::length(&public_key) == 32, EINVALID_PUBLIC_KEY);

        if (!exists<PublicKeyPermissions>(account_addr)) {
            move_to(signer, PublicKeyPermissions { public_key_table: table::new() });
        };

        let permissions = borrow_global_mut<PublicKeyPermissions>(account_addr);
        assert!(
            !table::contains(&permissions.public_key_table, public_key),
            EENTRY_ALREADY_EXISTS
        );

        table::add(&mut permissions.public_key_table, public_key, true);
    }

    public entry fun revoke_public_key(
        signer: &signer, public_key: vector<u8>
    ) acquires PublicKeyPermissions {
        let account_addr = signer::address_of(signer);

        assert!(exists<PublicKeyPermissions>(account_addr), ENO_PERMISSIONS);

        let permissions = borrow_global_mut<PublicKeyPermissions>(account_addr);
        table::remove(&mut permissions.public_key_table, public_key);
    }
}
