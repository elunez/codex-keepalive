# Codex 定时唤醒

CLIProxyAPI 的 Codex 账号定时唤醒插件。插件在配置时间实时读取宿主中的 Codex 账号，为每个可用账号发送指定次数的轻量唤醒请求，并在管理页保留可筛选、可分页的执行日志。

## 功能

- 通过 CLIProxyAPI 原生 `host.auth.list` 获取 Codex 账号，不依赖 Keeper、SQLite 或其他插件。
- 按每天的一个或多个固定时间执行，默认三频为 `07:00,12:15,17:30`。
- 支持在插件页面手动执行；关闭自动执行后，手动执行仍可使用。
- 每个账号、每次请求独立生成 `0～N` 秒随机延迟；范围允许时会优先使用不重复的秒数。
- 支持限制并发请求数，避免大量账号同时访问上游。
- 不注册 `scheduler.pick`，不会停止用户调度、修改账号优先级或干预正常请求。
- 不读取或显示额度、费用和阈值。
- 每次唤醒向 Codex Responses API 发送固定的轻量消息 `hi`，使用流式响应确认请求完成，不保存对话内容。
- 执行日志保存到独立的 `logs.json`，最多保留最近 5000 条；管理页支持搜索、结果/触发方式筛选和服务端分页。
- 配置保存到独立的 `config.json`；通过 CPA“重新安装”更新插件时，即使宿主传入空配置，也会自动读取并恢复上一次配置。
- 页面只读取管理中心已保存的登录信息，不显示或持久化管理密钥。

## 插件配置

插件管理中只需要负责启用或停用插件；所有业务配置统一在“定时唤醒”页面右上角的“设置”按钮中维护，保存后立即热生效。

```yaml
plugins:
  codex-keepalive:
    enabled: true
    priority: 88
```

页面设置说明：

- `activation_enabled`：是否启用自动执行。
- `activation_times`：每日执行时间，使用英文逗号分隔的 `HH:mm`。
  页面提供三频预设 `07:00,12:15,17:30` 和四频预设 `07:00,12:15,17:30,23:45`。
- `activation_timezone`：IANA 时区。
- `activation_model`：唤醒请求使用的 Codex 模型。
- `activation_requests_per_run`：每个可用账号每轮请求次数，范围 `1～10`。
- `activation_random_delay_seconds`：随机延迟上限，范围 `0～3600`；`60` 表示在 `0～60` 秒内随机。
- `activation_concurrency`：同时执行的最大请求数，范围 `1～32`。

默认数据目录为 `~/.cli-proxy-api/plugins/codex-keepalive/`，其中 `config.json` 保存非敏感配置，`logs.json` 保存执行日志；不会保存账号凭据、token 或管理密钥。可通过未公开到插件表单的 `data_dir` 指定数据目录。

更新插件时直接在 CPA 插件管理中点击“重新安装”即可。只要数据目录未被删除，插件启动时会先读取 `config.json`，再用本次显式传入的配置覆盖对应字段，因此原有配置无需手动抄录。清空日志只会重置 `logs.json`，不会影响配置；若要恢复默认配置，可在插件管理中重新填写全部配置项。

## 构建

需要 Go 1.26.0 和目标平台可用的 C 编译器。CLIProxyAPI SDK 通过 Go Modules 获取。

```sh
make package VERSION=0.0.1
```

插件源码不包含架构相关逻辑，可分别为 Linux `amd64`、Linux `arm64`、macOS `amd64` 和 macOS `arm64` 构建。`c-shared` 使用 CGO，跨架构构建时需要对应的交叉编译器；仓库中的 GitHub Actions 使用对应架构的原生 runner 构建。

每次推送到 `main` 后，GitHub Actions 都会自动测试、构建并发布。第一次发布使用源码中的基准版本，之后自动递增补丁版本，例如 `v0.0.1`、`v0.0.2`、`v0.0.3`：

```sh
git push origin main
```

也可以手动推送符合 `v<major>.<minor>.<patch>` 格式的标签发布指定版本。Release 包含 CPA 插件商店要求的四个平台压缩包和 `checksums.txt`。压缩包根目录只有对应的 `codex-keepalive.so` 或 `codex-keepalive.dylib`。

## 安装

### 从插件商店安装

CLIProxyAPI 建议使用 `v7.2.142` 或更高版本。首次使用本仓库商店源时，在 `config.yaml` 的现有 `plugins` 节点中加入：

```yaml
plugins:
  enabled: true
  store-sources:
    - "https://raw.githubusercontent.com/elunez/codex-keepalive/main/registry.json"
```

重新加载配置后，在 CPA 管理中心打开“插件商店”，搜索“Codex 定时唤醒”或 `codex-keepalive`，点击安装。安装器会自动选择当前系统和架构，并把插件写入 `plugins.dir` 对应目录。

安装后在“插件管理”中启用插件即可；定时唤醒的业务参数请进入“定时唤醒”页面，点击右上角设置按钮进行配置。登录管理中心时需勾选“记住密码”，插件页面才能调用受保护的管理接口。

仓库已包含符合 CLIProxyAPI 官方商店规范的 `registry.json`。进入官方默认商店还需要在首个 GitHub Release 发布后，向 [`router-for-me/CLIProxyAPI-Plugins-Store`](https://github.com/router-for-me/CLIProxyAPI-Plugins-Store) 的 `registry.json` 提交相同插件条目。

### 手动安装

将动态库放到 CLIProxyAPI 对应系统和架构的插件运行目录，例如：

```text
plug/runtime/linux/amd64/codex-keepalive.so
plug/runtime/linux/arm64/codex-keepalive.so
plug/runtime/darwin/arm64/codex-keepalive.dylib
```

启用插件并重启或重新加载 CLIProxyAPI 后，管理菜单会出现“定时唤醒”。
