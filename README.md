# TiRTC Server SDK for Go

`go/` 是 `github.com/tangeai/tirtc-client-go/v2` 的开发工程。根 package `tirtc` 提供 headless RTC client，`storage` 子 package 提供云录像查询、回放和导出。Go 只投影 Runtime public C surface；Token 签发、刷新、连接状态和媒体处理都由 Runtime 完成。

当前支持 Go 1.25、`CGO_ENABLED=1` 下的 `darwin/arm64` 与 `linux/amd64`。每个 module 版本同时携带两个平台的 public header、动态库和许可证。

## 安装与构建

应用把 SDK 与构建工具固定到同一版本：

```bash
go get github.com/tangeai/tirtc-client-go/v2@<version>
go get -tool github.com/tangeai/tirtc-client-go/v2/cmd/tirtc-build@<version>
go tool tirtc-build build --output dist/bin/app ./cmd/app
```

生成的程序只从相邻的 `dist/lib/tirtc/` 加载 Native library。分发时保留整个 `dist/`，包括 `dist/share/licenses/tirtc/`。

## Client 与凭据

RTC 和云存分别创建 client，均在创建时接收 App ID、AK/SK、可写绝对 CacheDir 和可选 Endpoint：

```go
rtcClient, err := tirtc.NewClient(tirtc.ClientOptions{
    AppID: appID, AccessKeyID: accessKeyID, AccessKeySecret: accessKeySecret,
    CacheDir: cacheDir,
})

storageClient, err := storage.NewClient(storage.ClientOptions{
    AppID: storageAppID, AccessKeyID: storageAccessKeyID,
    AccessKeySecret: storageAccessKeySecret, CacheDir: cacheDir,
})
```

同一进程只允许一个 RTC client；它可以依次或同时拥有多个 Connection。云存允许多个独立 Client，每次查询、Replay 或 Export 显式传入 device ID。各 client 的应用身份和 Endpoint 可以不同；活跃产品共享的 CacheDir 与 `ConsoleLogEnabled` 必须一致。Runtime 内部签发和刷新短期 Token，普通 Go API 不暴露 Token 或 `UpdateToken`。

## 公开能力

根 package `tirtc` 提供主动连接、命令、流消息、订阅、四类 decoded/encoded Output、MP4 Recording、JPEG Snapshot 和日志上传。`Conn.Connect(ctx, deviceID)` 等待本次建连的唯一初始结果；初次失败只从返回值取得，连接成功后的断线继续走状态 callback。Context 取消会结束业务 attempt；Native cleanup barrier 尚未越过时 Close 返回 `ErrInUse`，可重试。

`storage` 提供录像自然日和范围查询、Replay、四类 Output、暂停/恢复/Seek/倍速、Replay Recording、Snapshot 和独立范围 Export。Export 提供覆盖进度、已确认缺口事件和最终 `ExportReport`；部分可播放 MP4 是成功结果，`Report.Complete` 用来区分完整与部分覆盖。

Frame callback 收到的 byte slice 和 plane 已脱离 C callback 生命周期。Callback mailbox 有界；媒体帧因背压丢弃后，下一帧以 `Discontinuity=true` 标记。

Export 缺口通知使用有界背压，`Wait` 等待已接受的用户回调排空；部分文件不使提前中断的进度变为 100%。本对象 observer 内调用 Export `Wait` 返回 `ErrInUse`，其他协程可以等待。

Replay 用户回调直接由排入 Go 队列的 Runtime callback task 调用，Seek/新 Play 的 generation 检查保留到交付时；同一 Replay handler 内的 Play/Seek/Stop 返回 `ErrInUse`；其他操作遇正在执行的 Replay 操作时也返回 `ErrInUse`，避免与回调排空互等。

Recording、Export 和 Snapshot 返回 Runtime cache 中的临时文件。应用用标准 `os`/`io` 保存到业务目录，随后调用结果对象的 `Delete()` 清理临时源文件。已成功发布的 Export 文件保留独立删除归属，可在 client 关闭后保存和删除；同一结果的副本共享成功删除状态。

## Examples

- [`example/client`](example/client/README.md) 展示单设备 RTC 签发、建连、四类 Output、控制消息、录像、截图和清理。
- [`example/storage`](example/storage/README.md) 展示云录像查询、四类 Output、回放控制、缺口事件、覆盖报告、录像、导出和截图。

两份 Example 都只 import 公开 module，凭据只从环境变量读取；它们也是候选包真实验收使用的 canonical clients。

## 验证与发布

```bash
go test -race ./...
```

`script/build_candidate.sh` 从同一批 Runtime 生成双 tuple candidate。`script/go_verify.sh` 运行两平台 contract 和单设备 RTC smoke；`tool/ti_cloud_storage.py` 运行两平台云存 Example；`tool/upload_logs.py` 从同一 candidate 执行一次真实日志上传并产生身份绑定 summary。`tool/release_gate.py` 要求完整六条产品 lane 与 UploadLogs 证据后才允许公开发布。

完整声明见 [Go RTC API Reference](../docs/rtc/api-reference/go.md) 和 [Go Ti Cloud Storage API Reference](../docs/cloud-storage/public/api-reference/go.md)。
