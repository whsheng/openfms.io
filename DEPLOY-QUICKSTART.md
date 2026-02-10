# OpenFMS 快速部署指南 (Cloud Shell)

> 无需本地安装任何工具，5分钟完成部署

---

## 🚀 三步部署

### 第 1 步：推送代码到 GitHub

```bash
# 在本地项目目录执行
cd /Users/whsheng/works/openfms.io
git init
git add .
git commit -m "Initial commit"
git remote add origin https://github.com/YOUR-USERNAME/openfms.git
git push -u origin main
```

### 第 2 步：打开 Cloud Shell

```
1. 访问 https://console.cloud.google.com/
2. 点击右上角 ▶_ 图标（Activate Cloud Shell）
3. 等待 Cloud Shell 启动
```

### 第 3 步：运行部署命令

在 Cloud Shell 中复制粘贴：

```bash
cd ~ && git clone https://github.com/YOUR-USERNAME/openfms.git && cd openfms && chmod +x deploy-cloud-shell.sh && ./deploy-cloud-shell.sh
```

---

## ✅ 部署完成

脚本会自动输出访问地址：

```
🌐 访问地址：
  Web 界面: http://34.81.XX.XX
  API 文档: http://34.81.XX.XX:3000/swagger/index.html
  Grafana:  http://34.81.XX.XX:3001 (admin/admin)

🔑 默认账号：admin / admin
```

---

## 💰 费用

- **月费用**: ~$45 USD (~¥320 CNY)
- **新用户**: 有 $300 免费额度

---

## 📚 详细文档

- 完整部署指南: `docs/deployment-cloud-shell.md`
- 项目总结: `docs/project-summary-analysis.md`

---

## ⚡ 常用命令

```bash
# 查看日志
gcloud compute ssh openfms-server --zone=asia-east1-a --command='cd ~/openfms && sudo docker-compose logs -f'

# 重启服务
gcloud compute ssh openfms-server --zone=asia-east1-a --command='cd ~/openfms && sudo docker-compose restart'

# 停止实例（停止计费）
gcloud compute instances stop openfms-server --zone=asia-east1-a

# 启动实例
gcloud compute instances start openfms-server --zone=asia-east1-a
```

---

**有问题？** 查看 `docs/deployment-cloud-shell.md` 常见问题部分。
