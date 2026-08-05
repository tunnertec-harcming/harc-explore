# Harc 听感进化闭环（v0.1）

> 定位：**合规进化**，不是全网扒音频。  
> 阶段：第一步已有 stem+规则；本版先用 **规则引擎** 跑通闭环，第二步再换模型。

---

## 1. 一句话

系统只学习「用户怎么用」和「自有/授权内容表现如何」，持续优化 **推荐与参数**；播放文件永远来自自有 Pack，不收录公网盗版音频。

---

## 2. 学什么 / 不学什么

| 可以学 | 不可以碰 |
|--------|----------|
| 播放、暂停、完播、切换 Pack、停留时长 | 爬取 Spotify/YouTube/网盘等版权音频入库 |
| 时段（昼/夜）、所选模式 | 把未授权录音当 stem 下发 |
| 显式反馈（喜欢 / 太吵 / 太闷，预留） | 存储可还原原曲的整段爬取音频 |
| 匿名聚合：哪类 Pack 留存更好 | 声称「全网收录」 |

---

## 3. 闭环流程

```
播放端上报事件 (play/heartbeat/switch/stop)
        ↓
云端 Evolution Store（本期内存；可换 DB）
        ↓
规则策略（第二步 → 模型）
        ↓
输出小结果：推荐 pack_id + engine_profile delta
        ↓
播放端仍用 stem 引擎执行（失败则回退默认包）
```

**硬约束：** 模型/规则失败时，行为与无个性化完全一致。

---

## 4. API

Base：与现有云端相同。Feature flag：`EVOLUTION_ENABLED=1`（默认开启规则版）。

### 4.1 上报事件

`POST /v1/evolution/events`

```json
{
  "device_id": "web-preview-demo",
  "session_id": "uuid",
  "events": [
    {
      "type": "play_start",
      "ts": 1710000000,
      "mode": "sleep",
      "pack_id": "sleep-deep-night",
      "duration_min": 60
    },
    {
      "type": "heartbeat",
      "ts": 1710000060,
      "mode": "sleep",
      "pack_id": "sleep-deep-night",
      "listened_sec": 60
    },
    {
      "type": "pack_switch",
      "ts": 1710000100,
      "mode": "sleep",
      "pack_id": "sleep-soft-rain",
      "from_pack_id": "sleep-deep-night"
    },
    {
      "type": "play_stop",
      "ts": 1710000200,
      "mode": "sleep",
      "pack_id": "sleep-soft-rain",
      "listened_sec": 100
    }
  ]
}
```

响应：`{ "accepted": N }`

### 4.2 推荐

`POST /v1/evolution/recommend`

```json
{
  "device_id": "web-preview-demo",
  "mode": "sleep",
  "hour_local": 23
}
```

响应：

```json
{
  "pack_id": "sleep-soft-rain",
  "score": 0.82,
  "source": "rules",
  "reason": "night_affinity+dwell"
}
```

无历史时：`source: "default"`，返回该模式默认包。

### 4.3 个性化参数补丁

`POST /v1/evolution/personalize`

```json
{
  "device_id": "web-preview-demo",
  "pack_id": "sleep-deep-night"
}
```

响应：

```json
{
  "pack_id": "sleep-deep-night",
  "profile_delta": {
    "energy": -0.03,
    "brightness": -0.04,
    "masking": 0.05,
    "space": 0.04,
    "event_density": -0.02
  },
  "source": "rules",
  "reason": "late_night_soften"
}
```

播放端：`final = clamp(base + delta, 0..1)`（tempo 等非 0–1 字段本期不改）。

### 4.4 洞察（调试）

`GET /v1/evolution/insights?device_id=`

返回该设备各 pack 累计收听秒数、切换次数等，便于网页预览展示「进化中」。

---

## 5. 本期规则策略（可被模型替换）

1. **默认：** `DefaultPackIDs[mode]`  
2. **亲和：** 同 mode 下 `listened_sec` 最高的 pack 加权  
3. **夜间（22–5 点）：** 偏好 sleep 包；personalize 降低 energy/brightness，略增 masking/space  
4. **办公时段（9–18）：** 若 mode=focus，略提 energy，降 event_density  
5. **早切惩罚：** `listened_sec < 45` 的 pack_switch 降低该 from_pack 分  

第二步：把 2–5 换成模型打分，**请求/响应 schema 不变**。

---

## 6. 内容进化（与播放学习分开）

| 管道 | 输入 | 输出 |
|------|------|------|
| 授权/自采 | 音效库、自录 | 新人审 stem → Pack |
| 离线辅助生成 | brief / 标签 | 候选音频 → 人审+调音 |
| 公网 | 仅允许：公开文本趋势、标签统计 | **方向洞察**，不入库音频 |

禁止：自动爬取音频写入 `assets/stems`。

---

## 7. 隐私

- 本期 `device_id` 为预览用本地随机 ID（localStorage）  
- 不上报原始麦克风、通讯录、精确 GPS  
- 后续：可清除、可关闭进化、数据留存周期

---

## 8. 验收

1. 多次播放同一 pack 后，`recommend` 倾向该 pack  
2. 夜间 `personalize` 返回偏软 delta  
3. 关掉事件上报时，推荐回到 default  
4. 播放链路仍只拉自有 `/stems/...`  
