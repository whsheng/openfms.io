#!/bin/bash
# OpenFMS 部署验证脚本
# 在本地运行，验证 Cloud Build 部署状态

echo "========================================"
echo "OpenFMS 部署验证工具"
echo "========================================"
echo ""

# 检查 gcloud
echo "检查 gcloud 配置..."
PROJECT_ID=$(gcloud config get-value project 2>/dev/null)
if [ -z "$PROJECT_ID" ]; then
    echo "❌ 未设置项目 ID"
    echo "请运行: gcloud config set project YOUR-PROJECT-ID"
    exit 1
fi
echo "✓ 项目 ID: $PROJECT_ID"
echo ""

INSTANCE_NAME="openfms-server"
ZONE="asia-east1-a"

# 检查实例状态
echo "检查 GCE 实例状态..."
INSTANCE_STATUS=$(gcloud compute instances describe $INSTANCE_NAME --zone=$ZONE --format='get(status)' 2>/dev/null)
if [ -z "$INSTANCE_STATUS" ]; then
    echo "❌ 实例不存在: $INSTANCE_NAME"
    echo "请检查 Cloud Build 是否成功执行"
    exit 1
fi

echo "✓ 实例状态: $INSTANCE_STATUS"

if [ "$INSTANCE_STATUS" != "RUNNING" ]; then
    echo "⚠️  实例未运行，尝试启动..."
    gcloud compute instances start $INSTANCE_NAME --zone=$ZONE
    sleep 10
fi

# 获取外部 IP
echo ""
echo "获取访问地址..."
EXTERNAL_IP=$(gcloud compute instances describe $INSTANCE_NAME --zone=$ZONE --format='get(networkInterfaces[0].accessConfigs[0].natIP)')
echo "✓ 外部 IP: $EXTERNAL_IP"
echo ""

# 服务健康检查
echo "========================================"
echo "服务健康检查"
echo "========================================"
echo ""

SERVICES=(
    "Web Frontend:http://$EXTERNAL_IP:80"
    "API Server:http://$EXTERNAL_IP:3000/health"
    "Swagger Docs:http://$EXTERNAL_IP:3000/swagger/index.html"
    "Prometheus:http://$EXTERNAL_IP:9090"
    "Grafana:http://$EXTERNAL_IP:3001"
)

ALL_PASSED=true

for service in "${SERVICES[@]}"; do
    IFS=':' read -r name url <<< "$service"
    echo -n "检查 $name ... "
    
    # 尝试访问
    if curl -s -o /dev/null -w "%{http_code}" "$url" | grep -qE "(200|301|302)"; then
        echo "✓ 正常"
        echo "  URL: $url"
    else
        echo "❌ 无法访问"
        echo "  URL: $url"
        ALL_PASSED=false
    fi
    echo ""
done

# API 详细检查
echo "========================================"
echo "API 接口测试"
echo "========================================"
echo ""

echo -n "测试 /health 端点 ... "
HEALTH_RESPONSE=$(curl -s http://$EXTERNAL_IP:3000/health)
if echo "$HEALTH_RESPONSE" | grep -q "ok"; then
    echo "✓ 通过"
    echo "  响应: $HEALTH_RESPONSE"
else
    echo "❌ 失败"
    echo "  响应: $HEALTH_RESPONSE"
    ALL_PASSED=false
fi
echo ""

echo -n "测试 Swagger 文档 ... "
if curl -s http://$EXTERNAL_IP:3000/swagger/index.html | grep -q "swagger"; then
    echo "✓ 正常"
else
    echo "❌ 无法加载"
    ALL_PASSED=false
fi
echo ""

# 容器状态检查
echo "========================================"
echo "容器状态检查"
echo "========================================"
echo ""

gcloud compute ssh $INSTANCE_NAME --zone=$ZONE --command="
    echo '容器列表:'
    sudo docker ps --format 'table {{.Names}}\t{{.Status}}\t{{.Ports}}'
    echo ''
    echo '容器统计:'
    sudo docker system df
" 2>/dev/null || echo "❌ 无法 SSH 到实例"

echo ""

# 日志查看
echo "========================================"
echo "最近日志"
echo "========================================"
echo ""

echo "API 服务日志 (最近 10 行):"
gcloud compute ssh $INSTANCE_NAME --zone=$ZONE --command="
    sudo docker logs --tail=10 openfms-api 2>/dev/null || echo '容器未运行'
" 2>/dev/null

echo ""

# 总结
echo "========================================"
echo "验证结果"
echo "========================================"
echo ""

if [ "$ALL_PASSED" = true ]; then
    echo "✅ 所有检查通过！部署成功！"
    echo ""
    echo "🌐 访问地址:"
    echo "  Web 界面:   http://$EXTERNAL_IP"
    echo "  API 文档:   http://$EXTERNAL_IP:3000/swagger/index.html"
    echo "  Grafana:    http://$EXTERNAL_IP:3001 (admin/admin)"
    echo ""
    echo "🔑 默认账号:"
    echo "  用户名: admin"
    echo "  密码: admin"
else
    echo "⚠️  部分检查失败，请查看上方详情"
fi

echo ""
echo "常用命令:"
echo "  查看日志:   ./view-logs.sh"
echo "  SSH 登录:   gcloud compute ssh $INSTANCE_NAME --zone=$ZONE"
echo "  重启服务:   gcloud compute ssh $INSTANCE_NAME --zone=$ZONE --command='cd ~ && sudo docker-compose restart'"
echo ""

# 保存信息
cat > deployment-urls.txt << EOF
OpenFMS 部署访问地址
====================
生成时间: $(date)
项目ID: $PROJECT_ID
实例名: $INSTANCE_NAME
外部IP: $EXTERNAL_IP

访问地址：
- Web:     http://$EXTERNAL_IP
- API:     http://$EXTERNAL_IP:3000
- Swagger: http://$EXTERNAL_IP:3000/swagger/index.html
- Grafana: http://$EXTERNAL_IP:3001 (admin/admin)

验证结果: $([ "$ALL_PASSED" = true ] && echo "全部通过" || echo "部分失败")
EOF

echo "✓ 访问地址已保存到: deployment-urls.txt"
