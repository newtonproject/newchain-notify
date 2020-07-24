package notify

import (
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"strings"

	"github.com/eclipse/paho.mqtt.golang"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	log "github.com/sirupsen/logrus"
)

type Tx struct {
	from common.Address
	tx   *types.Transaction
}

type NotifyConfig struct {
	Server      string
	Username    string
	Password    string
	ClientID    string
	QoS         byte
	Topic       string
	PrefixTopic string // for publish and only for n_address
}

type Notify struct {
	s *NotifyConfig
	p *NotifyConfig

	Logger *log.Logger
	quit   chan struct{}
}

type TransferTx struct {
	From        common.Address  `json:"from"`
	To          *common.Address `json:"to"`
	Value       *big.Int        `json:"value"`
	Hash        common.Hash     `json:"hash"`
	Data        []byte          `json:"data"`
	BlockNumber *big.Int        `json:"blockNumber"`
}

// UnmarshalJSON decodes from json format to a TransferTx.
func (c *TransferTx) UnmarshalJSON(data []byte) error {
	type Tx struct {
		From  common.Address  `json:"from"`
		To    *common.Address `json:"to"`
		Value string          `json:"value"`
		Hash  common.Hash     `json:"hash"`
	}
	var tx Tx
	err := json.Unmarshal(data, &tx)
	if err != nil {
		return err
	}
	c.From = tx.From
	c.To = tx.To
	value, err := hexutil.DecodeBig(tx.Value)
	if err != nil {
		return err
	}
	c.Value = value
	c.Hash = tx.Hash

	return nil
}

// MarshalJSON encodes to json format.
func (c *TransferTx) MarshalJSON() ([]byte, error) {
	type Tx struct {
		From        common.Address  `json:"from"`
		To          *common.Address `json:"to"`
		Value       *hexutil.Big    `json:"value"`
		Hash        common.Hash     `json:"hash"`
		Data        hexutil.Bytes   `json:"data"`
		BlockNumber *hexutil.Big    `json:"blockNumber"`
	}

	enc := &Tx{
		From:        c.From,
		To:          c.To,
		Value:       (*hexutil.Big)(c.Value),
		Hash:        c.Hash,
		Data:        c.Data,
		BlockNumber: (*hexutil.Big)(c.BlockNumber),
	}

	return json.Marshal(&enc)
}

func (n *Notify) runSubscribeClient(f func(c mqtt.Client, message mqtt.Message)) error {
	if n.s.Topic == "" {
		n.quit <- struct{}{}
		return errors.New("not all topic set")
	}

	if n.Logger != nil {
		mqtt.ERROR = errorLogger{n.Logger}
	}
	opts := mqtt.NewClientOptions().AddBroker(n.s.Server).SetClientID(n.s.ClientID)
	opts.SetUsername(n.s.Username)
	opts.SetPassword(n.s.Password)
	opts.OnConnect = func(c mqtt.Client) {
		if token := c.Subscribe(n.s.Topic, n.s.QoS, f); token.Wait() && token.Error() != nil {
			n.Logger.Errorln(token.Error())
			n.quit <- struct{}{}
			return
		}
		n.Logger.Info("ActiveMQ Connected/Reconnected...")
	}

	c := mqtt.NewClient(opts)
	go func() {
		if token := c.Connect(); token.Wait() && token.Error() != nil {
			n.Logger.Errorln(token.Error())
			n.quit <- struct{}{}
			return
		}
	}()

	select {
	case <-n.quit:
		return nil
	}
}

func (n *Notify) publish(c mqtt.Client, tx *TransferTx) {
	if c == nil {
		n.Logger.Error("publish client is nil")
		return
	}
	payload, err := json.Marshal(tx)
	if err != nil {
		n.Logger.Error(err)
		return
	}
	n.Logger.WithFields(log.Fields{
		"publish": n.p.Topic,
	}).Info(string(payload))

	c.Publish(n.p.Topic, n.p.QoS, false, string(payload))
}

func (n *Notify) publishToBlockTopic(c mqtt.Client, tx *TransferTx, block int64) {
	if c == nil {
		n.Logger.Error("publish client is nil")
		return
	}
	payload, err := json.Marshal(tx)
	if err != nil {
		n.Logger.Error(err)
		return
	}
	var topic string
	if tx.To == nil {
		topic = fmt.Sprintf("%sContractCreate", n.p.PrefixTopic)
	} else {
		topic = fmt.Sprintf("%s%s/%d", n.p.PrefixTopic, strings.ToLower(tx.To.String()[2:]), block)
	}

	n.Logger.WithFields(log.Fields{
		"publish": topic,
	}).Info(string(payload))

	c.Publish(topic, n.p.QoS, false, string(payload))
}

func (n *Notify) getPublishClient() (mqtt.Client, error) {
	opts := mqtt.NewClientOptions().AddBroker(n.p.Server).SetClientID(n.p.ClientID)
	opts.SetUsername(n.p.Username)
	opts.SetPassword(n.p.Password)
	c := mqtt.NewClient(opts)

	go func() {
		if token := c.Connect(); token.Wait() && token.Error() != nil {
			n.Logger.Errorln(token.Error())
			n.quit <- struct{}{}
			return
		}
	}()

	return c, nil
}
