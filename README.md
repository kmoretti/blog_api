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

> 修改 `.env` 或 `data/config/*.json` 后需要重启后端服务才会生效；当前版本不支持完整热重载。

## Docker / Docker Compose 部署

这是推荐的生产部署方式。项目会从 GHCR 拉取已经构建好的单容器镜像，镜像内同时包含 Go API 服务和 Vue 管理面板，不需要在服务器上安装 Go、Node.js 或 pnpm。

当前镜像面向 `x86_64 / linux/amd64` 架构。GitHub Actions 每次构建 `main` 时会同时发布两个标签：

```text
ghcr.io/kmoretti/blog_api:latest
ghcr.io/kmoretti/blog_api:sha-a1b2c3d
```

其中 `latest` 始终指向最近一次构建，`sha-a1b2c3d` 固定对应某次 Git 提交。

以下命令默认在服务器上的项目目录执行。建议将 `README.md`、`docker-compose.yml`、`.env`、`data/` 等文件放在同一个目录中。

### 1. 环境要求

服务器需要安装：

- Docker Engine
- Docker Compose v2，也就是支持 `docker compose` 命令的版本
- `x86_64 / amd64` CPU 架构

检查 Docker 和 Compose 是否可用：

```bash
docker --version
docker compose version
```

如果服务器不是 `amd64` 架构，当前 GHCR 工作流发布的镜像不能直接作为该架构的部署镜像使用。

### 2. 准备部署目录和配置

进入项目目录，创建持久化目录并复制配置模板：

```bash
mkdir -p data/config data/image
cp .env.example .env
cp system_config.example.json data/config/system_config.json
```

可选地复制示例友链配置：

```bash
cp friend_list.example.json data/config/friend_list.json
```

生产环境启动前，必须编辑 `.env`，至少检查并修改以下配置：

```dotenv
WEB_PANEL_USER = "admin"
WEB_PANEL_PWD = "请修改为强密码"
JWT_SECRET = "请替换为长期保存的随机字符串"
```

`JWT_SECRET` 一旦用于生产环境，应在后续更新中保持不变，否则已有登录令牌会失效。不要将包含真实密码、Token 或密钥的 `.env` 提交到代码仓库。

建议使用随机值生成 JWT 密钥，例如：

```bash
openssl rand -hex 32
```

还需要编辑 `data/config/system_config.json`，将指纹密钥从默认值：

```json
"secret": "change-me"
```

替换为独立的随机值。不要在命令、日志、截图或公开文档中暴露这些凭据。

如果启用了 Telegram、Discord、OSS、邮箱或 Turnstile 等功能，还需要按照 `system_config.json` 和 `.env.example` 中的字段补充对应配置。配置文件修改后需要重新创建或重启容器才会生效。

### 3. 数据和配置的持久化范围

Compose 配置会使用以下挂载：

```text
宿主机 ./data  -> 容器 /app/data
宿主机 ./.env  -> 容器 /app/.env，只读
```

容器内的前端面板位于 `/app/panel`，它在镜像中提供，不在 `data/` 挂载目录中。不要创建或挂载宿主机 `panel/` 来替换它，否则无法按当前 Compose 配置正确提供面板。

通常需要备份和迁移的内容包括：

- `data/database.db`
- `data/config/`
- `data/image/`
- `data/resource/` 或其他由资源配置使用的目录
- `.env`

不要删除 `data/`，否则会丢失数据库、系统配置和上传资源。

### 4. 私有 GHCR 镜像登录

如果 GHCR 中的镜像是私有的，先使用具有 `read:packages` 权限的 GitHub Token 登录。不要把 Token 直接写在命令行参数中：

```bash
read -rsp 'GHCR_TOKEN: ' GHCR_TOKEN; export GHCR_TOKEN; echo
printf '%s' "$GHCR_TOKEN" | docker login ghcr.io -u YOUR_GITHUB_USERNAME --password-stdin
unset GHCR_TOKEN
```

