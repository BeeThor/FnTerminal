# 飞牛界面核对与匹配范围（2026-10-10）

直接读取FN Connect未登录入口声明的官方静态资源：

- CSS：https://static2.fnnas.com/connect/assets/BeJQwDfNoNFoedLu.css
- JS：https://static2.fnnas.com/connect/assets/1MPg8Gvv7C7Lrf46.js

公开CSS包含约3013种.semi-*选择器、--semi-*颜色及圆角变量；JS包含React/React DOM许可证注释与React18.3.1版本字符串。可以核实FN Connect这一公开前端使用React与Semi Design，不能据此证明全部fnOS桌面及各系统版本均采用相同库。

受保护的 NAS 桌面未完成实际访问核验。因此完整桌面组件库、定制主题参数与实际匹配效果均仍待真机验证。未读取Cookie、登录令牌或设备私有文件。

FnTerminal继续使用实际提取的Uniterm/Vue终端组件和Element Plus交互实现。引入MIT许可的官方[@douyinfe/semi-theme-default](https://github.com/douyinfe/semi-design/tree/main/packages/semi-theme-default) 2.104.0，仅编译global.scss主题变量；适配色板、14px控件字体、32px控件高度、3/6/12px圆角、菜单、弹窗及浅深色状态。没有引入React运行时，也没有复制飞牛的私有组件或声称已迁移Semi React组件。

生产前端保持全部资源离线；Semi MIT许可进入FPK的licenses/npm目录。初次使用默认浅色；已有浅深色保留，旧深蓝迁移为深色。终端配色独立，画布和边缘使用同一终端背景。

官方[getPlatformConfig](https://developer.fnnas.com/api/platform-config)和[前端调用约定](https://developer.fnnas.com/api/calling)提供主题读取/监听能力，但要求micro_app=true与较新系统支持。本次未改变应用加载模式、最低版本或增加后台API权限，不声称已经自动跟随NAS桌面主题。
