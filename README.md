# blog_api

是时候给 memos / fcircle / 随机图 api 扔到 土立土及木甬 里面去了

一个基于 `Go + Gin + SQLite + Vue3` 的博客周边 API 服务，覆盖：`memos`（动态）、`fcircle`（友链与 RSS 聚合）、`随机图 API`（图片仓库与图片直链）。

> 我有我的便利，你有你的自由
> 
> 后端 API 可独立运行，不依赖内置 `web` 面板；你也可以完全自己做一个面板对接这些接口。

## 这项目到底做了啥

- `memos`：动态内容（moments）增删改查、媒体绑定、点赞/取消点赞。
- `fcircle`：友链管理、友链 RSS 拉取、RSS 文章列表对外输出。
- `随机图 API`：图片入库、更新、删除、公开访问（`/api/public/image/*id`）。
- 资源管理：支持本地资源上传/删除，也支持 OSS 上传/删除。
- 管理后台：内置 `web/`（Vue3 + Element Plus）仅作为默认实现，可不用或自行替换。
- 反机器人能力：支持 Turnstile、指纹签发、邮箱验证码（按配置启用）。

## 快速启动

### 1. 准备配置

- 复制 `.env.example` 到 `.env` 并按需修改。
- 复制 `system_config.example.json` 为 `data/config/system_config.json`。
- 可选：复制 `friend_list.example.json` 为 `data/config/friend_list.json`（用于初始化友链）。

### 2. 启动后端

```bash
go run main.go
```

默认监听：`0.0.0.0:10024`。

### 3. 启动前端管理面板（可选）

```bash
cd web
npm install
npm run dev
```

开发访问：`http://localhost:5173/panel/login`。

构建发布：

```bash
cd web
npm run build
```

构建产物输出到 `data/panel`，由后端统一托管。

## 关键接口（示例）

- 公共接口：
  - `GET /api/public/moments/`
  - `GET /api/public/rss/`
  - `GET /api/public/friend/`
  - `GET /api/public/image/*id`

- 管理接口（JWT）：
  - `GET /api/action/moments`
  - `POST /api/action/rss`
  - `POST /api/action/image`
  - `POST /api/action/resource/local`

- 认证相关：
  - `POST /api/verify/passwd`
  - `POST /api/verify/email`
  - `POST /api/verify/turnstile`
  - `POST /api/verify/fingerprint`

## 目录结构（简版）

```text
.
├── main.go                # 程序入口
├── src/                   # 后端源码
├── migrations/            # SQLite SQL 迁移文件
├── data/config/           # JSON 配置
└── web/                   # 管理后台（Vue3）
```