如果镜像是公开的，可以跳过登录步骤。

### 5. 拉取镜像并启动服务

首次启动时执行：

```bash
docker compose pull blog-api
docker compose up -d blog-api
```

查看容器状态：

```bash
docker compose ps blog-api
```

服务配置：

- Compose 服务名：`blog-api`
- 容器端口：`10024`
- 宿主机端口：`10024`
- 重启策略：`unless-stopped`
- 平台：`linux/amd64`

容器健康检查使用公开接口：

```text
http://127.0.0.1:10024/api/public/verify_conf
```

查看健康检查结果：

```bash
docker inspect --format='{{json .State.Health}}' $(docker compose ps -q blog-api)
```

也可以从服务器主机直接请求：

```bash
curl --fail --silent --show-error http://127.0.0.1:10024/api/public/verify_conf
```

确认容器状态为 `healthy` 后，在浏览器访问：

```text
http://服务器地址:10024/panel/login
```

如果服务器前面配置了反向代理，建议只开放代理端口，并通过 HTTPS 访问管理面板。

### 6. 常用 Docker Compose 操作

查看实时日志：

```bash
docker compose logs -f blog-api
```

查看最近日志：

```bash
docker compose logs --tail=200 blog-api
```

停止服务但保留容器和数据：

```bash
docker compose stop blog-api
```

启动已停止的服务：

```bash
docker compose start blog-api
```

重启服务：

```bash
docker compose restart blog-api
```

停止并删除容器，但不会删除 `./data`：

```bash
docker compose down
```

不要使用会删除卷或宿主机数据的清理命令，除非你已经确认不再需要这些数据。

### 7. 更新镜像

#### 一键更新到最新版本

Compose 默认使用 `latest`，因此不设置 `IMAGE_TAG` 时，现有一键部署流程保持不变：

```bash
docker compose stop blog-api
docker compose pull blog-api
docker compose up -d blog-api
docker compose ps blog-api
```

也可以简化为：

```bash
docker compose pull blog-api
docker compose up -d blog-api
```

#### 固定部署某个提交版本

每次 `main` 构建都会发布类似下面的 SHA 标签：

```text
sha-a1b2c3d
```

指定 SHA 标签时，将 `IMAGE_TAG` 传给当前命令：

```bash
IMAGE_TAG=sha-a1b2c3d docker compose pull blog-api
IMAGE_TAG=sha-a1b2c3d docker compose up -d blog-api
```

这种写法只对当前命令生效，不会修改服务器上的 Compose 文件，也不会影响之后默认使用 `latest`。

如果希望服务器固定运行某个版本，可以在部署目录的 `.env` 中增加：

```dotenv
IMAGE_TAG=sha-a1b2c3d
```

之后普通命令会固定使用该版本：

```bash
docker compose pull blog-api
docker compose up -d blog-api
```

删除 `.env` 中的 `IMAGE_TAG` 后，会恢复默认的 `latest`。

#### 回滚到旧版本

回滚时将 `IMAGE_TAG` 设置为之前验证过的 SHA 标签：

```bash
IMAGE_TAG=sha-7f8e9ab docker compose pull blog-api
IMAGE_TAG=sha-7f8e9ab docker compose up -d blog-api
```

更新前建议先备份数据库。确认服务恢复为 `healthy` 后再继续提供流量。更新过程中会有短暂中断；当前 Compose 配置没有配置蓝绿发布或无中断升级。

查看当前容器实际使用的镜像标签和摘要：

```bash
docker inspect --format='{{.Config.Image}}' $(docker compose ps -q blog-api)
docker inspect --format='{{index .RepoDigests 0}}' $(docker compose ps -q blog-api)
```

查看 GHCR 中可用的镜像版本，可以访问仓库的 **Packages → Container images → blog_api → Versions**。

### 8. 更新前备份 SQLite

