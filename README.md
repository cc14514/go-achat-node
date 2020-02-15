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
>    例如 : 
>
>        achat --mailbox 16Uiu2HAm91Fc9psqiTTiEpDEPqxc8cLqj4fhjUqPK6LwGZvLKx3j console

子命令 `console` 可以在调试时得到一个 `shell` ，也可以单独启动节点进程并以 `attach` 子命令登陆节点 `shell` 进行交互


## RPC接口

节点提供 `http` 和 `websocket` 两种接口

### HTTP

>用来发送给指令，完成交互，协议与 `JSONRPC` 雷同；
>失败时统一返回如下格式的报文：
>```
>{
>	"id": "req-uuid",
>	"error": {
>		"code": "xxxx",
>		"message": "xxxx"
>	}
>}
>```

1. auth

认证接口，参数为节点启动时的 `--pwd` 参数对应的值，加入 pwd = 123456 则有如下交互

__请求：__
```
{
	"id": "uuid",
	"method": "auth",
	"params": ["123456"]
}
```

__响应：__

```
{
	"id": "uuid",
	"result": "1193c3a40299a61192c062f937ff4d531e3e3629"
}
```



### WEBSOCKET

* 客户端与节点保持长连接，用来收消息
