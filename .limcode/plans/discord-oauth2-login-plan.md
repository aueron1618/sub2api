## TODO LIST

<!-- LIMCODE_TODO_LIST_START -->
- [ ] 设计并接入 discord_connect 配置与系统设置链路（config defaults/validate、domain constants、setting service、DTO、admin/public handler）  `#p0-backend-config-settings`
- [ ] 实现 Discord OAuth start/callback/complete-registration 后端流程并注册路由，复用现有 invitation_required 机制  `#p0-backend-oauth-handler-routes`
- [ ] 通过 MCP context7 查阅并确认 Discord OAuth2 官方文档参数（点、scope、token auth、PKCE、redirect 规则、userinfo 字段）  `#p0-context7-discord-oauth-docs`
- [ ] 在管理设置页增加 Discord OAuth 配置卡片与保存逻辑，并补齐中英文 i18n 文案  `#p1-frontend-admin-settings-i18n`
- [ ] 新增 Discord OAuth 前端入口组件与回调页面，并接入 Login/Register/Router/Auth API  `#p1-frontend-auth-callback`
- [ ] 补充后端/前端相关测试并执行手工联调验收（成功、邀请码、拒绝授权、state/token 失败分支）  `#p2-tests-and-e2e-check`
<!-- LIMCODE_TODO_LIST_END -->

# Discord OAuth2 登录接入实施计划

## 1. 目标与范围

- 新增一个完整可用的 **Discord OAuth2 登录方式**。
- 覆盖以下层面：
  - 后端 OAuth 配置与校验
  - 后端 OAuth 启动/回调/补全注册流程
  - 前端登录/注册页入口、回调页、路由与文案
  - 管理后台系统设置（开关 + Client ID/Secret + Redirect URL）
  - 公共设置透出（用于前端决定是否显示 Discord 登录入口）
  - 相关测试与验收
- 严格遵循现有 LinuxDo OAuth 的项目模式，尽量做“同构扩展”，降低维护成本。

---

## 2. 官方文档基线（P0，先完成）

> 必须通过 MCP `context7` 获取并确认 Discord 官方 OAuth2 文档后再编码，避免端点/参数偏差。

需确认并记录到实现注释/PR 描述中的关键点：

1. 授权端点、Token 端点、UserInfo 端点
2. 必需 scope（`identify`、`email` 是否都要）
3. Token 交换认证方式（`client_secret_post` / basic / 其他）
4. 是否推荐/支持 PKCE，以及当前服务端客户端场景的建议
5. Redirect URI 的严格匹配规则
6. `/users/@me` 返回字段（`id`、`username`、`global_name`、`email` 可用性）

---

## 3. 现状映射（可复用模板）

### 后端现有 LinuxDo OAuth 链路

- Handler: `backend/internal/handler/auth_linuxdo_oauth.go`
- 路由:
  - `GET /api/v1/auth/oauth/linuxdo/start`
  - `GET /api/v1/auth/oauth/linuxdo/callback`
  - `POST /api/v1/auth/oauth/linuxdo/complete-registration`
- 配置与默认值:
  - `backend/internal/config/config.go` (`linuxdo_connect.*`)
- 系统设置（DB 覆盖 config）与透出:
  - `backend/internal/service/setting_service.go`
  - `backend/internal/service/settings_view.go`
  - `backend/internal/handler/admin/setting_handler.go`
  - `backend/internal/handler/dto/settings.go`
  - `backend/internal/handler/setting_handler.go`

### 前端现有 LinuxDo OAuth 链路

- 登录入口组件: `frontend/src/components/auth/LinuxDoOAuthSection.vue`
- 登录页/注册页接入:
  - `frontend/src/views/auth/LoginView.vue`
  - `frontend/src/views/auth/RegisterView.vue`
- 回调页: `frontend/src/views/auth/LinuxDoCallbackView.vue`
- 路由: `frontend/src/router/index.ts` (`/auth/linuxdo/callback`)
- API:
  - `frontend/src/api/auth.ts` (`completeLinuxDoOAuthRegistration`)
  - `frontend/src/api/admin/settings.ts`（LinuxDo 设置字段）
- 类型:
  - `frontend/src/types/index.ts` (`linuxdo_oauth_enabled`)
  - `frontend/src/stores/app.ts` 公共设置默认结构
- i18n:
  - `frontend/src/i18n/locales/zh.ts`
  - `frontend/src/i18n/locales/en.ts`
- 管理设置 UI:
  - `frontend/src/views/admin/SettingsView.vue`

---

## 4. 实施方案

## 4.1 后端配置与设置模型

### 4.1.1 配置结构（config）

在 `backend/internal/config/config.go` 参照 `LinuxDoConnectConfig` 增加 `DiscordConnectConfig`并挂到根配置：

- 新增字段建议：
  - `Enabled`
  - `ClientID`
  - `ClientSecret`
  - `AuthorizeURL`
  - `TokenURL`
  - `UserInfoURL`
  - `Scopes`
  - `RedirectURL`
  - `FrontendURL`（建议默认 `/auth/discord/callback`）
  - `TokenAuthMethod`
  - `UsePKCE`
  - `UserInfoEmailPath`
  - `UserInfoIDPath`
  - `UserInfoUsernamePath`

