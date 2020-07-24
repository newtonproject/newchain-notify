package notify

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/eclipse/paho.mqtt.golang"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/rlp"
	log "github.com/sirupsen/logrus"
)

type PendingNotify struct {
	Notify
}

func NewPendingNotify(s, p *NotifyConfig, logger *log.Logger) (*PendingNotify, error) {
	if s == nil || p == nil {
		return nil, errors.New("subscribe or publish config can not be nil")
	}
	return &PendingNotify{
		Notify{
			s:      s,
			p:      p,
			Logger: logger,
			quit:   make(chan struct{}, 1),
		}}, nil
}

// MarshalJSON encodes to json format.
func (n *PendingNotify) MarshalJSON() ([]byte, error) {
	type config struct {
		Subscribe   *NotifyConfig
		Publish     *NotifyConfig
		LoggerLevel string
	}

	enc := &config{
		Subscribe:   n.s,
		Publish:     n.p,
		LoggerLevel: n.Logger.Level.String(),
	}

	return json.Marshal(&enc)
}

func (n *PendingNotify) Run() error {
	if n.Logger == nil {
		n.Logger = log.New()
	}

	ch := make(chan string, 10)
	onMessageReceived := func(c mqtt.Client, message mqtt.Message) {
		if message != nil && message.Topic() == n.s.Topic {
			ch <- string(message.Payload())
		}
	}

	if n.p.Topic == "" {
		n.quit <- struct{}{}
		return errors.New("publish topic set")
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
				n.handlerRawTransaction(pClient, raw)
			case <-n.quit:
				return
			}
		}
	}()

	return n.runSubscribeClient(onMessageReceived)
}

func (n *PendingNotify) handlerRawTransaction(c mqtt.Client, raw string) {
	tx, err := decodeTransaction(raw)
	if err != nil {
		n.Logger.Errorln(err)
		return
	}
	if tx == nil {
		n.Logger.Errorln(errors.New("tx is nil"))
		return
	}
	signer := types.NewEIP155Signer(tx.ChainId())
	from, err := signer.Sender(tx)
	if err != nil {
		n.Logger.Errorln(err)
		return
	}
	if from == (common.Address{}) {
		n.Logger.Errorln(errors.New("from address is nil"))
		return
	}

	aTx := &TransferTx{
		From:  from,
		To:    tx.To(),
		Value: tx.Value(),
		Hash:  tx.Hash(),
	}

	n.publish(c, aTx)
	n.publishToBlockTopic(c, aTx, 0)
}

func decodeTransaction(hexParam string) (*types.Transaction, error) {
	encodedTx, err := hexutil.Decode(hexParam)
	if err != nil {
		return nil, err
	}
	if len(encodedTx) <= 0 {
		return nil, fmt.Errorf("decode transaction error")
	}
	tx := new(types.Transaction)
	if err := rlp.DecodeBytes(encodedTx, tx); err != nil {
		return nil, err
	}

	return tx, nil
}
