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

当前镜像面向 `x86_64 / linux/amd64` 架构：

```text
ghcr.io/kmoretti/blog_api:latest
```

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
read -rsp 'GITHUB_TOKEN: ' GITHUB_TOKEN; export GITHUB_TOKEN; echo
printf '%s' "$GITHUB_TOKEN" | docker login ghcr.io -u YOUR_GITHUB_USERNAME --password-stdin
unset GITHUB_TOKEN
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

`latest` 是可变标签，发布新版本后再次拉取可能得到不同内容。更新前建议先备份数据库，然后执行：

```bash
docker compose stop blog-api
docker compose pull blog-api
docker compose up -d blog-api
docker compose ps blog-api
```

确认服务恢复为 `healthy` 后再继续提供流量。更新过程中会有短暂中断；当前 Compose 配置没有配置蓝绿发布或无中断升级。

如果需要确认实际使用的镜像摘要，可以执行：

```bash
docker inspect --format='{{index .RepoDigests 0}}' $(docker compose ps -q blog-api)
```

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

#### 容器无法启动

先查看完整启动日志：

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

## 本地开发：启动前端管理面板

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
