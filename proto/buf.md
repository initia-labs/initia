# Protobufs

This is the public protocol buffers API for the [Initia](https://github.com/initia-labs/initia).

## npm Package

TypeScript definitions are published to npm as [`@initia/initia-proto`](https://www.npmjs.com/package/@initia/initia-proto).

- **Tagged releases** (`v*`) are published as `latest` (e.g. `1.0.0`).
- **Main branch** pushes are published as `canary` (e.g. `0.0.0-canary.<short-sha>`).

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