项目当前默认未启用 WAL。为了避免复制过程中数据库仍被写入，备份前先停止服务：

```bash
docker compose stop blog-api
backup_timestamp=$(date +%Y%m%d-%H%M%S)
cp data/database.db "data/database.db.bak-$backup_timestamp"
[ -f data/database.db-wal ] && cp data/database.db-wal "data/database.db-wal.bak-$backup_timestamp"
[ -f data/database.db-shm ] && cp data/database.db-shm "data/database.db-shm.bak-$backup_timestamp"
```

备份完成后再启动服务：

```bash
docker compose start blog-api
```

如果 `data/database.db` 不存在，说明服务尚未成功初始化或配置路径发生了变化，应先查看日志，不要直接覆盖或删除 `data/`。

### 9. 权限、网络和安全注意事项

- 当前镜像默认以 `root` 用户运行。若使用 rootless Docker、用户命名空间、SELinux、NFS root squash 或其他受限运行环境，请确保容器进程可以读写宿主机 `data/` 目录。
- 防火墙需要允许宿主机的 `10024` 端口，或者允许反向代理访问该端口。生产环境不建议直接将管理面板暴露到公网而不加 HTTPS 或访问控制。
- PWA 的安装能力和部分浏览器功能需要安全上下文，生产环境建议通过反向代理配置 TLS，并使用 HTTPS 访问 `/panel/login`。
- 不要把 `.env`、数据库备份、上传图片、配置文件或包含密钥的日志上传到公共仓库。
- `PPROF_ENABLED` 生产环境建议保持为 `false`。

### 10. 常见问题排查

#### GitHub Actions 推送 GHCR 时出现 `permission_denied: write_package`

如果 GitHub Actions 在构建完成后出现以下错误：

```text
denied: permission_denied: write_package
```

说明 Docker 构建本身已经完成，但用于推送 GHCR 的 `GITHUB_TOKEN` 没有目标镜像包的写权限。按以下顺序检查：

1. 仓库的 **Settings → Actions → General → Workflow permissions** 允许工作流使用读写权限。若组织策略禁止工作流写入，需要由组织管理员调整策略。
2. 工作流的发布任务必须包含 `packages: write` 权限。当前 `.github/workflows/docker-publish.yml` 同时在工作流和 `publish` job 中声明了该权限。
3. 如果 `ghcr.io/kmoretti/blog_api` 已经存在，打开该 GHCR Package 的 **Package settings → Manage Actions access**，将 `kmoretti/blog_api` 仓库加入可访问仓库，并确认包没有被绑定到其他仓库而拒绝当前仓库访问。
4. 确认工作流运行的仓库确实是 `kmoretti/blog_api`，且推送触发分支为 `main`。
5. 如果仓库或组织启用了更严格的 Actions、Packages 或私有仓库策略，需要使用具有 `write:packages` 权限的个人访问令牌登录，或先解除该策略限制。不要把个人访问令牌硬编码到工作流文件中，应使用 GitHub Actions Secret。

工作流权限修正后，需要重新推送一次 `main` 分支或在 Actions 页面重新运行失败的工作流。该错误与 `libc6`、前端构建警告和 Go 编译无关。



```bash
docker compose logs --tail=200 blog-api
```

重点检查：

- `.env` 是否存在且格式正确
- `data/config/system_config.json` 是否存在且为合法 JSON
- `data/` 是否具有读写权限
- 宿主机是否为 `amd64` 架构
- `10024` 端口是否已被其他服务占用

#### 状态不是 `healthy`

确认容器是否仍在运行，并直接请求健康检查接口：

```bash
docker compose ps blog-api
curl --fail --silent --show-error http://127.0.0.1:10024/api/public/verify_conf
```

如果接口无法访问，查看日志和容器健康检查记录：

```bash
docker inspect --format='{{json .State.Health}}' $(docker compose ps -q blog-api)
docker compose logs --tail=200 blog-api
```

