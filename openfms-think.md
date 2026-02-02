---
date: 2026-02-02
project: OpenFMS
status: Phase 1 (Launched)
tags:
  - #Strategy
  - #Branding
  - #Operations
  - #SEO
  - #Golang
---

# OpenFMS - 开源车队管理系统技术方案

## 1. 项目背景与定位

### 1.1 市场痛点
- 传统车队管理软件（包括开源的Traccar）UI与技术框架老旧，费用不菲
- 商用闭源车队管理软件（如 Samsara, Geotab）昂贵且数据封闭
- 开源方案（如 Traccar）在UI/UX和特定场景落地（电动车队、冷链）上有优化空间

### 1.2 目标用户
- 中小企业/物流公司：需要低成本车队管理方案，基础功能完全免费
- 开发者/集成商：需要灵活的底座来开发定制化解决方案

### 1.3 核心目标
- 面向中小企业/物流公司的轻量级、现代化、低成本车队管理平台
- 初期锚定1000台车以内，架构支持横向扩展
- 极低运营成本（单机VPS可跑），适合私有化部署
- 抛弃旧时代UI，采用React/Vue + Mapbox/Leaflet
- 商业模式：核心免费 + 授权私有化 + 硬件捆绑 + 高级功能收费

### 1.4 设备支持
- 支持多类型、多厂家设备接入（类似Wialon平台）
- 车载设备定义：GPS Tracker、DashCam（可定位、传输视频）、MDVR（可定位、传视频）
- 主要协议：JT808（国内部标）、JT1078（音视频流）

---

## 2. 技术栈选型

| 模块 | 选型 | 理由 |
|------|------|------|
| 后端API | Golang (Gin/Echo) | 高并发、二进制部署简单（无依赖）、内存占用极低 |
| 设备网关 | Golang (Net) | 原生协程处理TCP长连接，适合高并发I/O |
| 前端 | React / Vue 3 | 现代化组件库(Ant Design/Shadcn)，体验好，开发快 |
| 数据库 | TimescaleDB (PostgreSQL) | 完美解决时序轨迹存储，支持列式压缩（节省90%空间） |
| 缓存/状态 | Redis | 存储会话(Session)、设备影子(Shadow)、指令队列 |
| 消息队列 | Redis Streams / NATS | 解耦网关与业务逻辑，实现"轻网关"架构 |
| 地图引擎 | Mapbox GL JS / Leaflet | Mapbox视觉效果最好，Leaflet完全开源免费 |
| 部署 | Docker Compose | 一键交付，简化私有化部署流程 |

---

## 3. 架构设计

### 3.1 轻网关策略 (Light Gateway Pattern)

**网关职责：**
- 仅负责TCP/UDP连接维护、粘包处理(Framer)、协议解析(Codec)
- 不涉及数据库读写
- 解析出标准JSON后，直接推送到MQ

**业务解耦：**
- 网关与业务系统通过消息队列解耦
- 业务逻辑（鉴权、状态机、报警判断）上移

**协议适配：**
- 定义通用`ProtocolAdapter`接口
- 实现多协议（JT808, GT06, Wialon）的热插拔兼容

### 3.2 数据流转

**上行 (Uplink)：**
```
设备 -> 网关 -> (解析) -> MQ -> 业务消费者 -> TimescaleDB / Redis Shadow
```

**下行 (Downlink)：**
```
API -> Redis (查Session位置) -> MQ -> 网关 -> (编码) -> 设备
```

### 3.3 视频流策略

- 视频流（JT1078）与指令流物理分离，不经过Go网关
- 网关仅负责信令控制
- 设备推流直连流媒体服务（ZLMediaKit或SRS）
- 前端播放器直接连流媒体服务观看

---

## 4. 协议适配层设计

### 4.1 三层抽象架构

#### 第一层：网络帧边界 (PacketScanner)
解决粘包、拆包问题，从TCP流中切出完整数据包。

```go
type PacketScanner interface {
    // 从buffer中提取完整数据包
    // completePacket: 切割好的完整包（已去除转义、校验码等底层噪音）
    // restBuffer: 剩余未处理字节
    Scan(buffer []byte) (completePacket []byte, restBuffer []byte, err error)
}
```

#### 第二层：业务数据翻译 (ProtocolAdapter)
将完整包翻译成系统标准数据模型。

**标准数据模型：**
```go
type StandardMessage struct {
    DeviceID  string
    Type      string  // "AUTH", "LOCATION", "HEARTBEAT", etc.
    Timestamp int64
    Lat       float64
    Lon       float64
    Speed     float64
    Direction float64
    Extras    map[string]interface{} // 扩展字段：油量、温度、门开关
}

type StandardCommand struct {
    Type     string
    Params   map[string]interface{}
}
```

**适配器接口：**
```go
type ProtocolAdapter interface {
    // 解码：只负责翻译，不管业务对错
    Decode(packet []byte) (*StandardMessage, error)
    
    // 编码：把业务指令翻译成二进制
    Encode(cmd StandardCommand) ([]byte, error)
    
    // 心跳包检测（可选）
    IsHeartbeat(packet []byte) bool
    GenerateHeartbeatAck(packet []byte) []byte
}
```

