# Server 快速开始

本文档帮助你在最短时间内完成 Server 的部署和首次启动。

## 下载

从 [GitHub Releases](https://github.com/chainreactors/malice-network/releases/latest) 下载对应平台的 Server 二进制文件。

文件名格式为 `malice_network_[os]_[arch]`，例如：

- `malice_network_linux_amd64` — Linux x86_64
- `malice_network_darwin_arm64` — macOS Apple Silicon
- `malice_network_windows_amd64.exe` — Windows x86_64

!!! tip "网络问题"
    国内服务器下载 GitHub release 可能超时，建议配置代理：
    ```bash
    export http_proxy="http://127.0.0.1:1080"
    export https_proxy="http://127.0.0.1:1080"
    ```

## 首次启动

```bash
./malice-network -i <公网IP>
```

!!! important "IP 设置"
    `-i` 参数需要设置为 Client 可访问到的 IP 地址。公网服务器设置为公网 IP，内网环境设置为内网 IP。

首次启动后，Server 会自动完成：

1. 生成默认配置文件 `config.yaml`
2. 生成 CA 证书和加密密钥
3. 生成 Listener 凭证 `listener.auth`
4. 生成管理员凭证 `admin_<ip>.auth`
5. 启动 gRPC 服务（默认端口 `5004`）
6. 启动默认 Listener 和 Pipeline（TCP:5001, HTTP:8080）
7. 如果配置了 SaaS，自动编译对应 Implant

!!! warning "凭证安全"
    `.auth` 文件是认证凭证，请妥善保管。将 `admin_<ip>.auth` 发给操作员用于 Client 登录。

## 使用 quickstart 向导

首次使用或需要重新配置时，可以使用交互式向导：

```bash
./malice-network --quickstart
```

向导会引导完成 IP、端口、构建源等基础配置。

## 使用安装脚本（Linux）

!!! info "安装脚本会自动完成 Docker 安装、Server/Client 下载、构建镜像拉取和 systemd 配置"

```bash
curl -L "https://raw.githubusercontent.com/chainreactors/malice-network/master/install.sh" | sudo bash
```

安装脚本会交互式询问：

- **安装路径**：默认 `/iom`
- **IP 地址**：自动检测，可手动修改

安装完成后 Server 会以 systemd 服务运行。

## 防火墙配置

确保以下端口对 Client 可达：

| 端口 | 用途 |
|------|------|
| `5004` | gRPC（Client ↔ Server） |
| `5001` | TCP Pipeline（Implant 上线） |
| `8080` | HTTP Pipeline（Implant 上线） |

!!! tip "按需开放"
    仅需开放实际使用的 Pipeline 端口。如果不使用 HTTP Pipeline，无需开放 8080。

## 验证启动

Server 启动成功后，使用 Client 登录验证：

```bash
./iom login admin_<ip>.auth
```

登录后执行：

```bash
status          # 查看 Server 状态
listener        # 查看 Listener 列表
pipeline        # 查看 Pipeline 列表
```

## 启动模式

| 模式 | 命令 | 场景 |
|------|------|------|
| 标准启动 | `./malice-network -i <ip>` | Server + Listener 一起运行 |
| 仅 Server | `./malice-network --server-only` | Listener 独立部署时 |
| 仅 Listener | `./malice-network --listener-only` | 独立 Listener 节点 |
| 守护进程 | `./malice-network --daemon` | 后台运行 |
| 交互向导 | `./malice-network --quickstart` | 首次配置 |

## 下一步

- [Server 配置参考](index.md) — config.yaml 完整配置说明
- [Listener 架构](listeners.md) — Pipeline 配置与 TLS 机制
- [构建系统](build.md) — 构建源配置（Docker/Action/SaaS）
- [部署操作指南](../operations/deployment.md) — 详细部署步骤（VSCode GUI 等）