健康检查只验证 HTTP 服务和公开配置接口可访问，不等同于完整验证所有数据库或资源写入功能。

#### 登录后很快失效或重启容器后需要重新登录

检查 `.env` 中是否设置了稳定的 `JWT_SECRET`，并确认 Compose 同时加载了：

```yaml
env_file:
  - ./.env
```

当前配置还会将 `.env` 只读挂载到 `/app/.env`，两者分别用于环境变量读取和配置文件读取。

#### 管理面板显示 404

确认访问路径包含 `/panel/login`，而不是根路径 `/login`：

```text
http://服务器地址:10024/panel/login
```

同时确认没有将宿主机 `./data` 之外的目录错误挂载到 `/app/panel`。

#### 图片或配置修改后丢失

确认 Compose 服务包含：

```yaml
volumes:
  - ./data:/app/data
```

如果只删除并重新创建容器，`data/` 中的数据应继续保留；如果删除了宿主机 `data/`，则无法恢复其中的数据。

### 11. 从源码构建镜像（可选）

默认部署不需要本地构建，直接使用 GHCR 镜像即可。如果需要从当前源码构建：

```bash
docker build --platform linux/amd64 -t blog-api:local .
```

然后修改 `docker-compose.yml` 中的镜像地址，或使用临时 Compose 配置运行本地镜像。Dockerfile 会分别构建 Vue 面板和启用 CGO 的 Go 后端，最终运行时镜像只包含二进制文件、迁移文件和面板静态文件。

本项目当前的本地 Docker 构建和 Compose 运行验证不在本文档编写范围内，请以你的服务器环境实际执行结果为准。

## GitHub 仓库卡片与 API Token

动态中的 GitHub 仓库卡片会通过后端代理请求 GitHub 仓库元数据，前端不会接触 GitHub Token。代理接口为：

```text
GET /api/public/github/repository/:owner/:repo
```

如果未配置 Token，卡片仍可显示仓库基础信息，但 GitHub API 使用未认证请求，容易受到较低的速率限制。建议在服务器 `.env` 中配置只读 Token：

```dotenv
GH_TOKEN="github_pat_xxxxxxxxxxxxxxxxxxxx"
```

Token 使用建议：

- 使用 GitHub Fine-grained personal access token
- 只授予访问所需公开仓库的最小读取权限
- 不要将 Token 写入 `system_config.json`、前端变量、Dockerfile 或 Git 仓库
- 不要将真实 Token 写入 `.env.example`
- 当前 Compose 会通过 `env_file: ./.env` 将 Token 注入后端容器
- 修改 Token 后重启容器使其生效：

```bash
docker compose restart blog-api
```

如果 GitHub API 请求失败，卡片会降级显示仓库路径，不会影响动态页面的其他内容。



Docker 部署不需要执行以下步骤。只有在本地开发或修改前端时，才需要启动前端开发服务器：

```bash
cd web
pnpm install
pnpm run dev
```

开发访问：`http://localhost:5173/panel/login`。

构建发布版前端：

```bash
cd web
pnpm run build
```

构建产物输出到 `data/panel`，本地开发时由后端统一托管。

### 一键脚本（Windows）

项目根目录提供了几个 PowerShell 脚本，简化日常操作：

```powershell
# 首次使用 - 一键初始化配置（创建目录、复制配置、安装前端依赖）
.\setup.ps1

# 日常启动（任选其一）
.\start-backend.ps1       # 仅启动后端
.\start-frontend.ps1      # 仅启动前端开发服务器
.\start-all.ps1           # 同时启动前后端

# 构建前端发布版
.\build-frontend.ps1
```

> 脚本位于项目根目录 `../`，非 `api/` 目录下。依赖操作系统的 PowerShell 执行策略（若提示无法执行，先运行 `Set-ExecutionPolicy -Scope Process -ExecutionPolicy Bypass`）。

## API 接口文档

### 基础信息

默认服务地址：

```text
http://localhost:10024
```