- 在 `setDefaults()` 新增 `discord_connect.*` 默认值（端点以 context7 文档核准后写入）。
- 在 `Load()` 的 trim/normalize 阶段加入 `cfg.Discord.*` 清理。
- 在 `Validate()` 加入与 LinuxDo 平行的校验逻辑（必填项、URL 格式、token_auth_method 白名单、PKCE 约束、frontend redirect 校验）。

### 4.1.2 系统设置键与视图

在 `backend/internal/service/domain_constants.go` 增加：

- `discord_connect_enabled`
- `discord_connect_client_id`
- `discord_connect_client_secret`
- `discord_connect_redirect_url`

并在以下位置进行全链路接线：

1. `backend/internal/service/settings_view.go`
   - `SystemSettings` 增加 Discord Connect 字段
   - `PublicSettings` 增加 `DiscordOAuthEnabled bool`
2. `backend/internal/service/setting_service.go`
   - `GetPublicSettings` keys + 计算 `discordOAuthEnabled`
   - `GetPublicSettingsForInjection` 返回 JSON 字段
   - `UpdateSettings` 写入 `updates[...]`
   - `parseSettings` 读取 DB 覆盖 / fallback config
   - 新增 `GetDiscordConnectOAuthConfig(ctx)`（平行 LinuxDo）
3. `backend/internal/handler/dto/settings.go`
   - `SystemSettings`、`PublicSettings` DTO 加字段
4. `backend/internal/handler/setting_handler.go`
   - 公共设置响应映射加入 `discord_oauth_enabled`
5. `backend/internal/handler/admin/setting_handler.go`
   - `UpdateSettingsRequest` 增加 Discord 字段
   - `GetSettings` / `UpdateSettings` 响应映射
   - 参数验证（启用时 Client ID/Redirect URL/Secret 检查）
   - `diffSettings` 增加审计字段

---

## 4.2 后端 OAuth Handler 与路由

### 4..1 新增 Discord OAuth handler

新增文件建议：`backend/internal/handler/auth_discord_oauth.go`

实现以下方法（与 LinuxDo 对齐）：

- `DiscordOAuthStart`
- `DiscordOAuthCallback`
- `CompleteDiscordOAuthRegistration`
- `getDiscordOAuthConfig`

关键实现点：

1. 使用独立 cookie 命名空间与 path：`/api/v1/auth/oauth/discord`
2. state + (可选) PKCE 逻辑复用等安全策略
3. 回调错误统一经 fragment 回前端（结构与 LinuxDo 一致）
4. Token 交换与 userinfo 获取：
   - 按 context 核准的 Discord 端点和认证方式
   - 兼容 JSON 返回，记录 provider error 细节（避免泄漏敏感信息）
5. userinfo 解析策略：
   - subject: `id`
   - username: 优先 `global_name` / `username`（以文档为准）
   - email: 若为空，按安全策略兜底
6. 账号绑定安全：
   - 推荐沿用 LinuxDo 策略：**优先使用 subject 派生稳定合成邮箱**，避免第三方 email 直绑导致账号碰撞/接管风险
   - 若采取合成邮箱，需新增 Discord 专属 domain 常量及 reserved email 防护（见 4.4）
7. 邀请码分支：
   - 复用 `pending_oauth_token` + complete-registration 现有机制

### 4.2.2 路由注册

更新 `backend/internal/server/routes/auth.go`：

- `GET /auth/oauth/discord/start`
- `GET /auth/oauth/discord/callback`
- `POST /auth/oauth/discord/complete-registration`（增加限流器，命名如 `oauth-discord-complete`）

---

## 4.3 前端接入

### 4.3.1 登录入口组件

新增组件建议：`frontend/src/components/auth/DiscordOAuthSection.vue`

- 样式与 LinuxDo 结构保持一致（按钮 + 分隔线），但使用 Discord 品牌文案/图标颜色
- 跳转 URL：`/api/v1/auth/oauth/discord/start?redirect=...`

### 4.3.2 登录/注册页

- `frontend/src/views/auth/LoginView.vue`
  - 并列显示 Discord OAuth 入口（受 `discord_oauth_enabled` 控制）
- `frontend/src/views/auth/RegisterView.vue`
  - 同步接入 Discord OAuth 入口

### 4.3.3 回调页与路由

新增：`frontend/src/views/auth/DiscordCallbackView.vue`

- 逻辑基本对齐 LinuxDoCallback：
  - 解析 fragment token / error
  - `invitation_required` 时提交邀请码完成注册
  - 本地保存 refresh_token / expires_at

新增路由：

- `frontend/src/router/index.ts` 添加 `/auth/discord/callback`

### 4.3.4 API 与类型

- `frontend/src/api/auth.ts`
  - 新增 `completeDiscordOAuthRegistration`
