# Account Abstraction V1

## Prerequisites

Before using account abstraction, you need to:

1. **Deploy an authentication module** containing your custom `authenticate` function
2. **Register the authentication function** for your target account
3. **Set up the necessary permissions** (e.g., whitelist public keys)

## Register Authentication Function

To use the AA feature, a user must register their authentication function via `0x1::account_abstraction::add_authentication_function` for the target account.

```move
entry fun add_authentication_function(
    account: &signer,
    module_address: address,
    module_name: String,
    function_name: String
)
```

### Parameters

* `account`: The signer representing the account that will register this authentication function for signature verification
* `module_address`: The address where the authentication module is deployed
* `module_name`: The name of the module containing the authentication function
* `function_name`: The name of the authentication function

## Authentication Function Interface

The registered function must match this interface:

```move
fun authenticate(account: signer, signing_data: AbstractionAuthData): signer
```

### AbstractionAuthData

```move
enum AbstractionAuthData has copy, drop {
    V1 {
        digest: vector<u8>,
        authenticator: vector<u8>
    },
    DerivableV1 {
        digest: vector<u8>,
        abstract_signature: vector<u8>,
        abstract_public_key: vector<u8>
    }
}
```

## Signature Format

```json
{
  "function_info": {
    "module_address": "0xcafe",
    "module_name": "custom_authenticator",
    "function_name": "authenticate"
  },
  "auth_data": {
    "v1": {
      "signing_message_digest": "base64_digest",
      "authenticator": "base64_custom_auth_data"
    }
  }
}
```

* `signing_message_digest`: SHA3-256 hash of the signBytes with direct sign mode
* `authenticator`: Custom encoded authentication data

Encode the JSON string to base64 and place it into the `signature` field.

### Public Key

When building a transaction body, use the normal public key of the account that has enabled account abstraction. You do not need to use any special public key format.

### initia.js example

```ts
function createSignature(signDoc: SignDoc): SignatureV2 {
  const authData = createAuthData(Buffer.from(signDoc.toBytes()));
  return new SignatureV2(
    this.publicKey, // normal account PubKey
    new SignatureV2.Descriptor(
      new SignatureV2.Descriptor.Single(
        SignMode.SIGN_MODE_ACCOUNT_ABSTRACTION,
        Buffer.from(authData, 'utf-8').toString('base64')
      )
    ),
    signDoc.sequence
  );
}

function createAuthData(signBody: Buffer): string {
  return JSON.stringify({
    function_info: {
      module_address: "0xcafe",
      module_name: "custom_authenticator",
      function_name: "authenticate"
    },
    auth_data: {
      v1: {
        signing_message_digest: sha3_256(signBody).toString('base64'),
        authenticator: Buffer.from("hello world", "utf-8").toString('base64')
      }
    }
  });
}
```
