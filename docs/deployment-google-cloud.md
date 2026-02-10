# OpenFMS Google Cloud 部署指南

> 部署时间: 2026-02-08
> 适用版本: v1.2+

---

## 一、部署方案对比

| 方案 | 适用场景 | 复杂度 | 成本 | 扩展性 |
|------|----------|--------|------|--------|
| **A. GCE + Docker Compose** | 测试/小规模 | ⭐ 低 | 低 | 手动 |
| **B. GKE Standard** | 生产环境 | ⭐⭐⭐ 高 | 中 | 自动 |
| **C. Cloud SQL + GCE** | 生产推荐 | ⭐⭐ 中 | 中 | 手动 |
| **D. Cloud Run + Cloud SQL** | 无服务器 | ⭐⭐ 中 | 按需 | 自动 |

**推荐选择：**
- 🧪 **测试验证** → 方案 A（单台 GCE）
- 🏭 **生产环境** → 方案 C（Cloud SQL + GCE）

---

## 二、方案 A：GCE + Docker Compose（推荐测试）

### 2.1 架构图

```
┌─────────────────────────────────────────┐
│         Google Compute Engine           │
│              (e2-medium)                │
│                                         │
│  ┌──────────────┐  ┌──────────────┐    │
│  │   API 服务    │  │  Gateway 服务 │    │
│  │   :3000      │  │   :8080/8081  │    │
│  └──────────────┘  └──────────────┘    │
│                                         │
│  ┌──────────────┐  ┌──────────────┐    │
│  │   Web 前端    │  │   ZLMediaKit  │    │
│  │   :80        │  │   (视频服务)   │    │
│  └──────────────┘  └──────────────┘    │
│                                         │
│  ┌──────────────┐  ┌──────────────┐    │
│  │  PostgreSQL  │  │    Redis     │    │
│  │  (TimescaleDB)│  │              │    │
│  └──────────────┘  └──────────────┘    │
│                                         │
│  ┌──────────────┐  ┌──────────────┐    │
│  │     NATS     │  │  Prometheus  │    │
│  │              │  │   Grafana    │    │
│  └──────────────┘  └──────────────┘    │
└─────────────────────────────────────────┘
```

### 2.2 部署步骤

#### 步骤 1：创建 GCE 实例

```bash
# 在 Google Cloud Console 或使用 gcloud CLI

gcloud compute instances create openfms-server \
  --machine-type=e2-medium \
  --image-family=ubuntu-2204-lts \
  --image-project=ubuntu-os-cloud \
  --boot-disk-size=100GB \
  --boot-disk-type=pd-ssd \
  --tags=http-server,https-server,openfms \
  --zone=asia-east1-a
```

**配置建议：**
- **测试环境**: e2-medium (2 vCPU, 4GB 内存) - 约 $25/月
- **生产环境**: e2-standard-4 (4 vCPU, 16GB 内存) - 约 $100/月

#### 步骤 2：配置防火墙规则

```bash
# 允许 HTTP/HTTPS
gcloud compute firewall-rules create allow-http \
  --allow tcp:80,tcp:443 \
  --target-tags=openfms

# 允许 OpenFMS 端口
gcloud compute firewall-rules create allow-openfms \
  --allow tcp:3000,tcp:8080,tcp:8081,tcp:9090,tcp:3001 \
  --target-tags=openfms
```

#### 步骤 3：安装 Docker 和 Docker Compose

```bash
# SSH 进入实例
gcloud compute ssh openfms-server --zone=asia-east1-a

# 安装 Docker
sudo apt-get update
sudo apt-get install -y docker.io

# 安装 Docker Compose
sudo curl -L "https://github.com/docker/compose/releases/download/v2.23.0/docker-compose-$(uname -s)-$(uname -m)" -o /usr/local/bin/docker-compose
sudo chmod +x /usr/local/bin/docker-compose

# 启动 Docker
sudo systemctl start docker
sudo systemctl enable docker
sudo usermod -aG docker $USER

# 退出并重新登录
exit
```

#### 步骤 4：部署 OpenFMS

```bash
# 重新 SSH 登录
gcloud compute ssh openfms-server --zone=asia-east1-a

# 克隆代码（或上传代码）
cd ~
git clone https://github.com/your-org/openfms.git
# 或者使用 scp 上传本地代码
# gcloud compute scp --recurse ./openfms openfms-server:~ --zone=asia-east1-a

cd openfms

# 启动服务
docker-compose up -d

# 查看日志
docker-compose logs -f
```

#### 步骤 5：配置域名和 HTTPS（可选）

