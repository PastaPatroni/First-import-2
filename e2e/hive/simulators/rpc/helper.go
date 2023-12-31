// SPDX-License-Identifier: MIT
//
// # Copyright (c) 2023 Berachain Foundation
//
// Permission is hereby granted, free of charge, to any person
// obtaining a copy of this software and associated documentation
// files (the "Software"), to deal in the Software without
// restriction, including without limitation the rights to use,
// copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the
// Software is furnished to do so, subject to the following
// conditions:
//
// The above copyright notice and this permission notice shall be
// included in all copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND,
// EXPRESS OR IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES
// OF MERCHANTABILITY, FITNESS FOR A PARTICULAR PURPOSE AND
// NONINFRINGEMENT. IN NO EVENT SHALL THE AUTHORS OR COPYRIGHT
// HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER LIABILITY,
// WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING
// FROM, OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR
// OTHER DEALINGS IN THE SOFTWARE.

package main

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/ethereum/hive/hivesim"

	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/rpc"
)

// default timeout for RPC calls.
var rpcTimeout = 10 * time.Second

// TestClient is the environment of a single test.
type TestEnv struct {
	*hivesim.T
	RPC   *rpc.Client
	Eth   *ethclient.Client
	Vault *vault

	// This holds most recent context created by the Ctx method.
	// Every time Ctx is called, it creates a new context with the default
	// timeout and cancels the previous one.
	lastCtx    context.Context
	lastCancel context.CancelFunc
}

const (
	timeout = 5
	delay   = 100
)

// runHTTP runs the given test function using the HTTP RPC client.
func runHTTP(t *hivesim.T, c *hivesim.Client, v *vault, fn func(*TestEnv)) {
	// This sets up debug logging of the requests and responses.
	client := &http.Client{
		Transport: &loggingRoundTrip{
			t:     t,
			inner: http.DefaultTransport,
		},
	}

	//nolint: staticcheck // rpc.DialOptions requires ctx
	rpcClient, _ := rpc.DialHTTPWithClient(fmt.Sprintf("http://%v:8545/", c.IP), client)
	defer rpcClient.Close()
	env := &TestEnv{
		T:     t,
		RPC:   rpcClient,
		Eth:   ethclient.NewClient(rpcClient),
		Vault: v,
	}
	fn(env)
	if env.lastCtx != nil {
		env.lastCancel()
	}
}

// runWS runs the given test function using the WebSocket RPC client.
func runWS(t *hivesim.T, c *hivesim.Client, v *vault, fn func(*TestEnv)) {
	ctx, done := context.WithTimeout(context.Background(), timeout*time.Second)
	rpcClient, err := rpc.DialWebsocket(ctx, fmt.Sprintf("ws://%v:8546/", c.IP), "")
	done()
	if err != nil {
		t.Fatal("WebSocket connection failed:", err)
	}
	defer rpcClient.Close()

	env := &TestEnv{
		T:     t,
		RPC:   rpcClient,
		Eth:   ethclient.NewClient(rpcClient),
		Vault: v,
	}
	fn(env)
	if env.lastCtx != nil {
		env.lastCancel()
	}
}

// CallContext is a helper method that forwards a raw RPC request to
// the underlying RPC client. This can be used to call RPC methods
// that are not supported by the ethclient.Client.
func (t *TestEnv) CallContext(ctx context.Context, result interface{}, method string, args ...interface{}) error {
	return t.RPC.CallContext(ctx, result, method, args...)
}

// Ctx returns a context with the default timeout.
// For subsequent calls to Ctx, it also cancels the previous context.
func (t *TestEnv) Ctx() context.Context {
	if t.lastCtx != nil {
		t.lastCancel()
	}
	t.lastCtx, t.lastCancel = context.WithTimeout(context.Background(), rpcTimeout)
	return t.lastCtx
}

// func waitSynced(c *rpc.Client) error {
// 	var (
// 		err         error
// 		timeout     = 20 * time.Second
// 		end         = time.Now().Add(timeout)
// 		ctx, cancel = context.WithDeadline(context.Background(), end)
// 	)
// 	defer func() {
// 		cancel()
// 		if errors.Is(err, context.DeadlineExceeded) {
// 			err = fmt.Errorf("didn't sync within timeout of %v", timeout)
// 		}
// 	}()

