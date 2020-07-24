package cli

import (
	"os"

	"github.com/spf13/viper"
)

const defaultConfigFile = "./config.toml"
const defaultRPCURL = "https://rpc1.newchain.newtonproject.org"

func (cli *CLI) defaultConfig() {
	viper.BindPFlag("rpcURL", cli.rootCmd.PersistentFlags().Lookup("rpcURL"))
	viper.SetDefault("rpcURL", defaultRPCURL)

	viper.BindPFlag("Subscribe.Topic", cli.rootCmd.PersistentFlags().Lookup("subscribe"))
	viper.BindPFlag("Publish.Topic", cli.rootCmd.PersistentFlags().Lookup("publish"))
	viper.BindPFlag("Subscribe.ClientID", cli.rootCmd.PersistentFlags().Lookup("sid"))
	viper.BindPFlag("Publish.ClientID", cli.rootCmd.PersistentFlags().Lookup("pid"))

	viper.SetDefault("Subscribe.QoS", 1)
	viper.SetDefault("Publish.QoS", 1)

}

func (cli *CLI) setupConfig() error {

	//var ret bool
	var err error

	cli.defaultConfig()

	viper.SetConfigName(defaultConfigFile)
	viper.AddConfigPath(".")
	cfgFile := cli.config
	if cfgFile != "" {
		if _, err = os.Stat(cfgFile); err == nil {
			viper.SetConfigFile(cfgFile)
			err = viper.ReadInConfig()
		} else {
			// The default configuration is enabled.
			//fmt.Println(err)
			err = nil
		}
	} else {
		// The default configuration is enabled.
		err = nil
	}

	if rpcURL := viper.GetString("rpcURL"); rpcURL != "" {
		cli.rpcURL = rpcURL
	}

	return nil
}