使用 Cloudflare 或 Google Cloud Load Balancer 配置 HTTPS：

```bash
# 安装 certbot（使用 Let's Encrypt）
sudo apt-get install -y certbot

# 或者使用 Cloudflare Origin CA
# 下载证书并配置 nginx
```

---

## 三、方案 C：Cloud SQL + GCE（推荐生产）

### 3.1 架构图

```
┌──────────────────┐     ┌──────────────────┐
│   Cloud SQL      │     │  Compute Engine  │
│  (PostgreSQL)    │◄────│   (API/Web)      │
│   Managed        │     │   Docker Compose │
└──────────────────┘     └──────────────────┘
         ▲                        │
         │                        ▼
         │               ┌──────────────────┐
         │               │    Cloud NAT     │
         │               │   (固定出口IP)   │
         │               └──────────────────┘
         │
┌──────────────────┐
│    Memorystore   │
│     (Redis)      │
└──────────────────┘
```

### 3.2 部署步骤

#### 步骤 1：创建 Cloud SQL 实例

```bash
# 创建 PostgreSQL 实例
gcloud sql instances create openfms-db \
  --database-version=POSTGRES_15 \
  --tier=db-f1-micro \
  --storage-type=SSD \
  --storage-size=100GB \
  --region=asia-east1 \
  --availability-type=ZONAL

# 设置密码
gcloud sql users set-password postgres \
  --instance=openfms-db \
  --password=YourStrongPassword123

# 创建数据库
gcloud sql databases create openfms --instance=openfms-db
```

**连接字符串：**
```
host=/cloudsql/your-project:asia-east1:openfms-db
user=postgres
password=YourStrongPassword123
database=openfms
```

#### 步骤 2：创建 Memorystore (Redis)

```bash
# 创建 Redis 实例
gcloud redis instances create openfms-redis \
  --tier=basic \
  --size=5 \
  --region=asia-east1 \
  --redis-version=redis_7_0

# 获取连接信息
gcloud redis instances describe openfms-redis --region=asia-east1
```

#### 步骤 3：创建 GCE 实例（仅运行应用）

```bash
# 创建应用服务器（无需大存储）
gcloud compute instances create openfms-app \
  --machine-type=e2-medium \
  --image-family=ubuntu-2204-lts \
  --image-project=ubuntu-os-cloud \
  --boot-disk-size=20GB \
  --tags=openfms-app \
  --zone=asia-east1-a \
  --scopes=cloud-platform
```

#### 步骤 4：配置 Cloud SQL Proxy（推荐）

```bash
# SSH 进入应用服务器
gcloud compute ssh openfms-app --zone=asia-east1-a

# 下载 Cloud SQL Auth Proxy
wget https://dl.google.com/cloudsql/cloud_sql_proxy.linux.amd64 -O cloud_sql_proxy
chmod +x cloud_sql_proxy

# 运行代理（使用服务账号）
./cloud_sql_proxy -instances=your-project:asia-east1:openfms-db=tcp:5432 &
```

**优点：**
- 数据流量加密
- 无需配置防火墙规则
- 使用 IAM 认证

#### 步骤 5：修改配置文件

```yaml
# docker-compose.yml 修改数据库连接
services:
  api:
    environment:
      - DATABASE_URL=postgres://postgres:YourStrongPassword123@localhost:5432/openfms?sslmode=disable
      - REDIS_URL=10.0.0.3:6379  # Memorystore IP
```

---

## 四、成本估算

### 方案 A：单台 GCE（测试环境）

| 资源 | 规格 | 月费用 |
|------|------|--------|
| GCE e2-medium | 2 vCPU, 4GB | ~$25 |
| 磁盘 100GB SSD | - | ~$10 |
| 网络 egress | 100GB | ~$10 |
| **总计** | | **~$45/月** |

### 方案 C：分离架构（生产环境）

| 资源 | 规格 | 月费用 |
|------|------|--------|
| GCE e2-standard-2 | 2 vCPU, 8GB | ~$50 |
| Cloud SQL db-f1-micro | 共享 vCPU | ~$10 |
| Memorystore Redis 5GB | Basic | ~$20 |
| 磁盘 50GB | SSD | ~$5 |
| 网络 egress | 500GB | ~$50 |
| **总计** | | **~$135/月** |

### 省钱技巧

1. **使用抢占式实例** - 节省 60-90%（适合测试）
2. **预留实例** - 1年/3年承诺，节省 30-50%
3. **关闭非生产环境** - 使用 Cloud Scheduler 定时关机
4. **使用 Cloudflare** - 缓存静态资源，减少 egress 费用

---

## 五、监控和运维

