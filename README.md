# 管风琴联动验算台

管风琴联动（coupler）可以把一个键经多级移调传到同一音管；配置成环时递归会失控，
按路径累计又会重复计数。本验算台把联动传播建模为**“键盘＋键号”状态图上的可达闭包**，
让设计师在采纳配置前得到**不漏音、不重算**的联动核查结果。

- 前端：Vue 3 + TypeScript（导入/编辑配置 JSON、虚拟键盘、传播图、裁剪原因、音管核查单、快照管理）
- 后端：Go + Gin（校验、闭包求值、稳定路径裁决、快照存取）
- 运维：Docker Compose 一键联调，`verify` 一次性服务跑齐测试与生产构建

## 仓库结构

```
├── api/                    # Go/Gin 后端
│   ├── main.go             #   HTTP 路由
│   └── internal/core/      #   领域逻辑（纯函数，可单测）
│       ├── types.go        #   配置/请求/结果模型
│       ├── validate.go     #   定位校验
│       ├── verify.go       #   状态图闭包 + 裁决
│       ├── snapshot.go     #   快照仓库（内存 + 可选落盘）
│       └── verify_test.go  #   场景测试
├── web/                    # Vue 3 + TS 前端（Vite）
│   └── src/
│       ├── lib/layout.ts   #   传播图布局（纯函数，含单测）
│       └── components/     #   编辑器/虚拟键盘/传播图/核查单/快照
├── verify/Dockerfile       # 一次性服务：全部测试 + 生产构建
└── docker-compose.yml
```

## 配置 JSON 模式

```jsonc
{
  "keyboards": [
    { "id": "I", "midiMin": 36, "midiMax": 96 }   // 键盘：全局唯一非空 ID + 有效 MIDI 键域
  ],
  "stops": [
    {
      "id": "S-Prinzipal",        // 音栓：全局唯一非空 ID
      "keyboard": "I",            // 所属键盘
      "pipes": { "60": "P-8-060" } // 键号 -> 物理音管 ID
    }
  ],
  "couplers": [
    {
      "id": "C-I-II",   // 联动：全局唯一非空 ID
      "from": "I",       // 按键键盘
      "to": "II",        // 发声键盘
      "offset": 12,      // 加到当前键号上的移调量（可为负）
      "enabled": true    // 可省略，缺省视为 true
    }
  ]
}
```

核查输入为配置 + 按下的键 + 启用的音栓 ID 列表：

```jsonc
{
  "config": { /* 上文的配置 */ },
  "pressed": [{ "keyboard": "I", "key": 60 }],
  "stops": ["S-Prinzipal"]
}
```

## 核查算法（`api/internal/core/verify.go`）

1. **状态图**：节点是 `(键盘, 键号)`。按下的键是 0 边起点；每条启用的联动边
   `(from, to, offset)` 把状态 `(from, k)` 松弛到 `(to, k+offset)`，逐边累加移调。
2. **终止性**：不设任何深度上限。状态空间有限（键盘数 × 键域），采用 Dijkstra 式
   最优路径闭包，每个状态只确定一次，闭环（含零偏移自环）自然终止。
3. **越界裁剪**：目标键号越出发声键盘有效音域时，仅裁剪该分支并记录原因
   （目标键号、音域、联动 ID、路径），其余分支照常传播。
4. **音管去重**：状态 `(K, k)` 经 K 上每个启用音栓查映射发声；同一物理音管 ID
   无论被多少音栓/多少路径触达，只计一次。启用音栓在可达键位无映射时记入“潜在漏音”。
5. **稳定裁决**：每根音管的来源取所有候选中的最小者，依次比较
   **边数 → 联动 ID 序列 → 起始键号 → 音栓 ID**。
   联动 ID 序列与音栓 ID 按 Unicode 码点字典序比较（UTF-8 字节序与码点序一致，
   Go 字符串比较天然满足）。状态级最优路径同样按此序，保证全量结果确定可复现。

## HTTP API

| 方法 | 路径 | 说明 |
| --- | --- | --- |
| GET | `/api/healthz` | 健康检查 |
| POST | `/api/verify` | 核查；成功返回结果，`422` 返回 `[{path, message}]` 定位错误 |
| POST | `/api/snapshots` | 采纳后保存 `{name, input, result}` 快照 |
| GET | `/api/snapshots` | 快照摘要列表 |
| GET / DELETE | `/api/snapshots/:id` | 读取 / 删除快照 |

校验错误均带 JSON 路径定位，例如 `config.couplers[1].id`（重复联动 ID）、
`config.stops[0].keyboard`（未知键盘）、`config.keyboards[0].midiMax`（非法音域）、
`pressed[0].key`（按键越界）、`stops[1]`（未知音栓）。

快照默认保存在容器卷 `/data/snapshots.json`（`SNAPSHOT_FILE` 可改；置空则仅内存）。

## Docker Compose 联调

```bash
docker compose up --build          # web + api（verify 为一次性服务，可用 run 触发）
docker compose run --rm verify     # 跑齐 api/web 测试与生产构建，失败即非零退出
```

宿主端口可用环境变量覆盖：

```bash
WEB_PORT=9000 API_PORT=9001 docker compose up --build
# 前端 http://localhost:9000 （/api 由 nginx 反代到 api 服务）
# 后端 http://localhost:9001/api/healthz
```

## 本地开发

```bash
# 后端（Go 1.22+）
cd api && go test ./... && go run .          # 监听 :8080

# 前端（Node 20+）
cd web && npm ci && npm run test && npm run dev
# vite 开发服务器把 /api 代理到 http://localhost:${API_PORT:-8080}
```

## 测试覆盖

`api/internal/core/verify_test.go`：

- 无联动：按键只经本键盘启用音栓发声
- 多级移调：`I -(+12)→ II -(+7)→ III` 逐边累加
- 闭环终止：双向循环联动 + 零偏移自环，状态与音管不重复
- 边缘越界：越界分支裁剪并记录原因，其余分支不受影响
- 共享音管去重：多音栓/多路径触达同一音管只计一次
- 稳定路径裁决：边数 → 联动 ID 序列 → 起始键号 → 音栓 ID
- 定位校验：重复联动 ID、未知键盘、非法音域、未知音栓等
- 禁用联动不参与传播；缺映射记入漏音点；重复求值结果完全一致

`web/src/lib/layout.spec.ts`：传播图按边数分列、列内确定排序、
裁剪目标落列与原因携带、空闭包合法画布。
