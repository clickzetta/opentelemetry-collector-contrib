# cz-otel — ClickZetta OpenTelemetry Collector 命令行工具

一个用于配置和运行自定义 OpenTelemetry Collector（集成 ClickZetta 导出插件）的命令行工具。它负责配置管理、Collector 配置文件生成以及守护进程生命周期管理，让你只需几条简单命令即可将遥测数据导入 ClickZetta Lakehouse。

## 安装

**一键安装（推荐）：**

```bash
curl -fsSL https://raw.githubusercontent.com/clickzetta/opentelemetry-collector-contrib/main/exporter/clickzettaexporter/cz-otel/scripts/install.sh | bash
```

安装指定版本：

```bash
CZ_OTEL_VERSION=0.1.0 curl -fsSL https://raw.githubusercontent.com/clickzetta/opentelemetry-collector-contrib/main/exporter/clickzettaexporter/cz-otel/scripts/install.sh | bash
```

**手动安装：**

1. 从发布页面下载适合你系统的平台包。
2. 解压：`tar -xzf cz-otel-<version>-<os>-<arch>.tar.gz`
3. 将二进制文件复制到 `~/.cz-otel/bin/` 并添加到 PATH。

安装无需 root 权限。

## 快速开始

```bash
# 1. 交互式配置 — 提示输入服务地址、凭据、工作空间等
cz-otel config init

# 2. 以后台守护进程方式启动 Collector
cz-otel start

# 3. 查看 Collector 状态
cz-otel status
```

Collector 默认在 gRPC（端口 4317）和 HTTP（端口 4318）上接收 OTLP 数据。

## 命令参考

| 命令                              | 说明                                         |
|-----------------------------------|----------------------------------------------|
| `cz-otel config init`            | 交互式配置向导，设置所有配置字段             |
| `cz-otel config set <key> <value>` | 设置单个配置值                             |
| `cz-otel config get <key>`       | 获取已存储的配置值                           |
| `cz-otel config list`            | 列出所有配置值（密码已脱敏）                 |
| `cz-otel config validate`        | 验证所有必填字段是否已配置且有效             |
| `cz-otel config generate`        | 根据已存储的配置生成 Collector YAML 配置文件  |
| `cz-otel start`                  | 以后台守护进程方式启动 Collector             |
| `cz-otel start --foreground`     | 在前台运行 Collector                         |
| `cz-otel stop`                   | 停止正在运行的 Collector 守护进程            |
| `cz-otel restart`                | 使用当前配置重启 Collector                   |
| `cz-otel status`                 | 检查 Collector 是否正在运行（PID、运行时间） |
| `cz-otel logs`                   | 显示 Collector 最近 50 行日志                |
| `cz-otel logs --follow`          | 实时流式输出 Collector 日志                  |
| `cz-otel version`                | 打印 CLI 版本、Collector 版本和平台信息      |
| `cz-otel --help`                 | 显示所有命令的帮助信息                       |

## 配置

所有配置存储在 `~/.cz-otel/config.yaml`（macOS/Linux）或 `%APPDATA%\cz-otel\config.yaml`（Windows）。

| 配置项               | 必填   | 默认值         | 说明                                   |
|----------------------|--------|----------------|----------------------------------------|
| `service`            | 是     | —              | ClickZetta 服务地址                    |
| `username`           | 是     | —              | ClickZetta 用户名                      |
| `password`           | 是     | —              | ClickZetta 密码                        |
| `workspace`          | 是     | —              | 目标工作空间名称                       |
| `virtual_cluster`    | 是     | —              | 虚拟集群名称                           |
| `instance`           | 是     | —              | 实例名称                               |
| `schema`             | 否     | `public`       | 目标 Schema                            |
| `protocol`           | 否     | `https`        | 连接协议                               |
| `logs_table_name`    | 否     | `otel_logs`    | 日志数据表名                           |
| `traces_table_name`  | 否     | `otel_traces`  | 链路追踪数据表名                       |
| `metrics_table_name` | 否     | `otel_metrics` | 指标数据表名                           |
| `create_schema`      | 否     | `true`         | 如果 Schema 不存在则自动创建           |

配置文件在 Unix 系统上以 `0600` 权限存储，以保护凭据安全。

## 从源码构建

### 前置条件

- Go 1.22+
- OCB（OpenTelemetry Collector Builder）— 安装方式：
  ```bash
  go install go.opentelemetry.io/collector/cmd/builder@latest
  ```

### 构建目标

```bash
# 为当前平台构建 CLI 二进制文件
make build

# 使用 OCB 构建 Collector 二进制文件
make collector

# 为当前平台创建分发包（tar.gz）
make package

# 交叉编译并打包所有支持的平台
make package-all

# 运行测试
make test

# 清理构建产物
make clean
```

构建时可以覆盖版本号：
```bash
make build VERSION=1.2.3
make package VERSION=1.2.3
```

## 平台支持

| 平台            | 架构         | 包名                                      |
|-----------------|--------------|-------------------------------------------|
| macOS           | amd64        | `cz-otel-<version>-darwin-amd64.tar.gz`   |
| macOS           | arm64 (M1+)  | `cz-otel-<version>-darwin-arm64.tar.gz`   |
| Linux           | amd64        | `cz-otel-<version>-linux-amd64.tar.gz`    |
| Linux           | arm64        | `cz-otel-<version>-linux-arm64.tar.gz`    |
| Windows         | amd64        | `cz-otel-<version>-windows-amd64.tar.gz`  |

## 目录结构

安装完成后，会创建以下目录结构：

```
~/.cz-otel/
├── bin/
│   ├── cz-otel                    # CLI 二进制文件
│   └── otelcol-clickzetta         # Collector 二进制文件
├── logs/
│   └── collector.log              # Collector 日志输出
├── config.yaml                    # 用户配置文件
├── otel-collector-config.yaml     # 生成的 Collector 配置文件
└── collector.pid                  # PID 文件（运行时存在）
```

Windows 系统的基础目录为 `%APPDATA%\cz-otel\`。
