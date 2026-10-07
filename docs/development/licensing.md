# 仓库许可说明

> Doc type: development guide
>
> Status: Active
>
> Owner: Platform maintainers
>
> Last verified: 2026-10-07

本仓库按作品来源和类型分别授权。根目录 [LICENSE](../../LICENSE) 提供 GPLv3 正文；本页说明它与其他许可证的边界。

## 许可范围

| 内容 | 许可证 | 范围与要求 |
|---|---|---|
| YourTJ 原创软件和仓库文档 | GPL-3.0-only | 包括论坛新增代码、移动应用、状态站、共享模块和构建脚本。第三方文件按其许可证处理。 |
| GooseForum 上游原创内容 | MIT | 保留 [`apps/gooseforum/LICENSE`](../../apps/gooseforum/LICENSE) 和已有的版权、许可证声明。 |
| YourTJ 对 GooseForum 的原创改动 | GPL-3.0-only | 只许可 YourTJ 持有或已获授权的改动。组合分发时履行 GPLv3 义务，上游 MIT 授权仍有效。 |
| Wiki 页面与课程评价数据 | CC BY-NC-SA 4.0 | 适用于 YourTJ 持有或已获授权的原创内容，不改变第三方作品、商标、隐私或人格权。 |
| 其他用户内容 | 作者保留权利 | 论坛话题、回复、私信、头像和附件不因根目录许可证自动取得公开再利用许可。 |
| YourTJ 品牌名称、商标和标识 | 不在软件或数据许可范围内 | GPL 与 CC 许可均不授权商标使用、品牌背书或对外代表项目。 |
| 第三方组件、素材和商标 | 各自声明的许可证或权利 | 保留随附的 `LICENSE`、`NOTICE`、来源和署名，不由本页中的 GPL 或 CC 许可覆盖。 |

