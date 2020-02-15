# go-alibp2p-chat

## 介绍

社交 Dapp Lib

## 启动参数

节点可以运行于 pc 或嵌入式环境中，启动时需要提供一些基本参数，具体如下

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
   --homedir value, -d value  home dir (default: "/tmp")
   --pwd value                passwd for subcmd attach
   --mailbox value            recv offline message
   --help, -h                 show help
   --version, -v              print the version
```

> 开发和调试时如果是本地单节点运行，则只关心 `--pwd` 和 `--mailbox` 两个参数即可，一个用于给 `rpc` 通道加密，另一个用于指定 `mailbox` 节点 `id` 来收离线消息
>
> 例如 : 
>    achat --mailbox 16Uiu2HAm91Fc9psqiTTiEpDEPqxc8cLqj4fhjUqPK6LwGZvLKx3j console