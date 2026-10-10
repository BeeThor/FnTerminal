# FnTerminal 开发与运行说明

**Intel NAS 请安装 `dist/FnTerminal-0.9.0-x86.fpk`。** ARM64 对应 `dist/FnTerminal-0.9.0-arm.fpk`。两个包由飞牛官方 fnpack 1.2.3 生成，包含 Linux 静态 Go 二进制、离线前端、根目录 logo.png 派生图标及第三方许可证。

**0.9.0 已通过隔离 Linux 的真实 SSH/PTY/启停测试与本机 Chrome 流程验证，尚未在真实 fnOS 安装回归。** 本版完善窄窗口、长文本和弹窗滚动布局，保持Semi官方主题与单提示符重连行为；保留连接配置迁移、草稿保护、删除撤销及全部权限边界。详见 [测试结果](TEST_RESULTS.md)。

## 安装

1. 在电脑核对 `dist/SHA256SUMS`，进入 fnOS **应用中心 → 设置 → 手动安装应用**。
2. Intel 选择 **FnTerminal-0.9.0-x86.fpk**；已有旧版时按应用中心升级流程上传0.9.0。
3. 启动后从桌面/应用卡片打开 FnTerminal，访问路径为原 NAS 域名的 `/app/fnterminal`。
4. 使用已登录的 fnOS 管理员账户。首次打开显示 **SSH 连接弹窗**：本机主机填 `127.0.0.1`，端口填写 NAS 实际 SSH 端口（例如 `10022`），用户名填写自己的 NAS 用户名（允许数字开头）。点击连接后，在终端确认指纹、输入密码。应用服务仍以 **fnterminal 专属应用账户，非root** 运行。
5. 下次打开默认自动尝试最近使用的已保存连接；可在连接弹窗关闭“打开应用时自动连接最近使用的配置”，之后每次显示配置弹窗。点击标签栏 **＋** 新建连接，终端菜单/标签右键可 **编辑当前 SSH 连接**。保存配置不改变活动会话，“保存并重新连接”只结束当前标签的旧连接。配置保存在当前浏览器，仅含名称、主机、端口和用户名，不保存密码。新版左侧连接列表支持按名称、地址、端口和用户名搜索，选中后在右侧编辑，保存反馈显示具体目标。
6. **高级诊断 → 独立应用账户 Shell** 仅供应用目录诊断；它不是日常 NAS 管理入口。
7. 若终端断连，可直接点击“编辑连接”修正主机/端口，或在终端菜单打开 **连接诊断与日志**，可查看具体阶段和下载诊断。若应用本身启动失败，请保留应用中心显示的具体错误和服务日志：`/var/apps/fnterminal/var/fnterminal.log`。也可运行只读诊断脚本 [diagnose-fnos.sh](../scripts/diagnose-fnos.sh)。不要通过改成root或Socket 666/777来“修复”。

安装、升级、卸载、重启及安全验证步骤见 [FNOS_TEST_PLAN.md](FNOS_TEST_PLAN.md)。应用中心卸载时的数据保留/删除行为需真机验证，本包不自行递归删除用户数据。

## 界面来源