许可日期追溯至 **2026-08-06**，即 Hub 仓库的初始化日期。GitHub [仓库元数据](https://api.github.com/repos/YourTongji/YourTJ-Hub)和[首个提交](https://github.com/YourTongji/YourTJ-Hub/commit/808a0eae9caf3260f30c9e491525b24bed2c2c9b)均记录该日期。
权利人已书面授权将历史原创软件和数据纳入上述许可。该日期不表示后来创建的 Wiki 仓库或页面在当日已经存在。
在该日期之后创作的作品，自作品产生后按其范围适用许可。

Wiki 内容的来源是公开的 [`YourTJ-Wiki`](https://github.com/YourTongji/YourTJ-Wiki) 仓库；Hub 将其作为只读投影展示。
本项目许可说明同样适用于经授权并由 Hub 展示的 Wiki 页面和课程评价数据。

## 许可兼容与分发

MIT 是宽松许可证，可与 GPLv3 组合。分发包含 GooseForum 上游代码的修改版本时，保留原 MIT 声明，并对 YourTJ 原创部分履行 GPL-3.0-only 条款。
GPL 授权不会撤销上游作者已经提供的 MIT 权利。

CC BY-NC-SA 4.0 与 GPL 软件应作为独立作品分发。不要把受 CC BY-NC-SA 约束的数据改编后作为 GPL 软件源代码的一部分发布。
单独再发布 Wiki 或课评数据时，须注明作者和来源、提供许可链接、标明修改，并以相同许可分享改编内容；不得用于商业目的。
匿名评价不得暴露账号身份，署名应使用内容作者选择的昵称或匿名标识。
Creative Commons 当前没有指定与 CC BY-NC-SA 4.0 兼容的非 CC 许可证；如将数据改编或合并进其他许可作品，须先取得许可人额外授权。
平台若加入广告、收费或其他商业用途，也须重新评估该数据许可证的非商业条件。

CC 4.0 许可覆盖著作权和适用的数据库权利，不自动授予商标、隐私、人格权或第三方内容权利。
若数据包含他人作品、个人信息或可识别身份，发布方仍须取得所需授权并遵守隐私规则。
CC BY-NC-SA 4.0 不可撤销，作者应在提交前确认许可范围。

GPLv3 允许收费分发，但交付 GPL 软件时，须提供对应版本的完整源代码和许可证文本。
论坛的 Linux、macOS 和 Windows 二进制归档随附 `licenses/GPL-3.0-only/LICENSE`（根目录 GPLv3 正文）、`LICENSE`（上游 MIT 正文）
和 [LICENSES.md](../../apps/gooseforum/LICENSES.md)（分发范围说明）；上游 MIT 文件不代表整个修改版按 MIT 授权。
对 GPLv3 所定义的 User Product，还须提供安装修改版本所需的信息。
仓库内的 iOS 发布流程会提交 App Store 和 TestFlight 版本。Apple [标准 EULA](https://www.apple.com/legal/internet-services/itunes/dev/stdeula/) 默认限制应用的转让、修改和再分发，
同时允许开源组件依照其许可证行使权利；开发者提交自定义 EULA 时，还须满足 Apple 的最低条款。
`Current`：维护者已核对 App Store Connect 的 License Agreement，当前应用采用 Apple 标准 EULA。
`Partial`：GPLv3 与实际 iOS 分发条件的兼容性仍需核验。标准 EULA 的开源组件例外不能单独证明整款应用的分发兼容；
发布者仍须确认终端用户不受限制 GPL 权利的条款约束，并按 GPLv3 提供对应源代码及满足适用条件时所需的安装信息。
若无法确认，应暂停该 GPL 应用的 App Store/TestFlight 发布并寻求法律审查。

服务条款默认文案说明 Wiki 和课程评价数据的 CC BY-NC-SA 4.0 许可。管理员可以在站点设置中另行配置服务条款；
部署者须确认线上条款与本页一致。已保存的自定义条款继续使用数据库中的文案；未保存自定义条款的现有站点
在升级后直接使用新的内嵌默认文案，损坏的配置 JSON 也会回退到默认值。默认文案不只在新建或重置站点时生效。

## 第三方许可清单

以下是仓库内明确随附许可证的第三方内容。未复制进仓库的依赖包继续适用各自上游许可证。

| 路径 | 已声明的许可证 |
|---|---|
| `apps/mobile/packages/ui_kit/assets/LICENSE-lucide.txt` | ISC；Feather 衍生图标按文件内 MIT 声明授权 |
| `apps/mobile/packages/ui_kit/assets/LICENSE-reicon.txt` | MIT |
| `apps/mobile/packages/ui_kit/assets/LICENSE-simple-icons.txt` | CC0 1.0 Universal |
| `apps/mobile/third_party/home_widget/LICENSE` | BSD 3-Clause |
| `apps/gooseforum/app/console/stickerpresets/preset_stickers/LICENSE-EmojiPackage.txt` | Apache-2.0 |
| `apps/gooseforum/app/console/stickerpresets/preset_stickers/LICENSE-WXMemeStickers.txt` | MIT |
| `apps/gooseforum/app/console/stickerpresets/preset_stickers/LICENSE-flowerhd.txt` | CC BY 4.0 |

[贴纸资源 NOTICE.md](../../apps/gooseforum/app/console/stickerpresets/preset_stickers/NOTICE.md)
和上述许可证文件列明资源来源、署名和使用范围。新增依赖前应检查其直接与传递依赖许可证，
并确认没有把不兼容的代码或数据合并进 GPL 发行物。

## 参考资料

- [GNU GPLv3](https://www.gnu.org/licenses/gpl-3.0.en.html) 与 [GNU 许可证兼容说明](https://www.gnu.org/licenses/license-compatibility.en.html)。
- [CC BY-NC-SA 4.0 法律文本](https://creativecommons.org/licenses/by-nc-sa/4.0/legalcode.en) 与 [兼容许可证清单](https://creativecommons.org/compatible-licenses/)。
- [Creative Commons 软件与数据库 FAQ](https://creativecommons.org/faq/index.html#can-i-apply-a-creative-commons-license-to-software)。
- [Apple Developer Program License Agreement](https://developer.apple.com/support/terms/apple-developer-program-license-agreement/) 与 [App Store 标准 EULA](https://www.apple.com/legal/internet-services/itunes/dev/stdeula/)。
- 本文格式遵循[中文技术文档写作说明](https://github.com/ruanyf/document-style-guide)。
