# CPA Codex Bridge：无 RT 账户的 chat2api 降级方案 (v3)

## 背景

用户从 ChatGPT 网页端手动抓取的 `accessToken` (AT) 无法通过 Codex 专有端点发起对话，但对网页版 Conversation API 仍然有效。本方案将在 CPA 中实现智能降级，自动将无 RT 的 codex 账户路由到 chat2api 中间件。

---

## chat2api 项目分析 ([lanqian528/chat2api](https://github.com/lanqian528/chat2api))

### ✅ 兼容性评估：协议层完美匹配

| 维度 | chat2api 能力 | CPA 需求 | 匹配度 |
|:---|:---|:---|:---|
| **API 格式** | 标准 OpenAI `/v1/chat/completions` | `OpenAICompatExecutor` 期望的格式 | ✅ 100% |
| **认证方式** | `Authorization: Bearer <AccessToken>` | CPA 注入 `api_key` 作为 Bearer Token | ✅ 100% |
| **流式支持** | SSE `data:` 格式流式输出 | `OpenAICompatExecutor.ExecuteStream` 标准 SSE 解析 | ✅ 100% |
| **部署方式** | Docker 一键部署，端口 5005 | CPA 通过 `base_url` 配置指向 | ✅ 无缝 |

### ⚠️ 关键发现：chat2api 模型列表严重过时

> [!WARNING]
> **chat2api 的 `api/models.py` 仍停留在 GPT-4o 时代！** 它的 `model_proxy` 字典只包含：
> `gpt-3.5-turbo`, `gpt-4`, `gpt-4o`, `gpt-4o-mini`, `o1`, `o3-mini` 等已退役模型。
>
> **但这不影响核心功能。** chat2api 的协议核心 `chatFormat.py` 将模型名**透传**给 ChatGPT 的 `/backend-api/conversation` 端点。`model_proxy` 仅用于生成 `system_fingerprint`（一个无关紧要的响应元数据字段）。因此，传入 `gpt-5.4` 等新模型名后，chat2api 仍能正常工作。

### 当前 ChatGPT Web 端模型现状 (2026年4月)

| Web 端分类 | 内部模型 slug | 说明 |
|:---|:---|:---|
| **Instant** | `gpt-5.3` / `gpt-5.3-instant` | 日常快速任务 |
| **Standard** | `gpt-5.4` | 旗舰通用模型 |
| **Thinking** | `gpt-5.4` (thinking mode) | 复杂推理 |
| **Pro** | `gpt-5.4` (pro mode) | 高级推理（Pro 计划专属） |
| **Mini** | `gpt-5.4-mini` | 速率限制时的自动降级替代 |

> 旧模型（GPT-4o, GPT-4.1, GPT-5 Instant, GPT-5.1）已于 2026年2月13日从 ChatGPT 平台**全部退役**。

---

## 模型名称映射策略（更新）

### CPA Codex 模型 → ChatGPT Web 模型映射表