### 5.1 使用 Google Cloud Monitoring

```bash
# 安装 Cloud Monitoring Agent
curl -sSO https://dl.google.com/cloudagents/add-monitoring-agent-repo.sh
sudo bash add-monitoring-agent-repo.sh
sudo apt-get install -y stackdriver-agent
sudo service stackdriver-agent start
```

### 5.2 设置告警

在 Google Cloud Console：
1. 进入 **Monitoring** → **Alerting**
2. 创建告警策略：
   - CPU 使用率 > 80%
   - 内存使用率 > 85%
   - 磁盘使用率 > 90%

### 5.3 备份策略

```bash
# 自动备份 Cloud SQL
gcloud sql instances patch openfms-db \
  --backup-start-time=03:00 \
  --enable-bin-log

# 创建磁盘快照（每日）
gcloud compute disks snapshot openfms-server \
  --snapshot-names=openfms-backup-$(date +%Y%m%d) \
  --zone=asia-east1-a
```

---

## 六、快速部署脚本

### 一键部署脚本

```bash
#!/bin/bash
# deploy-to-gcp.sh

PROJECT_ID="your-project-id"
ZONE="asia-east1-a"
INSTANCE_NAME="openfms-server"

echo "=== OpenFMS Google Cloud 部署脚本 ==="

# 设置项目
gcloud config set project $PROJECT_ID

# 创建实例
echo "创建 GCE 实例..."
gcloud compute instances create $INSTANCE_NAME \
  --machine-type=e2-medium \
  --image-family=ubuntu-2204-lts \
  --image-project=ubuntu-os-cloud \
  --boot-disk-size=100GB \
  --tags=http-server,https-server,openfms \
  --zone=$ZONE

# 配置防火墙
echo "配置防火墙..."
gcloud compute firewall-rules create allow-openfms \
  --allow tcp:80,tcp:443,tcp:3000,tcp:8080,tcp:8081 \
  --target-tags=openfms

# 等待实例启动
sleep 30

# 安装 Docker
echo "安装 Docker..."
gcloud compute ssh $INSTANCE_NAME --zone=$ZONE --command="
  sudo apt-get update
  sudo apt-get install -y docker.io docker-compose git
  sudo systemctl start docker
  sudo usermod -aG docker \$USER
"

# 克隆代码并启动
echo "部署 OpenFMS..."
gcloud compute ssh $INSTANCE_NAME --zone=$ZONE --command="
  git clone https://github.com/your-org/openfms.git
  cd openfms
  docker-compose up -d
"

# 获取外部 IP
EXTERNAL_IP=$(gcloud compute instances describe $INSTANCE_NAME \
  --zone=$ZONE \
  --format='get(networkInterfaces[0].accessConfigs[0].natIP)')

echo "=== 部署完成 ==="
echo "访问地址:"
echo "  Web 界面: http://$EXTERNAL_IP"
echo "  API 服务: http://$EXTERNAL_IP:3000"
echo "  Grafana:  http://$EXTERNAL_IP:3001"
```

---

## 七、常见问题

### Q1: 如何更新部署？

```bash
# SSH 进入实例
gcloud compute ssh openfms-server --zone=asia-east1-a

cd ~/openfms
# 拉取最新代码
git pull origin main

# 重启服务
docker-compose down
docker-compose up -d --build
```

### Q2: 如何查看日志？

```bash
# 实时日志
docker-compose logs -f

# 查看特定服务
docker-compose logs -f api

# 保存日志到文件
docker-compose logs > openfms-$(date +%Y%m%d).log
```

### Q3: 如何备份数据？

```bash
# 备份 PostgreSQL
docker exec openfms-postgres pg_dump -U postgres openfms > backup.sql

# 下载到本地
gcloud compute scp openfms-server:~/openfms/backup.sql ./backup.sql
```

### Q4: 性能优化建议？

1. **启用 Cloud CDN** - 缓存静态资源
2. **使用 Cloud Load Balancer** - 分发流量
3. **数据库优化** - Cloud SQL 只读副本
4. **启用缓存** - Redis 集群模式

---

## 八、总结

**推荐方案：**
- 🧪 **测试验证** → 方案 A（单台 GCE，约 $45/月）
- 🏭 **生产环境** → 方案 C（Cloud SQL + GCE，约 $135/月）

**部署时间预估：**
- 首次部署：30-60 分钟
- 后续更新：5-10 分钟

**下一步：**
1. 选择部署方案
2. 准备 Google Cloud 环境
3. 运行部署脚本
4. 访问验证

---

**文档维护**: OpenFMS Team  
**最后更新**: 2026-02-08
