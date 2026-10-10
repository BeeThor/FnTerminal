# 安全问题

FnTerminal 提供真实交互式 Shell。服务默认以独立应用账户运行，NAS 用户权限通过 SSH 二次认证获得；该账户不是恶意程序沙箱，不同管理员共享应用账户的 HOME。

目前 0.9.0 为预发布：真实 fnOS 安装、网关身份、注销、升级卸载和设备权限边界仍待验证。详细模型见 [登录态与权限](docs/DEVELOPMENT.md#登录态与权限) 与 [安全验收](docs/FNOS_TEST_PLAN.md)。

发现鉴权绕过、凭据泄露或提权问题时，请使用仓库 **Security → Report a vulnerability** 私下报告。提供版本、系统环境、最小复现与影响范围；不要在公开 Issue 中提交密码、令牌、私钥或完整 NAS 诊断数据。普通功能问题可通过 Issues 反馈。