生产环境请将示例地址替换为实际域名，例如：

```text
https://blog-api.2005815.xyz
```

所有接口路径都以 `/api` 开头。接口通常返回统一结构：

```json
{
  "code": 200,
  "message": "success",
  "data": {}
}
```

分页数据通常包含：

```json
{
  "items": [],
  "total": 0,
  "page": 1,
  "page_size": 20
}
```

GitHub 仓库代理接口直接透传 GitHub API 的 JSON 响应，不使用上面的统一响应包装。

### 认证方式

#### 管理端 JWT

登录成功后，从响应中的 `token` 获取 JWT：

```bash
curl -X POST http://localhost:10024/api/verify/passwd \
  -H "Content-Type: application/json" \
  -d '{"username":"admin","password":"你的面板密码"}'
```

后续管理接口使用：

```http
Authorization: Bearer <JWT>
```

JWT 默认有效期为 24 小时。`JWT_SECRET` 配置改变后，原有 JWT 会失效。

#### Turnstile 验证

Turnstile 启用后，登录或验证接口需要传递 token。支持以下位置：

```http
X-Turnstile-Token: <turnstile-token>
```

或 JSON：

```json
{
  "turnstile_token": "<turnstile-token>"
}
```

查询前端是否启用 Turnstile：

```bash
curl http://localhost:10024/api/public/verify_conf
```

#### 反机器人 Token 和指纹 Token

部分公开接口需要反机器人 Token：

```http
X-Antibot-Token: <token>
```

也支持：

```http
Authorization: Bearer <antibot-token>
```

动态 reaction 接口还需要指纹 Token：

```http
X-Fingerprint-Token: <fingerprint-token>
```

或：

```http
Authorization: Fingerprint <fingerprint-token>
```

### 认证与验证接口

| 方法 | 路径 | 认证 | 说明 |
| --- | --- | --- | --- |
| `POST` | `/api/verify/passwd` | Turnstile（启用时） | 管理员登录，返回 JWT |
| `POST` | `/api/verify/email` | 反机器人 Token | 不传 `code` 时发送邮箱验证码，传 `code` 时校验邮箱验证码 |
| `POST` | `/api/verify/turnstile` | Turnstile（启用时） | 签发验证 Token |
| `POST` | `/api/verify/fingerprint` | 反机器人 Token | 创建或获取客户端指纹 Token |

管理员登录请求：

```json
{
  "username": "admin",
  "password": "你的面板密码"
}
```

邮箱验证码请求：

```json
{
  "email": "user@example.com"
}
```

邮箱验证码校验：

```json
{
  "email": "user@example.com",
  "code": "123456"
}
```

### 公开接口

公开接口不需要管理员 JWT。

#### 系统和 GitHub

| 方法 | 路径 | 说明 |
| --- | --- | --- |
| `GET` | `/api/public/verify_conf` | 返回 Turnstile 是否启用及前端 Site Key |
| `GET` | `/api/public/github/repository/:owner/:repo` | 获取 GitHub 仓库元数据 |

GitHub 代理示例：

```bash
curl https://blog-api.2005815.xyz/api/public/github/repository/kmoretti/blog_api
```

`:owner` 和 `:repo` 只允许字母、数字、`.`、`_`、`-`，仓库路径支持可选的 `.git` 后缀。后端会从服务器 `.env` 的 `GH_TOKEN` 读取 GitHub Token，前端不会接触该密钥。

#### 动态 Moments

| 方法 | 路径 | 说明 |
| --- | --- | --- |
| `GET` | `/api/public/moments/` | 分页获取公开动态 |
| `POST` | `/api/public/moments/:id/reactions` | 为动态添加 reaction |
| `DELETE` | `/api/public/moments/:id/reactions` | 删除动态 reaction |

动态查询参数：

| 参数 | 默认值 | 说明 |
| --- | --- | --- |
| `page` | `1` | 页码 |
| `page_size` | `10` | 每页数量，最大 `100` |