界面**实际提取自 [ys-ll/uniterm](https://github.com/ys-ll/uniterm)**，Apache-2.0 许可，上游提交 `071df19012a0b4ba26404a92d204a90b327eff55`。

复用其 AppHeader、TabsList、TabItem、Menu 组件、MobileKeyBar、终端区域/搜索样式、主题目录、中文词条和 JetBrains Mono 字体，替换 Wails 桌面绑定以适配 fnOS。原始源码、哈希、改动记录与许可均保存在 `third_party/uniterm/`，见 [提取说明](UPSTREAM.md)。本次不包含上游AI、数据库、SFTP或桌面窗口控制等未移植功能。

- 同时最多6个标签；支持拖拽排序、双击重命名、锁定、右键复制会话/重连/关闭其它标签。
- 终端搜索、选区复制、文字输入/粘贴框、导出、浅色/深色UI及多种终端配色、字号设置。
- 移动快捷键含Esc、Tab、Ctrl、方向键、Ctrl+C/Ctrl+D/Enter；中文使用xterm输入法处理，也可在文字框中先完成输入再发送。
- 粘贴框不会自动添加回车；系统直接粘贴保留xterm的括号粘贴模式。剪贴板权限不可用时提供手动粘贴/复制入口。
- 重新连接会创建新Shell，不恢复已结束的进程。全部管理员共享应用账户及HOME，不隔离彼此的应用文件。

开发检查生成的原始日志与截图保留在本地 `test-results/`，默认不进入公开源码归档。公开预览见 [连接弹窗](images/connection-desktop.png) 和 [手机布局](images/connection-mobile.png)，均为本机浏览器测试，不是fnOS实机截图。

## 飞牛UI匹配范围

直接检查飞牛FN Connect公开静态资源，确认其使用React18.3.1与Semi Design（存在.semi-*组件选择器和--semi-*主题变量）。完整fnOS桌面尚未完成真机核对，因此**尚未确认完整fnOS桌面的组件库及具体主题定制**，不能把FN Connect结果冒充全部桌面的事实。[核对证据](FNOS_UI_REVIEW.md)。

FnTerminal保留实际提取的Uniterm/Vue终端组件与Element Plus交互组件，引入MIT许可的官方Semi主题包2.104.0，统一颜色、字体、32px控件、3/6/12px圆角、弹窗、菜单及浅深色状态。这是同源设计变量和视觉规范的匹配，**并非迁移到Semi React组件**。首次默认浅色，已有浅深色主题保持（旧深蓝设置迁移为深色）；终端配色独立，画布与边缘保持同色。

重连结束旧会话并清空旧屏幕、输入模式及光标状态；原输出需重连前通过“导出终端内容”保存。旧的异步输出解析完成后才重置，新会话不会接在旧提示符后；不改写Shell真实输出。尺寸无变化时不重复通知PTY，减少无效重绘。

## 窄窗口与长文本

弹窗固定标题、关闭按钮和底部操作区，正文独立滚动；短窗口和横屏无需滚动到最底部才能保存或连接。窄屏默认折叠已保存连接列表，点击展开后仍可搜索、导入导出和选择。320px窗口将端口与用户名纵向排列，主要连接操作占完整一行。

长名称在连接卡片中最多显示两行，完整内容可从悬停提示或编辑输入框查看；长地址/用户名按单行省略，编辑输入框可水平移动光标。保存反馈、错误、事件及原始诊断会自动换行，超长正文可滚动。底部目标地址省略显示，完整地址保留在悬停提示；断连操作按钮独立排列，终端搜索栏与标签菜单也按窗口限制宽度。

本机Chrome覆盖320px窄屏、320px低高度、横屏和常见桌面尺寸，检查正文无横向溢出、标题/按钮在视口内且可点击；模拟360px可见区域下也检查了菜单和弹窗边界；真实fnOS窗口和手机键盘仍待验证。

## 配置迁移与误操作保护

连接弹窗的导入/导出按钮使用本机JSON文件，不上传到外部服务。备份仅含连接名称、主机、端口和用户名，不含密码、令牌、私钥、配置ID或登录态。导入采用合并模式：原有连接保留，同一规范化地址/端口/用户名的连接跳过；文件最多64KiB、合并后最多30条。未知字段、凭据、无效地址和超限文件整体拒绝，校验/存储失败不改已有配置。

切换连接、新建配置、删除或关闭弹窗时，若表单有未保存修改，会提供“继续编辑 / 放弃修改”。保存或有意使用临时连接可正常离开。删除最近一条配置后，在当前弹窗内可撤销；退出弹窗后撤销记录清空，不影响现有终端会话。

诊断窗口显示当前连接原因、目标、服务版本、平台、账户、活动会话和最近事件；原始JSON默认折叠，仍支持下载。刷新有10秒超时，关闭后取消读取，HTTP拒绝与异常响应明确提示，不展示伪造的健康状态。

## 登录态与权限

生产程序仅监听 `${TRIM_APPDEST}/app.sock`，无TCP终端端口、Docker或额外系统资源。`packaging/config/privilege` 固定 `run-as=package`、账号/组 `fnterminal`，不加入sudo/docker组。

HTTP、前端资产、WS、会话续期均先校验Linux `SO_PEERCRED` 对端UID，再严格检查唯一的数字 `X-Trim-Userid` 与 `X-Trim-Isadmin: true`。UI里的 `allUsers=false` 仅控制入口展示。WS和POST续期还检查同源Origin；WS不能从消息里修改用户身份。

默认Socket0600且仅信任UID0。**官方文档未明确网关进程身份，这一默认值仍待真机确认。** 若网关实际为非root，需管理员在独立测试机核实其真实UID/GID后，在 `${TRIM_PKGETC}/gateway.conf` 写入一行 `UID GID`（例如由实际核实值填写，不提供猜测数值），重启并复测。配置GID后Socket0660仍只接受可信UID；应用UID自身不可设为网关。无法验证时保持失败关闭。

0775的安装目录只有在root/包用户所有、当前专属组独占且可从/etc/passwd和/etc/group核验时才接受。共享组、外部属主、world-write仍拒绝。生命周期脚本若由系统root调用，会通过runuser切回固定fnterminal账户；Go服务和PTY始终拒绝root。

前端每20秒通过网关重新POST续期，服务端租约60秒。NAS注销/撤销管理员身份后无法续期，最长约60秒回收；具体网关行为待实机确认。浏览器后台节流、断网可能导致会话提前关闭。

## 会话管理

| 控制项 | 默认 |
|---|---|
| 全局/每用户PTY数 | 12 / 6 |
| 创建会话 | 每用户10次突发、约10次/分钟；全局30次突发、约30次/分钟 |
| 无终端输入空闲超时 | 15分钟（输出、ping、续期不会延长） |
| 会话最大寿命 | 8小时 |
| 登录态租约 | 60秒 |
| 输入 | 单帧16KiB，64KiB/s、128KiB突发；帧120/s、240突发 |
| 输出 | 1MiB/s、2MiB突发；未ACK窗口约128KiB，通过PTY背压限流 |
| WS保活/网络写入 | ping约20秒、pong45秒、写入10秒 |

连接采用短期CSRF准备凭据与绑定UID/Origin的一次性WS票据，能够适配代理改写Host；Shell真正启动后才显示已连接。

Linux每个PTY拥有独立subreaper监督器，负责Shell及后台/双重fork后代的退出回收；服务TERM关闭所有WS并等待回收，再清理被接管的后代。界面诊断记录最近200条本用户及服务公共事件，支持下载；结构化元信息也写入应用日志。日志仅记录UID、随机会话标识、PID、起止时间与原因，不记录按键、密码或终端输出。日志在服务重新启动时超过5MiB轮换一次。

这是管理员使用的真实Shell，专属账号并非容器或恶意程序沙箱；管理员主动攻击监督器、SIGKILL、内核不可中断任务或系统断电等情况下，不作绝对回收保证。Linux监督器、真实UDS网关、SSH二次认证与生命周期启停已在隔离Linux验证；真实NAS重启/卸载仍待验证。

默认入口直接使用 SSH 连接弹窗，经 NAS sshd 二次认证进入对应用户会话。服务账户是 Web/PTY 的运行身份；只有高级诊断 Shell 才显示应用账户提示符。应用不会自动开启 SSH、不默认提权、不存 NAS 密码。首次连接要求确认主机指纹，已知密钥变化拒绝。SSH 登录后权限由目标账户控制。

连接配置可编辑名称、主机、端口和用户名；本机默认 `127.0.0.1:22`，自定义10022等端口贯穿预检、首次连接、重连和下次自动连接。错误提示包含实际目标，不再把端口不可达直接断言为“服务未开启”。如在高级诊断 Shell 中手动使用命令，正确语法是 `ssh -p 10022 用户名@主机IP`，不能写 `用户名@主机IP:10022`。`nas-login 10022` 同样支持自定义端口。隔离 Linux 已验证22/10022和数字用户名；实际 NAS SSH兼容性仍待验证。

0.9.0恢复损坏的浏览器配置；存储写入不可用时，设置仅在当前页面生效，“连接”仍可使用临时配置，明确显示未保存。保存/删除只有在写入成功后才更新连接列表，不会显示虚假的保存结果。主机字段提前拒绝`localhost:10022`等混写，保留有效IPv4、IPv6和域名支持。

空闲、寿命、登录态过期和服务停止分别显示原因；续期网络错误与明确鉴权拒绝分开提示，均关闭会话。SSH标签显示“终端通道已连接”，指纹确认和身份认证仍需在终端完成。SSH层每15秒无服务器数据时发送保活，连续3次无响应后按OpenSSH规则断开，不延长15分钟应用空闲超时。[OpenSSH官方参数说明](https://man.openbsd.org/ssh_config#ServerAliveInterval)。

移动快捷键触摸轻点在松手时发送，滑动只滚动、不输入；按钮扩大并增加安全区留白。Chrome真实触摸事件与模拟可见视口变化已检查；真实手机输入法和软键盘体验仍待验证。

删除“关于 FnTerminal”菜单，第三方许可证继续随包分发；图标保留根目录logo.png的透明通道。

## 源码与构建

- Go 1.22+（本次实际构建Go1.27.1）；Node.js 20.19+/22.12+（本次Node26.10.0）；Python3 + Pillow。构建机器需联网，NAS运行不依赖Node/npm/Python。
- Go/npm依赖版本及校验和分别锁在go.sum、package-lock.json。
- 图标来自根目录logo.png，构建中生成保留透明背景的64px/256px应用中心与桌面图标。

```bash
python3 -m pip install -r requirements-build.txt
npm ci --ignore-scripts --no-audit --no-fund
./scripts/download-fnpack.sh
./scripts/check.sh
./scripts/build-fpk.sh x86
./scripts/build-fpk.sh arm
python3 scripts/package-source.py
# 完整本机发布检查（需已安装Playwright Chromium；或设置FNTERMINAL_CHROME=system）
./scripts/release.sh
```

macOS沙箱限制临时监听时，Go测试/浏览器测试需要获准使用本机临时Socket/HTTP端口。这仅在测试二进制中存在，生产程序不包含TCP测试入口。

可选实际浏览器测试：

```bash
npm run build
mkdir -p .tools
go test -c -o .tools/browser-tests ./cmd/fnterminal
npx playwright install chromium
npm run test:browser
# 已安装Chrome时也可 FNTERMINAL_CHROME=system npm run test:browser
```

`tests/browser.mjs` 自动启动本地测试夹具并在结束时停止。测试夹具注入的是模拟网关身份，不证明NAS认证；只验证真实PTY与界面行为。

`build-fpk.sh` 为每个架构使用独立staging目录，不改源manifest架构；使用官方fnpack生成包后核对app.tgz的MD5、静态ELF64架构、必需资源、JS/CSS/字体引用、执行位、图标、权限和网关入口，并输出逐包verification.json及SHA256SUMS。fnpack下载记录SHA256，官方未提供独立签名校验，本地哈希不冒充官方认证。

自动化CI见 [.github/workflows/build.yml](../.github/workflows/build.yml)，包含Linux专用对端身份/脱离进程回收测试与两个架构打包；本次尚未远程运行CI。源码梳理与安全/产品修正见 [ARCHITECTURE_REVIEW.md](ARCHITECTURE_REVIEW.md)。

`release.sh`依次运行Go/前端检查、真实浏览器回归、可用的Apple Silicon内存Linux验证、两个FPK构建、版本/哈希匹配报告及源码归档；任何一步失败即停止。报告始终保留fnOS/真实手机待验证状态，不用本机测试替代安装验收。

可在Apple Silicon macOS用 `./scripts/test-linux-vm.sh` 重放独立Linux验证；该脚本只创建内存测试机，不访问NAS。


详细结果：[TEST_RESULTS.md](TEST_RESULTS.md)，官方核对：[OFFICIAL_REVIEW.md](OFFICIAL_REVIEW.md)，变化：[CHANGELOG.md](../CHANGELOG.md)。
