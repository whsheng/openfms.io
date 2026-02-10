# OpenFMS - 开源车队管理系统

<p align="center">
  <img src="docs/logo.png" alt="OpenFMS Logo" width="200">
</p>

<p align="center">
  <strong>轻量级、现代化的开源车队管理平台</strong>
</p>

<p align="center">
  <a href="#功能特性">功能特性</a> •
  <a href="#技术栈">技术栈</a> •
  <a href="#快速开始">快速开始</a> •
  <a href="#文档">文档</a> •
  <a href="#贡献">贡献</a>
</p>

---

## 项目简介

OpenFMS 是一个面向中小企业和物流公司的开源车队管理系统(Fleet Management System)。它采用现代化的技术栈，支持多种GPS设备协议，提供实时监控、历史轨迹回放、电子围栏等核心功能。

### 核心优势

- 🚀 **轻量级架构** - 单机VPS即可支持1000+车辆
- 💰 **极低运营成本** - 开源免费，无授权费用
- 🔌 **多协议支持** - JT808、GT06、Wialon等
- 📱 **现代化UI** - React + Ant Design，体验流畅
- 🗄️ **时序数据库** - TimescaleDB高效存储GPS轨迹
- 🐳 **一键部署** - Docker Compose快速启动

## 功能特性

### 已实现功能 (v1.2)

- ✅ **设备接入** - JT808/GT06/Wialon多协议支持
- ✅ **实时监控** - Mapbox GL JS 地图，车辆位置实时更新
- ✅ **设备管理** - 设备增删改查
- ✅ **历史轨迹** - 轨迹查询与回放
- ✅ **用户认证** - JWT Token鉴权
- ✅ **轻网关架构** - 网关与业务解耦
- ✅ **电子围栏** - 圆形/多边形围栏，进出检测报警
- ✅ **报警中心** - 实时报警、规则配置、统计分析
- ✅ **用户权限** - RBAC 角色权限管理
- ✅ **视频服务** - JT1078音视频流，实时播放/回放
- ✅ **报表统计** - 里程、停车、驾驶行为分析
- ✅ **系统监控** - Prometheus + Grafana监控

### 开发中功能 (v2.0)

- 🚧 **多租户 SaaS** - 租户隔离、计费系统
- 🚧 **License 授权** - 授权验证、功能限制
- 🚧 **移动端 App** - React Native/小程序
- 🚧 **AI 分析** - 驾驶行为评分、预测性维护

## 技术栈

| 模块 | 技术选型 |
|------|----------|
| 设备网关 | Go (Gin/Net) |
| API服务 | Go (Gin) + GORM |
| 前端 | React 18 + Ant Design 5 |
| 数据库 | TimescaleDB (PostgreSQL) |
| 缓存 | Redis |
| 消息队列 | NATS |
| 地图 | Mapbox GL JS / Leaflet |
| 部署 | Docker Compose |

## 快速开始

### 环境要求

- Docker 20.10+
- Docker Compose 2.0+
- 2核CPU / 4GB内存 / 50GB存储

### 一键部署

```bash
# 克隆仓库
git clone https://github.com/your-org/openfms.git
cd openfms

# 启动服务
docker-compose up -d

# 查看日志
docker-compose logs -f
```

### 访问服务

| 服务 | 地址 | 说明 |
|------|------|------|
| Web界面 | http://localhost | 前端管理界面 |
| API服务 | http://localhost:3000 | REST API |
| 网关TCP | localhost:8080 | 设备接入端口 |
| 网关HTTP | http://localhost:8081 | 网关管理接口 |
| NATS监控 | http://localhost:8222 | 消息队列监控 |

### 默认账号

- 用户名: `admin`
- 密码: `admin`

## 项目结构

```
openfms/
├── gateway/          # 设备网关服务 (Go)
│   ├── cmd/gateway/  # 入口程序
│   ├── internal/     # 内部包
│   │   ├── adapter/  # 协议适配器
│   │   ├── config/   # 配置
│   │   ├── protocol/ # 协议定义
│   │   └── server/   # TCP服务器
│   ├── Dockerfile
│   └── go.mod
├── api/              # API服务 (Go)
│   ├── cmd/api/      # 入口程序
│   ├── internal/     # 内部包
│   │   ├── handler/  # HTTP处理器
│   │   ├── model/    # 数据模型
│   │   ├── service/  # 业务逻辑
│   │   └── config/   # 配置
│   ├── Dockerfile
│   └── go.mod
├── web/              # 前端应用 (React)
│   ├── src/
│   │   ├── components/  # 组件
│   │   ├── pages/       # 页面
│   │   ├── services/    # API服务
│   │   └── stores/      # 状态管理
│   ├── Dockerfile
│   └── package.json
├── database/         # 数据库脚本
│   └── init/         # 初始化SQL
├── docker-compose.yml
└── README.md
```

## 开发指南

### 本地开发

```bash
# 1. 启动基础设施
docker-compose up -d postgres redis nats

# 2. 运行网关
cd gateway
go run cmd/gateway/main.go

# 3. 运行API服务 (新终端)
cd api
go run cmd/api/main.go

# 4. 运行前端 (新终端)
cd web
npm install
npm run dev
```

### 添加新协议支持

1. 在 `gateway/internal/adapter/` 创建适配器文件
2. 实现 `ProtocolAdapter` 接口
3. 在 `Detector` 中添加协议识别逻辑

示例:
```go
type MyProtocolAdapter struct{}

func (a *MyProtocolAdapter) Decode(packet []byte) (*protocol.StandardMessage, error) {
    // 实现解码逻辑
}

func (a *MyProtocolAdapter) Encode(cmd protocol.StandardCommand) ([]byte, error) {
    // 实现编码逻辑
}
```

## 架构设计

### 轻网关模式

```
设备 -> 网关 -> (解析) -> NATS -> API服务 -> TimescaleDB/Redis
              <- (ACK) <-        <- 指令下发 <-
```

- **网关职责**: 仅负责TCP连接维护、协议解析
- **业务解耦**: 鉴权、状态机、报警判断上移
- **水平扩展**: 网关无状态，可部署多实例

### 数据流

**上行 (Uplink)**:
1. 设备发送GPS数据到网关
2. 网关解析为 `StandardMessage`
3. 发布到 NATS `fms.uplink.LOCATION`
4. API服务消费并写入TimescaleDB
5. 更新Redis设备影子

**下行 (Downlink)**:
1. API接收指令请求
2. 查询Redis获取设备所在网关
3. 发布到 NATS `gateway.downlink.{node_id}`
4. 网关接收并编码发送给设备

## 文档

- [部署指南](docs/deployment.md)
- [API文档](docs/api.md)
- [协议规范](docs/protocol.md)
- [开发文档](docs/development.md)
- [v1.1 开发总结](docs/v1.1-summary.md)
- [v1.2 开发总结](docs/v1.2-summary.md)

## 贡献

欢迎提交Issue和PR！

1. Fork 本仓库
2. 创建特性分支 (`git checkout -b feature/amazing-feature`)
3. 提交更改 (`git commit -m 'Add some amazing feature'`)
4. 推送分支 (`git push origin feature/amazing-feature`)
5. 创建 Pull Request

## 许可证

[MIT](LICENSE) © OpenFMS Team

## 致谢

- [TimescaleDB](https://www.timescaledb.com/) - 时序数据库
- [NATS](https://nats.io/) - 消息队列
- [Gin](https://gin-gonic.com/) - Go Web框架
- [Ant Design](https://ant.design/) - UI组件库