查询动态：

```bash
curl "http://localhost:10024/api/public/moments/?page=1&page_size=10"
```

reaction 类型：

```text
👍  👎  ❤  👀  💩
```

添加 reaction：

```bash
curl -X POST http://localhost:10024/api/public/moments/1/reactions \
  -H "Content-Type: application/json" \
  -H "X-Antibot-Token: <antibot-token>" \
  -H "X-Fingerprint-Token: <fingerprint-token>" \
  -d '{"reaction":"👍"}'
```

#### 友链

| 方法 | 路径 | 认证 | 说明 |
| --- | --- | --- | --- |
| `GET` | `/api/public/friend/` | 无 | 分页查询公开友链 |
| `GET` | `/api/public/friend/:id` | 无 | 查询单条公开友链 |
| `GET` | `/api/public/friend/self` | 邮箱 Token | 查询当前邮箱对应的友链 |
| `POST` | `/api/public/friend` | 邮箱 Token 或 JWT | 自助提交友链 |
| `PUT` | `/api/public/friend/:id` | 邮箱 Token 或 JWT | 修改自己提交的友链 |

友链查询参数：

```text
status       状态筛选
search       名称或链接搜索
is_died      是否已失联
page         页码，默认 1
page_size    每页数量，默认 20，最大 1000
```

自助提交友链示例：

```bash
curl -X POST http://localhost:10024/api/public/friend \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer <email-token>" \
  -d '{
    "name":"我的博客",
    "link":"https://example.com",
    "avatar":"https://example.com/avatar.png",
    "description":"博客简介",
    "email":"user@example.com",
    "enable_rss":true
  }'
```

#### RSS 与图片

| 方法 | 路径 | 说明 |
| --- | --- | --- |
| `GET` | `/api/public/rss/` | 分页获取 RSS 文章 |
| `GET` | `/api/public/image/*id` | 获取随机图片、指定图片或图片元数据 |

RSS 查询参数：

```text
rss_id          按 RSS ID 筛选
friend_link_id  按友链 ID 筛选
page            页码，默认 1
page_size       每页数量，默认 10
```

图片接口行为：

```text
GET /api/public/image/
    随机返回一张图片

GET /api/public/image/123
    返回 ID 为 123 的图片

GET /api/public/image/123?type=metadata
    返回图片元数据 JSON
```

图片默认通过 HTTP 302 跳转到最终图片 URL。

### 管理接口 `/api/action`

所有管理接口都需要：

```http
Authorization: Bearer <JWT>
```

#### 系统

| 方法 | 路径 | 说明 |
| --- | --- | --- |
| `GET` | `/api/status` | 获取系统状态 |
| `PUT` | `/api/action/config` | 更新 JSON 系统配置 |

#### 友链、RSS、图片

| 方法 | 路径 | 说明 |
| --- | --- | --- |
| `GET` | `/api/action/friend` | 分页获取完整友链 |
| `GET` | `/api/action/friend/:id` | 获取单条完整友链 |
| `POST` | `/api/action/friend` | 创建友链 |
| `PUT` | `/api/action/friend/:id` | 更新友链 |
| `DELETE` | `/api/action/friend/:id` | 删除友链 |
| `GET` | `/api/action/rss` | 分页获取 RSS 配置 |
| `POST` | `/api/action/rss` | 创建 RSS 配置 |
| `PUT` | `/api/action/rss/:id` | 更新 RSS 配置 |
| `DELETE` | `/api/action/rss/:id` | 删除 RSS 配置 |
| `GET` | `/api/action/image` | 分页获取图片 |
| `POST` | `/api/action/image` | 创建图片记录 |
| `PUT` | `/api/action/image/:id` | 更新图片记录 |
| `DELETE` | `/api/action/image/:id` | 删除图片记录 |

创建 RSS：

```json
{
  "rss_url": "https://example.com/feed.xml",
  "friend_link_id": 1,
  "name": "示例 RSS"
}
```