// 	ec := ethclient.NewClient(c)
// 	for {
// 		var progress *ethereum.SyncProgress
// 		progress, err = ec.SyncProgress(ctx)
// 		if err != nil {
// 			return err
// 		}
// 		var head uint64
// 		head, err = ec.BlockNumber(ctx)
// 		if err != nil {
// 			return err
// 		}
// 		if progress == nil && head > 0 {
// 			return nil // success!
// 		}
// 		time.Sleep(delay * time.Millisecond)
// 	}
// }

// // Naive generic function that works in all situations.
// // A better solution is to use logs to wait for confirmations.
// //nolint: gocognit // function is long since it has a lot of checks
// func waitForTxConfirmations(t *TestEnv, txHash common.Hash, n uint64) (*types.Receipt, error) {
// 	var (
// 		receipt    *types.Receipt
// 		startBlock *types.Block
// 		err        error
// 	)

// 	for i := 0; i < 90; i++ {
// 		receipt, err = t.Eth.TransactionReceipt(t.Ctx(), txHash)
// 		if err != nil && !errors.Is(err, ethereum.NotFound) {
// 			return nil, err
// 		}
// 		if receipt != nil {
// 			break
// 		}
// 		time.Sleep(time.Second)
// 	}
// 	if receipt == nil {
// 		return nil, ethereum.NotFound
// 	}

// 	if startBlock, err = t.Eth.BlockByNumber(t.Ctx(), nil); err != nil {
// 		return nil, err
// 	}

// 	for i := 0; i < 90; i++ {
// 		var currentBlock *types.Block
// 		currentBlock, err = t.Eth.BlockByNumber(t.Ctx(), nil)
// 		if err != nil {
// 			return nil, err
// 		}

// 		//nolint: nestif // will fix this soon
// 		if startBlock.NumberU64()+n >= currentBlock.NumberU64() {
// 			var checkReceipt *types.Receipt
// 			checkReceipt, err = t.Eth.TransactionReceipt(t.Ctx(), txHash)
// 			if checkReceipt != nil {
// 				if bytes.Equal(receipt.PostState, checkReceipt.PostState) {
// 					return receipt, nil
// 				}
// 				// chain reorg
// 				if _, err = waitForTxConfirmations(t, txHash, n); err != nil {
// 					t.Fatal(err)
// 				}
// 			} else {
// 				return nil, err
// 			}
// 		}

// 		time.Sleep(time.Second)
// 	}

// 	return nil, ethereum.NotFound
// }

// loggingRoundTrip writes requests and responses to the test log.
type loggingRoundTrip struct {
	t     *hivesim.T
	inner http.RoundTripper
}

func (rt *loggingRoundTrip) RoundTrip(req *http.Request) (*http.Response, error) {
	// Read and log the request body.
	reqBytes, err := io.ReadAll(req.Body)
	if err != nil {
		return nil, err
	}
	err = req.Body.Close()
	if err != nil {
		return nil, err
	}
	rt.t.Logf(">>  %s", bytes.TrimSpace(reqBytes))
	reqCopy := *req
	reqCopy.Body = io.NopCloser(bytes.NewReader(reqBytes))

	// Do the round trip.
	resp, err := rt.inner.RoundTrip(&reqCopy)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close() //#nosec:G307 // this is a test.

	// Read and log the response bytes.
	respBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	respCopy := *resp
	respCopy.Body = io.NopCloser(bytes.NewReader(respBytes))
	rt.t.Logf("<<  %s", bytes.TrimSpace(respBytes))
	return &respCopy, nil
}

// func loadGenesis() *types.Block {
// 	contents, err := os.ReadFile("init/genesis.json")
// 	if err != nil {
// 		panic(fmt.Errorf("can't to read genesis file: %w", err))
// 	}
// 	var genesis core.Genesis
// 	if err = json.Unmarshal(contents, &genesis); err != nil {
// 		panic(fmt.Errorf("can't parse genesis JSON: %w", err))
// 	}
// 	return genesis.ToBlock()
// }

// // diff checks whether x and y are deeply equal, returning a description
// // of their differences if they are not equal.
// func diff(x, y interface{}) string {
// 	var d string
// 	for _, l := range pretty.Diff(x, y) {
// 		d += l + "\n"
// 	}
// 	return d
// }
