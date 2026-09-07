# TiRTC Go SDK

`go/` 是 `github.com/tangeai/tirtc-client-go/v2` 的开发工程。根 package `tirtc` 提供 headless RTC client，`storage` 子 package 提供云录像查询、回放和导出。两者都只消费 Runtime public C surface，不重新实现连接、媒体处理、录像或云端访问。

当前支持 Go 1.25、`CGO_ENABLED=1` 下的 `darwin/arm64` 与 `linux/amd64`。每个 module 版本同时携带两个平台的 public header、动态库和第三方许可证。应用使用同版本 `tirtc-build` 生成可搬移的 `dist/bin` 与 `dist/lib/tirtc`，不需要在目标机器全局安装 Runtime。

## 安装与构建

在应用 module 中选择同一个 SDK 版本作为 library 和 build tool：

```bash
go get github.com/tangeai/tirtc-client-go/v2@<version>
go get -tool github.com/tangeai/tirtc-client-go/v2/cmd/tirtc-build@<version>
go tool tirtc-build build --output dist/bin/app ./cmd/app
```

成功后，`dist/bin/app` 只从相邻的 `dist/lib/tirtc/` 加载 Native library。分发应用时必须保留整个 `dist/`，包括 `dist/share/licenses/tirtc/`。

`InitOptions.CacheDir` 是必填的可写绝对路径。RTC 与 Ti Cloud Storage 可以使用不同的 App ID 和 Endpoint，但同一进程中的两者必须使用相同的 CacheDir 和 `ConsoleLogEnabled`。配置冲突在创建目录或启动 Runtime 前返回 `ErrAlreadyInitialized`。

## 公开能力

根 package `tirtc` 提供：

- `Init`、`Shutdown` 和 `UploadLogs`；
- 主动连接、命令、流消息、订阅和关键帧请求；
- decoded/encoded Audio/Video Output；
- MP4 RecordingTask 和 JPEG Snapshot；
- 稳定 error sentinel，以及保留 Native 错误码的 `*Error`。

`storage` 提供：

- 独立的 `Init`、`Shutdown` 和设备授权 `CloudStorage`；
- 录像自然日、录像时间段查询和 `UpdateToken`；
- Replay、decoded/encoded Output、暂停、恢复、Seek、倍速和当前时间；
- 回放录像、范围导出和 Snapshot。

Frame callback 收到的字节和 plane 已脱离 C callback 生命周期；调用方可以在 callback 返回后继续读取。SDK 的 callback mailbox 有界，媒体帧因背压丢弃后，下一帧以 `Discontinuity=true` 标记不连续。

Recording、Export 和 Snapshot 返回 Runtime cache 中的临时文件。应用先用标准 `os`/`io` API 把文件复制或移动到自己的目录，再调用结果对象的 `Delete()` 删除临时源文件。Go SDK 不接管相册或系统媒体目录。

## Examples

[`example/client`](example/client/README.md) 展示 RTC 主动连接、四种 Output、控制消息、关键帧请求、录像、截图和逆序关闭。

[`example/storage`](example/storage/README.md) 展示云录像日期与范围查询、显式 Token 更新和重试、四种 Output、回放控制、录像、范围导出、截图和逆序关闭。

两个 Example 都只 import 公开 module，Secret 只从环境变量读取，普通配置通过 flag 传入。它们也是候选包真实运行验收的标准 client，不存在另一套私有 acceptance 程序。

## 验证与发布边界

仓内开发测试：

```bash
go test -race ./...
```

`script/build_candidate.sh` 从同一次 Runtime candidate 生成双平台本地 module proxy。`script/go_verify.sh` 验证 Darwin/Linux contract 和两条真实 RTC smoke；`tool/ti_cloud_storage.py` 在两个平台运行公开 Ti Cloud Storage Example。具体输入和完成信号由仓库 `go-test` Skill 定义。

公开源码由 `tool/project_release.py` 从 allowlist 投影。投影会拒绝 symlink、Git LFS pointer、超限文件、仓库私有工具，以及源码或 metadata 中会形成本机依赖的绝对路径；Native 可搬移性由 RPATH、install name 和动态依赖检查负责。编译器写入动态库的源码字符串不参与加载，因此不作为 candidate 或发布门禁。发布 module、tag、外部 Samples 或上传日志都需要用户明确授权。

完整声明见 [Go RTC API Reference](../docs/rtc/api-reference/go.md) 和 [Go Ti Cloud Storage API Reference](../docs/cloud-storage/public/api-reference/go.md)，实现边界见 [Go SDK 交付设计](../docs/cloud-storage/internal/go-sdk-design.md)。
