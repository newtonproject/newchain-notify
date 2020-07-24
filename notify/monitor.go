package notify

import (
	"context"
	"encoding/json"
	"errors"
	"io/ioutil"
	"math/big"
	"os"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/rpc"
	log "github.com/sirupsen/logrus"
	"github.com/newtonproject/newchain-notify/tracer"
)

type MonitorNotify struct {
	Notify

	rpcURL       string
	blockDelay   int64
	ec           *ethclient.Client
	enableTracer bool
	traceConfig  *tracer.TraceConfig
}

func NewMonitorNotify(p *NotifyConfig, rpcURL string, blockDelay int64, enableTracer bool, traceConfig *tracer.TraceConfig, logger *log.Logger) (*MonitorNotify, error) {
	if p == nil {
		return nil, errors.New("publish config can not be nil")
	}
	return &MonitorNotify{
		Notify: Notify{
			p:      p,
			Logger: logger,
			quit:   make(chan struct{}, 1),
		},
		blockDelay:   blockDelay,
		rpcURL:       rpcURL,
		enableTracer: enableTracer,
		traceConfig:  traceConfig,
	}, nil
}

// MarshalJSON encodes to json format.
func (n *MonitorNotify) MarshalJSON() ([]byte, error) {
	type config struct {
		Subscribe   *NotifyConfig
		Publish     *NotifyConfig
		RPCURL      string
		DelayBlock  int64
		LoggerLevel string
	}

	enc := &config{
		Publish:     n.p,
		RPCURL:      n.rpcURL,
		DelayBlock:  n.blockDelay,
		LoggerLevel: n.Logger.Level.String(),
	}
	n.p.Topic = "-"

	return json.Marshal(&enc)
}

func (n *MonitorNotify) Run() error {
	if n.Logger == nil {
		n.Logger = log.New()
	}

	start, err := n.loadBlockHeight()
	if err != nil {
		return err
	}

	n.monitorBlock(start)

	return nil
}

