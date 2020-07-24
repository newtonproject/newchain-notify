package cli

import (
	"github.com/spf13/cobra"
)

func (cli *CLI) buildRootCmd() {
	if cli.rootCmd != nil {
		cli.rootCmd.ResetFlags()
		cli.rootCmd.ResetCommands()
	}

	rootCmd := &cobra.Command{
		Use:              cli.name,
		Short:            cli.name + " is a commandline client for the NewChain blockchain",
		Run:              cli.help,
		PersistentPreRun: cli.setup,
	}
	cli.rootCmd = rootCmd

	// Global flags
	rootCmd.PersistentFlags().StringVarP(&cli.config, "config", "c", defaultConfigFile, "The `path` to config file")
	rootCmd.PersistentFlags().StringP("rpcURL", "i", defaultRPCURL, "Geth json rpc or ipc `url`")

	rootCmd.PersistentFlags().StringP("subscribe", "s", "", "the subscribe topic to use")
	rootCmd.PersistentFlags().StringP("publish", "p", "", "the publish topic to use")
	rootCmd.PersistentFlags().String("sid", "", "the client ID use for subscribe")
	rootCmd.PersistentFlags().String("pid", "", "the client ID use for publish")

	// Basic commands
	rootCmd.AddCommand(cli.buildVersionCmd()) // version
	rootCmd.AddCommand(cli.buildInitCmd())    // init

	rootCmd.AddCommand(cli.buildPendingCmd())  // pending
	rootCmd.AddCommand(cli.buildTransferCmd()) // run

	rootCmd.AddCommand(cli.buildMonitorCmd()) // monitor

}
