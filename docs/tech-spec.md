# Harc Soundscape — 技术规格

> 版本：v0.1  
> 对应 PRD：`docs/prd.md`  
> 本期实现：云端 API + 网页预览引擎

---

## 1. 系统架构

```
┌────────────────────────────────────────────┐
│  Web Preview (Vite + React)                │
│  · 模式/Pack UI                            │
│  · Web Audio SoundscapeEngine              │
│  · 拉取云端配置驱动参数                     │
└─────────────────┬──────────────────────────┘
                  │ REST JSON
┌─────────────────▼──────────────────────────┐
│  Cloud API (Go, net/http)                  │
│  · /modes  /packs  /packs/{id}  /health    │
│  · 内存内容库（可换 DB/对象存储）           │
└─────────────────┬──────────────────────────┘
                  │ （后续）
┌─────────────────▼──────────────────────────┐
│  Speaker Firmware (2G, 后续)               │
│  · 本地 stem 混音 · 缓存淘汰 · 断网可播    │
└────────────────────────────────────────────┘
```

**原则：** 云端下发「声明式引擎配置」；播放端（网页或音箱）负责实时混音。网页与音箱共享同一套 Pack schema。

---

## 2. 仓库结构

```
/
├── docs/
│   ├── prd.md
│   └── tech-spec.md
├── apps/
│   ├── api/                 # Go 云端
│   │   ├── cmd/server/main.go
│   │   ├── internal/
│   │   │   ├── models/      # 数据结构
│   │   │   ├── data/        # 模式与 Pack 种子数据
│   │   │   └── handlers/    # HTTP handlers
│   │   └── go.mod
│   └── web/                 # Vite React 预览
│       ├── src/
│       │   ├── audio/       # SoundscapeEngine
│       │   ├── api/
│       │   ├── components/
│       │   └── styles/
│       └── package.json
└── README.md
```

---

## 3. 数据模型

### 3.1 Mode

```ts
type ModeId = "sleep" | "focus" | "atmosphere" | "relax";

interface Mode {
  id: ModeId;
  name: string;           // 中文展示名
  tagline: string;
  duration_options_min: number[]; // -1 = 无限
  default_duration_min: number;
  palette: {
    bg0: string;
    bg1: string;
    accent: string;
    glow: string;
  };
}
```

### 3.2 Engine Profile（核心）

映射到 Endel 式参数骨架，播放端据此调制各层。

```ts
interface EngineProfile {
  energy: number;          // 0–1
  event_density: number;   // 0–1
  tempo_bpm: number;       // 0 = 无拍感
  brightness: number;      // 高频相对量 0–1
  masking: number;         // 噪声遮掩 0–1
  space: number;           // 空间感/混响感 0–1
  variation_period_sec: number; // 微变周期
  phases: Phase[];         // 入场/维持/收束
}

interface Phase {
  id: "intro" | "sustain" | "outro";
  duration_ratio: number;  // 相对总时长；sustain 可吃满剩余
  energy_mul: number;
  density_mul: number;
}
```

### 3.3 Layer（预览期为程序化描述）

```ts
type LayerKind =
  | "drone"
  | "noise"
  | "pulse"
  | "texture"
  | "event"
  | "bed";

interface Layer {
  id: string;
  kind: LayerKind;
  gain: number;            // 0–1 基础增益
  /** 程序化预览参数；音箱端改为 stem_url */
  synth?: {
    type: "sine" | "triangle" | "sawtooth" | "square" | "brown" | "pink" | "white";
    freq_hz?: number;
    lfo_hz?: number;
    lfo_depth?: number;
  };
  /** 后续真实资源 */
  stem_url?: string;
  loop: boolean;
}
```

### 3.4 Pack

```ts
interface Pack {
  id: string;
  mode: ModeId;
  name: string;
  description: string;
  tier: "free" | "premium";
  version: string;
  engine_profile: EngineProfile;
  layers: Layer[];
  /** 预估安装体积（音箱缓存用），字节 */
  approx_bytes: number;
}
```

---

## 4. Stem / 资源命名规范（音箱端前瞻）

```
packs/{pack_id}/v{version}/{layer_id}.{ext}

示例：
packs/sleep-deep-night/v1/bed_brown.opus
packs/sleep-deep-night/v1/drone_low.opus
packs/focus-clear/v1/pulse_soft.opus
```

约定：
- `ext`：优先 `opus`，备选 `aac`
- 循环层：文件内已是无缝 loop 点；元数据可另附 `loop.json`
- 单层循环建议 8–32s
- 单 Pack 目标 30–80MB（音箱）；网页预览不下载 stem

---

## 5. 云端 API

Base URL（本地）：`http://127.0.0.1:8000`

| Method | Path | 说明 |
|--------|------|------|
| GET | `/health` | `{ status: "ok" }` |
| GET | `/modes` | 模式列表 |
| GET | `/packs?mode={id}` | 按模式筛 Pack；`mode` 可选 |
| GET | `/packs/{pack_id}` | Pack 详情含 layers + engine_profile |
| GET | `/v1/speaker/bootstrap` | （预留）音箱启动：默认 packs + 配置 |

