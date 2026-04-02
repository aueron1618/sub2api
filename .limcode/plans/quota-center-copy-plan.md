## TODO LIST

<!-- LIMCODE_TODO_LIST_START -->
- [ ] 更新 router/index.ts 中 /subscriptions 与 /admin/subscriptions 的 meta.title 兜底文案为配额导向表达  `#p0-route-fallback-title`
- [ ] 更新 zh.ts 的 nav/admin.subscriptions/subscriptionProgress/userSubscriptions/allowedGroupsHint 文案，将“订阅管理”体系改为“配额中心”体系  `#p0-zh-copy-core`
- [ ] 同步调整 en.ts 对应文案（Quota Center / My Quotas 等），保持双语一致  `#p1-en-copy-alignment`
- [ ] 评估并统一 purchase 与 admin settings purchase 区块中“充值/订阅”文案是否改为“充值/购买配额”  `#p1-purchase-wording-scope`
- [ ] 可选：将 backend/internal/server/routes/admin.go 中“订阅管理”注释改为“配额中心”  `#p2-backend-comment-cleanup`
- [ ] 执行关键词回归检查（订阅管理/我的订阅），确认仅保留必要技术语义文案且用户侧覆盖完成  `#p2-verify-and-grep`
<!-- LIMCODE_TODO_LIST_END -->

# 配额中心文案替换实施计划

## 1. 背景与目标

用户侧与管理侧当前存在较多“订阅管理 / 我的订阅”表达。目标是在**不修改任何业务逻辑、接口、路由路径、配置键、枚举值**的前提下，将用户可见文案调整为“配额中心 / 我的配额”导向表达，并保持中英文一致。

## 2. 约束与原则

- **仅改文案**，不改：
  - 路由路径：`/subscriptions`、`/admin/subscriptions`
  - 技术键名：`admin.subscriptions.*`、`userSubscriptions.*`、`subscriptionProgress.*`
  - 配置键：`purchase_subscription_enabled`、`purchase_subscription_url`、`default_subscriptions`
  - 计费类型语义 `subscription`（技术语义保留）
- 用户可见文案优先“配额”表达；技术/实现语义保留“subscription”键名。
- 双语一致：`zh.ts` 与 `en.ts` 同步更新。

## 3. 变更范围（文件级）

### 3.1 路由 fallback 标题
- `frontend/src/router/index.ts`
  - `/subscriptions` 的 `meta.title`: `My Subscriptions` → `My Quotas`
  - `/admin/subscriptions` 的 `meta.title`: `Subscription Management` → `Quota Center`

### 3.2 中文文案主改（P0）
- `frontend/src/i18n/locales/zh.ts`
  - `nav.subscriptions`: 订阅管理 → 配额中心
  - `nav.mySubscriptions`: 我的订阅 → 我的配额
  - `subscriptionProgress.*`: 统一为“配额”措辞（如“查看配额详情”“暂无有效配额”等）
  - `userSubscriptions.*`: 标题/说明/空态/提示/成功失败文案改为“配额”措辞
  - `admin.subscriptions.*`: 标题、描述、按钮、表单提示、空态、toast、guide 全面改为“配额中心/配额”措辞
  - `admin.users.allowedGroupsHint`: 将“订阅管理中配置”改为“配额中心中配置”

### 3.3 英文对齐（P1）
- `frontend/src/i18n/locales/en.ts`
  - `nav.subscriptions`: `Subscriptions` → `Quota Center`
  - `nav.mySubscriptions`: `My Subscriptions` → `My Quotas`
  - `subscriptionProgress.*`: 改为 Quota 体系（如 `View quota details`, `No active quotas`）
  - `userSubscriptions.*`: 改为 `My Quotas` / quota-oriented 表达
  - `admin.subscriptions.*`: `Subscription Management` 体系改为 `Quota Center` 体系
  - `allowedGroupsHint`: 改为“subscription groups configured in Quota Center”的表达

### 3.4 充值/购买配额文案统一评估（P1）
- 候选范围：
  - `nav.buySubscription`
  - `purchase.*`（用户页）
  - `admin.settings.site.purchase.*`（管理后台设置页对应 i18n）
- 推荐落地：
  - 中文统一为“充值/购买配额”
  - 英文统一为“Top up / Buy Quota”
- 注意：仅改展示文案，**不改任何 purchase/subscription 相关配置键名**。

### 3.5 可后端注释清理（P2）
- `backend/internal/server/routes/admin.go`
  - 注释 `// 订阅管理` → `// 配额中心`

## 4. 文案合理化策略（避免机械替换）

- 面向用户可见入口与页面标题：优先“配额中心 / 我的配额”。
- 面向操作动作：
  - `Assign/Adjust/Revoke Subscription` 语义改为“分配/调整/撤销配额（授权）”表达，保留操作含义。
- 面向技术约束提示（例如“计费类型为 subscription”）：可采用“订阅计费（配额周期）”这种兼容写法，避免歧义。

## 5. 回归与验收

### 5.1 关键词回归检查
- 执行关键词检索并人工判断残留是否属于必要技术语义：
  - `订阅管理`
  - `我的订阅`
  - `Subscription Management`
  - `My Subscriptions`
- 预期：用户可见主路径（导航、页标题、空态、按钮、提示）完成配额化。

### 5.2 功能不回归检查（轻量）
- 前端仅文案改动，不应影响编译与运行。
- 关键页面人工检查：
  - 用户：侧边栏、Header 迷你进度、`/subscriptions`、`/purchase`
  - 管理：`/admin/subscriptions`、系统设置 purchase 区块

## 6. 风险与处理

- **风险**：过度替换导致技术语义不清（如 subscription billing type）。
  - **处理**：仅替换用户可见文案，不动技术键和接口字段。
- **风险**：中英文不一致。
  - **处理**：同一批次同步修改 `zh.ts` 与 `en.ts` 对应 key。