#### 第三层：协议探测器 (Detector)
连接建立初期识别协议类型。

```go
type Detector interface {
    // 嗅探：查看前几个字节判断协议类型
    // 0x7E开头 -> JT808
    // 0x78 0x78开头 -> GT06
    Match(headerBytes []byte) (ProtocolAdapter, bool)
}
```

### 4.2 高并发处理要点

**TCP粘包/拆包处理：**
- 利用JT808特性：以`0x7E`开头，以`0x7E`结尾
- 使用`bufio.Scanner`或自定义`SplitFunc`
- 在数据进入业务逻辑层前，先经过Frame Decoder
- 转义还原（Unescape）：`0x7d 0x02`还原成`0x7e`，`0x7d 0x01`还原成`0x7d`

**内存管理：**
- 使用`sync.Pool`复用buffer
- 避免频繁的创建销毁byte数组

**异步处理模型：**
- 不在收到数据包的Goroutine里做数据库入库操作
- 架构：`TCP Handler` -> `Decode` -> `Push to Channel/Queue` -> `Return`
- Worker Goroutines消费队列数据（入库、报警判断）

---

## 5. Redis数据结构设计

### 5.1 会话路由表 (Session Routing)
记录设备连在哪个网关节点的哪个Socket上。

- **Key格式：** `fms:sess:{sim_no}` (例：`fms:sess:13900000001`)
- **数据结构：** String
- **Value格式：** `{gateway_id}:{conn_id}:{client_ip}` (例：`node-01:48292:10.0.2.5`)
- **过期策略：** 300秒（5分钟），设备登录/心跳时续期

### 5.2 设备影子 (Device Shadow)
维护设备最新状态镜像，供前端Dashboard查询。

- **Key格式：** `fms:shadow:{sim_no}`
- **数据结构：** Hash
- **Fields：**
  - `lat`: 纬度
  - `lon`: 经度
  - `spd`: 速度
  - `dir`: 方向
  - `st`: 状态位（ACC、报警等掩码）
  - `fuel`: 油量
  - `ts`: 最后更新时间戳
- **过期策略：** 24小时

### 5.3 指令待答复池 (Command Pending Pool)
暂存下发指令等待设备ACK。

- **Key格式：** `fms:cmd:{sim_no}:{serial_no}` (例：`fms:cmd:13900000001:100`)
- **数据结构：** String
- **Value格式：** JSON字符串 `{ "type": "STOP_ENGINE", "req_ts": 167888888, "user_id": "admin" }`
- **过期策略：** 30秒（超时时间）

### 5.4 地理空间索引 (Geo-Fencing)
用于实时计算"谁在禁区里"或"找最近的车辆"。

- **Key格式：** `fms:geo:online`
- **数据结构：** GEO (Sorted Set)
- **Member：** `{sim_no}`
- **操作：** `GEOADD`更新位置，`GEORADIUS`/`GEOSEARCH`查询

### 5.5 并发锁
防止并发下发冲突。

- **Key格式：** `fms:lock:{sim_no}`
- **过期策略：** 1秒

---

## 6. TimescaleDB表结构设计

### 6.1 核心轨迹表

```sql
-- 启用扩展
CREATE EXTENSION IF NOT EXISTS timescaledb;
CREATE EXTENSION IF NOT EXISTS postgis;

-- 创建轨迹表
CREATE TABLE vehicle_positions (
    time        TIMESTAMPTZ       NOT NULL,  -- 时间戳
    device_id   VARCHAR(20)       NOT NULL,  -- 设备号(sim_no)
    lat         DOUBLE PRECISION  NOT NULL,  -- 纬度
    lon         DOUBLE PRECISION  NOT NULL,  -- 经度
    speed       SMALLINT,                    -- 速度
    angle       SMALLINT,                    -- 方向(0-360)
    flags       INTEGER,                     -- 状态位(ACC、报警等掩码)
    extras      JSONB                        -- 扩展：油量、温度、门磁
);

-- 转化为Hypertable（按时间切片）
-- 1000台车建议7天一个Chunk；百万台车建议1天或4小时一个Chunk
SELECT create_hypertable('vehicle_positions', 'time', chunk_time_interval => interval '7 days');

-- 复合索引（查询速度保证）
-- 场景：查"某辆车"在"某段时间"的轨迹
CREATE INDEX ix_device_time ON vehicle_positions (device_id, time DESC);
```

### 6.2 压缩策略

```sql
-- 开启压缩（按device_id分组压缩，按时间排序）
ALTER TABLE vehicle_positions SET (
    timescaledb.compress,
    timescaledb.compress_segmentby = 'device_id',
    timescaledb.compress_orderby = 'time DESC'
);

-- 自动压缩策略：数据写入7天后自动压缩
SELECT add_compression_policy('vehicle_positions', INTERVAL '7 days');
```

