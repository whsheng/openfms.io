#!/bin/bash
# OpenFMS 方案A - GCE + Docker Compose 一键部署脚本
# 使用前请替换以下变量：

# ==================== 配置区域 ====================
PROJECT_ID="your-project-id"        # 替换为您的 GCP 项目ID
ZONE="asia-east1-a"                  # 可用区（推荐 asia-east1-a 台湾）
INSTANCE_NAME="openfms-server"       # 实例名称
MACHINE_TYPE="e2-medium"             # 机器类型 (2 vCPU, 4GB 内存)
DISK_SIZE="100GB"                    # 磁盘大小
# =================================================

# 颜色输出
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo -e "${GREEN}========================================${NC}"
echo -e "${GREEN}  OpenFMS Google Cloud 部署脚本 (方案A) ${NC}"
echo -e "${GREEN}========================================${NC}"
echo ""

# 检查 gcloud 是否安装
if ! command -v gcloud &> /dev/null; then
    echo -e "${RED}错误: gcloud CLI 未安装${NC}"
    echo "请先安装 Google Cloud SDK:"
    echo "  macOS: brew install google-cloud-sdk"
    echo "  其他: https://cloud.google.com/sdk/docs/install"
    exit 1
fi

# 检查是否登录
if ! gcloud auth list --filter=status:ACTIVE --format="value(account)" | grep -q "@"; then
    echo -e "${YELLOW}请先登录 Google Cloud:${NC}"
    gcloud auth login
fi

# 设置项目
echo -e "${YELLOW}设置项目: $PROJECT_ID${NC}"
gcloud config set project $PROJECT_ID

echo ""
echo -e "${GREEN}步骤 1/6: 创建 GCE 实例...${NC}"
echo "   实例名: $INSTANCE_NAME"
echo "   类型: $MACHINE_TYPE"
echo "   区域: $ZONE"
echo ""

gcloud compute instances create $INSTANCE_NAME \
  --machine-type=$MACHINE_TYPE \
  --image-family=ubuntu-2204-lts \
  --image-project=ubuntu-os-cloud \
  --boot-disk-size=$DISK_SIZE \
  --boot-disk-type=pd-ssd \
  --tags=http-server,https-server,openfms \
  --zone=$ZONE \
  --metadata-from-file startup-script=<(cat << 'EOF'
#!/bin/bash
# 启动脚本 - 自动安装 Docker
apt-get update
apt-get install -y docker.io docker-compose git curl
systemctl start docker
systemctl enable docker
usermod -aG docker ubuntu
EOF
)

if [ $? -ne 0 ]; then
    echo -e "${RED}创建实例失败${NC}"
    exit 1
fi

echo -e "${GREEN}✓ 实例创建成功${NC}"
echo ""

# 获取实例 IP
echo -e "${GREEN}步骤 2/6: 配置防火墙规则...${NC}"

# 检查防火墙规则是否存在
if ! gcloud compute firewall-rules describe allow-openfms --format="value(name)" &>/dev/null; then
    gcloud compute firewall-rules create allow-openfms \
      --allow tcp:80,tcp:443,tcp:3000,tcp:8080,tcp:8081,tcp:9090,tcp:3001 \
      --target-tags=openfms \
      --description="Allow OpenFMS ports"
    echo -e "${GREEN}✓ 防火墙规则创建成功${NC}"
else
    echo -e "${YELLOW}防火墙规则已存在，跳过${NC}"
fi

echo ""
echo -e "${GREEN}步骤 3/6: 等待实例启动...${NC}"
echo "   (约需 30-60 秒)"
echo ""

# 等待实例启动
sleep 30

# 获取外部 IP
EXTERNAL_IP=$(gcloud compute instances describe $INSTANCE_NAME \
  --zone=$ZONE \
  --format='get(networkInterfaces[0].accessConfigs[0].natIP)')

echo -e "${GREEN}✓ 实例外部 IP: $EXTERNAL_IP${NC}"
echo ""

# 等待 SSH 可用
echo -e "${GREEN}步骤 4/6: 安装 Docker 环境...${NC}"
echo ""

for i in {1..10}; do
    if gcloud compute ssh $INSTANCE_NAME --zone=$ZONE --command="echo 'SSH ready'" &>/dev/null; then
        break
    fi
    echo "   等待 SSH 服务启动... ($i/10)"
    sleep 5
