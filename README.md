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

节点提供 `http`(只接收 post 请求) 和 `websocket` 两种接口，URL如下：

* http://localhost:[rpcport]/rpc 
* ws://localhost:[rpcport]/chat

### HTTP

>用来发送给指令，完成交互，协议与 `JSONRPC` 相同；
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

#### auth

认证接口，返回一个 `token` 调用其他接口时使用，建立 `websocket` 长连接时也要提供正确的 `token`，参数为节点启动时的 `--pwd` 参数对应的值，加入 pwd = 123456 则有如下交互

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

#### sendmsg

向指定节点 `jid` 发送 `content` 消息, 即 `params = [jid,content]`,具体如下：

__请求：__

```
{
	"id": "efda2cb1-fa4c-431a-b6c3-655aafafb1d6",
	"token": "3fcbd15aa4556e80e46a651e84a2737214097f1c",
	"method": "sendmsg",
	"params": ["16Uiu2HAmN2eZ9DLJhccS1R49Qc1tpdGMdbC8uWwzUCUAfRpRvEvd", "hello"]
}
```

__响应：__

```
{"result":"success","Id":"efda2cb1-fa4c-431a-b6c3-655aafafb1d6"}
```

#### user

> 用户接口用来对用户信息进行增删改查，用户信息包含如下属性
>```
>{
>	"id": "用户JID",
>	"name": "用户名",
>	"comment": "备注",
>	"gender": 1 // 性别：0 女，1 男
>}
>```
>
> 提供如下操作：
>
> * user_put : 添加用户信息，可以用于修改 `自己` 的用户信息或者添加 `好友` 信息
> * user_get : 根据 user.id 获取用户信息
> * user_del : 根据 user.id 来删除用户信息
> * user_query : 获取全部用户信息

##### user_put

__请求：__

```
{
	"id": "b1b09207-b92a-410f-9aaf-bc486bb1565f",
	"token": "e379f924be7548b43c2f2273db9549e47c752872",
	"method": "user_put",
	"params": [{
		"comment": "测试账户",
		"gender": "1",
		"id": "16Uiu2HAmN2eZ9DLJhccS1R49Qc1tpdGMdbC8uWwzUCUAfRpRvEvd",
		"name": "test1"
	}]
}
```

__响应：__

```
{"result":"success","id":"b1b09207-b92a-410f-9aaf-bc486bb1565f"}
```


##### user_get

__请求：__

```
{
	"id": "b1b01749-c113-45b1-aa86-2b1461ab4bd3",
	"token": "e379f924be7548b43c2f2273db9549e47c752872",
	"method": "user_get",
	"params": ["16Uiu2HAmN2eZ9DLJhccS1R49Qc1tpdGMdbC8uWwzUCUAfRpRvEvd"]
}
```

__响应：__

```
{
	"result": {
		"id": "16Uiu2HAmN2eZ9DLJhccS1R49Qc1tpdGMdbC8uWwzUCUAfRpRvEvd",
		"name": "test1",
		"gender": 1,
		"comment": "测试账户"
	},
	"id": "b1b01749-c113-45b1-aa86-2b1461ab4bd3"
}
```

##### user_del

__请求：__

```
{
	"id": "b1b01749-c113-45b1-aa86-2b1461ab4bd3",
	"token": "e379f924be7548b43c2f2273db9549e47c752872",
	"method": "user_del",
	"params": ["16Uiu2HAmN2eZ9DLJhccS1R49Qc1tpdGMdbC8uWwzUCUAfRpRvEvd"]
}
```

__响应：__

```
{"result":"success","id":"b1b09207-b92a-410f-9aaf-bc486bb1565f"}
```


##### user_query

__请求：__

```
{
	"id": "a2430ddb-16f2-49e1-99ca-6589dff93750",
	"token": "e379f924be7548b43c2f2273db9549e47c752872",
	"method": "user_query"
}
```

__响应：__

```
{
	"result": [{
		"id": "16Uiu2HAmN2eZ9DLJhccS1R49Qc1tpdGMdbC8uWwzUCUAfRpRvEvd",
		"name": "test1",
		"gender": 1,
		"comment": "测试账户"
	}],
	"id": "a2430ddb-16f2-49e1-99ca-6589dff93750"
}
```


### WEBSOCKET

客户端与节点保持长连接，用来收消息

当第一次打开连接时需要以如下格式来报文发送 `token` 以完成 `openstream` 操作

```
{
	"id": "8f2930d0-8e64-42d2-b2a9-e4ec6dc78f67",
	"method": "open",
	"token": "c9074e7a1255926709f5e2b24e1ee6dbd6c34874"
}
```

返回

```
{
	"envelope": {
		"id": "2543ac95-0769-4501-a6f6-5b0df1edf031",
		"type": "4",
		"ct": "1581754241"
	},
	"payload": {
		"attrs": [{
			"key": "method",
			"val": "open"
		}, {
			"key": "result",
			"val": "success"
		}]
	},
	"vsn": "0.0.2"
}
```

第一个返回报文的 `envelope.type == 4`, 表示 SYS 类型的消息 


成功时 

    `payload.attrs[1] == {"key":"result","val":"success"}`

失败时

    `payload.attrs[1] == {"key":"error","val":"# error reason #"}`,并且会关闭连接通道
    

当 `open stream` 成功了以后，这个长连接通道上就会得到如下格式的聊天消息

```
{
	"envelope": {
		"id": "465e1f04-99f7-442b-bac5-da33aa7e7caa",
		"from": "16Uiu2HAmN2eZ9DLJhccS1R49Qc1tpdGMdbC8uWwzUCUAfRpRvEvd",
		"to": "16Uiu2HAkvsWx5Byt8RCXs2ScrmPCZteHjcdQhzxdHCVYTjtYxPYr",
		"type": "1",
		"ct": "1581756469"
	},
	"payload": {
		"content": "hello world"
	},
	"vsn": "0.0.2"
}
```