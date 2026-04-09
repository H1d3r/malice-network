# 贡献指南

本文档介绍如何为 malice-network 项目进行开发和贡献。

## 参与方式

### Issue Reporter

通过深度使用 IoM 发现问题：

- 提交 bug 报告（附带复现步骤）
- 提出功能需求和改进建议
- 反馈不合理的设计和低级 bug
- 指出文档中的错误描述、歧义等

### Contributor

协助解决具体问题：

1. 分析并定位问题
2. 编写修复代码
3. 完成测试验证
4. 提交 Pull Request

### Core Contributor

参与新功能开发和架构优化：

1. 发起需求并讨论技术方案
2. 实现完整功能模块
3. 参与 Code Review 和迭代优化

## 环境配置

??? important "Go 开发环境"
    **版本要求**: Go >= 1.20

    ```bash
    go version
    ```

??? important "protobuf 环境"
    === "Linux"

        ```bash
        apt install -y protobuf-compiler
        protoc --version  # 确保版本 >= 3
        ```

    === "macOS"

        ```bash
        brew install protobuf
        protoc --version
        ```

    === "Windows"

        ```bash
        winget install protobuf
        protoc --version
        ```

    **protobuf Go 插件**:
    ```bash
    go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@v1.3.0
    go install google.golang.org/protobuf/cmd/protoc-gen-go@v1.34.1
    ```

## 项目设置

```bash
# Fork 并克隆仓库
git clone --recurse-submodules https://github.com/your-username/malice-network.git
cd malice-network
git remote add upstream https://github.com/chainreactors/malice-network.git

# 安装依赖
go mod tidy

# 编译验证
go build ./server/
go build ./client/
```

## Client 开发

### Command 开发

IoM 的命令开发基于四个步骤：**功能函数 → 命令包装 → 插件注册 → Cobra 定义**。

#### 1. 编写功能函数

```go
func Env(rpc clientrpc.MaliceRPCClient, session *core.Session) (*clientpb.Task, error) {
    task, err := rpc.Env(session.Context(), &implantpb.Request{
        Name: consts.ModuleEnv,
    })
    if err != nil {
        return nil, err
    }
    return task, err
}
```

#### 2. 编写命令包装函数

```go
func EnvCmd(cmd *cobra.Command, con *repl.Console) error {
    session := con.GetInteractive()
    task, err := Env(con.Rpc, session)
    if err != nil {
        return err
    }
    session.Console(task, string(*con.App.Shell().Line()))
    return nil
}
```

#### 3. 注册到 MAL 插件系统

```go
func RegisterEnvFunc(con *repl.Console) {
    con.RegisterImplantFunc(
        consts.ModuleEnv,
        Env,
        "benv",
        func(rpc clientrpc.MaliceRPCClient, sess *core.Session) (*clientpb.Task, error) {
            return Env(rpc, sess)
        },
        output.ParseKVResponse,
        output.FormatKVResponse)
}
```

#### 4. 定义 Cobra 命令

```go
envCmd := &cobra.Command{
    Use:   consts.ModuleEnv,
    Short: "List environment variables",
    RunE: func(cmd *cobra.Command, args []string) error {
        return EnvCmd(cmd, con)
    },
    Annotations: map[string]string{
        "depend": consts.ModuleEnv,
        "ttp":    "T1134",
    },
}
```

### Annotations 标注

| 标注 | 说明 |
|------|------|
| `depend` | 依赖的 Module 名称 |
| `ttp` | MITRE ATT&CK 技术 ID |

### 补全系统

使用 [carapace](https://github.com/carapace-sh/carapace) 实现动态补全：

```go
common.BindFlagCompletions(httpCmd, func(comp carapace.ActionMap) {
    comp["listener"] = common.ListenerIDCompleter(con)
    comp["cert-name"] = common.CertNameCompleter(con)
})
```

## Server 开发

### 扩展 Proto 协议

优先使用通用的 `Request` / `Response` proto：

```protobuf
message Request {
  string name = 1;
  string input = 2;
  repeated string args = 3;
  map<string, string> params = 4;
  bytes bin = 5;
}

message Response {
  string output = 1;
  string error = 2;
  map<string, string> kv = 3;
  repeated string array = 4;
}
```

如果通用 proto 无法满足，才考虑修改 proto：

1. 在 `proto/implant/implantpb/implant.proto` 添加新 message
2. 将 message 添加到 `Spite.body` oneof 中
3. 在 `helper/types/message.go` 添加对应常量
4. 在 `proto/services/clientrpc/service.proto` 添加 RPC 定义

!!! warning "Proto 修改规范"
    Proto 变更在 `external/IoM-go` 子模块内进行，不要手动编辑生成的 Go 代码。变更后需更新子模块引用并执行 `go mod tidy`。

### 添加 RPC Handler

**普通 RPC**：

```go
func (rpc *Server) NewCommand(ctx context.Context, req *implantpb.Request) (*clientpb.Task, error) {
    greq, err := newGenericRequest(ctx, req)
    if err != nil {
        return nil, err
    }
    ch, err := rpc.asyncGenericHandler(ctx, greq)
    if err != nil {
        return nil, err
    }
    go greq.HandlerAsyncResponse(ch, types.MsgResponse)
    return greq.Task.ToProtobuf(), nil
}
```

**流式 RPC**（实时输出）：

```go
func (rpc *Server) Execute(ctx context.Context, req *implantpb.ExecRequest) (*clientpb.Task, error) {
    greq, err := newGenericRequest(ctx, req)
    if err != nil {
        return nil, err
    }
    if !req.Realtime {
        ch, err := rpc.GenericHandler(ctx, greq)
        if err != nil {
            return nil, err
        }
        go greq.HandlerResponse(ch, types.MsgExec)
    } else {
        greq.Count = -1
        _, out, err := rpc.StreamGenericHandler(ctx, greq)
        if err != nil {
            return nil, err
        }
        go func() {
            for resp := range out {
                if resp.GetExecResponse().End {
                    greq.Task.Finish(resp, "")
                    break
                }
                greq.HandlerSpite(resp)
            }
        }()
    }
    return greq.Task.ToProtobuf(), nil
}
```

## PR 合并流程

1. **角色分配**：每个复杂功能分配一个 Maintainer 和至少一个 Assignee
2. **Review 流程**：Maintainer 完成后通知 Assignee 进行 review 和测试
3. **文档要求**：
   - PR 中附上测试截图和用法说明
   - 新功能需添加对应的 help 信息
   - 系统性功能 PR 通过后立即编写相关文档

## Pre-commit 检查

```bash
go vet ./...                            # 静态分析
go test ./... -count=1 -timeout 300s    # 测试
CGO_ENABLED=0 go build ./...            # 编译验证
```

## 相关文档

- [测试文档](testing.md) — 测试框架与规范
- [协议文档](protocol/) — Proto 协议规范
- [Client 架构](../client/) — Client 机制与设计
- [Server 架构](../server/) — Server 架构与配置
