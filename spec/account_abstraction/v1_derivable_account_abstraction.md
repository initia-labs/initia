# Derivable Account Abstraction

[Derivable Account Abstraction (DAA)](https://github.com/aptos-foundation/AIPs/blob/main/aips/aip-113.md) is a standard for account abstraction that enables custom authentication schemes by registering a `derivable_authentication_function`.

DAA differs from vanilla [Account Abstraction (AA)](./v1_account_abstraction.md) in that, for a given `derivable_authentication_function`, it defines how to deterministically derive the account address from an `abstract_public_key`, which can be done off-chain.

In contrast, vanilla AA is enabled for a specific pre-existing account by explicitly registering an on-chain `authentication_function` and submitting a transaction, which involves extra steps and costs gas for each account.

This allows registering secondary authentication schemes with identical user experience to the native ones. More specifically, this provides a flexible and secure way to manage cross-chain signatures.

## Abstract Public Key

The abstract public key is a custom data structure that represents the authentication method:

- Ethereum Abstract PubKey: `|bcs(ethereum_addr_bytes)|bcs(domain_string)|`
- Solana Abstract PubKey: `|bcs(solana_base58_pubkey_bytes)|bcs(domain_string)|`

## Abstract Signature

### Ethereum Abstract Signature

1. Construct message which is following SIWE format.

   ```go
   func constructMessage(
     ethereumAddress string,
     domain string,
     digestHex string,
     issuedAt string,
     scheme string,
     chainId string,
   ) ([]byte, error) {
     message := fmt.Sprintf(`
   <domain> wants you to sign in with your Ethereum account:
   <ethereum_address>
   
   Please confirm you explicitly initiated this request from <domain>. You are approving to execute transaction on Initia blockchain (<chain_id>).
   
   URI: <scheme>://<domain>
   Version: 1
   Chain ID: <chain_id>
   Nonce: <digest>
   Issued At: <issued_at>`, domain, ethereumAddress, domain, chain_id, scheme, domain, chainId, digestHex, issuedAt)
     msgLen := len(message)
   
     prefix := []byte("\x19Ethereum Signed Message:\n")
     msgLenBytes := []byte(strconv.Itoa(msgLen))
   
     var fullMessage []byte
     fullMessage = append(fullMessage, prefix...)
     fullMessage = append(fullMessage, msgLenBytes...)
     fullMessage = append(fullMessage, []byte(message)...)
   
     return fullMessage, nil
   }
   ```

2. Sign constructed message with Ethereum PrivKey in EIP191.
3. Construct abstract signature

   ```move
   enum SIWEAbstractSignature has drop {
     /// Deprecated, use MessageV2 instead
     MessageV1 {
       /// The date and time when the signature was issued
       issued_at: String,
       /// The signature of the message
       signature: vector<u8>
     },
     MessageV2 {
       /// The scheme in the URI of the message, e.g. the scheme of the website that requested the signature (http, https, etc.)
       scheme: String,
       /// The date and time when the signature was issued
       issued_at: String,
       /// The signature of the message
       signature: vector<u8>
     }
   }
   ```

   ```go
   func createAbstractSignature(
     scheme string,
     issuedAt string,
     signature []byte,
   ) ([]byte, error) {
     var res []byte
    
     // bcs encoding
     encodedType, err := vmtypes.SerializeUint8(0x01)
     if err != nil {
       return nil, err
     }
   
     // bcs encoding
     encodedScheme, err := vmtypes.SerializeString(scheme)
     if err != nil {
       return nil, err
     }
   
     // bcs encoding
     encodedIssuedAt, err := vmtypes.SerializeString(issuedAt)
     if err != nil {
       return nil, err
     }
    
     // bcs encoding
     encodedSignature, err := vmtypes.SerializeBytes(signature)
     if err != nil {
       return nil, err
     }
   
     res = append(res, encodedType...) // MessageV2 type
     res = append(res, encodedScheme...)
     res = append(res, encodedIssuedAt...)
     res = append(res, encodedSignature...)
     return res, nil
   }
   ```

### Solana Abstract Signature

1. Construct message which is following SIWS format.

   ```go
   func constructMessage(
     base58PubKey string,
     domain string,
     digestHex string,
     chainId string,
   ) []byte {
     return []byte(fmt.Sprintf(`
   <domain> wants you to sign in with your Solana account:
   <base58_pubkey>
   
   Please confirm you explicitly initiated this request from <domain>. You are approving to execute transaction on Initia blockchain (<chain_id>).
   
   Nonce: <digest>`, domain, base58PubKey, domain, chainId, digestHex))
   }
   ```

2. Sign constructed message with Solana PrivKey.
3. Construct abstract signature

   ```move
   enum SIWSAbstractSignature has drop {
     MessageV1 {
       signature: vector<u8>
     }
   }
   ```

   ```go
   func createAbstractSignature(
     signature []byte,
   ) ([]byte, error) {
     var res []byte

     // bcs encoding
     encodedType, err := vmtypes.SerializeUint8(0x00)
     if err != nil {
       return nil, err
     }

     // bcs encoding
     encodedSignature, err := vmtypes.SerializeBytes(signature)
     if err != nil {
       return nil, err
     }

     res = append(res, encodedType...)
     res = append(res, encodedSignature...)
     return res, nil
   }
   ```

## Constructing DAA Signature

For Derivable Account Abstraction, the signature structure is:

```json
{
  "function_info": {
    "module_address": "0x1",
    "module_name": "ethereum_derivable_account",
    "function_name": "authenticate"
  },
  "auth_data": {
    "derivable_v1": {
      "signing_message_digest": "c2lnbmluZ19tZXNzYWdlX2RpZ2VzdF92MQ==",
      "abstract_signature": "YXV0aGVudGljYXRvcl92MQ==",
      "abstract_public_key": "YXV0aGVudGljYXRvcl92MQ=="
    }
  }
}
```

**Field Descriptions:**

- `signing_message_digest`: SHA3-256 hash of the signBytes with direct sign mode
- `abstract_signature`: abstract signature
- `abstract_public_key`: abstract public key

Encode the JSON string to base64 and place it into the `signature` field.

### Public Key

To use derivable account abstraction, you must use the `/initia.crypto.v1.derivable.PubKey` type defined in [proto/initia/crypto/v1/derivable/keys.proto](../../proto/initia/crypto/v1/derivable/keys.proto). This public key type contains:

- `module_address`: The address where the authentication module is deployed
- `module_name`: The name of the module containing the authentication function  
- `function_name`: The name of the authentication function
- `abstract_public_key`: The abstract public key bytes, typically containing the base58-encoded public key and domain

The address from the pubkey is derived with following way

```go
type PubKey struct {
  ModuleAddress string `protobuf:"bytes,1,opt,name=module_address, json=moduleAddress,proto3" json:"module_address,omitempty"`
  ModuleName    string `protobuf:"bytes,2,opt,name=module_name,json=moduleName,proto3" json:"module_name,omitempty"`
  FunctionName  string `protobuf:"bytes,3,opt,name=function_name,json=functionName,proto3" json:"function_name,omitempty"`
 // normally |pubkey|domain|
  AbstractPublicKey []byte `protobuf:"bytes,4,opt,name=abstract_public_key,json=abstractPublicKey,proto3" json:"abstract_public_key,omitempty"`
}

// Address returns the address of the derived public key.
// This function implementation should align with `0x1::account_abstraction::derive_account_address` in MoveVM.
//
// Format:
// sha3_256(
//
// bcs(module_address),
// bcs(module_name),
// bcs(function_name),
// bcs(abstract_public_key),
// DERIVABLE_ABSTRACTION_DERIVED_SCHEME
//
// )
func (pubKey PubKey) Address() crypto.Address {
  bytes := pubKey.Bytes()

  hasher := sha3.New256()
  hasher.Write(bytes)
  hash := hasher.Sum(nil)

  return crypto.Address(hash)
}

// Bytes returns the bytes of the derived public key.
//
// Format:
// bcs(module_address) | bcs(module_name) | bcs(function_name) | bcs(abstract_public_key) | DERIVABLE_ABSTRACTION_DERIVED_SCHEME
func (pubKey PubKey) Bytes() []byte {
  fInfo, err := vmtypes.NewFunctionInfo(pubKey.ModuleAddress, pubKey.ModuleName, pubKey.FunctionName)
  if err != nil {
    panic(fmt.Sprintf("failed to create function info: %v", err))
  }

  bytes, err := fInfo.BcsSerialize()
  if err != nil {
    panic(fmt.Sprintf("failed to serialize function info: %v", err))
  }

  pubkeyBytes, err := vmtypes.SerializeBytes(pubKey.AbstractPublicKey)
  if err != nil {
    panic(fmt.Sprintf("failed to serialize abstract public key: %v", err))
  }

  const DERIVABLE_ABSTRACTION_DERIVED_SCHEME = byte(0x5)
  bytes = append(append(bytes, pubkeyBytes...),  DERIVABLE_ABSTRACTION_DERIVED_SCHEME)

  return bytes
}
```

### initia.js example

```ts
function createSignature(signDoc: SignDoc): SignatureV2 {
  const authData = createAuthData(Buffer.from(signDoc.toBytes()));
  return new SignatureV2(
    this.publicKey, // derivable PubKey
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
  const message = constructMessage(...)
  const signature = pk.sign(message)
  const abstractSignature = constructAbstractSignature(signature)
  return JSON.stringify({
    function_info: {
      module_address: "0xcafe",
      module_name: "custom_authenticator",
      function_name: "authenticate"
    },
    auth_data: {
      derivable_v1: {
        signing_message_digest: sha3_256(signBody).toString('base64'),
        abstract_signature: abstractSignature.toBuffer().toString('base64')
        abstract_public_key: pubkey.toBuffer().toString('base64')
      }
    }
  });
}
```
