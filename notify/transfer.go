package notify

import (
	"context"
	"encoding/json"
	"errors"
	"math/big"
	"time"

	"github.com/eclipse/paho.mqtt.golang"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	log "github.com/sirupsen/logrus"
	"github.com/newtonproject/newchain-notify/queue"
)

type TxAge struct {
	tx  *TransferTx
	age uint64
	c   mqtt.Client
}

type TransaferNotify struct {
	Notify

	rpcURL string
	block  int64
	ec     *ethclient.Client

	// blockCh chan types.Block
	// q   *queue.Queue
}

func NewTransferNotify(s, p *NotifyConfig, rpcURL string, block int64, logger *log.Logger) (*TransaferNotify, error) {
	if s == nil || p == nil {
		return nil, errors.New("subscribe or publish config can not be nil")
	}
	return &TransaferNotify{
		Notify: Notify{
			s:      s,
			p:      p,
			Logger: logger,
			quit:   make(chan struct{}, 1),
		},
		block:  block,
		rpcURL: rpcURL,
		// blockCh:    make(chan types.Block, 1),
		// q:      queue.New(),
	}, nil
}

// MarshalJSON encodes to json format.
func (n *TransaferNotify) MarshalJSON() ([]byte, error) {
	type config struct {
		Subscribe   *NotifyConfig
		Publish     *NotifyConfig
		RPCURL      string
		DelayBlock  int64
		LoggerLevel string
	}

	enc := &config{
		Subscribe:   n.s,
		Publish:     n.p,
		RPCURL:      n.rpcURL,
		DelayBlock:  n.block,
		LoggerLevel: n.Logger.Level.String(),
	}
	n.p.Topic = "-"

	return json.Marshal(&enc)
}

func (n *TransaferNotify) Run() error {
	if n.Logger == nil {
		n.Logger = log.New()
	}

	ec, err := ethclient.Dial(n.rpcURL)
	if err != nil {
		return err
	}

	blockCh := make(chan *types.Block, 10)
	q := queue.New()
	go n.getBlockTicker(ec, n.block, blockCh)
	go n.runBlockCheck(q, blockCh)

	ch := make(chan string, 10)
	onMessageReceived := func(c mqtt.Client, message mqtt.Message) {
		if message != nil && message.Topic() == n.s.Topic {
			ch <- string(message.Payload())
		}
	}

	pClient, err := n.getPublishClient()
	if err != nil {
		n.quit <- struct{}{}
		return err
	}
	if pClient == nil {
		n.quit <- struct{}{}
		return errors.New("publish client nil")
	}

	go func() {
		for {
			select {
			case raw := <-ch:
				n.Logger.WithFields(log.Fields{
					"subscribe": n.s.Topic,
				}).Info(raw)
				tx, err := decodeTransferTx(raw)
				if err != nil {
					n.Logger.Errorln(err)
					continue
				}
				if tx == nil {
					n.Logger.Errorln(errors.New("tx is nil"))
					continue
				}
				q.Push(TxAge{tx: tx, age: 0, c: pClient})
			case <-n.quit:
				return
			}
		}
	}()

	return n.runSubscribeClient(onMessageReceived)
}

func (n *TransaferNotify) getBlockTicker(ec *ethclient.Client, blockDelay int64, blockCh chan *types.Block) {
	if blockDelay < 0 {
		blockDelay = 0
	}

	ctx := context.Background()
	block, err := ec.BlockByNumber(ctx, nil)
	if err != nil {
		n.Logger.Errorln(err)
		n.quit <- struct{}{}
		return
	}
	number := block.Number()
	// number = number - blockDelay
	parenBlock, err := ec.BlockByNumber(ctx, big.NewInt(0).Sub(number, big.NewInt(1)))
	if err != nil {
		n.Logger.Error(err)
		n.quit <- struct{}{}
		return
	}

	blockPeriod := int64(block.Time() - parenBlock.Time())
	if blockPeriod <= 0 {
		n.Logger.Errorln("get block period error", blockPeriod)
		n.quit <- struct{}{}
		return
	}
	n.Logger.Printf("blockPeriod is : %d second", blockPeriod)

	// try get the latest
	block, err = ec.BlockByNumber(ctx, nil)
	if err != nil {
		n.Logger.Errorln(err)
		n.quit <- struct{}{}
		return
	}

	if blockDelay > 0 {
		number = big.NewInt(0).Sub(block.Number(), big.NewInt(blockDelay))
	} else {
		number = block.Number()
	}
	var bTime int64
	n.Logger.Println("latest: ", block.NumberU64(), "use: ", number.Uint64())
	for {
		block, err = ec.BlockByNumber(ctx, number)
		if err != nil {
			if err.Error() == "not found" {
				continue
			}
			n.Logger.Errorln(err, number.String())
			continue
		}
		bTime = int64(block.Time())
		n.Logger.Debugln(block.NumberU64())

		// checkTx()
		blockCh <- block

		now := time.Now().UnixNano()
		sleep := (blockPeriod*(blockDelay+1)+bTime)*1000 - now/int64(time.Millisecond)
		// if sleep < 0 {
		if sleep < -100 { // -0.1ms
			n.Logger.Debugln("try reset the block")
			block, err = ec.BlockByNumber(ctx, nil)
			if err != nil {
				n.Logger.Errorln(err)
				continue
			}

			if blockDelay > 0 {
				number = big.NewInt(0).Sub(block.Number(), big.NewInt(blockDelay))
				block, err = ec.BlockByNumber(ctx, number)
				if err != nil {
					if err.Error() == "not found" {
						continue
					}
					n.Logger.Errorln(err, number.String())
					continue
				}

			}
			blockCh <- block
			bTime = int64(block.Time())
			now = time.Now().UnixNano()
			sleep = (blockPeriod*(blockDelay+1)+bTime)*1000 - now/int64(time.Millisecond)
		}

		n.Logger.Debugln(" sleep: ", time.Duration(sleep)*time.Millisecond)
		time.Sleep(time.Duration(sleep) * time.Millisecond)

		number = number.Add(number, big.NewInt(1))

		select {
		case <-n.quit:
			return
		default:
		}
	}

}

func (n *TransaferNotify) runBlockCheck(q *queue.Queue, blockCh chan *types.Block) {
	var blockList []*types.Block
	limitBlock := n.block + 10
	limitTx := uint64(limitBlock)
	for block := range blockCh {
		if block == nil {
			n.Logger.Errorln("get nil block")
			continue
		}
		blockList = append(blockList, block)

		size := q.Size()
		for i := 0; (!q.Empty()) && (i < size); i++ {
			func() {
				p := q.Pop()
				if p == nil {
					return
				}
				txAge := p.(TxAge)
				tx := txAge.tx
				for _, b := range blockList {
					for _, t := range b.Transactions() {
						if t.Hash() == tx.Hash {
							n.publishToBlockTopic(txAge.c, tx, n.block+1)
							return
						}
					}
				}

				if txAge.age > limitTx {
					n.Logger.Warnln("discard transaction ", tx.Hash.String())
					return
				}
				txAge.age++

				q.Push(txAge)
			}()
		}

		if int64(len(blockList)) > limitBlock {
			blockList = blockList[int64(len(blockList))-limitBlock:]
		}
	}
}

func decodeTransferTx(hexParam string) (*TransferTx, error) {
	tx := new(TransferTx)
	if err := json.Unmarshal([]byte(hexParam), tx); err != nil {
		return nil, err
	}

	return tx, nil
}
