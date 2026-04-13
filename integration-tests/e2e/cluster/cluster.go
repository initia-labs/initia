package cluster

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"testing"
	"time"
)

const (
	MaxNodeCount    = 10
	defaultBasePort = 26000
	defaultStride   = 20
)

type ClusterOptions struct {
	NodeCount      int
	AccountCount   int
	ChainID        string
	BasePort       int
	PortStride     int
	BinaryPath     string
	MemIAVL        bool
	TimeoutCommit  time.Duration // 0 = use default (2s)
	ValidatorCount int           // 0 or 1 = single validator (current behavior)
	MaxBlockGas    int64         // 0 = no limit; >0 sets block max_gas in genesis (mainnet: 200000000)
	NoAllowQueued  bool          // true = omit --allow-queued flag (for pre-proxy baseline binaries)
}

type Node struct {
	Index   int
	Name    string
	Home    string
	Ports   NodePorts
	PeerID  string
	LogPath string

	cmd     *exec.Cmd
	logFile *os.File
}

type AccountMeta struct {
	Address       string
	AccountNumber uint64
	Sequence      uint64
}

type TxResult struct {
	Code   int64
	TxHash string
	RawLog string
	Err    error
}

type Cluster struct {
	t     *testing.T
	opts  ClusterOptions
	bin   string
	repo  string
	root  string
	nodes []*Node

	valAddresses []string
	accounts     map[string]string

	mu     sync.Mutex
	closed bool
}

func NewCluster(ctx context.Context, t *testing.T, opts ClusterOptions) (*Cluster, error) {
	t.Helper()

	if opts.NodeCount < 1 || opts.NodeCount > MaxNodeCount {
		return nil, fmt.Errorf("node count must be 1..%d, got %d", MaxNodeCount, opts.NodeCount)
	}
	if opts.AccountCount < 1 {
		opts.AccountCount = 3
	}
	if opts.ChainID == "" {
		opts.ChainID = "testnet"
	}
	if opts.BasePort == 0 {
		opts.BasePort = defaultBasePort
	}
	if opts.PortStride == 0 {
		opts.PortStride = defaultStride
	}

	repoRoot, err := findRepoRoot()
	if err != nil {
		return nil, err
	}

	binPath := opts.BinaryPath
	if binPath == "" {
		binPath = filepath.Join(t.TempDir(), "initiad")
		if err := buildInitiad(ctx, repoRoot, binPath); err != nil {
			return nil, err
		}
	}

	c := &Cluster{
		t:        t,
		opts:     opts,
		bin:      binPath,
		repo:     repoRoot,
		root:     t.TempDir(),
		nodes:    make([]*Node, 0, opts.NodeCount),
		accounts: map[string]string{},
	}

	if err := c.initNodes(ctx); err != nil {
		return nil, err
	}
	if err := c.configureNodes(ctx); err != nil {
		return nil, err
	}
	return c, nil
}

func (c *Cluster) Start(ctx context.Context) error {
	for _, n := range c.nodes {
		if err := c.startNode(ctx, n); err != nil {
			c.Close()
			return err
		}
	}
	return nil
}

func (c *Cluster) Logf(format string, args ...any) {
	if c.t != nil {
		c.t.Logf(format, args...)
	}
}

func (c *Cluster) Close() {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.closed {
		return
	}
	c.closed = true

	for _, n := range c.nodes {
		if n.cmd == nil || n.cmd.Process == nil {
			if n.logFile != nil {
				_ = n.logFile.Close()
			}
			continue
		}

		_ = n.cmd.Process.Signal(syscall.SIGTERM)
	}

	deadline := time.Now().Add(10 * time.Second)
	for _, n := range c.nodes {
		if n.cmd == nil {
			if n.logFile != nil {
				_ = n.logFile.Close()
			}
			continue
		}

		done := make(chan error, 1)
		go func(cmd *exec.Cmd) {
			done <- cmd.Wait()
		}(n.cmd)

		select {
		case <-done:
		case <-time.After(time.Until(deadline)):
			if n.cmd.Process != nil {
				_ = n.cmd.Process.Kill()
				<-done
			}
		}

		if n.logFile != nil {
			_ = n.logFile.Close()
		}
	}
}

func (c *Cluster) WaitForReady(ctx context.Context, timeout time.Duration) error {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	for {
		select {
		case <-ctx.Done():
			return fmt.Errorf("wait for network ready timed out: %w", ctx.Err())
		default:
		}

		allHealthy := true
		for _, n := range c.nodes {
			h, _, err := c.nodeStatus(ctx, n)
			if err != nil || !h {
				allHealthy = false
				break
			}
		}
		if !allHealthy {
			time.Sleep(800 * time.Millisecond)
			continue
		}

		h1, err := c.latestHeight(ctx, 0)
		if err != nil {
			time.Sleep(800 * time.Millisecond)
			continue
		}
		time.Sleep(2 * time.Second)
		h2, err := c.latestHeight(ctx, 0)
		if err != nil {
			time.Sleep(800 * time.Millisecond)
			continue
		}
		if h2 > h1 && h2 > 1 {
			return nil
		}
	}
}

func (c *Cluster) WaitForMempoolEmpty(ctx context.Context, timeout time.Duration) error {
	return c.waitForMempoolEmpty(ctx, timeout, len(c.nodes))
}

// WaitForValidatorMempoolEmpty waits until the mempool is empty on validator
// nodes only. CList mempool can leave txs stranded on non-validator nodes
// that never get cleared, so this avoids false timeout failures.
func (c *Cluster) WaitForValidatorMempoolEmpty(ctx context.Context, timeout time.Duration) error {
	valCount := min(max(1, c.opts.ValidatorCount), len(c.nodes))
	return c.waitForMempoolEmpty(ctx, timeout, valCount)
}

func (c *Cluster) waitForMempoolEmpty(ctx context.Context, timeout time.Duration, nodeCount int) error {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	for {
		select {
		case <-ctx.Done():
			return fmt.Errorf("wait for mempool empty timed out: %w", ctx.Err())
		default:
		}

		allEmpty := true
		for i := range nodeCount {
			n, err := c.unconfirmedTxCount(ctx, i)
			if err != nil || n != 0 {
				allEmpty = false
				break
			}
		}
		if allEmpty {
			return nil
		}
		time.Sleep(500 * time.Millisecond)
	}
}

func (c *Cluster) NodeCount() int {
	return len(c.nodes)
}

// NodeRPCPort returns the RPC port for the given node index.
func (c *Cluster) NodeRPCPort(index int) (int, error) {
	n, err := c.getNode(index)
	if err != nil {
		return 0, err
	}
	return n.Ports.RPC, nil
}

// LatestHeight returns the latest block height from the given node.
func (c *Cluster) LatestHeight(ctx context.Context, nodeIndex int) (int64, error) {
	return c.latestHeight(ctx, nodeIndex)
}

