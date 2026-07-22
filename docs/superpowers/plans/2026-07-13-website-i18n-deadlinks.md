# Website Bilingual Polish + Markdown Dead Link Check & Fix Implementation Plan

> **Status: ✅ COMPLETED — PR #3 merged to main**
> Steps use checkbox (`- [x]`) syntax.

**Goal:** 验证并完善 website 的简体中文/English 双语支持（调研已确认 i18n 基础设施与内容质量已完整到位，本计划聚焦死链治理），自主安装 markdown 死链检查工具 lychee，扫描全部 markdown 文件中的死链（外链 + 站内链接）并修复所有失效链接。

**Architecture:** `cargo install lychee` 安装 Rust 实现的链接检查器 → 对 58 个 markdown 文件（website/docs 下 56 个 + 根 README.md + README.zh-CN.md）运行 lychee，用 `--exclude` 过滤文档中的示例/占位 URL（example.com、本地 IP、镜像演示域名）→ lychee 输出死链清单（含文件:行号 + 失效 URL）→ 按清单逐条修复（站内悬空链接改指向真实存在页面；失效外链替换为归档源或移除）→ 重新运行 lychee 验证零死链 → 提交并以 PR 形式自主合并到 main。数据流：markdown 文件 → lychee 扫描 → 死链报告 → 逐条修复 → 复扫验证 → 提交合并。

**Tech Stack:** lychee (Rust, via cargo 1.96.0), VitePress 1.6.4 站点路由, GitHub CLI (gh) 2.x 用于 PR 合并

**Risks:**
- lychee 对文档中演示镜像源配置的示例 URL（`example.com`、`127.0.0.1:7890`、`10.0.0.1:8080`、`gems.corp.internal`、`gems.example.com`、`gems.ruby-china.com`、`mirrors.tuna.tsinghua.edu.cn`、`mirrors.aliyun.com`）会误报为死链 → 缓解：Task 2 Step 1 用 `--exclude` 逐一排除这些示例域名与 IP
- 站内相对链接（`./xxx`、`../xxx`）由 VitePress 路由解析，lychee 默认不解析 markdown 间相对跳转 → 缓解：Task 2 Step 2 用 `lychee --base` 指向 website/docs 并配合 `--remap` 让相对路径可被解析，外加独立 shell 脚本验证每个站内链接目标文件是否存在
- 外链检查因网络/速率限制慢或误判临时不可达为死链 → 缓解：lychee 自带请求缓存与限流，`--timeout 30 --max-redirects 10`，对每个报告的死链二次人工复核（WebFetch 验证）后再修
- 修复外链可能改坏示例代码语义 → 缓解：示例 URL（演示镜像源）一律保留不修，只修真实失效外链与悬空站内链接

---

### Task 1: 安装 lychee 死链检查工具

**Depends on:** None
**Files:**
- Create: 无（仅安装二进制）

- [x] **Step 1: 通过 cargo 安装 lychee — Rust 实现的高速链接检查器**
- [x] **Step 2: 验证 lychee 可执行且版本正常**
- [x] **Step 3: 提交**（无代码变更）

---

### Task 2: 运行死链检查并生成报告

**Depends on:** Task 1
**Files:**
- Create: `website/.lycheeignore`（排除规则文件）
- Create: `docs/superpowers/artifacts/2026-07-13-deadlinks-report.md`（死链报告，供 Task 3 修复依据）

- [x] **Step 1: 创建 lychee 排除规则文件 — 过滤文档中的示例/占位 URL**
- [x] **Step 2: 运行 lychee 扫描 website/docs 全部 markdown — 检查外链与相对站内链接**
- [x] **Step 3: 对根 README.md 与 README.zh-CN.md 单独运行死链检查 — 含 shields.io/pkg.go.dev 徽章链接**
- [x] **Step 4: 合并并人工复核死链报告 — 区分真死链与误报**
- [x] **Step 5: 提交**

---

### Task 3: 验证零死链并提交工件

**Depends on:** Task 2
**Files:**
- Create: `website/.lycheeignore`（已创建）
- Create: `docs/superpowers/artifacts/2026-07-13-deadlinks-report.md`（已创建）
- Modify: 无（零真死链，无需代码修复）

- [x] **Step 1: 分析死链报告，逐条分类确认真伪**
- [x] **Step 2: 重运行 lychee 复扫确认 — 验证零死链基线**
- [x] **Step 3: 提交工件 — .lycheeignore 配置 + 死链审计报告 + 计划文档**

---

### Task 4: 以 PR 形式自主合并到 main

**Depends on:** Task 3
**Files:**
- Modify: 无代码改动（仅 git/gh 操作）

- [x] **Step 1: 创建分支并推送 — 在专用分支上提交所有死链治理工作**
- [x] **Step 2: 创建 PR — 链接 lychee 配置 + 死链修复**
- [x] **Step 3: 等待 CI 通过后自主合并 PR — merge commit 保留可追溯历史**
- [x] **Step 4: 同步本地 main 并验证**
- [x] **Step 5: 提交**
