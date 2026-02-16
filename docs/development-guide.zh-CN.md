# go-achat-node 开发指南

## 1. 项目简介
`go-achat-node` 是一个基于 `go-alibp2p` 的 P2P Chat/社交 DApp 节点库与示例节点程序，提供：

- P2P 消息收发（普通消息、群相关能力的底层实现）
- 离线消息 Mailbox（通过 P2P 协议写入/查询/清理）
- 本地 RPC 网关（HTTP + WebSocket），便于桌面/移动端或脚本接入
- 本地持久化（LevelDB）

仓库包含两部分：

- 根目录：核心库（Go module：`github.com/cc14514/go-achat-node`，Go 1.19）
- `app/achat`：示例/参考用的节点可执行程序（Go module：`achat`，Go 1.19，提供 CLI、console、attach）

## 2. 架构说明

### 2.1 组件划分

- P2P 网络层（`go-alibp2p`）
  - 负责节点发现、连接、请求/响应式消息投递、handler 注册等
  - 主要协议 ID（见 `service.go`）：
    - 普通消息：`/chat/normal/0.0.1`
    - 群消息：`/chat/group/0.0.1`
    - Mailbox 写入：`/chat/mailbox/put/0.0.1`
    - Mailbox 查询：`/chat/mailbox/query/0.0.1`
    - Mailbox 清理：`/chat/mailbox/clean/0.0.1`
    - 群相关（Mailbox 内维护）：`/chat/mailbox/group/update/0.0.1`、`/chat/mailbox/group/member/0.0.1` 等

- ChatService（核心服务）
  - 入口：`NewChatService(ctx, myid, homedir, p2pservice)`
  - 启动：`ChatService.Start()`
    - 注册普通消息 handler
    - 启动 mailbox 子服务（本地 LevelDB + P2P handler）
  - 发送：`ChatService.SendMsg(msg)`
    - 优先直连投递到 `to.Peerid()`
    - 失败时 fallback 投递到 `to.Mailid()`（离线邮箱）

- Mailbox（离线消息与群数据）
  - 存储：LevelDB，位于 `${homedir}/mailbox`
  - 功能：put（写入）、query（查询）、clean（清理）

- RPC Server（本地网关，`rpc/`）
  - 监听：`127.0.0.1:${rpcport}`（仅本机回环，见 `rpc/server.go`）
  - HTTP：`POST /rpc`（JSON-RPC 风格）
  - WebSocket：`/chat`（收消息推送）
  - 鉴权：`auth(pwd)` 生成 token，后续 HTTP/WS 需携带 token
    - 注意：当启动参数 `--pwd` 为空时，`auth` 会直接放行

- CLI 示例程序（`app/achat/cmd/achat`）
  - `console`：启动节点 + 自动 attach 进入交互 shell
  - `attach`：连接到本地 RPC 并进入交互 shell
  - `bootnode`：以 bootnode 模式启动（代码中会关闭 discover 并清空 bootnodes）

### 2.2 默认端口与访问路径

- P2P 端口：`--port` 默认 `24000`
- RPC 端口：`--rpcport` 默认 `9990`
- RPC 实际地址（仅本机）：
  - HTTP：`POST http://localhost:${rpcport}/rpc`
  - WebSocket：`ws://localhost:${rpcport}/chat`

## 3. 目录结构

```text
go-achat-node/
  README.md
  GROUP.md
  go.mod
  go.sum

  types.go
  service.go
  mailbox.go
  mailbox_group.go

  rpc/
    server.go
    types.go
    user.go
    group.go

  ldb/
    database.go
    interface.go

  app/achat/
    go.mod
    go.sum
    cmd/achat/
      main.go
      console.go
```

## 4. 依赖说明

### 4.1 运行时依赖

- Go：`1.19`
- LevelDB：通过 `goleveldb` 使用本地文件存储（不需要外置数据库）
- 网络：需要能够进行 libp2p 连接（NAT/防火墙会影响直连效果）

### 4.2 关键第三方库（节选）

- `github.com/cc14514/go-alibp2p`：P2P 服务实现
- `github.com/syndtr/goleveldb`：本地 KV 存储
- `github.com/tendermint/go-amino`：编解码
- `golang.org/x/net/websocket`：RPC 的 WS 通道
- `github.com/urfave/cli`：示例程序 CLI 框架（仅 `app/achat`）

## 5. 运行方式（开发/调试）

> RPC 仅绑定 `127.0.0.1`，默认只允许本机访问。

### 5.1 启动节点（console 模式）

在示例程序模块目录运行：

```bash
cd app/achat
go run ./cmd/achat --pwd 123456 console
```

常用可选参数（带默认值）：

- `--rpcport 9990`：RPC 端口
- `--port 24000`：P2P 端口
- `--homedir /tmp` 或 `-d /tmp`：数据目录（LevelDB 会落在其子目录）
- `--mailbox <peerid>`：离线消息“邮箱节点 id”（作为 JID 的 mailbox 部分使用）
- `--bootnodes a,b,c`：以逗号分隔覆盖默认 bootnodes
- `--networkid 1`：网络隔离 id

建议本地多节点调试时显式区分端口与 `--homedir`：

```bash
# 节点 A
go run ./cmd/achat --pwd 123456 --port 24001 --rpcport 9991 --homedir /tmp/achat-a console

# 节点 B
go run ./cmd/achat --pwd 123456 --port 24002 --rpcport 9992 --homedir /tmp/achat-b console
```

### 5.2 attach 到已运行节点

