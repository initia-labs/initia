# 🧩 Account Abstraction in Initia

Initia introduces **Account Abstraction (AA)** by adapting [Aptos's Account Abstraction model](https://aptos.dev/en/build/sdks/ts-sdk/account/account-abstraction) to the **Cosmos SDK** ecosystem, powered by the **MoveVM**.

Account Abstraction enables powerful custom logic for transaction authentication, offering flexibility far beyond traditional key-based schemes.

For a deeper conceptual understanding, refer to Aptos's guides:

* [Account Abstraction V1](https://aptos.dev/en/build/sdks/ts-sdk/account/account-abstraction)
* [Derivable Account Abstraction](https://aptos.dev/en/build/sdks/ts-sdk/account/derivable-account-abstraction)

## 🔍 Supported Modes

Initia supports the following AA modes:

* [`V1`](./v1_account_abstraction.md): Register custom authenticator functions
* [`Derivable V1`](./v1_derivable_account_abstraction.md): Deterministic addresses derived from abstract public keys

## Features

* Define custom authentication logic using Move smart contracts
* Support various signature schemes beyond traditional cryptographic keys
* Implement complex multi-signature or threshold-based authentication
* Create deterministic account addresses using abstract public keys (DAA)
