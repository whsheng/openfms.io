#!/bin/bash
# OpenFMS Cloud Shell 部署脚本
# 在 Google Cloud Shell 中运行

set -e  # 遇到错误立即退出

# ==================== 配置 ====================
INSTANCE_NAME="openfms-server"
ZONE="asia-east1-a"  # 台湾区域，国内访问快
MACHINE_TYPE="e2-medium"
DISK_SIZE="100GB"
# =============================================

# 颜色
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

clear
echo -e "${GREEN}========================================${NC}"
echo -e "${GREEN}  OpenFMS Cloud Shell 部署脚本          ${NC}"
echo -e "${GREEN}========================================${NC}"
echo ""

# 检查是否在 Cloud Shell 中
if [ -z "$CLOUD_SHELL" ] && [ -z "$DEVSHELL_PROJECT_ID" ]; then
    echo -e "${YELLOW}警告: 未检测到 Cloud Shell 环境${NC}"
    echo "此脚本专为 Google Cloud Shell 设计"
    echo "继续执行可能出现问题"
    echo ""
    read -p "是否继续? (y/N): " confirm
    if [[ $confirm != [yY] ]]; then
        exit 1
    fi
fi

# 获取当前项目
PROJECT_ID=$(gcloud config get-value project)
echo -e "${BLUE}当前项目: ${PROJECT_ID}${NC}"
echo ""

# 检查项目ID
if [ "$PROJECT_ID" = "" ] || [ "$PROJECT_ID" = "(unset)" ]; then
    echo -e "${RED}错误: 未设置项目 ID${NC}"
    echo "请运行: gcloud config set project YOUR-PROJECT-ID"
    exit 1
fi

echo -e "${GREEN}步骤 1/7: 检查并启用必要 API...${NC}"
gcloud services enable compute.googleapis.com cloudresourcemanager.googleapis.com --quiet
echo -e "${GREEN}✓ API 已启用${NC}"
echo ""

echo -e "${GREEN}步骤 2/7: 创建 GCE 实例...${NC}"
echo -e "  实例名称: ${YELLOW}$INSTANCE_NAME${NC}"
echo -e "  机器类型: ${YELLOW}$MACHINE_TYPE${NC}"
echo -e "  区域: ${YELLOW}$ZONE${NC}"
echo -e "  磁盘: ${YELLOW}$DISK_SIZE${NC}"
echo ""

# 检查实例是否已存在
if gcloud compute instances describe $INSTANCE_NAME --zone=$ZONE --quiet &>/dev/null; then
    echo -e "${YELLOW}实例 $INSTANCE_NAME 已存在${NC}"
    read -p "是否删除并重新创建? (y/N): " recreate
    if [[ $recreate == [yY] ]]; then
        echo "删除现有实例..."
        gcloud compute instances delete $INSTANCE_NAME --zone=$ZONE --quiet
    else
        echo -e "${GREEN}使用现有实例继续部署...${NC}"
    fi
else
    # 创建新实例
    echo "正在创建实例..."
    gcloud compute instances create $INSTANCE_NAME \
        --machine-type=$MACHINE_TYPE \
        --image-family=ubuntu-2204-lts \
        --image-project=ubuntu-os-cloud \
        --boot-disk-size=$DISK_SIZE \
        --boot-disk-type=pd-ssd \
        --tags=http-server,https-server,openfms \
        --zone=$ZONE \
        --metadata startup-script='#!/bin/bash
            apt-get update
            apt-get install -y docker.io docker-compose git curl
            systemctl start docker
            systemctl enable docker
            usermod -aG docker ubuntu
        ' \
        --quiet
    
    echo -e "${GREEN}✓ 实例创建成功${NC}"
fi

echo ""
echo -e "${GREEN}步骤 3/7: 配置防火墙规则...${NC}"

# 创建防火墙规则（如果不存在）
if ! gcloud compute firewall-rules describe allow-openfms --quiet &>/dev/null; then
    gcloud compute firewall-rules create allow-openfms \
        --allow tcp:80,tcp:443,tcp:3000,tcp:8080,tcp:8081,tcp:9090,tcp:3001,tcp:554,tcp:1935 \
        --target-tags=openfms \
        --description="OpenFMS service ports" \
        --quiet
    echo -e "${GREEN}✓ 防火墙规则创建成功${NC}"