// UnconfirmedTxCount returns the number of unconfirmed transactions in the given node's mempool.
func (c *Cluster) UnconfirmedTxCount(ctx context.Context, nodeIndex int) (int64, error) {
	return c.unconfirmedTxCount(ctx, nodeIndex)
}

// BlockResult holds the data extracted from a block query.
type BlockResult struct {
	TxHashes  []string
	BlockTime time.Time
}

// QueryBlock queries a specific block by height from the given node and returns tx hashes and block time.
func (c *Cluster) QueryBlock(ctx context.Context, nodeIndex int, height int64) (BlockResult, error) {
	n, err := c.getNode(nodeIndex)
	if err != nil {
		return BlockResult{}, err
	}

	url := fmt.Sprintf("http://127.0.0.1:%d/block?height=%d", n.Ports.RPC, height)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return BlockResult{}, err
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return BlockResult{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return BlockResult{}, fmt.Errorf("block query status code %d", resp.StatusCode)
	}

	var decoded struct {
		Result struct {
			Block struct {
				Header struct {
					Time string `json:"time"`
				} `json:"header"`
				Data struct {
					Txs []string `json:"txs"`
				} `json:"data"`
			} `json:"block"`
		} `json:"result"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&decoded); err != nil {
		return BlockResult{}, fmt.Errorf("failed to decode block response: %w", err)
	}

	blockTime, err := time.Parse(time.RFC3339Nano, decoded.Result.Block.Header.Time)
	if err != nil {
		return BlockResult{}, fmt.Errorf("failed to parse block time %q: %w", decoded.Result.Block.Header.Time, err)
	}

	txHashes := make([]string, 0, len(decoded.Result.Block.Data.Txs))
	for idx, txBase64 := range decoded.Result.Block.Data.Txs {
		txBytes, decErr := base64Decode(txBase64)
		if decErr != nil {
			return BlockResult{}, fmt.Errorf("failed to decode tx at height=%d index=%d: %w", height, idx, decErr)
		}
		hash := sha256Hash(txBytes)
		txHashes = append(txHashes, strings.ToUpper(hash))
	}

	return BlockResult{
		TxHashes:  txHashes,
		BlockTime: blockTime,
	}, nil
}

func (c *Cluster) AccountNames() []string {
	names := make([]string, 0, len(c.accounts))
	for i := 1; i <= c.opts.AccountCount; i++ {
		names = append(names, fmt.Sprintf("acc%d", i))
	}
	return names
}

func (c *Cluster) ValidatorAddress() string {
	if len(c.valAddresses) == 0 {
		return ""
	}
	return c.valAddresses[0]
}

func (c *Cluster) ValidatorAddresses() []string {
	return c.valAddresses
}

func (c *Cluster) AccountAddress(name string) (string, error) {
	addr, ok := c.accounts[name]
	if !ok {
		return "", fmt.Errorf("unknown account: %s", name)
	}
	return addr, nil
}

// AccountAddressHex returns the hex-encoded address for the given account name.
// Useful for Move named-addresses which require hex format (0x...).
func (c *Cluster) AccountAddressHex(ctx context.Context, name string) (string, error) {
	addr, err := c.AccountAddress(name)
	if err != nil {
		return "", err
	}
	out, err := c.exec(ctx, "debug", "addr", addr)
	if err != nil {
		return "", err
	}
	// parse "Address (hex): <HEX>" line
	for line := range strings.SplitSeq(string(out), "\n") {
		line = strings.TrimSpace(line)
		if after, ok := strings.CutPrefix(line, "Address (hex):"); ok {
			hexAddr := strings.TrimSpace(after)
			return "0x" + strings.ToLower(hexAddr), nil
		}
	}
	return "", fmt.Errorf("failed to parse hex address from debug output: %s", string(out))
}

func (c *Cluster) RepoPath(parts ...string) string {
	all := make([]string, 0, len(parts)+1)
	all = append(all, c.repo)
	all = append(all, parts...)
	return filepath.Join(all...)
}

func (c *Cluster) QueryAccountMeta(ctx context.Context, nodeIndex int, address string) (AccountMeta, error) {
	node, err := c.getNode(nodeIndex)
	if err != nil {
		return AccountMeta{}, err
	}

	url := fmt.Sprintf("http://127.0.0.1:%d/cosmos/auth/v1beta1/accounts/%s", node.Ports.API, address)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return AccountMeta{}, err
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return AccountMeta{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return AccountMeta{}, fmt.Errorf("account query status code %d", resp.StatusCode)
	}

	var decoded map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&decoded); err != nil {
		return AccountMeta{}, fmt.Errorf("failed to parse account query output: %w", err)
	}

	accountAny, ok := decoded["account"]
	if !ok {
		return AccountMeta{}, errors.New("missing account field")
	}

	accountNumber, ok := findUintField(accountAny, "account_number")
	if !ok {
		accountNumber, ok = findUintField(accountAny, "accountNumber")
	}
	if !ok {
		return AccountMeta{}, errors.New("account_number not found")
	}
	sequence, ok := findUintField(accountAny, "sequence")
	if !ok {
		return AccountMeta{}, errors.New("sequence not found")
	}

	return AccountMeta{
		Address:       address,
		AccountNumber: accountNumber,
		Sequence:      sequence,
	}, nil
}

func (c *Cluster) SendBankTxWithSequence(ctx context.Context, fromName, toAddress, amount string, accountNumber, sequence, gasLimit uint64, viaNode int) TxResult {
	node, err := c.getNode(viaNode)
	if err != nil {
		return TxResult{Err: err}
	}
	c.t.Logf(
		"[send] from=%s to=%s amount=%s account_number=%d sequence=%d via_node=%d rpc_port=%d",
		fromName, toAddress, amount, accountNumber, sequence, viaNode, node.Ports.RPC,
	)

	args := []string{
		"tx", "bank", "send", fromName, toAddress, amount,
		"--chain-id", c.opts.ChainID,
		"--node", fmt.Sprintf("http://127.0.0.1:%d", node.Ports.RPC),
		"--home", c.nodes[0].Home,
		"--keyring-backend", "test",
		"--gas-prices", "0.15uinit",
		"--gas", strconv.FormatUint(gasLimit, 10),
		"--offline",
	}
	if !c.opts.NoAllowQueued {
		args = append(args, "--allow-queued")
	}
	args = append(args,
		"--broadcast-mode", "sync",
		"--account-number", strconv.FormatUint(accountNumber, 10),
		"--sequence", strconv.FormatUint(sequence, 10),
		"--yes",
		"--output", "json",
	)

	out, err := c.exec(ctx, args...)
	if err != nil {
		c.t.Logf(
			"[send] failed from=%s sequence=%d err=%v",
			fromName, sequence, err,
		)
		return TxResult{Err: err}
	}

	res, err := parseTxResultFromOutput(out)
	if err != nil {
		c.t.Logf("[send] parse-failed from=%s sequence=%d output=%s", fromName, sequence, strings.TrimSpace(string(out)))
		return TxResult{Err: err}
	}
	c.t.Logf(
		"[send] result from=%s sequence=%d code=%d txhash=%s raw_log=%q",
		fromName, sequence, res.Code, res.TxHash, res.RawLog,
	)
	return res
}

func (c *Cluster) MovePublish(ctx context.Context, fromName string, moduleFiles []string, viaNode int) TxResult {
	node, err := c.getNode(viaNode)
	if err != nil {
		return TxResult{Err: err}
	}
	fromAddr, err := c.AccountAddress(fromName)
	if err != nil {
		return TxResult{Err: err}
	}
	meta, err := c.QueryAccountMeta(ctx, viaNode, fromAddr)
	if err != nil {
		return TxResult{Err: err}
	}
	estimatedGas, err := c.MoveEstimatePublishGas(ctx, fromName, moduleFiles, meta.AccountNumber, meta.Sequence, viaNode)
	if err != nil {
		return TxResult{Err: err}
	}

	args := []string{"tx", "move", "publish"}
	args = append(args, moduleFiles...)
	args = append(args,
		"--from", fromName,
		"--chain-id", c.opts.ChainID,
		"--node", fmt.Sprintf("http://127.0.0.1:%d", node.Ports.RPC),
		"--home", c.nodes[0].Home,
		"--keyring-backend", "test",
		"--gas-prices", "0.15uinit",
		"--gas", strconv.FormatUint(estimatedGas, 10),
		"--offline",
	)
	if !c.opts.NoAllowQueued {
		args = append(args, "--allow-queued")
	}
	args = append(args,
		"--account-number", strconv.FormatUint(meta.AccountNumber, 10),
		"--sequence", strconv.FormatUint(meta.Sequence, 10),
		"--broadcast-mode", "sync",
		"--yes",
		"--output", "json",
	)

	out, err := c.exec(ctx, args...)
	if err != nil {
		return TxResult{Err: err}
	}
	res, err := parseTxResultFromOutput(out)
	if err != nil {
		return TxResult{Err: err}
	}
	c.t.Logf("[move-publish] from=%s seq=%d gas=%d files=%v code=%d txhash=%s", fromName, meta.Sequence, estimatedGas, moduleFiles, res.Code, res.TxHash)
	return res
}

func (c *Cluster) MoveEstimatePublishGas(
	ctx context.Context,
	fromName string,
	moduleFiles []string,
	accountNumber, sequence uint64,
	viaNode int,
) (uint64, error) {
	_ = accountNumber
	_ = sequence

	node, err := c.getNode(viaNode)
	if err != nil {
		return 0, err
	}
	args := []string{"tx", "move", "publish"}
	args = append(args, moduleFiles...)
	args = append(args,
		"--from", fromName,
		"--chain-id", c.opts.ChainID,
		"--node", fmt.Sprintf("http://127.0.0.1:%d", node.Ports.RPC),
		"--home", c.nodes[0].Home,
		"--keyring-backend", "test",
		"--gas-prices", "0.15uinit",
		"--gas", "auto",
		"--gas-adjustment", "1.2",
		"--generate-only",
		"--yes",
		"--output", "json",
	)

	out, err := c.exec(ctx, args...)
	if err != nil {
		return 0, err
	}
	gas, err := parseEstimatedGas(out)
	if err != nil {
		return 0, err
	}
	gas += 200_000 // for fee payment
	c.t.Logf("[move-publish-estimate] from=%s seq=%d gas=%d", fromName, sequence, gas)
	return gas, nil
}

func (c *Cluster) MoveExecuteJSONWithSequence(
	ctx context.Context,
	fromName, moduleAddress, moduleName, functionName string,
	typeArgs, args []string,
	accountNumber, sequence uint64,
	viaNode int,
) TxResult {
	node, err := c.getNode(viaNode)
	if err != nil {
		return TxResult{Err: err}
	}
	estimatedGas, err := c.MoveEstimateExecuteJSONGasWithSequence(
		ctx,
		fromName,
		moduleAddress,
		moduleName,
		functionName,
		typeArgs,
		args,
		accountNumber,
		sequence,
		viaNode,
	)
	if err != nil {
		return TxResult{Err: err}
	}
	typeArgsJSON, err := json.Marshal(typeArgs)
	if err != nil {
		return TxResult{Err: err}
	}
	moveArgsJSON, err := json.Marshal(args)
	if err != nil {
		return TxResult{Err: err}
	}

	execArgs := []string{
		"tx", "move", "execute-json",
		moduleAddress,
		moduleName,
		functionName,
		"--type-args", string(typeArgsJSON),
		"--args", string(moveArgsJSON),
		"--from", fromName,
		"--chain-id", c.opts.ChainID,
		"--node", fmt.Sprintf("http://127.0.0.1:%d", node.Ports.RPC),
		"--home", c.nodes[0].Home,
		"--keyring-backend", "test",
		"--gas-prices", "0.15uinit",
		"--gas", strconv.FormatUint(estimatedGas, 10),
		"--offline",
	}
	if !c.opts.NoAllowQueued {
		execArgs = append(execArgs, "--allow-queued")
	}
	execArgs = append(execArgs,
		"--account-number", strconv.FormatUint(accountNumber, 10),
		"--sequence", strconv.FormatUint(sequence, 10),
		"--broadcast-mode", "sync",
		"--yes",
		"--output", "json",
	)

	out, err := c.exec(ctx, execArgs...)
	if err != nil {
		return TxResult{Err: err}
	}
	res, err := parseTxResultFromOutput(out)
	if err != nil {
		return TxResult{Err: err}
	}
	c.t.Logf("[move-exec-json-seq] from=%s seq=%d gas=%d %s::%s::%s args=%v code=%d txhash=%s", fromName, sequence, estimatedGas, moduleAddress, moduleName, functionName, args, res.Code, res.TxHash)
	return res
}

func (c *Cluster) SendMoveExecuteJSONWithGas(
	ctx context.Context,
	fromName, moduleAddress, moduleName, functionName string,
	typeArgs, args []string,
	accountNumber, sequence, gasLimit uint64,
	viaNode int,
) TxResult {
	node, err := c.getNode(viaNode)
	if err != nil {
		return TxResult{Err: err}
	}
	typeArgsJSON, err := json.Marshal(typeArgs)
	if err != nil {
		return TxResult{Err: err}
	}
	moveArgsJSON, err := json.Marshal(args)
	if err != nil {
		return TxResult{Err: err}
	}

	execArgs := []string{
		"tx", "move", "execute-json",
		moduleAddress,
		moduleName,
		functionName,
		"--type-args", string(typeArgsJSON),
		"--args", string(moveArgsJSON),
		"--from", fromName,
		"--chain-id", c.opts.ChainID,
		"--node", fmt.Sprintf("http://127.0.0.1:%d", node.Ports.RPC),
		"--home", c.nodes[0].Home,
		"--keyring-backend", "test",
		"--gas-prices", "0.15uinit",
		"--gas", strconv.FormatUint(gasLimit, 10),
		"--offline",
	}
	if !c.opts.NoAllowQueued {
		execArgs = append(execArgs, "--allow-queued")
	}
	execArgs = append(execArgs,
		"--account-number", strconv.FormatUint(accountNumber, 10),
		"--sequence", strconv.FormatUint(sequence, 10),
		"--broadcast-mode", "sync",
		"--yes",
		"--output", "json",
	)

	out, err := c.exec(ctx, execArgs...)
	if err != nil {
		return TxResult{Err: err}
	}
	res, err := parseTxResultFromOutput(out)
	if err != nil {
		return TxResult{Err: err}
	}
	c.t.Logf("[move-exec-json-gas] from=%s seq=%d gas=%d %s::%s::%s args=%v code=%d txhash=%s", fromName, sequence, gasLimit, moduleAddress, moduleName, functionName, args, res.Code, res.TxHash)
	return res
}

func (c *Cluster) MoveEstimateExecuteJSONGasWithSequence(
	ctx context.Context,
	fromName, moduleAddress, moduleName, functionName string,
	typeArgs, args []string,
	accountNumber, sequence uint64,
	viaNode int,
) (uint64, error) {
	_ = accountNumber

	node, err := c.getNode(viaNode)
	if err != nil {
		return 0, err
	}
	typeArgsJSON, err := json.Marshal(typeArgs)
	if err != nil {
		return 0, err
	}
	moveArgsJSON, err := json.Marshal(args)
	if err != nil {
		return 0, err
	}

	out, err := c.exec(ctx,
		"tx", "move", "execute-json",
		moduleAddress,
		moduleName,
		functionName,
		"--type-args", string(typeArgsJSON),
		"--args", string(moveArgsJSON),
		"--from", fromName,
		"--chain-id", c.opts.ChainID,
		"--node", fmt.Sprintf("http://127.0.0.1:%d", node.Ports.RPC),
		"--home", c.nodes[0].Home,
		"--keyring-backend", "test",
		"--gas-prices", "0.15uinit",
		"--gas", "auto",
		"--gas-adjustment", "1.2",
		"--generate-only",
		"--yes",
		"--output", "json",
	)
	if err != nil {
		return 0, err
	}
	gas, err := parseEstimatedGas(out)
	if err != nil {
		return 0, err
	}
	gas += 200_000 // for fee payment
	c.t.Logf("[move-exec-json-estimate] from=%s seq=%d %s::%s::%s gas=%d", fromName, sequence, moduleAddress, moduleName, functionName, gas)
	return gas, nil
}

func (c *Cluster) MoveQueryResources(ctx context.Context, owner string, viaNode int) ([]byte, error) {
	node, err := c.getNode(viaNode)
	if err != nil {
		return nil, err
	}
	return c.exec(ctx,
		"query", "move", "resources", owner,
		"--node", fmt.Sprintf("http://127.0.0.1:%d", node.Ports.RPC),
		"--output", "json",
	)
}

func (c *Cluster) MoveQueryViewJSON(ctx context.Context, moduleOwner, moduleName, functionName string, typeArgs, args []string, viaNode int) ([]byte, error) {
	node, err := c.getNode(viaNode)
	if err != nil {
		return nil, err
	}
	typeArgsJSON, err := json.Marshal(typeArgs)
	if err != nil {
		return nil, err
	}
	moveArgsJSON, err := json.Marshal(args)
	if err != nil {
		return nil, err
	}

	return c.exec(ctx,
		"query", "move", "view-json",
		moduleOwner,
		moduleName,
		functionName,
		"--type-args", string(typeArgsJSON),
		"--args", string(moveArgsJSON),
		"--node", fmt.Sprintf("http://127.0.0.1:%d", node.Ports.RPC),
		"--output", "json",
	)
}

func (c *Cluster) BuildMoveModule(
	ctx context.Context,
	packagePath string,
	moduleName string,
	namedAddresses map[string]string,
) (string, error) {
	installDir := filepath.Join(c.t.TempDir(), "move-build")
	args := []string{
		"move", "build",
		"--path", packagePath,
		"--install-dir", installDir,
		"--skip-fetch-latest-git-deps",
	}

	if len(namedAddresses) > 0 {
		pairs := make([]string, 0, len(namedAddresses))
		for name, addr := range namedAddresses {
			pairs = append(pairs, fmt.Sprintf("%s=%s", name, addr))
		}
		sort.Strings(pairs)
		args = append(args, "--named-addresses", strings.Join(pairs, ","))
	}

	if _, err := c.exec(ctx, args...); err != nil {
		return "", err
	}

	target := moduleName + ".mv"
	var found string
	err := filepath.WalkDir(installDir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		if d.Name() == target {
			found = path
			return filepath.SkipAll
		}
		return nil
	})
	if err != nil {
		return "", err
	}
	if found == "" {
		return "", fmt.Errorf("built module %s not found under %s", target, installDir)
	}
	return found, nil
}

func (c *Cluster) initNodes(ctx context.Context) error {
	for i := 0; i < c.opts.NodeCount; i++ {
		ports, err := allocatePorts(i, c.opts.BasePort, c.opts.PortStride)
		if err != nil {
			return err
		}
		n := &Node{
			Index: i,
			Name:  fmt.Sprintf("node%d", i),
			Home:  filepath.Join(c.root, fmt.Sprintf("node%d", i)),
			Ports: ports,
		}
		if _, err := c.exec(ctx, "init", n.Name, "--home", n.Home, "--chain-id", c.opts.ChainID); err != nil {
			return err
		}
		c.nodes = append(c.nodes, n)
	}

	// Resolve node IDs early — needed for gentx memos in multi-validator setup.
	for _, n := range c.nodes {
		out, err := c.exec(ctx, "comet", "show-node-id", "--home", n.Home)
		if err != nil {
			return err
		}
		n.PeerID = strings.TrimSpace(string(out))
	}

	baseHome := c.nodes[0].Home

	// Determine number of validators.
	valCount := 1
	if c.opts.ValidatorCount > 1 {
		valCount = min(c.opts.ValidatorCount, c.opts.NodeCount)
	}

	if valCount <= 1 {
		return c.initSingleValidator(ctx, baseHome)
	}
	return c.initMultiValidator(ctx, baseHome, valCount)
}

// initSingleValidator is the original single-validator setup path.
func (c *Cluster) initSingleValidator(ctx context.Context, baseHome string) error {
	if _, err := c.exec(ctx, "keys", "add", "val", "--keyring-backend", "test", "--home", baseHome); err != nil {
		return err
	}
	valAddr, err := c.keyAddress(ctx, "val")
	if err != nil {
		return err
	}
	c.valAddresses = []string{valAddr}

	if err := c.addAccountKeys(ctx, baseHome); err != nil {
		return err
	}

	if _, err := c.exec(ctx,
		"genesis", "add-genesis-account", "val", "1000000000000000uinit",
		"--home", baseHome, "--keyring-backend", "test",
	); err != nil {
		return err
	}

	if err := c.addAccountGenesisAccounts(ctx, baseHome); err != nil {
		return err
	}

	if _, err := c.exec(ctx,
		"genesis", "gentx", "val", "500000000000uinit",
		"--home", baseHome, "--keyring-backend", "test", "--chain-id", c.opts.ChainID,
	); err != nil {
		return err
	}
	if _, err := c.exec(ctx, "genesis", "collect-gentxs", "--home", baseHome); err != nil {
		return err
	}

	return c.distributeGenesis(baseHome)
}

// initMultiValidator sets up N validators across the first N nodes.
func (c *Cluster) initMultiValidator(ctx context.Context, baseHome string, valCount int) error {
	// 1. Create all validator keys in baseHome keyring.
	for i := range valCount {
		name := fmt.Sprintf("val%d", i)
		if _, err := c.exec(ctx, "keys", "add", name, "--keyring-backend", "test", "--home", baseHome); err != nil {
			return err
		}
		addr, err := c.keyAddress(ctx, name)
		if err != nil {
			return err
		}
		c.valAddresses = append(c.valAddresses, addr)
	}

	if err := c.addAccountKeys(ctx, baseHome); err != nil {
		return err
	}

	// 2. Add genesis accounts for all validators.
	for i := range valCount {
		name := fmt.Sprintf("val%d", i)
		if _, err := c.exec(ctx,
			"genesis", "add-genesis-account", name, "1000000000000000uinit",
			"--home", baseHome, "--keyring-backend", "test",
		); err != nil {
			return err
		}
	}

	if err := c.addAccountGenesisAccounts(ctx, baseHome); err != nil {
		return err
	}

	// 3. Create gentxs — each validator's gentx must be created from its own node home
	//    so the correct node ID ends up in the memo.
	baseKeyringDir := filepath.Join(baseHome, "keyring-test")
	baseGenesisPath := filepath.Join(baseHome, "config", "genesis.json")

	for i := range valCount {
		nodeHome := c.nodes[i].Home
		name := fmt.Sprintf("val%d", i)

		// Copy keyring from baseHome to this node's home so gentx can sign.
		nodeKeyringDir := filepath.Join(nodeHome, "keyring-test")
		if err := copyDir(baseKeyringDir, nodeKeyringDir); err != nil {
			return fmt.Errorf("copy keyring to node%d: %w", i, err)
		}

		// Copy genesis.json from baseHome to this node's home.
		if err := copyFile(baseGenesisPath, filepath.Join(nodeHome, "config", "genesis.json")); err != nil {
			return fmt.Errorf("copy genesis to node%d: %w", i, err)
		}

		// Create gentx from this node's home (embeds correct peer ID in memo).
		if _, err := c.exec(ctx,
			"genesis", "gentx", name, "500000000000uinit",
			"--home", nodeHome, "--keyring-backend", "test", "--chain-id", c.opts.ChainID,
		); err != nil {
			return fmt.Errorf("gentx for %s: %w", name, err)
		}

		// Copy the gentx file back to baseHome/config/gentx/.
		nodeGentxDir := filepath.Join(nodeHome, "config", "gentx")
		baseGentxDir := filepath.Join(baseHome, "config", "gentx")
		if i > 0 { // node0 == baseHome, no copy needed
			if err := copyDir(nodeGentxDir, baseGentxDir); err != nil {
				return fmt.Errorf("copy gentx from node%d: %w", i, err)
			}
		}
	}

	// 4. Collect gentxs in baseHome.
	if _, err := c.exec(ctx, "genesis", "collect-gentxs", "--home", baseHome); err != nil {
		return err
	}

	// 5. Distribute genesis and keyring to all nodes.
	if err := c.distributeGenesis(baseHome); err != nil {
		return err
	}

	// Copy keyring to all validator nodes so they can sign transactions.
	for i := 1; i < len(c.nodes); i++ {
		nodeKeyringDir := filepath.Join(c.nodes[i].Home, "keyring-test")
		if err := copyDir(baseKeyringDir, nodeKeyringDir); err != nil {
			return fmt.Errorf("distribute keyring to node%d: %w", i, err)
		}
	}

	return nil
}

// addAccountKeys creates test account keys in baseHome.
func (c *Cluster) addAccountKeys(ctx context.Context, baseHome string) error {
	for i := 1; i <= c.opts.AccountCount; i++ {
		name := fmt.Sprintf("acc%d", i)
		if _, err := c.exec(ctx, "keys", "add", name, "--keyring-backend", "test", "--home", baseHome); err != nil {
			return err
		}
		addr, err := c.keyAddress(ctx, name)
		if err != nil {
			return err
		}
		c.accounts[name] = addr
	}
	return nil
}

// addAccountGenesisAccounts adds genesis accounts for all test accounts.
func (c *Cluster) addAccountGenesisAccounts(ctx context.Context, baseHome string) error {
	for i := 1; i <= c.opts.AccountCount; i++ {
		name := fmt.Sprintf("acc%d", i)
		if _, err := c.exec(ctx,
			"genesis", "add-genesis-account", name, "1000000000000000uinit",
			"--home", baseHome, "--keyring-backend", "test",
		); err != nil {
			return err
		}
	}
	return nil
}

// patchGenesisBlockGas sets consensus_params.block.max_gas in genesis.json.
func patchGenesisBlockGas(genesisPath string, maxGas int64) error {
	bz, err := os.ReadFile(genesisPath)
	if err != nil {
		return err
	}
	var genesis map[string]any
	if err := json.Unmarshal(bz, &genesis); err != nil {
		return fmt.Errorf("unmarshal genesis: %w", err)
	}

	// Navigate to consensus_params.block (or consensus.params.block for newer SDK)
	// and set max_gas as a string (CometBFT convention).
	gasStr := strconv.FormatInt(maxGas, 10)

	patched := false
	if cp, ok := genesis["consensus"].(map[string]any); ok {
		if params, ok := cp["params"].(map[string]any); ok {
			if block, ok := params["block"].(map[string]any); ok {
				block["max_gas"] = gasStr
				patched = true
			}
		}
	}
	if cp, ok := genesis["consensus_params"].(map[string]any); ok {
		if block, ok := cp["block"].(map[string]any); ok {
			block["max_gas"] = gasStr
			patched = true
		}
	}
	if !patched {
		return fmt.Errorf("genesis has no consensus_params.block or consensus.params.block to patch max_gas")
	}

	out, err := json.MarshalIndent(genesis, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal genesis: %w", err)
	}
	return os.WriteFile(genesisPath, out, 0o600)
}

// distributeGenesis patches genesis if needed and copies it from baseHome to all other nodes.
func (c *Cluster) distributeGenesis(baseHome string) error {
	baseGenesis := filepath.Join(baseHome, "config", "genesis.json")

	if c.opts.MaxBlockGas > 0 {
		if err := patchGenesisBlockGas(baseGenesis, c.opts.MaxBlockGas); err != nil {
			return fmt.Errorf("patch genesis max_gas: %w", err)
		}
	}

	for i := 1; i < len(c.nodes); i++ {
		n := c.nodes[i]
		if err := copyFile(baseGenesis, filepath.Join(n.Home, "config", "genesis.json")); err != nil {
			return err
		}
	}
	return nil
}

func (c *Cluster) configureNodes(_ context.Context) error {
	for _, n := range c.nodes {
		cfgPath := filepath.Join(n.Home, "config", "config.toml")
		appPath := filepath.Join(n.Home, "config", "app.toml")

		if err := setTOMLValue(cfgPath, "rpc", "laddr", fmt.Sprintf("\"tcp://127.0.0.1:%d\"", n.Ports.RPC)); err != nil {
			return err
		}
		if err := setTOMLValue(cfgPath, "p2p", "laddr", fmt.Sprintf("\"tcp://127.0.0.1:%d\"", n.Ports.P2P)); err != nil {
			return err
		}
		if err := setTOMLValue(cfgPath, "p2p", "allow_duplicate_ip", "true"); err != nil {
			return err
		}
		if err := setTOMLValue(cfgPath, "p2p", "addr_book_strict", "false"); err != nil {
			return err
		}

		if err := setTOMLValue(appPath, "api", "enable", "true"); err != nil {
			return err
		}
		if err := setTOMLValue(appPath, "api", "swagger", "true"); err != nil {
			return err
		}
		if err := setTOMLValue(appPath, "api", "address", fmt.Sprintf("\"tcp://127.0.0.1:%d\"", n.Ports.API)); err != nil {
			return err
		}
		if err := setTOMLValue(appPath, "grpc", "address", fmt.Sprintf("\"127.0.0.1:%d\"", n.Ports.GRPC)); err != nil {
			return err
		}

		memiavlValue := "false"
		if c.opts.MemIAVL {
			memiavlValue = "true"
		}
		if err := setTOMLValue(appPath, "memiavl", "enable", memiavlValue); err != nil {
			return err
		}

		if c.opts.TimeoutCommit > 0 {
			commitMs := fmt.Sprintf("\"%dms\"", c.opts.TimeoutCommit.Milliseconds())
			if err := setTOMLValue(cfgPath, "consensus", "timeout_commit", commitMs); err != nil {
				return err
			}
		}
	}

	for _, n := range c.nodes {
		peers := make([]string, 0, len(c.nodes)-1)
		for _, other := range c.nodes {
			if other.Index == n.Index {
				continue
			}
			peers = append(peers, fmt.Sprintf("%s@127.0.0.1:%d", other.PeerID, other.Ports.P2P))
		}
		if len(peers) == 0 {
			continue
		}
		cfgPath := filepath.Join(n.Home, "config", "config.toml")
		if err := setTOMLValue(cfgPath, "p2p", "persistent_peers", fmt.Sprintf("\"%s\"", strings.Join(peers, ","))); err != nil {
			return err
		}
	}

	return nil
}

func (c *Cluster) startNode(ctx context.Context, n *Node) error {
	logPath := filepath.Join(n.Home, "node.log")
	f, err := os.Create(logPath)
	if err != nil {
		return err
	}

	//nolint:gosec // c.bin is a test-controlled binary path from ClusterOptions or local build output.
	cmd := exec.CommandContext(ctx, c.bin, "start", "--home", n.Home)
	cmd.Stdout = f
	cmd.Stderr = f
	if err := cmd.Start(); err != nil {
		_ = f.Close()
		return err
	}

	n.LogPath = logPath
	n.logFile = f
	n.cmd = cmd
	return nil
}

func (c *Cluster) nodeStatus(ctx context.Context, n *Node) (bool, int64, error) {
	url := fmt.Sprintf("http://127.0.0.1:%d/status", n.Ports.RPC)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return false, 0, err
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return false, 0, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return false, 0, fmt.Errorf("status code %d", resp.StatusCode)
	}

	var decoded struct {
		Result struct {
			SyncInfo struct {
				LatestBlockHeight string `json:"latest_block_height"`
				CatchingUp        bool   `json:"catching_up"`
			} `json:"sync_info"`
		} `json:"result"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&decoded); err != nil {
		return false, 0, err
	}

	h, err := strconv.ParseInt(decoded.Result.SyncInfo.LatestBlockHeight, 10, 64)
	if err != nil {
		return false, 0, err
	}
	return !decoded.Result.SyncInfo.CatchingUp, h, nil
}