done

# 安装 Docker
gcloud compute ssh $INSTANCE_NAME --zone=$ZONE --command="
  sudo apt-get update
  sudo apt-get install -y docker.io docker-compose git
  sudo systemctl start docker
  sudo systemctl enable docker
  sudo usermod -aG docker \$USER
  docker --version
  docker-compose --version
" || {
    echo -e "${RED}Docker 安装失败${NC}"
    exit 1
}

echo -e "${GREEN}✓ Docker 安装成功${NC}"
echo ""

# 部署 OpenFMS
echo -e "${GREEN}步骤 5/6: 部署 OpenFMS...${NC}"
echo ""

# 方式1：使用 git 克隆（推荐）
echo "   方式：Git 克隆代码"
gcloud compute ssh $INSTANCE_NAME --zone=$ZONE --command="
  cd /home/ubuntu
  if [ ! -d 'openfms' ]; then
    git clone https://github.com/your-username/openfms.git
  fi
  cd openfms
  git pull
  
  # 创建环境变量文件
  cat > .env << 'ENVFILE'
# 数据库配置
DATABASE_URL=postgres://postgres:postgres@postgres:5432/openfms?sslmode=disable

# Redis 配置
REDIS_URL=redis:6379

# NATS 配置
NATS_URL=nats://nats:4222

# JWT 密钥
JWT_SECRET=your-secret-key-change-in-production

# 环境
ENV=production
ENVFILE

  # 启动服务
  sudo docker-compose up -d
" || {
    echo -e "${RED}部署失败${NC}"
    exit 1
}

# 等待服务启动
echo ""
echo -e "${GREEN}步骤 6/6: 等待服务启动...${NC}"
echo "   (约需 60 秒)"
echo ""

sleep 30

# 检查服务状态
gcloud compute ssh $INSTANCE_NAME --zone=$ZONE --command="
  cd /home/ubuntu/openfms
  sudo docker-compose ps
"

echo ""
echo -e "${GREEN}========================================${NC}"
echo -e "${GREEN}  部署完成！                            ${NC}"
echo -e "${GREEN}========================================${NC}"
echo ""
echo -e "访问地址："
echo -e "  ${YELLOW}Web 界面:${NC}  http://$EXTERNAL_IP"
echo -e "  ${YELLOW}API 服务:${NC}  http://$EXTERNAL_IP:3000"
echo -e "  ${Yellow}Swagger:${NC}   http://$EXTERNAL_IP:3000/swagger/index.html"
echo -e "  ${YELLOW}Grafana:${NC}   http://$EXTERNAL_IP:3001 (admin/admin)"
echo ""
echo -e "默认账号："
echo -e "  用户名: admin"
echo -e "  密码: admin"
echo ""
echo -e "${YELLOW}常用命令：${NC}"
echo -e "  查看日志:  gcloud compute ssh $INSTANCE_NAME --zone=$ZONE --command='cd /home/ubuntu/openfms && sudo docker-compose logs -f'"
echo -e "  重启服务:  gcloud compute ssh $INSTANCE_NAME --zone=$ZONE --command='cd /home/ubuntu/openfms && sudo docker-compose restart'"
echo -e "  进入实例:  gcloud compute ssh $INSTANCE_NAME --zone=$ZONE"
echo ""
echo -e "${YELLOW}注意：${NC}"
echo -e "  1. 首次启动可能需要 2-3 分钟初始化数据库"
echo -e "  2. 如需 HTTPS，请配置域名并申请 SSL 证书"
echo -e "  3. 生产环境请修改默认密码和密钥"
echo ""

# 保存信息到文件
cat > deployment-info.txt << EOF
OpenFMS 部署信息
================
部署时间: $(date)
项目ID: $PROJECT_ID
实例名: $INSTANCE_NAME
区域: $ZONE
外部IP: $EXTERNAL_IP

访问地址：
- Web: http://$EXTERNAL_IP
- API: http://$EXTERNAL_IP:3000
- Swagger: http://$EXTERNAL_IP:3000/swagger/index.html
- Grafana: http://$EXTERNAL_IP:3001

默认账号：
- 用户名: admin
- 密码: admin

SSH 连接：
gcloud compute ssh $INSTANCE_NAME --zone=$ZONE
EOF

echo -e "${GREEN}部署信息已保存到: deployment-info.txt${NC}"
