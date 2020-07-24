package tracer

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/rpc"
)

func TestTracer(t *testing.T) {
	rpcurl := "https://rpc1.newchain.newtonproject.org"

	ctx := context.Background()
	c, err := rpc.DialContext(ctx, rpcurl)
	if err != nil {
		fmt.Println(err)
		return
	}

	// hash := common.HexToHash("0x33fcd1b345b466dddec615edb041efc614d35cdbdc3434584db4fcb083f5fa81") // tx
	hash := common.HexToHash("0xde86ad8dca01972ca209dcafc021fb7b2a4c8063cace28ee9e63e52a1cd2ed27") // internal tx

	tjs, err := TracerJS()
	if err != nil {
		fmt.Println(err)
		return
	}

	config := struct {
		Tracer  string `json:"tracer"`
		Timeout string `json:"timeout"`
		Reexec  uint64 `json:"reexec"`
	}{
		Tracer:  string(tjs),
		Timeout: "5m",
		Reexec:  1024, // blocks
	}

	var raw json.RawMessage
	err = c.CallContext(ctx, &raw, "debug_traceTransaction", hash, config)
	if err != nil {
		fmt.Println(err)
		return
	} else if len(raw) == 0 {
		fmt.Println(ethereum.NotFound)
		return
	}

	fmt.Println(string(raw))
}