func (c *Cluster) latestHeight(ctx context.Context, nodeIndex int) (int64, error) {
	n, err := c.getNode(nodeIndex)
	if err != nil {
		return 0, err
	}
	_, h, err := c.nodeStatus(ctx, n)
	return h, err
}

func (c *Cluster) unconfirmedTxCount(ctx context.Context, nodeIndex int) (int64, error) {
	n, err := c.getNode(nodeIndex)
	if err != nil {
		return 0, err
	}
	url := fmt.Sprintf("http://127.0.0.1:%d/num_unconfirmed_txs", n.Ports.RPC)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return 0, err
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("status code %d", resp.StatusCode)
	}

	var decoded struct {
		Result struct {
			Total string `json:"total"`
		} `json:"result"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&decoded); err != nil {
		return 0, err
	}

	return strconv.ParseInt(decoded.Result.Total, 10, 64)
}

func (c *Cluster) getNode(index int) (*Node, error) {
	if index < 0 || index >= len(c.nodes) {
		return nil, fmt.Errorf("invalid node index %d", index)
	}
	return c.nodes[index], nil
}

func (c *Cluster) keyAddress(ctx context.Context, name string) (string, error) {
	out, err := c.exec(ctx,
		"keys", "show", name,
		"-a",
		"--keyring-backend", "test",
		"--home", c.nodes[0].Home,
	)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

func (c *Cluster) exec(ctx context.Context, args ...string) ([]byte, error) {
	//nolint:gosec // c.bin is a test-controlled binary path from ClusterOptions or local build output.
	cmd := exec.CommandContext(ctx, c.bin, args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("%s %s failed: %w\n%s", c.bin, strings.Join(args, " "), err, string(out))
	}
	return out, nil
}

func buildInitiad(ctx context.Context, repoRoot, outPath string) error {
	cmd := exec.CommandContext(ctx, "go", "build", "-o", outPath, "./cmd/initiad")
	cmd.Dir = repoRoot
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("go build failed: %w\n%s", err, string(out))
	}
	return nil
}

func findRepoRoot() (string, error) {
	wd, err := os.Getwd()
	if err != nil {
		return "", err
	}

	current := wd
	for {
		if _, err := os.Stat(filepath.Join(current, "go.mod")); err == nil {
			return current, nil
		}
		next := filepath.Dir(current)
		if next == current {
			break
		}
		current = next
	}
	return "", errors.New("go.mod not found from current directory")
}

func copyFile(src, dst string) error {
	bz, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	return os.WriteFile(dst, bz, 0o600) //nolint:gosec
}

// copyDir copies all files from srcDir into dstDir (non-recursive).
// dstDir is created if it does not exist. Existing files are overwritten.
func copyDir(srcDir, dstDir string) error {
	if err := os.MkdirAll(dstDir, 0o700); err != nil {
		return err
	}
	entries, err := os.ReadDir(srcDir)
	if err != nil {
		return err
	}
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		if err := copyFile(filepath.Join(srcDir, e.Name()), filepath.Join(dstDir, e.Name())); err != nil {
			return err
		}
	}
	return nil
}

func parseTxResultFromOutput(out []byte) (TxResult, error) {
	var txResp map[string]any
	if err := json.Unmarshal(out, &txResp); err != nil {
		jsonOut, extractErr := extractJSONObject(out)
		if extractErr != nil {
			return TxResult{}, fmt.Errorf("failed to parse tx response: %w", err)
		}
		if err := json.Unmarshal(jsonOut, &txResp); err != nil {
			return TxResult{}, fmt.Errorf("failed to parse extracted tx response: %w", err)
		}
	}

	code, _ := findIntField(txResp, "code")
	txHash, _ := txResp["txhash"].(string)
	rawLog, _ := txResp["raw_log"].(string)
	return TxResult{
		Code:   code,
		TxHash: txHash,
		RawLog: rawLog,
	}, nil
}

func findUintField(v any, key string) (uint64, bool) {
	switch vv := v.(type) {
	case map[string]any:
		if raw, ok := vv[key]; ok {
			switch x := raw.(type) {
			case string:
				n, err := strconv.ParseUint(x, 10, 64)
				if err == nil {
					return n, true
				}
			case float64:
				return uint64(x), true
			}
		}
		for _, child := range vv {
			if n, ok := findUintField(child, key); ok {
				return n, true
			}
		}
	case []any:
		for _, child := range vv {
			if n, ok := findUintField(child, key); ok {
				return n, true
			}
		}
	}
	return 0, false
}

func findIntField(v any, key string) (int64, bool) {
	switch vv := v.(type) {
	case map[string]any:
		if raw, ok := vv[key]; ok {
			switch x := raw.(type) {
			case string:
				n, err := strconv.ParseInt(x, 10, 64)
				if err == nil {
					return n, true
				}
			case float64:
				return int64(x), true
			}
		}
		for _, child := range vv {
			if n, ok := findIntField(child, key); ok {
				return n, true
			}
		}
	case []any:
		for _, child := range vv {
			if n, ok := findIntField(child, key); ok {
				return n, true
			}
		}
	}
	return 0, false
}

func extractJSONObject(out []byte) ([]byte, error) {
	s := strings.TrimSpace(string(out))
	start := strings.IndexByte(s, '{')
	end := strings.LastIndexByte(s, '}')
	if start == -1 || end == -1 || end <= start {
		return nil, errors.New("json object not found in output")
	}
	return []byte(s[start : end+1]), nil
}

func base64Decode(s string) ([]byte, error) {
	return base64.StdEncoding.DecodeString(s)
}

func sha256Hash(data []byte) string {
	h := sha256.Sum256(data)
	return hex.EncodeToString(h[:])
}

func parseEstimatedGas(out []byte) (uint64, error) {
	var txResp map[string]any
	if err := json.Unmarshal(out, &txResp); err == nil {
		if n, ok := findUintField(txResp, "gas_limit"); ok && n > 0 {
			return n, nil
		}
		if n, ok := findUintField(txResp, "gasLimit"); ok && n > 0 {
			return n, nil
		}
		if n, ok := findUintField(txResp, "gas_wanted"); ok && n > 0 {
			return n, nil
		}
		if n, ok := findUintField(txResp, "gasWanted"); ok && n > 0 {
			return n, nil
		}
	}

	re := regexp.MustCompile(`gas estimate:\s*([0-9]+)`)
	m := re.FindSubmatch(out)
	if len(m) == 2 {
		n, err := strconv.ParseUint(string(m[1]), 10, 64)
		if err == nil && n > 0 {
			return n, nil
		}
	}

	return 0, fmt.Errorf("failed to parse estimated gas from output: %s", strings.TrimSpace(string(out)))
}

// SignedTx holds a pre-signed, encoded transaction ready for HTTP broadcast.
type SignedTx struct {
	Account       string
	Sequence      uint64
	TxBase64      string
	TxHash        string
	AccountNumber uint64
}

// GenerateSignedBankTx generates a signed, base64-encoded bank send transaction offline.
// Uses --generate-only to build, tx sign to sign, and tx encode to produce the final bytes.
func (c *Cluster) GenerateSignedBankTx(ctx context.Context, fromName, toAddress, amount string, accountNumber, sequence, gasLimit uint64) (SignedTx, error) {
	// 1. Generate unsigned tx JSON
	genArgs := []string{
		"tx", "bank", "send", fromName, toAddress, amount,
		"--chain-id", c.opts.ChainID,
		"--home", c.nodes[0].Home,
		"--keyring-backend", "test",
		"--gas-prices", "0.15uinit",
		"--gas", strconv.FormatUint(gasLimit, 10),
		"--account-number", strconv.FormatUint(accountNumber, 10),
		"--sequence", strconv.FormatUint(sequence, 10),
		"--generate-only",
	}
	if !c.opts.NoAllowQueued {
		genArgs = append(genArgs, "--allow-queued")
	}

	unsignedJSON, err := c.exec(ctx, genArgs...)
	if err != nil {
		return SignedTx{}, fmt.Errorf("generate-only: %w", err)
	}

	// 2. Write unsigned tx to temp file and sign
	tmpFile, err := os.CreateTemp("", "unsigned-tx-*.json")
	if err != nil {
		return SignedTx{}, err
	}
	defer os.Remove(tmpFile.Name())

	if _, err := tmpFile.Write(unsignedJSON); err != nil {
		tmpFile.Close()
		return SignedTx{}, err
	}
	tmpFile.Close()

	signArgs := []string{
		"tx", "sign", tmpFile.Name(),
		"--from", fromName,
		"--chain-id", c.opts.ChainID,
		"--home", c.nodes[0].Home,
		"--keyring-backend", "test",
		"--offline",
		"--account-number", strconv.FormatUint(accountNumber, 10),
		"--sequence", strconv.FormatUint(sequence, 10),
	}

	signedJSON, err := c.exec(ctx, signArgs...)
	if err != nil {
		return SignedTx{}, fmt.Errorf("sign: %w", err)
	}

	// 3. Encode to base64
	signedFile, err := os.CreateTemp("", "signed-tx-*.json")
	if err != nil {
		return SignedTx{}, err
	}
	defer os.Remove(signedFile.Name())

	if _, err := signedFile.Write(signedJSON); err != nil {
		signedFile.Close()
		return SignedTx{}, err
	}
	signedFile.Close()

	encodeArgs := []string{
		"tx", "encode", signedFile.Name(),
		"--home", c.nodes[0].Home,
	}

	encodedOut, err := c.exec(ctx, encodeArgs...)
	if err != nil {
		return SignedTx{}, fmt.Errorf("encode: %w", err)
	}

	txBase64 := strings.TrimSpace(string(encodedOut))

	// Compute tx hash from raw bytes
	txBytes, err := base64.StdEncoding.DecodeString(txBase64)
	if err != nil {
		return SignedTx{}, fmt.Errorf("decode base64 for hash: %w", err)
	}
	hash := sha256.Sum256(txBytes)

	return SignedTx{
		Account:       fromName,
		Sequence:      sequence,
		TxBase64:      txBase64,
		TxHash:        strings.ToUpper(hex.EncodeToString(hash[:])),
		AccountNumber: accountNumber,
	}, nil
}

// GenerateSignedMoveExecTx generates a signed, base64-encoded Move execute transaction offline.
func (c *Cluster) GenerateSignedMoveExecTx(
	ctx context.Context,
	fromName, moduleAddress, moduleName, functionName string,
	typeArgs, args []string,
	accountNumber, sequence, gasLimit uint64,
) (SignedTx, error) {
	typeArgsJSON, err := json.Marshal(typeArgs)
	if err != nil {
		return SignedTx{}, err
	}
	moveArgsJSON, err := json.Marshal(args)
	if err != nil {
		return SignedTx{}, err
	}

	genArgs := []string{
		"tx", "move", "execute-json",
		moduleAddress, moduleName, functionName,
		"--type-args", string(typeArgsJSON),
		"--args", string(moveArgsJSON),
		"--from", fromName,
		"--chain-id", c.opts.ChainID,
		"--home", c.nodes[0].Home,
		"--keyring-backend", "test",
		"--gas-prices", "0.15uinit",
		"--gas", strconv.FormatUint(gasLimit, 10),
		"--account-number", strconv.FormatUint(accountNumber, 10),
		"--sequence", strconv.FormatUint(sequence, 10),
		"--generate-only",
	}
	if !c.opts.NoAllowQueued {
		genArgs = append(genArgs, "--allow-queued")
	}

	unsignedJSON, err := c.exec(ctx, genArgs...)
	if err != nil {
		return SignedTx{}, fmt.Errorf("generate-only: %w", err)
	}

	tmpFile, err := os.CreateTemp("", "unsigned-tx-*.json")
	if err != nil {
		return SignedTx{}, err
	}
	defer os.Remove(tmpFile.Name())

	if _, err := tmpFile.Write(unsignedJSON); err != nil {
		tmpFile.Close()
		return SignedTx{}, err
	}
	tmpFile.Close()

	signArgs := []string{
		"tx", "sign", tmpFile.Name(),
		"--from", fromName,
		"--chain-id", c.opts.ChainID,
		"--home", c.nodes[0].Home,
		"--keyring-backend", "test",
		"--offline",
		"--account-number", strconv.FormatUint(accountNumber, 10),
		"--sequence", strconv.FormatUint(sequence, 10),
	}

	signedJSON, err := c.exec(ctx, signArgs...)
	if err != nil {
		return SignedTx{}, fmt.Errorf("sign: %w", err)
	}

	signedFile, err := os.CreateTemp("", "signed-tx-*.json")
	if err != nil {
		return SignedTx{}, err
	}
	defer os.Remove(signedFile.Name())

	if _, err := signedFile.Write(signedJSON); err != nil {
		signedFile.Close()
		return SignedTx{}, err
	}
	signedFile.Close()

	encodeArgs := []string{
		"tx", "encode", signedFile.Name(),
		"--home", c.nodes[0].Home,
	}

	encodedOut, err := c.exec(ctx, encodeArgs...)
	if err != nil {
		return SignedTx{}, fmt.Errorf("encode: %w", err)
	}

	txBase64 := strings.TrimSpace(string(encodedOut))
	txBytes, err := base64.StdEncoding.DecodeString(txBase64)
	if err != nil {
		return SignedTx{}, fmt.Errorf("decode base64 for hash: %w", err)
	}
	hash := sha256.Sum256(txBytes)

	return SignedTx{
		Account:       fromName,
		Sequence:      sequence,
		TxBase64:      txBase64,
		TxHash:        strings.ToUpper(hex.EncodeToString(hash[:])),
		AccountNumber: accountNumber,
	}, nil
}

// BroadcastTxSync broadcasts a pre-signed transaction via HTTP to the given node's
// /broadcast_tx_sync endpoint. Returns the response code and tx hash.
func (c *Cluster) BroadcastTxSync(ctx context.Context, nodeIndex int, txBase64 string) (TxResult, error) {
	node, err := c.getNode(nodeIndex)
	if err != nil {
		return TxResult{}, err
	}

	// Use JSON-RPC POST for reliability.
	txBytes, err := base64.StdEncoding.DecodeString(txBase64)
	if err != nil {
		return TxResult{}, fmt.Errorf("decode tx base64: %w", err)
	}

	reqBody := fmt.Sprintf(`{"jsonrpc":"2.0","id":1,"method":"broadcast_tx_sync","params":{"tx":"%s"}}`, base64.StdEncoding.EncodeToString(txBytes))
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, fmt.Sprintf("http://127.0.0.1:%d", node.Ports.RPC), strings.NewReader(reqBody))
	if err != nil {
		return TxResult{}, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return TxResult{}, err
	}
	defer resp.Body.Close()

	var rpcResp struct {
		Result struct {
			Code int64  `json:"code"`
			Hash string `json:"hash"`
			Log  string `json:"log"`
		} `json:"result"`
		Error *struct {
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&rpcResp); err != nil {
		return TxResult{}, fmt.Errorf("decode broadcast response: %w", err)
	}

	if rpcResp.Error != nil {
		return TxResult{Err: fmt.Errorf("broadcast RPC error: %s", rpcResp.Error.Message)}, nil
	}

	return TxResult{
		Code:   rpcResp.Result.Code,
		TxHash: rpcResp.Result.Hash,
		RawLog: rpcResp.Result.Log,
	}, nil
}
