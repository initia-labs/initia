package oracle

import (
	"context"
	"fmt"
	"net/url"
	"strconv"
	"sync"
	"time"

	"cosmossdk.io/log"

	"github.com/skip-mev/slinky/oracle/config"
	oracleclient "github.com/skip-mev/slinky/service/clients/oracle"
	"github.com/skip-mev/slinky/service/metrics"
	oracleservertypes "github.com/skip-mev/slinky/service/servers/oracle/types"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	l2slinky "github.com/initia-labs/OPinit/x/opchild/l2slinky"
)

var _ oracleclient.OracleClient = (*GRPCClient)(nil)

// GRPCClient defines an implementation of a gRPC oracle client. This client can
// be used in ABCI++ calls where the application wants the oracle process to be
// run out-of-process. The client must be started upon app construction and
// stopped upon app shutdown/cleanup.
type GRPCClient struct {
	logger log.Logger
	mutex  sync.Mutex

	// address of remote oracle server
	addr string
	// underlying oracle client
	client oracleservertypes.OracleClient
	// underlying grpc connection
	conn *grpc.ClientConn
	// timeout for the client, Price requests will block for this duration.
	timeout time.Duration
	// metrics contains the instrumentation for the oracle client
	metrics metrics.Metrics
	// blockingDial is a parameter which determines whether the client should block on dialing the server
	blockingDial bool
}

// NewClientFromConfig creates a new grpc client of the oracle service with the given
// app configuration. This returns an error if the configuration is invalid.
func NewClientFromConfig(
	cfg config.AppConfig,
	logger log.Logger,
	metrics metrics.Metrics,
	opts ...oracleclient.Option,
) (oracleclient.OracleClient, error) {
	if err := cfg.ValidateBasic(); err != nil {
		return nil, err
	}

	if !cfg.Enabled {
		return &oracleclient.NoOpClient{}, nil
	}

	if logger == nil {
		return nil, fmt.Errorf("logger cannot be nil")
	}

	if metrics == nil {
		return nil, fmt.Errorf("metrics cannot be nil")
	}

	return NewClient(logger, cfg.OracleAddress, cfg.ClientTimeout, metrics, opts...)
}

// NewClient creates a new grpc client of the oracle service with the given
// address and timeout.
func NewClient(
	logger log.Logger,
	addr string,
	timeout time.Duration,
	metrics metrics.Metrics,
	opts ...oracleclient.Option,
) (oracleclient.OracleClient, error) {
	if logger == nil {
		return nil, fmt.Errorf("logger cannot be nil")
	}

	if _, err := url.ParseRequestURI(addr); err != nil {
		return nil, fmt.Errorf("invalid oracle address: %w", err)
	}

	if metrics == nil {
		return nil, fmt.Errorf("metrics cannot be nil")
	}

	if timeout <= 0 {
		return nil, fmt.Errorf("timeout must be positive")
	}

	client := &GRPCClient{
		logger:  logger,
		addr:    addr,
		timeout: timeout,
		metrics: metrics,
	}

	// apply options
	for _, opt := range opts {
		opt(client)
	}

	return client, nil
}

// NOOP
func (c *GRPCClient) Start(_ context.Context) error {
	return nil
}

// Stop stops the GRPC client. This method closes the connection to the remote.
func (c *GRPCClient) Stop() error {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	c.logger.Info("stopping oracle client")
	if c.conn == nil {
		return nil
	}

	err := c.conn.Close()
	c.logger.Info("oracle client stopped", "err", err)

	return err
}

// Prices returns the prices from the remote oracle service. This method blocks for the timeout duration configured on the client,
// otherwise it returns the response from the remote oracle.
func (c *GRPCClient) Prices(
	ctx context.Context,
	req *oracleservertypes.QueryPricesRequest,
	_ ...grpc.CallOption,
) (resp *oracleservertypes.QueryPricesResponse, err error) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	start := time.Now()
	defer func() {
		// Observe the duration of the call as well as the error.
		c.metrics.ObserveOracleResponseLatency(time.Since(start))
		c.metrics.AddOracleResponse(metrics.StatusFromError(err))
	}()

	// set deadline on the context
	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	// create client if the client is empty
	if c.client == nil {
		if err := c.createConnection(ctx); err != nil {
			return nil, err
		}
	}

	resp, err = c.client.Prices(ctx, req, grpc.WaitForReady(true))
	if err != nil {
		return nil, err
	}

	resp.Prices[l2slinky.ReservedCPTimestamp] = strconv.FormatInt(resp.Timestamp.UTC().UnixNano(), 10)
	return resp, nil
}

func (c *GRPCClient) createConnection(ctx context.Context) error {
	c.logger.Info("starting oracle client", "addr", c.addr)

	opts := []grpc.DialOption{
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	}

	if c.blockingDial {
		opts = append(opts, grpc.WithBlock())
	}

	// dial the client, but defer to context closure, if necessary
	var (
		conn *grpc.ClientConn
		err  error
		done = make(chan struct{})
	)
	go func() {
		defer close(done)
		conn, err = grpc.DialContext(ctx, c.addr, opts...)
	}()

	// wait for either the context to close or the dial to complete
	select {
	case <-ctx.Done():
		err = fmt.Errorf("context closed before oracle client could start: %w", ctx.Err())
	case <-done:
	}
	if err != nil {
		c.logger.Error("failed to dial oracle gRPC server", "err", err)
		return fmt.Errorf("failed to dial oracle gRPC server: %w", err)
	}

	c.client = oracleservertypes.NewOracleClient(conn)
	c.conn = conn

	c.logger.Info("oracle client started")

	return nil
}
