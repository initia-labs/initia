Release Notes
==================

Initia version 0.45.12 is now available from:

  <https://github.com/initia-labs/initia>

This release includes new features, various bug fixes and performance
improvements, as well as updated translations.

Please report bugs using the issue tracker at GitHub:

  <https://github.com/initia-labs/initia-labs/issues>

To receive security and update notifications, please join our discord channel. 


What's Changed 
==============
### Initia
- upgrade to cosmos-sdk v0.45.12 by @zkst in #102

### RPC and other APIs

- #25220 rpc: fix incorrect warning for address type p2sh-segwit in createmultisig
- #25237 rpc: Capture UniValue by ref for rpcdoccheck
- #25983 Prevent data race for pathHandlers
- #26275 Fix crash on deriveaddresses when index is 2147483647 (2^31-1)

### Build system

- #25201 windeploy: Renewed windows code signing certificate
- #25788 guix: patch NSIS to remove .reloc sections from installer stubs
- #25861 guix: use --build={arch}-guix-linux-gnu in cross toolchain
- #25985 Revert "build: Use Homebrew's sqlite package if it is available"

### GUI

- gui#631 Disallow encryption of watchonly wallets
- gui#680 Fixes MacOS 13 segfault by preventing certain notifications

### Tests

- #24454 tests: Fix calculation of external input weights

### Miscellaneous

- #26321 Adjust .tx/config for new Transifex CLI

Full Changelog: Initia-node-v1.0.0...Initia-node-v1.0.1 [link]