从 CPA 的 [models.json](file:///c:/Users/drs/Documents/CLIProxyAPI/internal/registry/models/models.json) 提取的 Codex 模型 ID 及其对应的 Web 端映射：

| CPA Codex 模型 ID | Web 端对照 slug | 映射策略 | 说明 |
|:---|:---|:---|:---|
| `gpt-5` | `gpt-5.3` | ⚠️ 近似 | GPT-5 基础版已退役，映射到最近的 Instant |
| `gpt-5-codex` | `gpt-5.3` | ⚠️ 近似 | Codex 专有变体，Web 端无直接对应 |
| `gpt-5-codex-mini` | `gpt-5.3` | ⚠️ 近似 | Mini 变体降级到 Instant |
| `gpt-5.1` | `gpt-5.3` | ⚠️ 近似 | 5.1 已退役 |
| `gpt-5.1-codex` | `gpt-5.3` | ⚠️ 近似 | 已退役 |
| `gpt-5.1-codex-mini` | `gpt-5.3` | ⚠️ 近似 | 已退役 |
| `gpt-5.1-codex-max` | `gpt-5.4` | ⚠️ 近似 | Max 级能力对应当前旗舰 |
| `gpt-5.2` | `gpt-5.4` | ✅ 直接 | 当前旗舰替代 |
| `gpt-5.2-codex` | `gpt-5.4` | ✅ 直接 | 当前旗舰替代 |
| `gpt-5.3-codex` | `gpt-5.4` | ✅ 直接 | 最新 Codex → 最新 Web |
| `gpt-5.3-codex-spark` | `gpt-5.3` | ⚠️ 近似 | Spark 是轻量级，映射到 Instant |
| `gpt-5.4` | `gpt-5.4` | ✅ 1:1 | 完全一致 |

### config.yaml 配置示例

```yaml
oauth-model-alias:
  codex-bridge:
    # === 最新模型 (1:1 映射) ===
    - name: "gpt-5.4"
      alias: "gpt-5.4"
    
    # === 近代模型 → 当前旗舰 ===
    - name: "gpt-5.4"
      alias: "gpt-5.2"
      fork: true
    - name: "gpt-5.4"
      alias: "gpt-5.2-codex"
      fork: true
    - name: "gpt-5.4"
      alias: "gpt-5.3-codex"
      fork: true
    - name: "gpt-5.4"
      alias: "gpt-5.1-codex-max"
      fork: true

    # === 旧模型 → Instant ===
    - name: "gpt-5.3"
      alias: "gpt-5"
      fork: true
    - name: "gpt-5.3"
      alias: "gpt-5-codex"
      fork: true
    - name: "gpt-5.3"
      alias: "gpt-5-codex-mini"
      fork: true
    - name: "gpt-5.3"
      alias: "gpt-5.1"
      fork: true
    - name: "gpt-5.3"
      alias: "gpt-5.1-codex"
      fork: true
    - name: "gpt-5.3"
      alias: "gpt-5.1-codex-mini"
      fork: true
    - name: "gpt-5.3"
      alias: "gpt-5.3-codex-spark"
      fork: true
```

> [!NOTE]
> **`fork: true`** 意味着 alias 作为额外模型注册，用户请求 `gpt-5.3-codex` 时，CPA 会自动将其路由到 chat2api 并以 `gpt-5.4` 请求。`name` 是发送给 chat2api 的实际模型名，`alias` 是客户端使用的名称。

---

## 用户决策整合

| 问题 | 决策 | 实现影响 |
|:---|:---|:---|
| Per-file 桥接覆盖 | ❌ 不需要，JSON 自动化获取不手动修改 | 仅通过全局配置 + RT 检测自动判断 |
| AT 过期行为 | ✅ 自动删除 auth.json | Step 6：过期检测 → 触发文件删除 |
| chat2api 选型 | ✅ `lanqian528/chat2api` | Docker 部署，端口 5005 |

---

## 实施计划 (6 步)

### Step 1: 配置层 — 添加 Bridge 配置项

#### [MODIFY] [config.go](file:///c:/Users/drs/Documents/CLIProxyAPI/internal/config/config.go)

在 `Config` 结构体中添加（约第 128 行附近，在 `Payload` 之前）：
```go
// CodexBridge configures the chat2api bridge for Codex accounts without refresh tokens.
CodexBridge CodexBridgeConfig `yaml:"codex-bridge" json:"codex-bridge"`
```

新增结构体：
```go
// CodexBridgeConfig configures the chat2api bridge fallback for Codex OAuth accounts
// that lack a refresh_token. When enabled, such accounts are automatically routed
// through the configured chat2api middleware instead of the Codex-specific endpoint.
type CodexBridgeConfig struct {
    // Enabled toggles the bridge fallback.
    Enabled bool `yaml:"enabled" json:"enabled"`
    // BaseURL is the chat2api middleware endpoint (e.g., "http://127.0.0.1:5005/v1").
    BaseURL string `yaml:"base-url" json:"base-url"`
    // AutoDeleteOnExpiry controls whether to delete the auth file when AT expires (401).
    AutoDeleteOnExpiry bool `yaml:"auto-delete-on-expiry" json:"auto-delete-on-expiry"`
}
```

---

### Step 2: Auth 加载层 — 为无 RT 的 Codex 文件打标记

#### [MODIFY] [filestore.go](file:///c:/Users/drs/Documents/CLIProxyAPI/sdk/auth/filestore.go)

在 `readAuthFile()` 方法中，当 `provider == "codex"` 时，检测 RT 并打标记：

```go
if strings.EqualFold(provider, "codex") {
    rt, _ := metadata["refresh_token"].(string)
    if strings.TrimSpace(rt) == "" {
        if auth.Metadata == nil {
            auth.Metadata = make(map[string]any)
        }
        auth.Metadata["_codex_no_rt"] = true
    }
}
```

---

### Step 3: 服务注册层 — Provider 透明改写

#### [MODIFY] [service.go](file:///c:/Users/drs/Documents/CLIProxyAPI/sdk/cliproxy/service.go)

在 `applyCoreAuthAddOrUpdate()` 中，`ensureExecutorsForAuth` 调用前插入 `applyCodexBridgeIfNeeded(auth)` 方法。检测无 RT 标记 + bridge 配置，改写 Provider 为 `codex-bridge`，将 `access_token` 映射为 `api_key`。

---

### Step 4: 执行器绑定 — 处理 `codex-bridge`

#### [MODIFY] [service.go](file:///c:/Users/drs/Documents/CLIProxyAPI/sdk/cliproxy/service.go)

在 `ensureExecutorsForAuthWithMode()` 的 `codex` 分支之后添加 `codex-bridge` 分支，注册 `OpenAICompatExecutor`。

---

### Step 5: Refresh 逻辑 — 自动跳过

`OpenAICompatExecutor.Refresh()` 已是 no-op。**无需修改。**

---

### Step 6: AT 过期自动删除 auth.json

当桥接请求收到 `401 Unauthorized` 时，对 `codex-bridge` 类型的 auth，异步调用 `FileTokenStore.Delete(auth.ID)` 删除文件。watcher 自动感知并移除账户。

#### [MODIFY] [conductor.go](file:///c:/Users/drs/Documents/CLIProxyAPI/sdk/cliproxy/auth/conductor.go)

在 401 错误处理分支中：
```go
if statusCode == 401 && strings.EqualFold(auth.Provider, "codex-bridge") {
    if s.cfg != nil && s.cfg.CodexBridge.AutoDeleteOnExpiry {
        log.Warnf("codex bridge: auth %s AT expired, deleting auth file", auth.ID)
        go func(authID string) {
            _ = s.store.Delete(context.Background(), authID)
        }(auth.ID)
    }
}
```

---

## chat2api 一键部署

```yaml
# docker-compose.yml
version: '3.8'
services:
  chat2api:
    image: lanqian528/chat2api:latest
    container_name: chat2api
    ports:
      - "5005:5005"
    environment:
      - HISTORY_DISABLED=true
      - CONVERSATION_ONLY=false
      - ENABLE_LIMIT=true
      - RANDOM_TOKEN=false
      - SCHEDULED_REFRESH=false
    restart: unless-stopped
```

---

## 完整数据流

```
用户请求 (model: gpt-5.3-codex)
    ↓
CPA oauth-model-alias: gpt-5.3-codex → gpt-5.4
    ↓
CPA Manager → 选择一个 codex-bridge 账户
    ↓
OpenAICompatExecutor.Execute()
    ↓ POST http://127.0.0.1:5005/v1/chat/completions
    ↓ Authorization: Bearer <access_token>
    ↓ Body: {"model": "gpt-5.4", "messages": [...]}
    ↓
chat2api 收到请求
    ↓ 透传 model="gpt-5.4" 到 chatgpt.com/backend-api/conversation
    ↓ 自动处理 PoW、会话管理
    ↓ 返回标准 OpenAI 格式响应
    ↓
CPA 收到响应 → 返回客户端
```

---

## Verification Plan

### Automated Tests
1. 单元测试 `applyCodexBridgeIfNeeded()`：有 RT / 无 RT / bridge 未启用
2. 集成测试：写入无 RT auth.json → 验证执行器注册为 `codex-bridge`
3. 过期清理测试：模拟 401 → 验证 auth.json 被删除

### Manual Verification
1. 部署 chat2api + CPA，配置 bridge
2. 放入无 RT 的 auth.json，观察日志
3. 发送对话请求，验证正常响应
4. 等待 AT 过期，验证 auth.json 被自动删除

---

## Open Questions (Remaining)

> [!IMPORTANT]
> **代理透传**：chat2api 需要独立配置 `PROXY_URL` 环境变量来访问 `chatgpt.com`，CPA 的代理配置不会透传。

确认以上方案后我将开始执行代码修改。
