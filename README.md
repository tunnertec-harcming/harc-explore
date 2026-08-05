# Harc Soundscape

沉浸式自适应声景（对标 Endel）。

## 文档

- [产品 PRD](docs/prd.md)
- [技术规格](docs/tech-spec.md)
- [进化闭环](docs/evolution.md)（合规：学行为，不扒全网音频）

## 代码结构

```
apps/api       Go 云端（modes/packs/stems/evolution）
apps/web       网页预览（Web Audio 播 stem）
apps/speaker   音箱参考客户端（缓存 + ffmpeg 混音）
docs/          PRD / 技术规格 / 进化
```

## 四模式

睡眠 · 办公 · 氛围 · 放松

## 一键命令

```bash
make test          # API + speaker 单测 + web build
make api           # :8000
make web           # :5173
make speaker-dry   # 拉包+缓存+计划（不播放）
make render        # 混出 /tmp/harc-preview.ogg
bash scripts/smoke.sh
```

## 分服务运行

### 1. 云端 API（Go）

```bash
cd apps/api
go test ./...
go run ./cmd/server
```

| Path | 说明 |
|------|------|
| `GET /health` | 健康检查 |
| `GET /modes` | 四模式 |
| `GET /packs?mode=` | 场景包 |
| `GET /packs/{id}` | 包详情 |
| `GET /packs/{id}/manifest` | 下载清单 |
| `GET /stems/{pack}/v2/{layer}.opus` | stem |
| `POST /v1/evolution/*` | 事件/推荐/个性化/洞察 |
| `GET /v1/speaker/bootstrap` | 音箱启动 |

### 2. 网页预览

```bash
cd apps/web && npm install && npm run dev
# http://127.0.0.1:5173
```

### 3. 音箱参考客户端

```bash
cd apps/speaker
go build -o ../../bin/harc-speaker ./cmd/harc-speaker
./bin/harc-speaker -mode sleep -dry-run
./bin/harc-speaker -mode relax -render /tmp/out.ogg -render-sec 6
# 真机有声卡时：
./bin/harc-speaker -mode focus -duration 25
```

流程：bootstrap → recommend/personalize → 下载 stem 到本地缓存（默认 150MB 上限）→ ffmpeg 多层 amix。

## 节奏

1. **第一步**：stem + 规则混音（已落地 web + api + speaker）
2. **第二步**：进化策略换模型（API schema 已预留）
