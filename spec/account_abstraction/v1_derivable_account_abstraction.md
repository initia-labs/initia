# Derivable Account Abstraction

Derivable Account Abstraction (DAA) allows creating deterministic account addresses from abstract public keys.

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
   ) ([]byte, error) {
       message := fmt.Sprintf(`
   <domain> wants you to sign in with your Solana account:
   <base58_pubkey>
   
   Please confirm you explicitly initiated this request from <domain>. You are approving to execute transaction on Initia blockchain (<chain_id>).
   
   Nonce: <digest>`, domain, base58PubKey, domain, chainId, digestHex)
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

### initia.js example

```ts
function createSignature(signDoc: SignDoc): SignatureV2 {
  const authData = createAuthData(Buffer.from(signDoc.toBytes()));
  return new SignatureV2(
    this.publicKey,
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
