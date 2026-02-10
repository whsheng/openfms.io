# OpenFMS Cloud Shell 部署指南

> 部署方式: GitHub + Google Cloud Shell
> 优势: 无需本地安装任何工具，浏览器即可完成全部操作
> 预估时间: 15-20 分钟

---

## 一、方案优势

| 传统方案 (本地 gcloud) | Cloud Shell 方案 (推荐) |
|----------------------|------------------------|
| ❌ 需要安装 gcloud SDK | ✅ 无需安装任何软件 |
| ❌ 需要配置本地环境 | ✅ 浏览器即可操作 |
| ❌ 可能有兼容性问题 | ✅ 官方预装环境 |
| ✅ 适合长期开发 | ✅ 适合快速测试部署 |

**Cloud Shell 免费提供：**
- 5GB 持久化存储
- 预装 gcloud、docker、kubectl、git、vim 等工具
- 基于 Web 的代码编辑器

---

## 二、准备工作（5分钟）

### 步骤 1：推送代码到 GitHub

#### 方式 A：HTTPS（推荐新手）
```bash
# 在本地项目目录执行
cd /Users/whsheng/works/openfms.io

# 初始化 git（如果还没有）
git init

# 添加所有文件
git add .

# 提交
git commit -m "Initial commit: OpenFMS v1.2"

# 在 GitHub 创建仓库后，关联远程仓库
git remote add origin https://github.com/YOUR-USERNAME/openfms.git

# 推送代码（会提示输入用户名和密码/Token）
git branch -M main
git push -u origin main
```

#### 方式 B：SSH（推荐）
```bash
# 1. 生成 SSH 密钥（如果还没有）
ssh-keygen -t ed25519 -C "your-email@example.com"

# 2. 复制公钥到 GitHub
# Settings → SSH and GPG keys → New SSH key
# 粘贴 ~/.ssh/id_ed25519.pub 的内容

# 3. 推送代码
git remote add origin git@github.com:YOUR-USERNAME/openfms.git
git push -u origin main
```

### 步骤 2：验证推送成功

访问 `https://github.com/YOUR-USERNAME/openfms`
确认代码已上传。

---

## 三、部署步骤（10分钟）

### 步骤 1：打开 Cloud Shell

```
1. 访问 https://console.cloud.google.com/
2. 点击右上角 "Activate Cloud Shell" 图标 (▶_)
3. 等待 Cloud Shell 启动（约 30 秒）
```

### 步骤 2：确认项目配置

```bash
# 查看当前项目
gcloud config get-value project

# 如果不是您的项目，请设置
# gcloud config set project YOUR-PROJECT-ID
```

### 步骤 3：克隆代码并部署

```bash
# 1. 克隆代码（替换为您的用户名）
cd ~
git clone https://github.com/YOUR-USERNAME/openfms.git

# 2. 进入项目
cd openfms

# 3. 运行部署脚本
chmod +x deploy-cloud-shell.sh
./deploy-cloud-shell.sh
```

### 步骤 4：等待部署完成

脚本会自动完成：
1. ✅ 创建 GCE 实例
2. ✅ 配置防火墙
3. ✅ 上传代码
4. ✅ 安装 Docker
5. ✅ 启动 OpenFMS
6. ✅ 验证服务

整个过程约 5-10 分钟。

---

## 四、部署后操作

### 访问服务

部署完成后，脚本会输出访问地址：

```
========================================
  部署完成！
========================================

🌐 访问地址：

  Web 界面:
    http://34.81.XX.XX

  API 服务:
    http://34.81.XX.XX:3000

  Swagger API 文档:
    http://34.81.XX.XX:3000/swagger/index.html

  Grafana 监控:
    http://34.81.XX.XX:3001
    默认账号: admin / admin
```

### 保存部署信息

部署信息会自动保存到：`~/openfms-deployment-info.txt`

### 常用操作

#### 查看日志
```bash
# 在 Cloud Shell 中运行
gcloud compute ssh openfms-server --zone=asia-east1-a --command='cd ~/openfms && sudo docker-compose logs -f'
```

#### 重启服务
```bash
gcloud compute ssh openfms-server --zone=asia-east1-a --command='cd ~/openfms && sudo docker-compose restart'
```

#### 停止实例（停止计费）
```bash
gcloud compute instances stop openfms-server --zone=asia-east1-a
```

#### 启动实例
```bash
gcloud compute instances start openfms-server --zone=asia-east1-a
```

---

## 五、更新部署

当您修改代码后，需要重新部署：

### 步骤 1：推送代码到 GitHub

```bash
# 在本地执行
cd /Users/whsheng/works/openfms.io
git add .
git commit -m "Update: xxx"
git push
```

### 步骤 2：在 Cloud Shell 中更新

```bash
# 1. 进入项目目录
cd ~/openfms

# 2. 拉取最新代码
git pull

# 3. 上传并更新
gcloud compute scp --recurse . openfms-server:~/openfms-new --zone=asia-east1-a

# 4. SSH 进入实例更新
gcloud compute ssh openfms-server --zone=asia-east1-a --command='
    cd ~
    rm -rf openfms
    mv openfms-new openfms
    cd openfms
    sudo docker-compose down
    sudo docker-compose up -d
'
```

