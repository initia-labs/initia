# Interchaintest

`Interchaintest` orchestrates Go tests that utilize Docker containers for multiple IBC-compatible blockchains.

It allows users to quickly spin up custom testnets and dev environments to test IBC, chain infrastructures, smart contracts, etc.

- Built-in suite of conformance tests to test high-level IBC compatibility between chain sets
- Easily construct customized tests in highly configurable environments
- Deployable as CI tests in production workflows

## Test Case
- [Initial Chain Start](./chain_start_test.go)
- [IBC Transfer between Gaia <> Initia](./ibc_transfer_test.go)
  <br/>
  ...

## Quick Start
Make sure you have Docker installed. For testing in local machine you need 2 steps:

1. Build a debug image with your code change
```bash
make docker-build-debug
```
2. Run Test-case you want to test. Example:
```bash
make ictest-ibc-transfer
```