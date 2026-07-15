# workbuddy-cli-proxy

把**腾讯 CodeBuddy**（`copilot.tencent.com`）封装成 [CLIProxyAPI](https://github.com/router-for-me/CLIProxyAPI)（CPA）插件。任何支持 OpenAI / Anthropic 协议的客户端（Claude Code、Cursor、Cline、SDK……）都能直接调用 CodeBuddy 背后的模型。

对 [Sliverkiss/cpa-plugin](https://github.com/Sliverkiss/cpa-plugin) 公开 `workbuddy.so` 的 clean-room 逆向重写，补齐了源码与多架构构建；workbuddy 的原始设计归属 Sliverkiss。

插件 ID：`workbuddy` · 模块：`github.com/WslzGmzs/workbuddy-cli-proxy`

## 工作原理

在 CPA 里注册为 `workbuddy` provider：

- **OAuth / 扫码登录** + token 刷新
- **手动 API Key** 凭据（上传 JSON 即可）
- 请求转发到 `copilot.tencent.com/v2/chat/completions`

## 模型

`glm-5.2` · `glm-5.1` · `glm-5v-turbo` · `kimi-k2.7` · `minimax-m3-pay` · `hy3` · `hy3-preview` · `hy3-preview-agent` · `deepseek-v4-pro` · `deepseek-v4-flash`

具体可用性以 CodeBuddy 账号权限为准。

## 安装

### A. 插件商店 / 在线安装（推荐）

仓库根目录提供符合 CPA 校验的 [`registry.json`](registry.json)（`schema_version: 1`，`install` 默认 `github-release`）。CPA 读到条目后，会去 **GitHub latest Release** 拉对应平台 zip。

#### 1）发布 Release 资产（安装前置）

打 tag 后 GitHub Actions 会产出：

- `workbuddy_<version>_<goos>_<goarch>.zip`（zip **根目录只有** `workbuddy.so` / `.dylib` / `.dll`）
- `checksums.txt`（sha256sum 格式）

```bash
git tag v0.2.0 && git push origin v0.2.0
```

没有 Release 资产时，商店能看到插件，但安装会失败。

#### 2）在 CPA 添加本仓库商店源（私有 / 自用，马上可测）

`config.yaml`：

```yaml
plugins:
  enabled: true
  dir: "plugins"
  # 额外商店源：指向 raw registry.json（GitHub / 自建 HTTP 均可）
  store-sources:
    - "https://raw.githubusercontent.com/WslzGmzs/workbuddy-cli-proxy/main/registry.json"
  configs:
    workbuddy: { enabled: true, priority: 100 }
```

然后管理端 **插件商店** 刷新，应能看到 **WorkBuddy (CodeBuddy)**；或：

```http
GET  /v0/management/plugin-store
POST /v0/management/plugin-store/workbuddy/install
```

（若同 ID 多源，可用 `?source=` 指定。）

本地未推送时，也可起静态文件服务挂载本仓库的 `registry.json`，把 `store-sources` 写成该 URL。

#### 3）官方商店（可选）

把 [`docs/plugin-store-entry.json`](docs/plugin-store-entry.json) 合并进  
[CLIProxyAPI-Plugins-Store/registry.json](https://github.com/router-for-me/CLIProxyAPI-Plugins-Store) 提 PR；合并后无需自配 `store-sources`。

#### 4）启用

安装成功后确认 `plugins.configs.workbuddy.enabled: true`，重启或热加载后日志出现 `plugin loaded ... plugin_id=workbuddy`。

### B. 本地编译

**前置**：CLIProxyAPI v7.2.x（带 CGO / 插件支持）、Go 1.26+、gcc；架构与 CPA 一致。

```bash
git clone https://github.com/WslzGmzs/workbuddy-cli-proxy.git
cd workbuddy-cli-proxy

# 当前平台
make build
# → dist/workbuddy.so | .dylib | .dll

# 指定平台并打成商店兼容 zip
make package VERSION=0.2.0 GOOS=linux GOARCH=amd64
```

也可手写：

```bash
CGO_ENABLED=1 GOOS=linux GOARCH=amd64 \
  go build -buildmode=c-shared -ldflags "-s -w -X main.pluginVersion=0.2.0" \
  -o workbuddy.so .
```

把产物放到 CPA 的 `plugins/`（或 `plugins/<goos>/<goarch>/`），启用配置后重启，日志出现 `plugin loaded ... plugin_id=workbuddy` 即成功。

## 凭据

### 1. 扫码 / OAuth（面板登录）

CPA 管理界面添加 workbuddy 凭据 → 扫码登录 CodeBuddy。登录后由宿主持久化（形态与 `workbuddy.json` 兼容）。

### 2. 管理页 / API 添加 API Key（推荐）

CPA 自带的 OAuth「回调 URL / 授权码」框 **不能** 用来贴 Key：宿主接口 `/v0/management/oauth-callback` 要求 `state` + `code`，面板常把内容塞进 `redirect_url` 且不带 `state`，请求在进插件前就被拒（`state is required`）。扫码登录不受影响。

#### 管理菜单

加载插件后，管理端会出现 **WorkBuddy API Key**（资源路径）：

```text
https://<cpa-host>/v0/resource/plugins/workbuddy/api-key
```

填写 Management Token + CodeBuddy API Key 即可保存。

#### HTTP API（需 management Bearer）

```bash
curl -X POST 'https://<cpa-host>/v0/management/workbuddy/api-key' \
  -H 'Authorization: Bearer <management-key>' \
  -H 'Content-Type: application/json' \
  -d '{"api_key":"YOUR_CODEBUDDY_API_KEY","user_id":"anonymous","prefix":"wb","proxy_url":"http://127.0.0.1:7890","priority":100}'
```

成功返回 `{"status":"ok","fileName":"workbuddy-key-....json",...}`，凭据经 `host.auth.save` 写入 CPA auth 目录。

### 3. 上传 auth JSON 文件

把 CodeBuddy 控制台的 API Key 写成 JSON，作为 CPA auth 文件上传 / 放入 auth 目录：

```json
{
  "type": "workbuddy",
  "auth_type": "api_key",
  "api_key": "YOUR_CODEBUDDY_API_KEY",
  "user_id": "anonymous",
  "domain": "copilot.tencent.com",
  "prefix": "wb",
  "proxy_url": "http://127.0.0.1:7890",
  "priority": 100
}
```

示例文件：[`examples/workbuddy-api-key.json`](examples/workbuddy-api-key.json)。

也支持简写字段 `apiKey`。API Key 模式不会走 token refresh；请求头会带 `Authorization: Bearer <key>` 与 `X-API-Key`。

可选字段：`enterprise_id` / `endpoint`（当前仍默认上游 `copilot.tencent.com`）。

### CPA 标准凭据字段

与 CPA 面板 / synthesizer 一致，写在 auth JSON **根级**：

| 字段 | 作用 |
|------|------|
| `prefix` | 模型前缀（单段，无 `/`）→ `AuthData.Prefix` |
| `proxy_url` | 该凭据出站 HTTP 代理 → `AuthData.ProxyURL`，插件请求会走此代理 |
| `priority` | 调度优先级（int / 数字字符串）→ `Attributes["priority"]` + metadata |
| `disabled` | `true` 时 CPA **注销该凭据的模型绑定**，插件 execute 也拒绝 |
| `excluded_models` / `excluded-models` | 对该凭据隐藏的上游模型 id 列表（与 `oauth-excluded-models` 同类） |
| `model_aliases` / `model-aliases` | CPA OAuth 模型别名：`[{"name":"上游id","alias":"客户端名","force-mapping":false}]` |

OAuth 扫码结果默认不带这些；可在 CPA 面板 PATCH 凭据，或直接编辑 auth 文件后重新加载。Refresh 时会从 host metadata/attributes **保留**已设置的值。

#### 为何「禁用了还能拉模型」？

旧版 workbuddy 用 **`model.static` + ExecutorModelScopeBoth`**，模型挂在 **插件静态表** 上，不跟单条凭据走。CPA 禁用凭据时只 `UnregisterClient(auth.ID)`，**静态插件模型仍在**。

现已改为：

- `ExecutorModelScope = oauth`（仅凭据绑定模型）
- `model.static` 返回空列表
- `model.for_auth` 在凭据 `disabled` 时返回空；有活跃凭据时返回模型列表

因此：**所有 workbuddy 凭据都禁用 / 无凭据 → `/v1/models` 不应再出现 workbuddy 模型**；至少一条启用凭据 → 正常列出。

#### 模型别名 / 排除（可用 CPA 能力）

1. **全局**（`config.yaml`，CPA 原生）：

```yaml
oauth-model-alias:
  workbuddy:
    - name: hy3-preview-agent   # 上游真实 id
      alias: hy3                # 客户端请求名
      force-mapping: false
oauth-excluded-models:
  workbuddy:
    - minimax-m3-pay
```

2. **单凭据**（auth JSON / 管理页 / 面板字段）：

```json
{
  "type": "workbuddy",
  "auth_type": "api_key",
  "api_key": "...",
  "excluded_models": ["minimax-m3-pay"],
  "model_aliases": [
    {"name": "hy3-preview-agent", "alias": "hy3", "force-mapping": false}
  ]
}
```

插件会把这些写进 `Metadata` + `Attributes`，供 CPA 注册模型与路由时使用；execute 侧对 `excluded_models` 再做一次拒绝。

## 使用

CPA 默认端口 `8317`，客户端 API key 见 `config.yaml` 的 `api-keys`。

| 协议 | Base URL |
|------|----------|
| OpenAI | `http://<host>:8317/v1` |
| Anthropic | `http://<host>:8317`（不带 `/v1`，走 `x-api-key`） |

```bash
# Claude Code
export ANTHROPIC_BASE_URL=http://localhost:8317
export ANTHROPIC_API_KEY=<api-key>
export ANTHROPIC_MODEL=hy3-preview-agent
claude
```

```bash
curl http://localhost:8317/v1/chat/completions \
  -H "Authorization: Bearer <api-key>" -H "Content-Type: application/json" \
  -d '{"model":"hy3-preview-agent","messages":[{"role":"user","content":"你好"}],"stream":true}'
```

流式 / 非流式都支持；非流式会在内部转成上游流式再聚合（CodeBuddy 上游 `code 11101` 拒绝非流式）。

## Claude Code 兼容性

腾讯 CodeBuddy 的内容审核把 Claude Code 的两句固定 system 模板逐字加进了黑名单，命中即回「敏感内容」拒答：

- `You are Claude Code, Anthropic's official CLI for Claude.`（身份句）
- `Main branch (you will usually use this for PRs)`（git 注入句）

任何一字改动都绕过（精确匹配）。workbuddy 转发前会自动做最小改写（`CLI`→`CLI tool`、`Main branch`→`Default branch`）。若上游再加封禁句，改 `sanitizeBlockedTemplates`。

## 思考模式

hy3 系列（`hy3` / `hy3-preview` / `hy3-preview-agent`）自动强制 `reasoning_effort=high`。CodeBuddy 只对 `high` 真正开深度思考。思考内容走 SSE 的 `delta.reasoning_content`。

## 流式

真流式（async）：边读上游边通过 `host.stream.emit` 推给 CPA。

## 发布 / 插件商店

1. 推送 tag：`git tag v0.2.0 && git push origin v0.2.0`
2. GitHub Actions（`.github/workflows/build.yml`）构建多平台 zip + `checksums.txt` 并创建 Release
3. 向 [CLIProxyAPI-Plugins-Store](https://github.com/router-for-me/CLIProxyAPI-Plugins-Store) 提 PR，仅追加 `docs/plugin-store-entry.json` 中的条目到 `registry.json`（`repository` 必须是 `https://github.com/WslzGmzs/workbuddy-cli-proxy`）
4. 之后只需打新 tag 发版，商店会读 latest release，无需每次改 registry

规范摘要：

| 项 | 要求 |
|----|------|
| 插件 ID | `workbuddy`（与文件名 / zip 内库名一致） |
| Release tag | `v<version>`，如 `v0.2.0` |
| 资产名 | `workbuddy_<version>_<goos>_<goarch>.zip` |
| zip 内容 | 根目录仅 `workbuddy.so` / `.dylib` / `.dll` |
| 校验 | `checksums.txt`（sha256sum 格式） |

## License

MIT。
