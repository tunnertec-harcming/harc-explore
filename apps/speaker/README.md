# Harc Speaker（参考客户端）

面向 2G 内存音箱的 Go 参考实现：同一套云端 Pack/stem/进化协议。

## 能力

1. `bootstrap` 拉默认包  
2. `recommend` / `personalize`（进化）  
3. 按 `manifest` 下载 Opus stem 到本地缓存（默认 150MB，LRU 淘汰）  
4. 生成混音计划；`ffmpeg` 多层 `amix` 播放或渲染文件  

生产固件可用板端 DSP 替换 `internal/engine`，**不要改云端 schema**。

## 运行

```bash
# 需先启动 apps/api
go run ./cmd/harc-speaker -mode sleep -dry-run
go run ./cmd/harc-speaker -mode relax -render /tmp/out.ogg -render-sec 6
```