#### 动态与媒体

| 方法 | 路径 | 说明 |
| --- | --- | --- |
| `GET` | `/api/action/moments` | 分页获取管理端动态 |
| `POST` | `/api/action/moments` | 创建动态 |
| `PUT` | `/api/action/moments/:id` | 更新动态 |
| `DELETE` | `/api/action/moments/:id` | 删除动态 |
| `DELETE` | `/api/action/moments/:id/reactions` | 删除动态 reaction |
| `GET` | `/api/action/moments/media` | 分页获取动态媒体 |
| `POST` | `/api/action/moments/media` | 创建动态媒体 |
| `PUT` | `/api/action/moments/media/:id` | 更新动态媒体 |
| `DELETE` | `/api/action/moments/media/:id` | 删除动态媒体 |

创建动态示例：

```json
{
  "content": "今天完成了一个功能。",
  "tags": "开发,记录",
  "extension": "{\"type\":\"github\",\"payload\":{\"repo_url\":\"https://github.com/kmoretti/blog_api\"}}",
  "media": []
}
```

#### 资源上传与删除

资源接口支持本地存储和 OSS 存储：

| 方法 | 路径 | 请求格式 | 说明 |
| --- | --- | --- | --- |
| `GET` | `/api/action/resource/*file_path` | - | 获取资源列表或文件内容 |
| `POST` | `/api/action/resource/local` | `multipart/form-data` | 上传到本地存储，文件字段为 `file` |
| `POST` | `/api/action/resource/oss` | `multipart/form-data` | 上传到 OSS，文件字段为 `file` |
| `DELETE` | `/api/action/resource/local/*file_path` | - | 删除本地资源 |
| `DELETE` | `/api/action/resource/oss/*file_path` | - | 删除 OSS 资源 |

本地上传示例：

```bash
curl -X POST http://localhost:10024/api/action/resource/local \
  -H "Authorization: Bearer <JWT>" \
  -F "file=@./avatar.png" \
  -F "path=images"
```

OSS 上传示例：

```bash
curl -X POST http://localhost:10024/api/action/resource/oss \
  -H "Authorization: Bearer <JWT>" \
  -F "file=@./photo.jpg" \
  -F "path=moments"
```

#### 系统操作

| 方法 | 路径 | 说明 |
| --- | --- | --- |
| `POST` | `/api/action/system/restart` | 重启服务 |

### 条件启用的内部状态接口

当 `.env` 配置 `STATE_API_MASTER_PASSWORD` 非空时注册。所有接口都需要服务端状态主密码认证：

| 方法 | 路径 | 说明 |
| --- | --- | --- |
| `GET` | `/api/internal/states/:key` | 读取状态 |
| `PUT` | `/api/internal/states/:key` | 写入状态 |
| `DELETE` | `/api/internal/states/:key` | 删除状态 |

未配置 `STATE_API_MASTER_PASSWORD` 时，这组接口不会注册。

### 错误处理与调试

常见 HTTP 状态码：

| 状态码 | 含义 |
| --- | --- |
| `200` | 请求成功 |
| `201` | 创建成功，具体接口以实现为准 |
| `400` | 请求参数或请求体无效 |
| `401` | 未认证或 Token 无效 |
| `403` | 无权限、反机器人验证失败或上游拒绝 |
| `404` | 资源或路由不存在 |
| `429` | 请求过于频繁或上游限流 |
| `500` | 服务端错误 |
| `502` | 上游服务请求失败，例如 GitHub API 网络错误 |

排查接口问题时先查看后端日志：

```bash
docker compose logs --tail=200 blog-api
```

## 目录结构（简版）

```text
.
├── main.go                # 程序入口
├── src/                   # 后端源码
├── migrations/            # SQLite SQL 迁移文件
├── data/config/           # JSON 配置
└── web/                   # 管理后台（Vue3）
```
