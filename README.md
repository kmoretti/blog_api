# blog_api

一个基于 `Go + Gin + SQLite + Vue 3` 的博客周边 API 服务，提供：

- Memos 动态接口
- 友链管理与 RSS 聚合
- 随机图与图片资源接口
- 内置 Vue 3 管理面板
- Turnstile、邮箱验证码和反机器人能力

后端 API 可以独立运行，也可以使用或替换内置 `web/` 管理面板。

## 快速开始

### 本地运行

```bash
cp .env.example .env
cp system_config.example.json data/config/system_config.json
go run main.go
```

默认地址：`http://localhost:10024`

### Docker Compose

```bash
cp .env.example .env
cp system_config.example.json data/config/system_config.json
docker compose pull blog-api
docker compose up -d blog-api
```

管理面板地址：`http://localhost:10024/panel/login`

生产环境请修改 `.env` 中的管理账号、密码和 `JWT_SECRET`，并为 `data/config/system_config.json` 设置独立密钥。不要提交 `.env`、数据库、Token 或其他敏感信息。

## 常用接口

所有接口路径都以 `/api` 开头。

| 功能 | 接口 |
| --- | --- |
| 管理员登录 | `POST /api/verify/passwd` |
| 系统配置 | `GET /api/public/verify_conf` |
| 公开动态 | `GET /api/public/moments/` |
| 公开友链 | `GET /api/public/friend/` |
| 提交友链申请 | `POST /api/public/friend/apply` |
| 提交友链更新申请 | `POST /api/public/friend/update-apply` |
| 申请记录 | `GET /api/public/friend/submissions` |
| 公开图片 | `GET /api/public/image/:id` |

## 友链申请

匿名访客可以通过 `POST /api/public/friend/apply` 提交友链申请。申请默认进入 `pending` 状态，需要管理员审核后才会出现在公开友链列表中。

表单必填字段：

- 站点名称
- 站点地址
- 头像地址

可选字段包括站点描述、联系邮箱、站点截图、友链页面、RSS 地址和 RSS 开关。

完整字段说明、请求示例、Turnstile 配置、申请更新、审核状态和前端嵌入示例，请查看 [docs.md](docs.md)。

## 开发

```bash
cd web
pnpm install
pnpm run dev
```

前端开发地址：`http://localhost:5173/panel/login`

构建前端：

```bash
pnpm run build
```

## 文档

完整文档请查看 [docs.md](docs.md)，包括：

- Docker 部署和更新
- 配置与数据持久化
- 认证方式
- Memos、友链、图片和资源 API
- 友链申请表单完整字段
- Turnstile 与邮箱通知
- 公开申请列表
- 前端嵌入代码示例
- 常见问题排查
