package cli

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/newtonproject/newchain-notify/notify"
)

func (cli *CLI) buildTransferCmd() *cobra.Command {
	transferCmd := &cobra.Command{
		Use:   "transfer",
		Short: "Run as transfer server",
		DisableFlagsInUseLine: true,
		Run: func(cmd *cobra.Command, args []string) {
			logger := logrus.New()
			logger.Out = os.Stdout
			logLevel := viper.GetString("LogLevel")
			if logLevel == "" {
				logLevel = "info"
			}
			level, err := logrus.ParseLevel(logLevel)
			if err != nil {
				logger.Error(err)
				return
			}
			logger.SetLevel(level)

			delayBlock := viper.GetInt64("DelayBlock")

			s, err := getTransferNotifyConfig("Subscribe", delayBlock)
			if err != nil {
				logger.Errorln(err)
				return
			}
			p, err := getTransferNotifyConfig("Publish", delayBlock)
			if err != nil {
				logger.Errorln(err)
				return
			}
			n, err := notify.NewTransferNotify(s, p, cli.rpcURL, delayBlock, logger)
			if err != nil {
				logger.Errorln(err)
				return
			}

			b, err := json.MarshalIndent(n, "", "\t")
			if err != nil {
				logger.Errorln(err)
				return
			}
			logger.Printf("ActiveMQ Info is as follow: \n%s", b)

			if err := n.Run(); err != nil {
				logger.Errorln(err)
				return
			}
		},
	}

	transferCmd.Flags().Int64P("delay", "d", 0, "The number of delayed blocks")
	viper.BindPFlag("DelayBlock", transferCmd.Flags().Lookup("delay"))

	return transferCmd
}

func getTransferNotifyConfig(p string, delayBlock int64) (*notify.NotifyConfig, error) {
	server := viper.GetString(p + ".Server")
	if server == "" {
		return nil, fmt.Errorf("%s server is empty", p)
	}
	username := viper.GetString(p + ".Username")
	if username == "" {
		return nil, fmt.Errorf("%s username is empty", p)
	}
	password := viper.GetString(p + ".Password")
	if password == "" {
		return nil, fmt.Errorf("%s password is empty", p)
	}
	clientID := viper.GetString(p + ".ClientID")
	if clientID == "" {
		clientID = fmt.Sprintf("NotifyTransfer%s%d", p, delayBlock+1)
	}
	qos := viper.GetInt(p + ".QoS")
	if !(qos == 0 || qos == 1 || qos == 2) {
		return nil, fmt.Errorf("%s QoS only 0,1,2", p)
	}
	topic := viper.GetString(p + ".Topic")
	if topic == "" {
		if p == "Subscribe" {
			topic = "Pending"
		} else if p == "Publish" {
		} else {
			return nil, fmt.Errorf("%s topic is empty", p)
		}
	}

	prefixTopic := viper.GetString(p + ".PrefixTopic")

	return &notify.NotifyConfig{
		Server:      server,
		Username:    username,
		Password:    password,
		ClientID:    clientID,
		QoS:         byte(qos),
		Topic:       topic,
		PrefixTopic: prefixTopic,
	}, nil
}