```bash
cd app/achat
go run ./cmd/achat --pwd 123456 --rpcport 9990 attach
```

`attach` 会先调用 `auth` 获取 token，然后建立 WebSocket（`/chat`）并发送 `open`。

## 6. 构建方式

### 6.1 构建示例节点程序（achat）

```bash
cd app/achat
go build -o achat ./cmd/achat
```

运行：

```bash
./achat --pwd 123456 console
```

### 6.2 测试

- 根模块（核心库）：
  ```bash
  go test ./...
  ```
- `app/achat`（示例程序模块）：
  ```bash
  cd app/achat
  go test ./...
  ```

## 7. 开发流程（建议）

### 7.1 本地调试链路（推荐）

1. 启动一个节点（console），设置 `--pwd`，确认 RPC 可访问：
   - HTTP：`POST http://localhost:${rpcport}/rpc`
   - WS：`ws://localhost:${rpcport}/chat`
2. 调用 `auth` 获取 token
3. 调用 `myid` 获取本节点 JID/peerid
4. 启动第二个节点，重复上述步骤
5. 使用 `sendmsg` 或 console 的 `opensession` 发送消息
6. 用 WS 通道验证实时消息；离线场景结合 Mailbox 的 query/clean 逻辑验证

### 7.2 数据落盘位置（排查问题常用）

假设 `--homedir /tmp/achat-a`：

- 离线消息：`/tmp/achat-a/mailbox`（LevelDB）
- 用户信息：`/tmp/achat-a/user`（LevelDB）
- 群信息缓存：`/tmp/achat-a/group`（LevelDB）

## 8. RPC 接口说明

### 8.1 地址

- HTTP：`http://localhost:${rpcport}/rpc`
- WebSocket：`ws://localhost:${rpcport}/chat`

### 8.2 鉴权：auth

请求：

```json
{
  "id": "uuid",
  "method": "auth",
  "params": ["123456"]
}
```

响应（成功返回 token）：

```json
{
  "id": "uuid",
  "result": "token-hex"
}
```

说明：

- 启动参数 `--pwd` 为空时，`auth` 会直接成功并发 token
- 后续请求需携带 `token`（HTTP 请求体字段），WS 在首包 `open` 中携带 token

### 8.3 myid

```json
{
  "id": "uuid",
  "token": "token-hex",
  "method": "myid"
}
```

### 8.4 sendmsg

`params = [jid, content...]`，多段 content 会在服务端用空格拼接：

```json
{
  "id": "uuid",
  "token": "token-hex",
  "method": "sendmsg",
  "params": ["<to-jid>", "hello", "world"]
}
```

### 8.5 conns

返回当前 p2p 连接信息（直连、relay 等）：

```json
{
  "id": "uuid",
  "token": "token-hex",
  "method": "conns"
}
```

### 8.6 user_*（namespace 路由）

路由规则：`user_put` => namespace=`user`，fn=`put`。

- `user_put`：写入用户/好友或群信息（存 LevelDB）
- `user_get`：按 id 查询
- `user_del`：删除
- `user_query`：全量查询

### 8.7 group_create

`group_create` 会调用底层 Mailbox 的 group update，并将群信息写入 `user` 表作缓存。示例见 `GROUP.md`。

### 8.8 WebSocket 收消息

连接后首包需发送（method 固定为 `open`，并携带 token）：

```json
{
  "id": "uuid",
  "method": "open",
  "token": "token-hex"
}
```

随后服务端会：

- 返回一条系统消息表示 open 成功/失败
- 推送离线消息（如果有）
- 持续推送实时消息

## 9. 常见问题（FAQ）

### Q1：为什么从别的机器访问不了 RPC？
RPC Server 监听在 `127.0.0.1:${rpcport}`，只允许本机访问。需要远程访问时通常使用 SSH 端口转发，或自行修改监听地址（本指南不涉及改代码）。

### Q2：auth 总是成功或不生效？
当启动参数 `--pwd` 为空时，服务端 `auth` 会直接放行并发 token。请确保启动时设置了 `--pwd`，并在请求里传一致的密码。

### Q3：消息发不出去/收不到消息？
排查顺序建议：

1. 两端 `--networkid` 是否一致
2. P2P 端口 `--port` 是否被占用/被防火墙拦截
3. bootnodes 是否可达（或用 `--bootnodes` 指向可达节点）
4. NAT/运营商网络导致无法直连时，可能依赖 relay（连接质量受限）
5. 通过 `conns` 查看连接拓扑与 peer 地址

### Q4：离线消息为什么不工作？
离线投递依赖“收件人 JID 的 mailbox 部分（Mailid）”：

- JID 由 `peerid + mailboxid` 拼接（见 `types.go` 的 JID 规则与 `JID.Mailid()`）
- 发送失败时才会 fallback 到 `to.Mailid()` 的 mailbox put 协议

因此需要：

- 目标 JID 里包含有效 mailbox peerid
- 对应 mailbox 节点在线并运行同协议 handler（`ChatService.Start()` 会启动 mailbox 服务）

### Q5：LevelDB 报错/打不开？
常见原因是同一个 `--homedir` 被多个进程同时使用（LevelDB 文件锁冲突）。为每个节点配置独立 `--homedir`。

## 10. 参考文件

- `README.md`：启动参数、RPC/WS 示例
- `GROUP.md`：`group_create` 示例
- `app/achat/cmd/achat/main.go`：CLI 参数与启动流程
- `rpc/server.go`：RPC 监听地址、路由与 token 校验
- `service.go`、`mailbox.go`：P2P 协议与离线消息实现
