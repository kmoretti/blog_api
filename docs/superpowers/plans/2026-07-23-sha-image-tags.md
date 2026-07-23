# SHA 镜像标签实施计划

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 在保留 `latest` 一键部署的同时，为每次 main 构建发布 `sha-<短提交哈希>` 镜像标签，并支持生产环境按标签固定版本。

**Architecture:** GitHub Actions 继续发布 `latest`，同时使用 Docker metadata action 根据当前提交生成 `sha-<短哈希>`。Compose 使用 `${IMAGE_TAG:-latest}`，不设置变量时行为完全兼容现有部署，设置 `IMAGE_TAG` 时可指定固定版本并支持回滚。业务代码、数据库和运行时镜像不变。

**Tech Stack:** GitHub Actions, docker/metadata-action, Docker Compose variable interpolation, Markdown.

---

### Task 1: 发布 latest 和 SHA 镜像标签

**Files:**
- Modify: `.github/workflows/docker-publish.yml:32-39`

- [ ] 保留 `type=raw,value=latest`。
- [ ] 增加 `type=sha,prefix=sha-`，让每次 main 构建生成形如 `sha-a1b2c3d` 的短提交标签。
- [ ] 保持 `tags: ${{ steps.meta.outputs.tags }}`，使两个标签同时推送。

### Task 2: 让 Compose 默认 latest、可选固定版本

**Files:**
- Modify: `docker-compose.yml:3`

- [ ] 将固定镜像标签改为 `${IMAGE_TAG:-latest}`。
- [ ] 确保未设置 `IMAGE_TAG` 时 Compose 仍解析为 `ghcr.io/kmoretti/blog_api:latest`。
- [ ] 确保可以通过 `IMAGE_TAG=sha-a1b2c3d docker compose ...` 指定固定镜像。

### Task 3: 更新部署文档

**Files:**
- Modify: `README.md` 的 GHCR 部署、更新和回滚章节

- [ ] 说明 CI 同时发布 `latest` 和 `sha-<短提交哈希>`。
- [ ] 保留无需额外参数的 latest 一键部署命令。
- [ ] 增加查看可用 SHA 标签、固定 SHA 更新和 SHA 回滚命令。
- [ ] 说明 `IMAGE_TAG` 只在当前命令中生效，避免污染后续 shell 会话。
- [ ] 说明生产环境可在 `.env` 中设置 `IMAGE_TAG=sha-...`，但默认不设置时仍使用 latest。

### Task 4: 验证

**Files:**
- Test: `.github/workflows/docker-publish.yml`, `docker-compose.yml`, `README.md`

- [ ] 使用 YAML/文本检查确认 metadata action 同时包含 raw latest 和 sha 标签。
- [ ] 使用 `docker compose config` 验证默认标签和 `IMAGE_TAG` 覆盖标签。
- [ ] 运行 `git diff --check`。
- [ ] 确认没有修改业务代码、`.env` 密钥或数据挂载配置。
