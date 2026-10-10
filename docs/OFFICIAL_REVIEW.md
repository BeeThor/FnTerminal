# 官方规范核对（2026-10-08）

直接获取并核对飞牛官网的 `llms-full.txt` 和下列页面；没有把第三方归档当作官方原文。

| 来源 | 本项目采用的规则 |
|---|---|
| [fnpack](https://developer.fnnas.com/docs/cli/fnpack/) | 官方当前公布 1.2.3；`build --directory`；存在 app、cmd、config、wizard、manifest、64/256 图标 |
| [统一网关](https://developer.fnnas.com/docs/core-concepts/gateway-registration/) | `gatewayPrefix=/app/fnterminal`、`gatewaySocket=app.sock`；Socket 在 target；保留请求前缀；HTTP/WS 注入 `X-Trim-Userid`、`X-Trim-Isadmin` |
| [Native 案例](https://developer.fnnas.com/docs/examples/native/) | 使用 TRIM_APPDEST/PKGHOME/PKGVAR 路径；国内版统一网关示例 os_min_version=1.1.3100 |
| [权限](https://developer.fnnas.com/docs/core-concepts/privilege/) | `run-as=package`、专属账号/组；不加 Docker、sudo、系统文件共享或额外附加组 |
| [Manifest](https://developer.fnnas.com/docs/core-concepts/manifest/) | 二进制分别 platform=x86/arm；ctl_stop=true；不声明 service_port；兼容范围需要真实测试 |
| [生命周期](https://developer.fnnas.com/docs/core-concepts/framework/) | main start/stop/status；0=运行，3=停止；升级/卸载前停止；错误写 TRIM_TEMP_LOGFILE |
| [图标](https://developer.fnnas.com/docs/core-concepts/icon/) | 根 logo.png 缩小为64和256 PNG；同尺寸桌面图标一致；每个图标小于1024 KiB |

官方说明网关进行 NAS 登录校验，但**未公布网关进程 UID、运行组、Socket ACL、Header 覆盖规则及注销时已建立 WS 的失效时限**。这些都列为真机待验证，不靠 allUsers=false 替代鉴权。

默认仅信任 Linux `SO_PEERCRED.uid=0`，Socket 为0600；这是失败关闭的兼容候选值，并非已确认的 fnOS 网关身份。非 root 网关需要在测试机确认身份后配置窄化 GID，仍执行对端 UID 与管理员 Header 检查。应用账户本身不可被配置为可信网关。

0.3.0 修正：0775 target 不再一概拒绝。只有目录属主是 root/应用账户，属组是当前专属应用组，/etc/passwd 与 /etc/group 显示组内仅该包账户和 root 时才接受；其他共享组、其他属主及 world-write 仍拒绝。

`os_min_version=1.1.3100` 是官方 Native 示例的网关最低声明，**未在本项目设备上验证**。ARM64 安装、具体系统版本兼容范围均待验证。

0.4.0连接不再直接依赖代理保留Host。通过不开放CORS的JSON接口颁发CSRF准备凭据，Origin绑定一次性票据，再验证同UID的WS附着。该方案已通过Linux真实代理改写Host的验证，仍需真实fnOS网关回归。

2026-10-09再次直接获取官方fnpack及统一网关页面：fnpack仍列1.2.3，网关路径/Socket/Header规范未发现本次需要变更之处。0.5.0只改变连接入口与SSH目标配置，不扩展应用权限或终端监听端口。

2026-10-10再次获取官方fnpack和统一网关原文：fnpack仍列1.2.3，网关字段保持一致。0.6.0连接列表与可靠性优化仍采用独立package账户和Unix Socket管理员网关，未增加资源或端口声明。
