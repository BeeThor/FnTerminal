# Uniterm 界面提取记录

上游项目：[ys-ll/uniterm](https://github.com/ys-ll/uniterm)。采用 Apache-2.0，提交 `071df19012a0b4ba26404a92d204a90b327eff55`。源码归档根目录与 API 返回的提交前缀一致。

实际复用其 Vue 组件，不是仅引用名字或仿制截图：

- AppHeader：顶部布局、工具按钮与菜单样式；移除 Wails 窗口/AI/桌面功能，接上 fnOS 操作。
- TabsList：标签栏、溢出列表、滚轮滚动、拖拽重排；只保留终端标签拖拽。
- TabItem：标签皮肤、编辑名称、锁定、复制/重连/导出/关闭菜单；接入应用本地的标签状态。
- Menu、MenuItem、MenuDivider、MenuSubmenu：现成的菜单定位、子菜单与样式；修正部分图标尺寸的 TS 类型。
- MobileKeyBar、mobileCtrlKey：原有触摸快捷键和一次性 Ctrl；增加 Ctrl+C/Ctrl+D/Enter，修正 Ctrl+非方向键产生 undefined 的问题。
- BaseTerminal：终端区域、搜索和右键布局/样式；桌面 Wails 会话调用替换为认证后的 WebSocket、输出 ACK 和 HTTP 续期。
- useTerminal：原项目终端配色目录；仅提取纯主题函数。
- style.css、字体定义、中文菜单词条：保留上游视觉体系，离线内置 JetBrains Mono 字体。

FnTerminal 的 Vue 装配、标签存储、设置存储及 WebSocket transport 是 fnOS 平台适配代码。**没有宣称移植 Uniterm 的 SSH 管理器、AI、SFTP、数据库或其它30多种协议。** NAS 用户权限通过可编辑 SSH 连接进行二次认证，`nas-login` 为诊断命令入口。

完整原文件保存在 `third_party/uniterm/original/`；UPSTREAM.json 保存来源/归档 SHA256/每个原文件 SHA256；MODIFICATIONS.json 列出有改动的文件，改动文件顶行含注明。`python3 scripts/check-upstream.py` 校验原始文件及改动声明。许可证、上游第三方声明和 NOTICE 随 FPK 分发。

调研过 xTerminal 官网，但未发现公开的源代码/开源许可入口，因此没有抓取它的商业程序或把它称为可复用开源代码。Electerm 的未采用组件已移除。
