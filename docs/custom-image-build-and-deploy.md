# 私有镜像构建与部署操作说明

本文档描述本仓库（二次开发版 new-api，分支 `dev-custom`）从代码修改到私有 Docker 镜像构建、推送、部署的完整流程。

## 环境概览

| 角色 | 说明 |
| --- | --- |
| 开发机 | Windows，代码仓库位于 `C:\vueproject\new-api-cdn`，分支 `dev-custom`，远程 `origin` 为私有仓库 `https://github.com/54sww/new-api.git` |
| 构建/仓库服务器 | `172.168.11.11`（CentOS 7，root 登录），已安装 Docker 27.5.1（静态二进制 + systemd，安装于 2026-09） |
| 私有镜像仓库 | 服务器上运行的 `registry:2` 容器，地址 `172.168.11.11:5000`，数据持久化在宿主机 `/data/registry`，无认证，仅内网使用 |
| 代码目录（服务器） | `/opt/new-api` |

服务器上 Docker 相关的既有配置：

- `/etc/docker/daemon.json` 已配置 `registry-mirrors`（daocloud / 1ms.run 国内加速）和 `insecure-registries: ["172.168.11.11:5000"]`。
- Dockerfile 的 Go 构建阶段已默认使用 `GOPROXY=https://goproxy.cn,direct`（服务器无法访问 proxy.golang.org），可用 build-arg 覆盖。

## 版本号规则

每次构建使用同一个版本号字符串，同时用于两处：

1. **`VERSION` 文件**（仓库根目录）：Dockerfile 通过 ldflags 注入 `common.Version`，决定程序自报版本（前台"关于"页、`/api/status`）。
2. **镜像 tag**：`172.168.11.11:5000/new-api:<版本号>`。

格式：`v<基础版本>-custom.<构建日期YYYYMMDD>.<代码commit短hash>`

例如：`v0.13.1-custom.20260902.2768c491c`

## 一、开发机：提交代码并生成版本

```bash
cd C:/vueproject/new-api-cdn
# ...修改代码、git add / git commit...

# 生成版本号并写入 VERSION 文件（注意：最后一个 commit 的 hash 会进入版本号）
V="v0.13.1-custom.$(date +%Y%m%d).$(git rev-parse --short HEAD)"
echo -n "$V" > VERSION
git add VERSION
git commit -m "chore: set image version $V"
```

> `VERSION` 文件不要有多余换行/空格，前台直接展示其内容。

## 二、传输代码到服务器

服务器无法直接克隆私有仓库（GitHub 私有 + 网络限制），使用 git bundle 传输：

```bash
git bundle create /tmp/new-api.bundle dev-custom
scp /tmp/new-api.bundle root@172.168.11.11:/tmp/
```

## 三、服务器：更新代码并构建镜像

SSH 登录服务器后：

```bash
cd /opt
rm -rf new-api
git clone -b dev-custom /tmp/new-api.bundle new-api
rm /tmp/new-api.bundle   # bundle 含全部私有代码，用完即删

cd new-api
cat VERSION   # 确认版本号与预期一致

# 构建（tag 与 VERSION 一致；首次构建较慢，之后有层缓存）
docker build -t 172.168.11.11:5000/new-api:v0.13.1-custom.YYYYMMDD.<hash> .

# 构建日志排错（如构建过程另起了后台任务）
# tail -50 /var/log/new-api-build.log
```

> 服务器 git 为 1.8.3（CentOS 7 自带），上述"删除后重新 clone"是兼容写法。若服务器 git 较新，可用 `git fetch /tmp/new-api.bundle dev-custom && git reset --hard FETCH_HEAD` 增量更新。

## 四、推送镜像到私有仓库

```bash
docker push 172.168.11.11:5000/new-api:v0.13.1-custom.YYYYMMDD.<hash>

# 验证仓库中的镜像列表
curl -s http://172.168.11.11:5000/v2/new-api/tags/list
```

## 五、部署运行

在任意需要运行该服务的机器上（需先在 `/etc/docker/daemon.json` 配置 `"insecure-registries": ["172.168.11.11:5000"]` 并重启 Docker；Windows Docker Desktop 在 Settings → Docker Engine 中配置）：

```bash
docker pull 172.168.11.11:5000/new-api:v0.13.1-custom.YYYYMMDD.<hash>

docker run -d --name new-api --restart=always \
  -p 3000:3000 \
  -v /data/new-api:/data \
  172.168.11.11:5000/new-api:v0.13.1-custom.YYYYMMDD.<hash>
```

- SQLite 数据库默认落在容器 `/data`，已挂载到宿主机 `/data/new-api`；使用外部 MySQL/PostgreSQL 时通过环境变量配置（见 `docs/installation/`）。
- 升级版本：`docker pull` 新 tag 后 `docker rm -f new-api` 再用新 tag 重新 `docker run`（数据在宿主机卷中不受影响），或使用 docker-compose 管理。

## 六、验证本次修复（client_gone 计费）

镜像 `v0.13.1-custom.20260902.2768c491c` 及之后版本包含修复：流式请求客户端断连后继续读完上游，保留 usage/计费。

验证方法：

```bash
# 发起流式请求并在 2 秒后主动断开
curl -N --max-time 2 http://<部署地址>:3000/v1/chat/completions \
  -H "Authorization: Bearer <token>" -H "Content-Type: application/json" \
  -d '{"model":"<模型>","stream":true,"messages":[{"role":"user","content":"写一篇长文"}]}'
```

然后在管理后台日志页确认该请求：

- `end_reason` 为 `client_gone`，`end_error` 为 `context canceled`；
- 软错误列表**不再出现**大量重复的 `request context done: context canceled`；
- `completion_tokens` 不为 0（usage 已从流末尾读到并计费）。

## 常见问题

| 现象 | 处理 |
| --- | --- |
| 构建在 `go mod download` 超时失败 | 确认 Dockerfile builder2 阶段的 `GOPROXY` ARG 未被覆盖为不可达地址；默认 `goproxy.cn` |
| `docker push` 报 `http: server gave HTTP response to HTTPS client` | 该机器未配置 `insecure-registries`，按第五节配置后重启 Docker |
| 服务器重启后 registry 容器未运行 | 容器已设 `--restart=always`，检查 `systemctl status docker`；数据在 `/data/registry` |
| bundle clone 后看不到新提交 | bundle 是快照，每次更新需重新生成并传输 |

## 安全注意事项

- 私有仓库 `172.168.11.11:5000` **无认证**，任何内网可达机器均可推拉镜像；如需账号密码（htpasswd basic auth）或 TLS，需另行配置。
- 服务器使用 root 密码登录，密码出现过在对话/脚本中，建议尽快改密并改用 SSH 密钥。
- `/tmp/new-api.bundle` 与构建机上的镜像包含全部私有代码，传输/存留时注意清理。