- `frontend/src/types/index.ts`
  - `PublicSettings` 增加 `discord_o_enabled`
- `frontend/src/stores/app.ts`
  - public settings fallback 默认结构新增 `discord_oauth_enabled: false`
- `frontend/src/api/admin/settings.ts`
  - `SystemSettings` / `UpdateSettingsRequest` 增加 `discord_connect_*`

### 4.3.5 管理后台设置页

更新 `frontend/src/views/admin/SettingsView.vue`：

- 在 Security Tab 增加 Discord OAuth 配置卡片（与 LinuxDo 卡片风格一致）
- 表单字段：
  - `discord_connect_enabled`
  - `discord_connect_client_id`
  - `discord_connect_client_secret`
  - `discord_connect_client_secret_configured`
  - `discord_connect_redirect_url`
- 增加“自动生成回调地址并复制”按钮（`/api/v1/auth/oauth/discord/callback`）
- 保存 payload 与 load/reset 逻辑接线

---

## 4.4 安全与账号模型决策

1. **开放重定向防护**：延续当前相对路径白名单规则
2. **state/PKCE**：按 provider 能力与 token_auth_method约束执行
3. **邮箱碰撞风险**：
   - 若采用合成邮箱（推荐），新增：
     - `DiscordConnectSyntheticEmailDomain`（例如 `@discord-connect.invalid`）
     - `isReservedEmail` 扩展（拦截本地注册该后缀）
     - 对应测试扩展
4. **错误输出最小化**：前端展示友好错误，日志保留 provider 诊断信息

---

## 4.5 测试与验收

### 4.5.1 后端测试

- `backend/internal/config/config_test.go`
  - 增加 Discord 配置合法/非法校验用例（镜像 LinuxDo）
- `backend/internal/handler/auth_discord_oauth_test.go`（新增）
  - redirect sanitize
  - token/provider error parse
  - userinfo parse（含 email 缺失、username fallback）
- `backend/internal/service/auth_service_register_test.go`
  - 如采用合成邮箱，新增 Discord synthetic reserved email case
- `backend/internal/server/api_contract_test.go`
  - admin/public settings JSON 断言新增 Discord 字段

### 4.5.2 前端测试（若已有测试框架支持）

- 路由存在性（`/auth/discord/callback`）
- callback 参数解析与 invitation_required 分支
- settings 类型与默认值覆盖（止字段缺失导致渲染错误）

### 4.5.3 手工验收清单

1. 管理端开启 Discord OAuth，填写配置并保存成功
2. 登录页/注册页出现 Discord 登录按钮
3. 点击按钮可跳转 Discord 授权页
4. 回调成功后自动登录并跳转 redirect
5. 需要邀请码时进入补全注册流程并可成功完成
6. 异常路径（拒绝授权、state 不匹配、token 交换失败）均有可理解反馈

---

## 5. 变更文件清单（预计）

### 后端

- `backend/internal/config/config.go`
- `backend/internal/config/config_test.go`
- `backend/internal/service/domain_constants.go`
- `backend/internal/service/settings_view.go`
- `backend/internal/service/setting_service.go`
- `backend/internal/service/auth_service.go`（如扩展 reserved email）
- `backend/internal/service/auth_service_register_test.go`（如扩展 reserved email 测试）
- `backend/internal/handler/dto/settings.go`
- `backend/internal/handler/setting_handler.go`
- `backend/internal/handler/admin/setting_handler.go`
- `backend/internal/server/routes/auth.go`
- `backend/internal/handler/auth_discord_oauth.go`（新增）
- `backend/internal/handler/auth_discord_oauth_test.go`（新增）
- `backend/internal/server/api_contract_test.go`
- `deploy/config.example.yaml`

### 前端

- `frontend/src/components/auth/DiscordOAuthSection.vue`（新增）
- `frontend/src/views/auth/DiscordCallbackView.vue`（新增）
- `frontend/src/views/auth/LoginView.vue`
- `frontend/src/views/auth/RegisterView.vue`
- `frontend/src/router/index.ts`
- `frontend/src/api/auth.ts`
- `frontend/src/api/admin/settings.ts`
- `frontend/src/types/index.ts`
- `frontend/src/stores/app.ts`
- `frontend/src/views/admin/SettingsView.vue`
- `frontend/src/i18n/locales/zh.ts`
- `frontend/src/i18n/locales/en.ts`

---

## 6. 决策待确认（实现前）

1. 路由命名是否最终固定为：
   - `/api/v1/auth/oauth/discord/start|callback|complete-registration`
   - `/auth/discord/callback`
2. 是否明确沿用 LinuxDo 的邀请码补全流程（建议：是）
3. 是否采用 Discord 合成邮箱策略（建议：是，安全优先）
4. Discord scope 是否固定 `identify email`（待 context7 官方文档确认）

---

## 7. 里程碑执行顺序

1. context7 文档确认（阻塞项）
2. 后端 config + setting 链路打通
3. 后端 handler + routes 打通
4. 前端登录/回调接入
5. 管理端配置 UI + i18n 打通
6. 测试补齐与联调验收
