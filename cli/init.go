package cli

import (
	"fmt"

	"github.com/ethereum/go-ethereum/console"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func (cli *CLI) buildInitCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "init",
		Short: "Initialize config file",
		DisableFlagsInUseLine: true,
		Run: func(cmd *cobra.Command, args []string) {

			fmt.Println("Initialize config file")

			prompt := fmt.Sprintf("Enter file in which to save (%s): ", defaultConfigFile)
			configPath, err := console.Stdin.PromptInput(prompt)
			if err != nil {
				fmt.Println("PromptInput err:", err)
			}
			if configPath == "" {
				configPath = defaultConfigFile
			}
			cli.config = configPath

			rpcURLV := viper.GetString("rpcURL")
			prompt = fmt.Sprintf("Enter geth json rpc or ipc url (%s): ", rpcURLV)
			rpcURL, err := console.Stdin.PromptInput(prompt)
			if err != nil {
				fmt.Println("PromptInput err:", err)
			}
			if rpcURL == "" {
				rpcURL = rpcURLV
			}
			viper.Set("rpcURL", rpcURL)

			err = viper.WriteConfigAs(configPath)
			if err != nil {
				fmt.Println("WriteConfig:", err)
			} else {
				fmt.Println("Your configuration has been saved in ", configPath)
			}
		},
	}

	return cmd
}
