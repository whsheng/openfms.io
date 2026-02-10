# 推送代码并验证 Cloud Build 部署

> 本指南帮助您将代码推送到 GitHub 并触发 Cloud Build 自动部署

---

## 📋 前置检查

确保您已完成以下配置：

- [ ] GitHub 仓库已创建（如 `github.com/yourusername/openfms`）
- [ ] Google Cloud Build 触发器已创建并关联 GitHub
- [ ] Cloud Build 有权限部署到 Compute Engine

---

## 第一步：推送代码到 GitHub

### 方法 A：HTTPS（推荐新手）

```bash
# 1. 进入项目目录
cd /Users/whsheng/works/openfms.io

# 2. 初始化 Git（如果还没有）
git init

# 3. 添加所有文件
git add .

# 4. 提交
git commit -m "feat: add Cloud Build configuration

- Add cloudbuild.yaml for automated deployment
- Add GitHub Actions workflow
- Add verification scripts
- Update documentation"

# 5. 关联远程仓库（替换为您的用户名）
git remote add origin https://github.com/YOUR-USERNAME/openfms.git

# 6. 推送代码
# 会提示输入 GitHub 用户名和个人访问令牌
git branch -M main
git push -u origin main
```

### 方法 B：SSH（更安全）

```bash
# 如果已配置 SSH 密钥
git remote add origin git@github.com:YOUR-USERNAME/openfms.git
git push -u origin main
```

---

## 第二步：验证推送成功

```bash
# 在浏览器中访问您的仓库
open https://github.com/YOUR-USERNAME/openfms

# 确认以下文件已上传：
# - cloudbuild.yaml
# - .github/workflows/cloud-build-trigger.yml
# - verify-deployment.sh
# - README.md
```

---

## 第三步：查看 Cloud Build 构建

推送代码后，Cloud Build 会自动触发构建：

```
1. 访问 https://console.cloud.google.com/cloud-build/builds
2. 查看最新的构建记录
3. 等待构建完成（约 10-15 分钟）
```

构建步骤：
1. 拉取代码
2. 构建 API 镜像
3. 构建 Web 镜像
4. 推送镜像到 Container Registry
5. 部署到 GCE 实例
6. 健康检查

---

## 第四步：验证部署

### 方法 1：使用验证脚本（推荐）

```bash
# 在本地运行验证脚本
chmod +x verify-deployment.sh
./verify-deployment.sh
```

脚本会检查：
- ✓ 实例状态
- ✓ 外部 IP
- ✓ Web 服务
- ✓ API 服务
- ✓ Swagger 文档
- ✓ Grafana 监控
- ✓ 容器状态

### 方法 2：手动验证

```bash
# 获取实例 IP
EXTERNAL_IP=$(gcloud compute instances describe openfms-server \
  --zone=asia-east1-a \
  --format='get(networkInterfaces[0].accessConfigs[0].natIP)')

echo "访问地址: http://$EXTERNAL_IP"

# 测试各服务
curl http://$EXTERNAL_IP:3000/health
open http://$EXTERNAL_IP
open http://$EXTERNAL_IP:3000/swagger/index.html
```

---

## 第五步：访问应用

部署成功后，访问以下地址：

| 服务 | 地址 | 说明 |
|------|------|------|
| Web 界面 | `http://YOUR-IP` | 前端管理界面 |
| API 文档 | `http://YOUR-IP:3000/swagger/index.html` | Swagger 文档 |
| Grafana | `http://YOUR-IP:3001` | 监控仪表盘 (admin/admin) |
| Prometheus | `http://YOUR-IP:9090` | 指标采集 |

---

## 🔍 常见问题排查

### 问题 1：Cloud Build 未触发

**检查：**
```bash
# 1. 确认触发器配置
# Cloud Console → Cloud Build → 触发器
# 确认仓库已连接

# 2. 手动触发测试
gcloud builds submit --config=cloudbuild.yaml
```

### 问题 2：构建失败

**查看日志：**
```bash
# 查看最新构建日志
gcloud builds list --limit=1
gcloud builds log $(gcloud builds list --limit=1 --format='value(id)')
```

**常见问题：**
- Docker 构建失败 → 检查 Dockerfile
- 权限不足 → 检查 Cloud Build 服务账号权限
- 部署失败 → 检查 GCE 实例是否存在

### 问题 3：服务无法访问

**检查：**
```bash
# 1. 检查实例状态
gcloud compute instances describe openfms-server --zone=asia-east1-a

# 2. 检查防火墙规则
gcloud compute firewall-rules describe allow-openfms

# 3. SSH 登录检查容器
gcloud compute ssh openfms-server --zone=asia-east1-a --command="
  sudo docker ps
  sudo docker-compose logs
"
```

---

## 🔄 更新部署

当代码有更新时，只需推送代码：

```bash
# 1. 修改代码
# ...

# 2. 提交并推送
git add .
git commit -m "feat: xxx"
git push

# 3. 等待 Cloud Build 自动部署
# 约 10-15 分钟

# 4. 验证部署
./verify-deployment.sh
```

---

## 📊 监控构建状态

### 在 Cloud Console 查看

```
https://console.cloud.google.com/cloud-build/builds
```

### 使用命令行查看

```bash
# 查看构建列表
gcloud builds list

# 查看实时日志
gcloud builds log --stream $(gcloud builds list --limit=1 --format='value(id)')
```

---

## 🎉 部署成功标志

当看到以下信息时，表示部署成功：

```
✅ 所有检查通过！部署成功！

🌐 访问地址：
  Web 界面:   http://34.81.XX.XX
  API 文档:   http://34.81.XX.XX:3000/swagger/index.html
  Grafana:    http://34.81.XX.XX:3001

🔑 默认账号：
  用户名: admin
  密码: admin
```

---

## 💡 下一步

部署成功后，您可以：

1. **添加测试设备** - 访问 Web 界面添加 GPS 设备
2. **配置域名** - 使用 Cloudflare 配置自定义域名
3. **启用 HTTPS** - 配置 SSL 证书
4. **设置监控告警** - 在 Grafana 配置告警规则

---

## 📚 相关文档

- [Cloud Build 完整指南](docs/deployment-cloud-shell.md)
- [项目总结分析](docs/project-summary-analysis.md)
- [快速开始指南](DEPLOY-QUICKSTART.md)

---

**需要帮助？** 运行 `./verify-deployment.sh` 查看详细诊断信息。