---

## 六、常见问题

### Q1: 部署脚本执行失败？

**检查步骤：**
```bash
# 1. 确认项目ID
gcloud config get-value project

# 2. 确认有权限
gcloud auth list

# 3. 手动检查实例状态
gcloud compute instances list

# 4. 查看实例日志
gcloud compute instances get-serial-port-output openfms-server --zone=asia-east1-a
```

### Q2: 如何修改配置？

```bash
# SSH 进入实例
gcloud compute ssh openfms-server --zone=asia-east1-a

# 编辑配置文件
cd ~/openfms
vim .env

# 重启服务
sudo docker-compose restart
```

### Q3: 如何备份数据？

```bash
# 备份数据库
gcloud compute ssh openfms-server --zone=asia-east1-a --command='
    cd ~/openfms
    sudo docker exec openfms-postgres pg_dump -U postgres openfms > backup-$(date +%Y%m%d).sql
'

# 下载到本地
gcloud compute scp openfms-server:~/openfms/backup-XXXXXX.sql ./ --zone=asia-east1-a
```

### Q4: 如何删除所有资源？

```bash
# 删除实例
gcloud compute instances delete openfms-server --zone=asia-east1-a

# 删除防火墙规则
gcloud compute firewall-rules delete allow-openfms
```

---

## 七、费用说明

### 预估费用（每月）

| 资源 | 规格 | 费用 |
|------|------|------|
| GCE 实例 | e2-medium (2vCPU, 4GB) | ~$25 |
| 磁盘 | 100GB SSD | ~$10 |
| 网络 | 预估 100GB 出站 | ~$10 |
| **总计** | | **~$45** |

### 省钱技巧

1. **使用免费额度**
   - 新用户有 $300 免费额度
   - 可用 6-12 个月

2. **非工作时间停止实例**
   ```bash
   # 设置定时关机（Cloud Scheduler）
   # 工作日 18:00 关机
   # 工作日 09:00 开机
   ```

3. **使用抢占式实例**
   - 节省 60-90%
   - 适合测试环境

4. **配置 Cloudflare**
   - 缓存静态资源
   - 减少出站流量

---

## 八、架构图

```
┌─────────────────────────────────────────────────────────────┐
│                      Google Cloud                           │
│                                                             │
│  ┌─────────────────────────────────────────────────────┐   │
│  │              GCE Instance (e2-medium)               │   │
│  │                                                     │   │
│  │  ┌──────────┐  ┌──────────┐  ┌──────────┐         │   │
│  │  │   API    │  │  Gateway │  │   Web    │         │   │
│  │  │  :3000   │  │ :8080/81 │  │   :80    │         │   │
│  │  └──────────┘  └──────────┘  └──────────┘         │   │
│  │                                                     │   │
│  │  ┌──────────┐  ┌──────────┐  ┌──────────┐         │   │
│  │  │ Postgres │  │  Redis   │  │   NATS   │         │   │
│  │  │(Timescale│  │          │  │          │         │   │
│  │  └──────────┘  └──────────┘  └──────────┘         │   │
│  │                                                     │   │
│  │  ┌──────────┐  ┌──────────┐                       │   │
│  │  │Prometheus│  │ Grafana  │                       │   │
│  │  │  :9090   │  │  :3001   │                       │   │
│  │  └──────────┘  └──────────┘                       │   │
│  └─────────────────────────────────────────────────────┘   │
│                                                             │
│  External IP: 34.81.XX.XX                                  │
└─────────────────────────────────────────────────────────────┘
```

---

## 九、下一步建议

### 测试验证（立即）
1. 访问 Web 界面
2. 添加测试设备
3. 验证 GPS 数据接收
4. 测试报警功能

### 生产准备（后续）
1. 配置 HTTPS（使用 Cloudflare 或 Let's Encrypt）
2. 配置域名
3. 设置监控告警
4. 配置自动备份

### 优化升级（后续）
1. 分离数据库到 Cloud SQL
2. 使用 Cloud Load Balancer
3. 配置 CDN
4. 多区域部署

---

## 十、获取帮助

如有问题，可以通过以下方式获取帮助：

1. **查看日志**
   ```bash
   gcloud compute ssh openfms-server --zone=asia-east1-a --command='cd ~/openfms && sudo docker-compose logs'
   ```

2. **检查服务状态**
   ```bash
   gcloud compute ssh openfms-server --zone=asia-east1-a --command='cd ~/openfms && sudo docker-compose ps'
   ```

3. **重启服务**
   ```bash
   gcloud compute ssh openfms-server --zone=asia-east1-a --command='cd ~/openfms && sudo docker-compose restart'
   ```

---

**部署脚本**: `deploy-cloud-shell.sh`  
**文档维护**: OpenFMS Team  
**最后更新**: 2026-02-08
