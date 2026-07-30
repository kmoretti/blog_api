# 项目配置与数据导入导出设计

## 1. 目标与范围

为 `blog_api` 提供一套管理面板可操作的配置/数据迁移能力，方便：

- 完整备份与恢复（灾备、整机迁移）
- 跨实例迁移业务数据（友链、动态、RSS 等）
- 批量编辑或初始化系统配置

## 2. 高层架构

采用「方案 C」：完整归档与分表导入导出组合。

```
┌─────────────────────────────────────────┐
│              Settings 页面               │
│  ┌──────────────┐  ┌──────────────────┐ │
│  │  完整备份    │  │  分模块导入导出  │ │
│  │  - 导出 ZIP  │  │  - 系统配置      │ │
│  │  - 恢复 ZIP  │  │  - 友链          │ │
│  └──────────────┘  │  - 动态          │ │
│                    │  - RSS           │ │
│                    │  - 图片清单      │ │
│                    └──────────────────┘ │
└─────────────────────────────────────────┘
                    │
        ┌───────────┴───────────┐
        ▼                       ▼
  /api/action/backup/*    /api/action/export/:module
  (ZIP 归档)              /api/action/import/:module
                          (JSON 合并)
```

## 3. 功能模块

### 3.1 完整备份（Full Backup）

**导出**

- `POST /api/action/backup/export`
- 将 `data/` 目录打包为 `blog_api_backup_<timestamp>.zip`
- 包含：`database.db`、WAL/SHM 文件、`config/`、`image/`、`resource/` 等
- 流式下载，避免大文件占用内存

**导入（恢复）**

- `POST /api/action/backup/import`
- 接收 ZIP 文件
- 流程：
  1. 校验 ZIP 结构（至少包含 `database.db`）
  2. 将当前 `data/` 复制到 `data.bak.<timestamp>`
  3. 解压 ZIP 覆盖 `data/`
  4. 返回成功响应，提示用户重启服务以释放 SQLite 文件占用
- **模式：仅替换，不合并**

### 3.2 分模块导入导出（JSON）

**system_config 特殊处理**

- 导出：直接读取 `data/config/system_config.json`
- 导入：
  - `replace`：直接覆盖整个 JSON 文件
  - `skip`：如果文件已存在则不覆盖
- 注意：修改系统配置后通常需要重启服务才能生效

**支持的模块**

| 模块 | 来源 | 说明 |
|------|------|------|
| `system_config` | `data/config/system_config.json` | 系统配置 JSON |
| `friend_links` | `friend_link` 表 | 友链数据 |
| `moments` | `moments` 表 | 动态数据 |
| `friend_rss` | `friend_rss` / `rss_post` 表 | RSS 源与文章 |
| `images` | `image_repo` 表 | 图片元数据（不含实际文件） |

**导出**

- `GET /api/action/export/:module`
- 返回统一 JSON 结构：

```json
{
  "version": "1.0",
  "module": "friend_links",
  "exported_at": "2026-07-30T12:00:00Z",
  "count": 10,
  "items": [...]
}
```

**导入**

- `POST /api/action/import/:module`
- 接收 JSON 文件 + 合并策略参数 `strategy=replace|skip`
- `replace`：主键冲突时覆盖（`INSERT OR REPLACE`）
- `skip`：主键冲突时跳过（`INSERT OR IGNORE`）
- 默认：`replace`

## 4. API 设计

### 4.1 完整备份

```http
POST /api/action/backup/export
Authorization: Bearer <admin-jwt>

→ 200 OK
Content-Type: application/zip
Content-Disposition: attachment; filename="blog_api_backup_20260730_120000.zip"
```

```http
POST /api/action/backup/import
Authorization: Bearer <admin-jwt>
Content-Type: multipart/form-data

file=<zip>

→ 200 OK
{
  "code": 200,
  "message": "success",
  "data": {
    "backup_path": "data.bak.20260730_120000",
    "notice": "请重启服务以完成恢复"
  }
}
```

### 4.2 分模块导出

```http
GET /api/action/export/friend_links
Authorization: Bearer <admin-jwt>

→ 200 OK
Content-Type: application/json; charset=utf-8
Content-Disposition: attachment; filename="friend_links_20260730_120000.json"
```

### 4.3 分模块导入

```http
POST /api/action/import/friend_links?strategy=replace
Authorization: Bearer <admin-jwt>
Content-Type: multipart/form-data

file=<json>

→ 200 OK
{
  "code": 200,
  "message": "success",
  "data": {
    "imported": 10,
    "skipped": 0,
    "replaced": 2
  }
}
```

## 5. 前端 UI 设计

在 `/panel/settings` 页面新增「数据迁移」卡片：

### 5.1 完整备份区域

- 标题：完整备份
- 按钮：导出备份（下载 ZIP）
- 按钮：恢复备份（上传 ZIP）
- 警告文案：恢复备份会覆盖当前 data/ 目录，操作后请重启服务

### 5.2 分模块导入导出区域

- 标题：模块数据迁移
- 表格列：模块名称、导出按钮、导入按钮、合并策略选择（replace/skip）
- 模块列表：`系统配置`、`友链`、`动态`、`RSS`、`图片清单`

## 6. 安全与错误处理

### 6.1 权限

- 所有接口仅允许管理员 JWT 访问
- 非管理员返回 403

### 6.2 完整备份恢复安全

- 导入前必须校验 ZIP 包含 `database.db`
- 导入前自动创建 `data.bak.<timestamp>`
- 解压失败时回滚（恢复备份目录）
- 大文件限制：默认 500MB，可配置

### 6.3 分模块导入安全

- 校验 JSON 顶层字段 `version`、`module`、`items`
- 校验 `module` 与 URL 参数一致
- 所有写入在数据库事务中完成
- 导入计数返回：imported / skipped / replaced

### 6.4 常见错误

| 场景 | 响应 |
|------|------|
| 非 ZIP/JSON 文件 | 400 bad request |
| 文件过大 | 413 payload too large |
| ZIP 结构缺失 database.db | 400 invalid backup |
| module 不匹配 | 400 module mismatch |
| 写入失败 | 500 并保留原数据 |

## 7. 实现阶段

### Phase 1：后端骨架

- 新建 `src/handler/backup/backup.go`
- 新建 `src/service/backup/backup.go`
- 实现 ZIP 导出
- 注册路由（admin 中间件）

### Phase 2：完整备份恢复

- 实现 ZIP 导入（校验 + 备份 + 解压覆盖）
- 错误回滚

### Phase 3：分模块导出

- 为每个模块实现 JSON 序列化
- 统一包装格式

### Phase 4：分模块导入

- 为每个模块实现 `INSERT OR REPLACE` / `INSERT OR IGNORE`
- 返回导入统计

### Phase 5：前端 Settings 页面

- 在 `web/src/views/Settings.vue` 新增「数据迁移」区域
- 实现文件上传/下载

### Phase 6：测试与文档

- 单元测试 ZIP 打包/解压
- 集成测试各模块导入导出
- 更新 README API 文档

## 8. 边界与限制

- 完整备份恢复后需要重启服务，因为 SQLite 文件被进程占用
- 图片/资源实际文件只在完整备份中处理；分模块 `images` 仅迁移元数据
- 运行时表（fingerprints、reactions 等）暂不支持分模块迁移
- 导入的 JSON 版本若高于当前支持版本，返回 400
