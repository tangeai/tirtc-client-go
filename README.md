# TiRTC Server SDK for Go

`go/` 是 `github.com/tangeai/tirtc-client-go/v2` 的开发工程。根 package `tirtc` 提供 headless RTC client，`storage` 子 package 提供云录像查询、回放和导出。Go 只投影 Runtime public C surface；Token 签发、刷新、连接状态和媒体处理都由 Runtime 完成。

这里的 `Client` 表示调用 TiRTC 或 Ti Cloud Storage 服务并持有本地资源树的 SDK 根对象，不表示只负责接收媒体。Go SDK 只提供 AK/SK 托管鉴权；Connection 发布还是订阅媒体，都不改变它相对于 TiRTC 服务的 Client 角色。产品名中的 Server SDK 只描述典型部署位置和信任边界，不表示 `Client` 会监听或对外提供 TiRTC 服务。

当前支持 Go 1.25、`CGO_ENABLED=1` 下的 `darwin/arm64` 与 `linux/amd64`。每个 module 版本同时携带两个平台的 public header、动态库和许可证。

## 运行与构建环境

- macOS 只支持 Apple Silicon，最低版本是 macOS 11.5。
- Linux 只支持 glibc x86_64。当前候选制品的最高符号需求是 `GLIBC_2.35`、`GLIBCXX_3.4.30` 和 `CXXABI_1.3.9`，并在 Ubuntu 22.04 x86_64 容器中实际构建和启动。部署系统必须提供这些或更新的符号版本；Alpine/musl 不是受支持目标。
- 构建应用需要 Go 1.25、`CGO_ENABLED=1` 和本机 C 工具链。Linux 上的 `tirtc-build` 还需要 `patchelf`；macOS 使用 Xcode Command Line Tools 提供的 `install_name_tool` 和 `codesign`。完成构建后，部署时保留整个 `dist/`，不需要 Go 工具链。

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

两份 Example 都只 import 公开 module，凭据只从环境变量读取。

## 仓库入口

测试责任、实现、候选证据与缺口见 [TEST.html](https://gitlab.tange-ai.com/tirtc-nexus/tirtc/-/blob/develop/go/TEST.html)，执行使用 [go-test](https://gitlab.tange-ai.com/tirtc-nexus/tirtc/-/blob/develop/.agents/skills/go-test/SKILL.md)。公开发布使用 [sdk-release-publish](https://gitlab.tange-ai.com/tirtc-nexus/tirtc/-/blob/develop/.agents/skills/sdk-release-publish/SKILL.md)。完整公开声明以 Go 类型、Go doc 与两个 canonical Example 为准。
