# SHA 镜像标签设计

**目标：** 在保留 `latest` 一键部署的同时，为每次 main 构建发布可追踪、可回滚的 `sha-<短提交哈希>` 镜像标签。

**方案：** GitHub Actions 使用 Docker metadata action 同时生成 `latest` 和 `sha-<短提交哈希>` 标签。Docker Compose 默认使用 `${IMAGE_TAG:-latest}`，不设置变量时保持原有一键部署行为，设置 `IMAGE_TAG=sha-a1b2c3d` 时可以固定部署指定提交镜像。

**部署行为：**

```bash
docker compose pull blog-api
docker compose up -d blog-api
```

默认拉取 `latest`。固定版本时：

```bash
IMAGE_TAG=sha-a1b2c3d docker compose pull blog-api
IMAGE_TAG=sha-a1b2c3d docker compose up -d blog-api
```

不修改应用代码、数据库、数据挂载、环境变量或运行时镜像结构。
