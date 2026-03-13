# Protobufs

This is the public protocol buffers API for the [Initia](https://github.com/initia-labs/initia).

## npm Package

TypeScript definitions are published to npm as [`@initia/initia-proto`](https://www.npmjs.com/package/@initia/initia-proto) on every tagged release (`v*`).

### Installation

```bash
npm install @initia/initia-proto @bufbuild/protobuf
```

### Usage

```typescript
import { MsgExecuteSchema } from "@initia/initia-proto/initia/move/v1/tx_pb";
import { MsgDelegateSchema } from "@initia/initia-proto/initia/mstaking/v1/tx_pb";
```

The package requires `@bufbuild/protobuf` v2 as a peer dependency.