func (n *MonitorNotify) monitorBlock(startBlockNumber *big.Int) {
	pClient, err := n.getPublishClient()
	if err != nil {
		log.Errorln(err)
		return
	}
	if pClient == nil {
		log.Errorln(errors.New("publish client nil"))
		return
	}

	log.Println("Running NewChain Monitor...")
	c, err := rpc.Dial(n.rpcURL)
	if err != nil {
		log.Errorln(err)
		return
	}
	client := ethclient.NewClient(c)

	blockDelay := n.blockDelay
	ctx := context.Background()

	latestBlockNumber := big.NewInt(0)

	updateLatestBlockNumberFromNewChain := func() error {
		header, err := client.HeaderByNumber(ctx, nil)
		if err != nil {
			return err
		}
		if latestBlockNumber.Cmp(header.Number) < 0 {
			latestBlockNumber.Set(header.Number)
		}
		log.Infof("Latest block number is %d", latestBlockNumber.Uint64())

		return nil
	}

	header, err := client.HeaderByNumber(ctx, nil)
	if err != nil {
		n.Logger.Error(err)
		n.quit <- struct{}{}
		return
	}
	if latestBlockNumber.Cmp(header.Number) < 0 {
		latestBlockNumber.Set(header.Number)
	}
	log.Infof("Latest block number is %d", latestBlockNumber.Uint64())

	// number = number - blockDelay
	parenBlock, err := client.HeaderByNumber(ctx, big.NewInt(0).Sub(latestBlockNumber, big.NewInt(1)))
	if err != nil {
		n.Logger.Error(err)
		n.quit <- struct{}{}
		return
	}

	blockPeriod := int64(header.Time - parenBlock.Time)
	if blockPeriod <= 0 {
		n.Logger.Errorln("get block period error", blockPeriod)
		n.quit <- struct{}{}
		return
	}
	n.Logger.Printf("blockPeriod is : %d second", blockPeriod)

	currentBlockNumber := big.NewInt(0)
	if startBlockNumber == nil {
		if latestBlockNumber.Cmp(big.NewInt(blockDelay)) <= 0 {
			// just in case
			time.Sleep(time.Second * time.Duration(blockDelay))
			if err = updateLatestBlockNumberFromNewChain(); err != nil {
				log.Errorln(err)
				return
			}
		}
		currentBlockNumber.Sub(latestBlockNumber, big.NewInt(blockDelay))
	} else {
		currentBlockNumber.Set(startBlockNumber)
	}
	log.Infof("Monitor from block number	%d", currentBlockNumber.Uint64())

	getBlocks := func() error {
		latestBlock, err := client.BlockByNumber(ctx, nil)
		if err != nil {
			return err
		}
		if latestBlockNumber.Cmp(latestBlock.Number()) < 0 {
			latestBlockNumber.Set(latestBlock.Number())
		}
		log.Infof("Latest block number is %d", latestBlockNumber.Uint64())

		for big.NewInt(0).Add(currentBlockNumber, big.NewInt(blockDelay)).Cmp(latestBlockNumber) <= 0 {
			log.Infof("Try to handle block %s and the latest block number is %s", currentBlockNumber.String(), latestBlockNumber.String())

			var block *types.Block
			if latestBlockNumber.Cmp(currentBlockNumber) == 0 {
				block = latestBlock
			} else {
				block, err = client.BlockByNumber(ctx, currentBlockNumber)
				if err != nil {
					return err
				}
			}

			err = n.saveBlockHeight(currentBlockNumber)
			if err != nil {
				return err
			}
			currentBlockNumber.Add(currentBlockNumber, big.NewInt(1))

			log.Infof("Handle block %d with txs is %d", block.NumberU64(), block.Transactions().Len())

			txs := block.Transactions()
			txLen := txs.Len()
			if txLen == 0 {
				continue
			}

			for i := 0; i < txLen; i++ {
				tx := txs[i]

				tracerStatus := false
				if n.enableTracer {
					tracerStatus = true
					txsTrace, err := tracer.TraceTransaction(c, ctx, tx, n.traceConfig)
					if err != nil {
						log.Errorln(err)
						tracerStatus = false
					}
					if len(txsTrace) > 0 {
						for _, tt := range txsTrace {
							// push
							ttx := TransferTx{
								From:        tt.From,
								To:          tt.To,
								Value:       tt.Value,
								Hash:        tx.Hash(),
								Data:        tt.Input,
								BlockNumber: block.Number(),
							}

							n.publishToBlockTopic(pClient, &ttx, blockDelay+1)
						}
					} else {
						tracerStatus = false
					}
				}

				if !n.enableTracer || !tracerStatus {
					var from common.Address
					from, err = client.TransactionSender(ctx, tx, block.Hash(), uint(i))
					if err != nil {
						log.Warnln(err)
						continue
					}

					// push
					ttx := TransferTx{
						From:        from,
						To:          tx.To(),
						Value:       tx.Value(),
						Hash:        tx.Hash(),
						Data:        tx.Data(),
						BlockNumber: block.Number(),
					}

					n.publishToBlockTopic(pClient, &ttx, blockDelay+1)
				}
			}
		}

		return nil
	}

	go func() {
		// get block by number
		ticker := time.NewTicker(time.Duration(blockPeriod) * time.Second)
		for {
			select {
			case <-ticker.C:
				if err = getBlocks(); err != nil {
					log.Errorln(err)
					continue
				}
			}
		}

	}()

	select {}
}

// LatestHeight config file
const NewChainNotifyMonitorLatestHeight = ".BlockHeight"

func (n *MonitorNotify) saveBlockHeight(number *big.Int) error {
	if number == nil {
		return nil
	}

	return ioutil.WriteFile(NewChainNotifyMonitorLatestHeight, []byte(number.String()), 0644)
}

func (n *MonitorNotify) loadBlockHeight() (*big.Int, error) {
	if _, err := os.Stat(NewChainNotifyMonitorLatestHeight); os.IsNotExist(err) {
		return nil, nil
	}

	nByte, err := ioutil.ReadFile(NewChainNotifyMonitorLatestHeight)
	if err != nil {
		return nil, err
	}

	number, ok := big.NewInt(0).SetString(string(nByte), 10)
	if !ok {
		return nil, errors.New("convert height to big int error")
	}

	return number, nil
}
