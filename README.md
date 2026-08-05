# Harc Soundscape

沉浸式自适应声景（对标 Endel）。本期交付：**产品/技术文档 + Go 云端 API + 网页预览**。

## 文档

- [产品 PRD](docs/prd.md)
- [技术规格](docs/tech-spec.md)

## 四模式

睡眠 · 办公 · 氛围 · 放松

## 本地运行

### 1. 云端 API（Go）

```bash
cd apps/api
# 可选：重新生成预览 stem
# python3 scripts/generate_stems.py
go test ./...
go run ./cmd/server
# http://127.0.0.1:8000
```

主要接口：

| Path | 说明 |
|------|------|
| `GET /health` | 健康检查 |
| `GET /modes` | 四模式 |
| `GET /packs?mode=` | 场景包列表（含 `stem_url`） |
| `GET /packs/{id}` | 包详情 |
| `GET /packs/{id}/manifest` | 音箱下载清单 |
| `GET /stems/{pack}/v2/{layer}.opus` | stem 文件 |
| `POST /v1/evolution/events` | 上报播放行为（进化） |
| `POST /v1/evolution/recommend` | 推荐 Pack |
| `POST /v1/evolution/personalize` | 参数补丁 |
| `GET /v1/evolution/insights` | 调试洞察 |
| `GET /v1/speaker/bootstrap` | 音箱启动预留 |

### 2. 网页预览

```bash
cd apps/web
npm install
npm run dev
# http://127.0.0.1:5173  （/api 代理到 :8000）
```

浏览器打开后选择模式 → 开始预览。音频由 Web Audio 按云端下发的层配置程序化合成（非真实录音 stem）。

## 架构摘要

```
Web Preview  --REST-->  Go Cloud API  --(后续)-->  Speaker 2G Engine
```

云端下发声明式 Pack；网页/音箱负责实时混音。

**节奏：** 第一步无模型（stem + 规则）；进化闭环见 `docs/evolution.md`（本期规则版可跑，第二步换模型）。内容不「全网扒音频」。