**压缩效果：** GPS轨迹数据压缩率通常在90%-95%。

### 6.3 数据保留策略

```sql
-- 自动删除90天以前的数据
SELECT add_retention_policy('vehicle_positions', INTERVAL '90 days');
```

---

## 7. 核心功能模块

### 7.1 MVP阶段核心功能

1. **实时监控 (Real-time Tracking)**
   - 地图上车辆平滑移动（前端插值算法）
   - 点击车辆弹出信息卡片（速度、状态、最后位置）

2. **历史回放 (History Playback)**
   - 带进度条、倍速播放
   - 轨迹纠偏和降噪（Kalman Filter或抽稀算法）

3. **电子围栏 (Geofencing)**
   - 圆形和多边形围栏
   - 进出围栏实时Webhook推送或报警

4. **基础报表 (Basic Reports)**
   - 里程报表
   - 停车报表

### 7.2 鉴权流程（轻网关模式）

1. 设备发送鉴权包 -> 网关解析出DeviceID和AuthToken
2. 网关不查数据库，通过MQ发送`device.auth.request`
3. 业务系统查Postgres验证，写入Redis `fms:sess:{sim_no}`
4. 业务系统回复网关`Allow: true`
5. 网关封装JT808 `0x8001`通用应答发送回设备

### 7.3 指令下发流程

1. 前端API请求 -> 业务系统查Redis `fms:sess:{sim_no}`获取网关ID
2. 向MQ发送指令 `gateway.downlink.{node_id}`
3. 网关调用`ProtocolAdapter.Encode()`转成二进制流
4. 通过Socket发送给设备

---

## 8. 成本分析

### 8.1 硬件资源需求（1000台车规模）

- **CPU：** 2核 ~ 4核
- **内存：** 4 GB ~ 8 GB
- **硬盘：** 100 GB SSD
- **带宽：** 3 Mbps ~ 5 Mbps（不含视频流）

**成本估算：**
- 阿里云/腾讯云：约150-300元/月
- Hetzner/DigitalOcean：约$10-$20/月

### 8.2 软件授权成本

| 组件 | 协议 | 费用 |
|------|------|------|
| Golang | BSD | $0 |
| Redis | BSD | $0 |
| PostgreSQL | PostgreSQL | $0 |
| TimescaleDB | TSL / Apache 2 | $0 |
| Frontend | MIT | $0 |

**TimescaleDB注意：** TSL协议允许免费使用压缩等高级功能，前提是不能做"云数据库服务"来卖。做车队管理SaaS完全合规且免费。

### 8.3 隐形成本

**地图服务：**
- Google Maps API费用高昂
- **省钱方案：** 使用OpenStreetMap (OSM) + Leaflet/MapLibre（$0）

**存储扩容：**
- 严格执行Retention Policy（数据保留策略）
- 免费版/基础版：保留3个月
- 付费版：保留1年

---

## 9. 商业策略

### 9.1 商业模式

- **Open Core（开放核心）：** 核心功能免费，高级功能（高级报表、AI驾驶行为分析、多租户管理）收费
- **托管服务：** 帮不想自己运维服务器的公司托管系统
- **硬件捆绑：** 软件免费，通过售卖预配置好的硬件（GPS终端、行车记录仪）获利
- **授权私有化部署：** 企业版导入加密的License Key解锁限制

### 9.2 License授权机制

- 代码预埋License检查模块
- 免费版：限制车辆数（如10台）或限制账号数
- 企业版：导入加密的License Key解锁限制
- 绑定服务器MAC地址或CPU ID，防止"买一套，装十家"

### 9.3 视频服务计费

- GPS数据便宜，视频流极其烧钱
- 视频服务必须单独计费，或限制每月观看时长

---

## 10. 下一步行动计划

1. **初始化仓库：** 在GitHub/GitLab创建项目，上传`docker-compose.yml`
2. **MVP验证：** 编写第一个Go TCP Server，实现JT808协议的最简解析（收到`0x0200`并打印）
3. **跑通链路：** 让Go Server把解析的数据写入Redis，观察数据结构是否符合预期
4. **部署验证：** 使用Docker Compose一键部署完整环境

---

## 附录

### 协议支持计划

**MVP阶段（2种）：**
1. JT808（国内部标）- 搞定国内绝大多数设备和合规需求
2. Traccar协议（OsmAnd）或Watcher - 通用HTTP协议兜底，方便手机模拟测试

**后续扩展：**
- GT06、Wialon IPS、Teltonika等通过插件化架构接入

### 关键技术决策

1. **轻网关 vs 重网关：** 选择轻网关，业务逻辑上移，网关只负责协议转换
2. **状态处理：** 鉴权/会话状态放到上层业务系统处理
3. **视频分离：** JT1078视频流旁路设计，不经过Go网关
4. **数据库存储：** 使用TimescaleDB而非MySQL+InfluxDB组合，降低维护复杂度
