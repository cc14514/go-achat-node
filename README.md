# go-alibp2p-chat

#### 介绍
社交 Dapp Lib

#### 使用说明

```
$> achat --help
NAME:
   achat - 基于 go-alibp2p 的 chat

USAGE:
   achat [global options] command [command options] [arguments...]

VERSION:
   0.0.1

AUTHOR:
   liangc <cc14514@icloud.com>

COMMANDS:
   attach   attach to console
   console  start with console
   help, h  Shows a list of commands or help for one command

GLOBAL OPTIONS:
   --rpcport PORT             RPC server listening PORT (default: 9990)
   --port value               service tcp port (default: 24000)
   --networkid value          network id (default: 1)
   --homedir value, -d value  home dir (default: "/tmp")
   --pwd value                passwd for subcmd attach
   --mailbox value            recv offline message
   --bootnodes value          bootnode list split by ','
   --help, -h                 show help
   --version, -v              print the version
```