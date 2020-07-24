package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/newtonproject/newchain-notify/notify"
	"github.com/newtonproject/newchain-notify/tracer"
)

func (cli *CLI) buildMonitorCmd() *cobra.Command {
	pendingCmd := &cobra.Command{
		Use:                   "monitor",
		Short:                 "Run as monitor server, subscribe from RPC URL and publish to transferN",
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

			rpcUrl := cli.rpcURL
			blockDelay := viper.GetInt64("DelayBlock")

			p, err := getMonitorNotifyConfig("Publish", blockDelay)
			if err != nil {
				logger.Errorln(err)
				return
			}

			enableTracer := viper.GetBool("EnableTracer")

			traceConfig := &tracer.TraceConfig{}
			TracerTimeout := viper.GetString("TracerTimeout")
			if TracerTimeout != "" {
				if TracerTimeoutDuration, err := time.ParseDuration(TracerTimeout); err != nil {
					logger.Errorln(err)
					return
				} else if TracerTimeoutDuration > 0 {
					traceConfig.Timeout = &TracerTimeout
				}
			}

			TracerReexec := uint64(viper.GetInt64("TracerReexec"))
			if TracerReexec > 0 {
				traceConfig.Reexec = &TracerReexec
			}

			n, err := notify.NewMonitorNotify(p, rpcUrl, blockDelay, enableTracer, traceConfig, logger)
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
	return pendingCmd
}

func getMonitorNotifyConfig(p string, delayBlock int64) (*notify.NotifyConfig, error) {
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
		clientID = fmt.Sprintf("NotifyMonitor%s%d", p, delayBlock+1)
	}
	qos := viper.GetInt(p + ".QoS")
	if !(qos == 0 || qos == 1 || qos == 2) {
		return nil, fmt.Errorf("%s QoS only 0,1,2", p)
	}

	prefixTopic := viper.GetString(p + ".PrefixTopic")

	return &notify.NotifyConfig{
		Server:      server,
		Username:    username,
		Password:    password,
		ClientID:    clientID,
		QoS:         byte(qos),
		PrefixTopic: prefixTopic,
	}, nil
}