### 5.1 响应约定

- JSON，UTF-8  
- 错误：`{ "detail": "..." }`，HTTP 4xx/5xx  
- CORS：允许网页预览源（开发期 `*`）

### 5.2 音箱缓存策略（规格预留，本期不实现）

| 规则 | 值 |
|------|-----|
| 常驻 | 当前播放 Pack + 各模式默认 Pack（可配置） |
| 内存预算 | stems 解码常驻 ≤ 150MB |
| 淘汰 | LRU；睡眠长会话禁止中途淘汰当前 Pack |
| 更新 | 云端 `version` 变化则空闲时增量下载 |

---

## 6. 网页 SoundscapeEngine

### 6.1 职责

- 根据 `Pack.layers` + `engine_profile` 创建 Web Audio 图  
- 支持 play / pause / stop、模式或 Pack 切换淡入淡出  
- 按 `phases` 与 `variation_period_sec` 缓慢调制 gain / filter  

### 6.2 音频图（示意）

```
Layer nodes (oscillator | noise buffer)
    → GainNode (layer gain × profile)
    → BiquadFilter (brightness)
    → 可选 Delay/Convolver 简化空间感
    → Master Gain
    → DynamicsCompressor
    → destination
```

### 6.3 切换策略

1. 新 Pack 建第二套节点，master 交叉淡化 1.2–2.0s  
2. 旧图 disconnect + stop  
3. UI 参数面板绑定当前 `engine_profile` 的实时展示值（含 phase 乘数）

### 6.4 自动挂起策略

- 遵循浏览器 Autoplay：首次播放需用户手势  
- `visibilitychange` 时可降低能量，不强制停（可选）

---

## 7. 本地 2G 音箱映射（前瞻）

| 网页预览 | 音箱端 |
|----------|--------|
| Oscillator/Noise synth | Opus stem 解码 |
| 全 Pack 常驻内存小 | 流式/分块解码 + 小缓冲 |
| 云端每次进页拉取 | 启动 bootstrap + 差分更新 |
| 无限精细 LFO | 降采样调制，省 CPU |

云端 **schema 保持一致**，仅 `Layer.synth` vs `Layer.stem_url` 二选一为主路径。

---

## 8. 分步路线（模型放第二步）

### 第一步（当前主线，无模型）

- 声明式 Pack + 本地/网页实时 stem（或程序化）混音  
- 时间曲线、场景 phase、设备 EQ、调音迭代  
- Go API 只读下发配置与内容元数据  
- **禁止**：播放链路依赖任何在线推理

### 第二步（模型增强，不替换引擎）

模型只产出「小结果」，由现有引擎执行：

| 能力 | 输出 | 建议 API（预留） |
|------|------|------------------|
| 模式/Pack 推荐 | `pack_id` + 理由码 | `POST /v1/recommend` |
| 偏好 / 参数策略 | `engine_profile` 补丁（delta） | `POST /v1/personalize` |
| 意图理解（文本/语音转写） | mode + profile 微调 | `POST /v1/intent` |
| 环境噪声分类（可选） | masking/brightness 建议 | 端侧小模型或云端 |
| 离线内容辅助 | 新 stem 候选 → 人审入库 | 运营工具链，非播放 API |

**硬约束：**

1. 实时发声仍是 stem 调度；模型失败时回退规则默认包  
2. 不下发大模型权重到 2G 设备（第二步默认云端）  
3. 不做云端实时 AI 作曲作为主听感路径  

字段预留（第一步即可在 schema 中保持可扩展，不必实现）：

```json
{
  "personalization": {
    "profile_delta": { "energy": 0.05, "masking": 0.1 },
    "source": "rules" 
  }
}
```

第二步将 `source` 扩展为 `model:recommend` 等，网页/音箱无需改混音内核。

---

## 9. 安全与运维（本期最小）

- API 只读；无用户写入  
- 无密钥；演示环境勿暴露写接口  
- 日志：请求 path + latency  
- 后续：API Key、Pack 签名校验、CDN Token  
- 第二步模型接口：鉴权、限流、可关断开关（feature flag）

---

## 10. 开发与运行

```bash
# API (Go)
cd apps/api && go run ./cmd/server

# Web
cd apps/web && npm install && npm run dev
```

环境变量：
- `VITE_API_BASE` — 网页请求的 API 根路径（默认开发期 `/api`，由 Vite 代理到 `:8000`）
- `PORT` — Go API 端口（默认 `8000`）

---

## 11. 测试要点

| 项 | 期望 |
|----|------|
| `/modes` | 返回 4 个模式 |
| `/packs?mode=sleep` | 仅睡眠 Pack |
| 网页切模式 | 听感可辨；无爆音硬切 |
| 睡眠 长时 | 参数缓慢漂移，事件稀疏 |
| 办公 | 可感知轻脉冲 |
| CORS | 网页可跨域读 API |
| 第一步回归 | 关闭一切模型相关 flag 后，预览与 API 行为不变 |