else
    echo -e "${YELLOW}防火墙规则已存在${NC}"
fi

# 获取外部 IP
EXTERNAL_IP=$(gcloud compute instances describe $INSTANCE_NAME \
    --zone=$ZONE \
    --format='get(networkInterfaces[0].accessConfigs[0].natIP)' \
    --quiet)

echo ""
echo -e "${GREEN}步骤 4/7: 等待实例启动...${NC}"
echo -e "  外部 IP: ${YELLOW}$EXTERNAL_IP${NC}"
echo ""

# 等待 SSH 可用
for i in {1..20}; do
    if gcloud compute ssh $INSTANCE_NAME --zone=$ZONE --command="echo 'OK'" --quiet 2>/dev/null; then
        echo -e "${GREEN}✓ 实例已就绪${NC}"
        break
    fi
    echo -n "."
    sleep 3
done

echo ""
echo -e "${GREEN}步骤 5/7: 上传代码到实例...${NC}"
echo ""

# 打包当前目录（排除不需要的文件）
echo "打包代码..."
cd ~
rm -rf /tmp/openfms-deploy
mkdir -p /tmp/openfms-deploy

# 复制项目文件
cp -r ~/openfms/* /tmp/openfms-deploy/ 2>/dev/null || \
cp -r ~/cloudshell_open/openfms/* /tmp/openfms-deploy/ 2>/dev/null || {
    echo -e "${RED}错误: 无法找到项目代码${NC}"
    echo "请确保在 Cloud Shell 中克隆了代码:"
    echo "  git clone https://github.com/YOUR-USERNAME/openfms.git"
    exit 1
}

# 创建部署包
cd /tmp/openfms-deploy
tar czvf /tmp/openfms.tar.gz \
    --exclude='.git' \
    --exclude='web/node_modules' \
    --exclude='*.log' \
    --exclude='.DS_Store' \
    . > /dev/null 2>&1

# 上传到实例
echo "上传代码到 GCE 实例..."
gcloud compute scp /tmp/openfms.tar.gz $INSTANCE_NAME:~/ --zone=$ZONE --quiet

# 解压
gcloud compute ssh $INSTANCE_NAME --zone=$ZONE --command="
    rm -rf ~/openfms
    mkdir ~/openfms
    tar xzvf ~/openfms.tar.gz -C ~/openfms
    rm ~/openfms.tar.gz
" --quiet

echo -e "${GREEN}✓ 代码上传完成${NC}"
echo ""

echo -e "${GREEN}步骤 6/7: 安装 Docker 并部署...${NC}"
echo ""

# 在实例中执行部署
gcloud compute ssh $INSTANCE_NAME --zone=$ZONE --command="
    cd ~/openfms
    
    # 检查 Docker
    if ! command -v docker &> /dev/null; then
        echo '安装 Docker...'
        sudo apt-get update
        sudo apt-get install -y docker.io docker-compose git
        sudo systemctl start docker
        sudo systemctl enable docker
        sudo usermod -aG docker \$USER
    fi
    
    # 显示版本
    echo 'Docker 版本:'
    docker --version
    docker-compose --version
    
    # 创建环境变量文件
    cat > .env << 'EOF'
# Database
DATABASE_URL=postgres://postgres:postgres@postgres:5432/openfms?sslmode=disable

# Redis
REDIS_URL=redis:6379

# NATS
NATS_URL=nats://nats:4222

# JWT
JWT_SECRET=openfms-secret-key-change-in-production

# Environment
ENV=production
LOG_LEVEL=info
EOF
    
    # 启动服务
    echo '启动 OpenFMS 服务...'
    sudo docker-compose down 2>/dev/null || true
    sudo docker-compose up -d
    
    echo '等待服务初始化...'
    sleep 30
    
    # 检查服务状态
    echo ''
    echo '服务状态:'
    sudo docker-compose ps
    
    echo ''
    echo '容器日志 (最近20行):'
    sudo docker-compose logs --tail=20
" 

echo ""
echo -e "${GREEN}步骤 7/7: 验证部署...${NC}"
echo ""

# 健康检查
for i in {1..10}; do
    if curl -s http://$EXTERNAL_IP:3000/health | grep -q "ok"; then
        echo -e "${GREEN}✓ API 服务运行正常${NC}"
        break
    fi
    echo "等待服务启动... ($i/10)"
    sleep 5
done

echo ""
echo -e "${GREEN}========================================${NC}"
echo -e "${GREEN}  部署完成！                            ${NC}"
echo -e "${GREEN}========================================${NC}"
echo ""
echo -e "🌐 ${BLUE}访问地址：${NC}"
echo ""
echo -e "  ${YELLOW}Web 界面:${NC}"
echo -e "    http://$EXTERNAL_IP"
echo ""
echo -e "  ${YELLOW}API 服务:${NC}"
echo -e "    http://$EXTERNAL_IP:3000"
echo ""
echo -e "  ${YELLOW}Swagger API 文档:${NC}"
echo -e "    http://$EXTERNAL_IP:3000/swagger/index.html"
echo ""
echo -e "  ${YELLOW}Grafana 监控:${NC}"
echo -e "    http://$EXTERNAL_IP:3001"
echo -e "    默认账号: admin / admin"
echo ""
echo -e "  ${YELLOW}Prometheus:${NC}"
echo -e "    http://$EXTERNAL_IP:9090"
echo ""
echo -e "🔑 ${BLUE}默认登录账号：${NC}"
echo -e "  用户名: ${YELLOW}admin${NC}"
echo -e "  密码: ${YELLOW}admin${NC}"
echo ""
echo -e "⚙️  ${BLUE}常用命令：${NC}"
echo -e "  查看日志:  ${GREEN}gcloud compute ssh $INSTANCE_NAME --zone=$ZONE --command='cd ~/openfms && sudo docker-compose logs -f'${NC}"
echo -e "  重启服务:  ${GREEN}gcloud compute ssh $INSTANCE_NAME --zone=$ZONE --command='cd ~/openfms && sudo docker-compose restart'${NC}"
echo -e "  进入实例:  ${GREEN}gcloud compute ssh $INSTANCE_NAME --zone=$ZONE${NC}"
echo -e "  停止实例:  ${GREEN}gcloud compute instances stop $INSTANCE_NAME --zone=$ZONE${NC}"
echo ""
echo -e "💰 ${BLUE}费用提醒：${NC}"
echo -e "  预估月费用: ${YELLOW}~\$45 USD${NC} (约 ¥320 CNY)"
echo -e "  新用户有 ${YELLOW}\$300${NC} 免费额度"
echo ""
echo -e "⚠️  ${YELLOW}安全提醒：${NC}"
echo -e "  1. 生产环境请修改默认密码"
echo -e "  2. 建议配置 HTTPS"
echo -e "  3. 定期备份数据"
echo ""

# 保存部署信息
cat > ~/openfms-deployment-info.txt << EOF
OpenFMS 部署信息
================
部署时间: $(date)
项目ID: $PROJECT_ID
实例名称: $INSTANCE_NAME
区域: $ZONE
外部IP: $EXTERNAL_IP

访问地址：
- Web: http://$EXTERNAL_IP
- API: http://$EXTERNAL_IP:3000
- Swagger: http://$EXTERNAL_IP:3000/swagger/index.html
- Grafana: http://$EXTERNAL_IP:3001 (admin/admin)

SSH 连接：
gcloud compute ssh $INSTANCE_NAME --zone=$ZONE

常用命令：
# 查看日志
gcloud compute ssh $INSTANCE_NAME --zone=$ZONE --command='cd ~/openfms && sudo docker-compose logs -f'

# 重启服务
gcloud compute ssh $INSTANCE_NAME --zone=$ZONE --command='cd ~/openfms && sudo docker-compose restart'

# 停止实例（停止计费）
gcloud compute instances stop $INSTANCE_NAME --zone=$ZONE

# 启动实例
gcloud compute instances start $INSTANCE_NAME --zone=$ZONE

# 删除实例（谨慎操作）
gcloud compute instances delete $INSTANCE_NAME --zone=$ZONE
EOF

echo -e "📄 部署信息已保存到: ${YELLOW}~/openfms-deployment-info.txt${NC}"
echo ""
