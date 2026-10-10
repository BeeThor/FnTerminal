# 参与开发

先阅读 [架构说明](docs/ARCHITECTURE_REVIEW.md)、[开发与运行说明](docs/DEVELOPMENT.md) 和 [安装验收](docs/FNOS_TEST_PLAN.md)。

- 使用锁定的 Go/npm 依赖，运行 `./scripts/check.sh`，修改终端或界面时运行浏览器回归。
- 不扩大默认账户权限、不暴露终端 TCP 入口、不存储密码；鉴权不可确认时保持失败关闭。
- 修改提取的 Uniterm 文件时保留顶行改动声明，运行 `python3 scripts/check-upstream.py`，保留上游原始文件及许可证。
- 新功能应说明实际检查环境。没有经过真实 fnOS 或手机验证的行为明确标为待验证。
- `dist/`、依赖、缓存及 `test-results/` 不提交。公开截图与日志先清除账户、地址、凭据及其它私人信息；发布脚本默认排除原始测试证据。

Pull Request 请描述问题、最终行为及相应验证结果；成品统一放入 GitHub Releases。
