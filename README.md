## NewChainNotify

This is a commandline client for the NewChain blockchain.
It's designed to be easy-to-use. It contains the following:
* Subscribe topic `RawTransaction` and publish to `Pending`
* Subscribe topic `Pending` and publish to `Transfer`
* Monitor blocks from JSON RPC URL and publish to `PrefixTopic/<address>/<ConfirmedBlockNumber>`

## Reference material

1. [debug_traceTransaction](https://github.com/ethereum/go-ethereum/wiki/Management-APIs#debug_tracetransaction)
2. [blockscout](https://github.com/poanetwork/blockscout)

## QuickStart

### Download from releases

Binary archives are published at https://release.cloud.diynova.com/newton/NewChainNotify/.

### Building the source

#### Windows

install command

```bash
git clone https://github.com/newtonproject/newchain-notify.git && cd newchain-notify && make install
```

run NewChainNotify

```bash
%GOPATH%/bin/newchain-notify.exe
```

#### Linux or Mac

install:

```bash
git clone https://github.com/newtonproject/newchain-notify.git && cd newchain-notify && make install
```
run NewChainNotify

```bash
$GOPATH/bin/newchain-notify
```

### Usage

#### Help

Use command `NewChainNotify help` to display the usage.

```bash
Usage:
  NewChainNotify [flags]
  NewChainNotify [command]

Available Commands:
  help        Help about any command
  init        Initialize config file
  monitor     Run as monitor server, subscribe from RPC URL and publish to transferN
  pending     Run as pending server, subscribe from RawTransaction and publish to transfer0
  transfer    Run as transfer server
  version     Get version of NewChainNotify CLI

Flags:
  -c, --config path        The path to config file (default "./config.toml")
  -h, --help               help for NewChainNotify
      --pid string         the client ID use for publish
  -p, --publish string     the publish topic to use
  -i, --rpcURL url         Geth json rpc or ipc url (default "https://rpc1.newchain.newtonproject.org")
      --sid string         the client ID use for subscribe
  -s, --subscribe string   the subscribe topic to use

Use "NewChainNotify [command] --help" for more information about a command.

```

#### Use config.toml

You can use a configuration file to simplify the command line parameters.

One available configuration file `config.toml` is as follows:


```conf
rpcurl = "https://rpc1.newchain.newtonproject.org/"

LogLevel = "info"
DelayBlock = 3 # for transfer and monitor
EnableTracer = true # enable tracer to trace transaction
#TracerTimeout = "5s" # the timeout to trace transaction, default: 5s
#TracerReexec = 128 # the number of blocks to be reexecuted, default: 128,

[Subscribe]
    Server = "url"
    Username = "username"
    Password = "password"
    #ClientID = "notify" # Default "notify"
    #QoS = 1 # 0, 1, 2, Default 1,
    #Topic = "RawTransaction"

[Publish]
    Server = "url"
    Username = "username"
    Password = "password"
    PrefixTopic = "newton/" # only for 0_address topic
    #ClientID = "notify" # Default "notify"
    #QoS = 1 # 0, 1, 2, Default 1,
    #Topic = "RawTransaction"
```

If you want to trace transactions's internal tx, set `EnableTracer = true`.

And Make sure `debug_traceTransaction` in the whitelist of the `NewChainGuard`,

The `debug_traceTransaction` will first use the blocks state cache,
then some number of blocks will be reexecuted base on `EnableTracer`.

You can use the flags `--cache`, `--cache.trie`, `--trie-cache-gens` 
of `geth` to change the size and number of blocks in cache.

### Pending
```bash
# Subscribe topic `RawTransaction` and publish to `Pending`
newchain-notify pending

# Subscribe topic `Pending` and publish to `Transfer`
newchain-notify transfer --id transfer

# Subscribe topic `Pending` and publish to `Transfer`
# Check with 3 block delay
newchain-notify transfer -b 3 --id transfer3

# Subscribe topic `Transfer2` and publish to `Transfer3`
newchain-notify transfer -b 3 --id transfer3 -s Transfer2 -p Transfer3
```

### Monitor

```bash
# Monitor RPC URL and publish to `PrefixTopic/<address>/<ConfirmedBlockNumber>`
newchain-notify monitor
```

* Tips:
    * You need to specify different IDs with `--id` when there are multiple programs are running at the same time. 