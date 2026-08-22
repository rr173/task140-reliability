# task140-reliability

可靠性分析与安全评估引擎，提供 FTA、FMEA、RBD 和失效数据拟合的 HTTP 接口。

## 本地验证

```bash
go test ./...
go run . --smoke-test
```

## Docker

```bash
bash build_benzhi_docker.sh task140-reliability linux/amd64
docker run --rm task140-reliability go run . --smoke-test
```
