# 移动端视觉 QA

## Messages

**对比目标**

- source visual truth path: `/var/folders/tt/nrdb7rh94p3dkvnpvcgwddb80000gn/T/codex-clipboard-d24ebb50-3b2a-4fa6-8285-bb5c98116394.png`
- implementation screenshot path: `apps/mobile/packages/forum_app/test/golden/golden/pages/messages_page.png`
- 新建私信 implementation screenshot path: `apps/mobile/packages/forum_app/test/golden/golden/pages/messages_new_chat.png`
- iOS 键盘 implementation screenshot path: `/tmp/yourtj-messages-keyboard.png`
- combined comparison path: `/tmp/yourtj-messages-design-qa.png`
- viewport: Flutter golden `390 × 844` CSS px，deviceScaleFactor `1`；实机复核为 iPhone 17 Pro / iOS 26.3。
- source pixels: `3138 × 2480`；implementation pixels: `390 × 844`。
- density normalization: 桌面 source 等比缩放到 `1068 × 844`，移动端 implementation 保持 `390 × 844`，合成一张 `1458 × 844` 对比图。由于 source 是桌面三栏、implementation 是移动端单栏，这不是逐像素克隆；只比较信息层级、密度、语义顺序、组件语言和视觉 token，避免对响应式结构做伪精确判断。
- state: source 为有会话且打开对话；主 implementation 为有两条会话的列表态；另复核空会话态、新建私信态、搜索聚焦态和软件键盘弹出态。

**Findings**

- 当前没有未解决的 P0 / P1 / P2 差异。
- 字体与排版：移动端保持 Web 的粗标题、14px 会话名、弱化时间与预览层级；未把桌面三栏字号直接缩小搬运。中英文标题、搜索占位和空状态文案没有截断。
- 间距与布局节奏：顶栏、搜索区、会话行使用紧凑的 56/44/64px 节奏；分隔线、8px 圆角与白色内容面映射现有 Gf/TDesign token。移动端采用单列进入对话，这是对桌面三栏的预期响应式收敛。
- 色彩与 token：白色主画布、低对比边线、primary 蓝色选中/CTA、弱化 meta 文本与 Web 一致；新建私信 scrim 和面板保持同一中性色体系。
- 图像与资产：实现继续使用 API 头像 URL 和 `GfAvatar`，没有用 emoji、手绘 SVG 或占位图替代产品资产。golden 在离线环境显示组件自带的头像 fallback，属于确定性测试约束，不是生产态资源替换。
- 文案与内容：`Messages / 私信`、会话搜索、新建私信、私信对话与首条消息提示均与 Web 语义一致；不再用空字符串创建一条伪消息。
- 交互与可用性：新建私信、用户搜索、会话搜索、打开已有会话、首条真实消息创建会话、空输入禁用发送、软件键盘避让均完成实测或回归测试。

**Focused Region Comparison**

- 对比图中重点检查了 Web 左侧会话列表与移动端上半屏：标题/新建入口、搜索框、头像、未读点、姓名、时间和消息预览的顺序与层级一致。
- 新建私信弹层在 source 截图中没有对应打开态，因此不做像素级匹配；其 48px 标题行、44px 搜索框、40px 头像行和无箭头列表来自同仓库 `apps/gooseforum/resource/src/site/pages/MessagesPage.vue` 的现有结构，并用 golden 与 iOS 键盘截图复核。
- 对话正文未单独生成 golden；代码与 live simulator 已检查左右头像、气泡、私信副标题、空会话提示和底部输入器。该部分的 Web source 在同一张 source 截图中足够清晰，无需再裁切局部图。

**Comparison History**

1. 第一轮发现：
   - P0：TDesign `TPopup` 内直接承载 Material 输入组件，点击 New message 报 `No Material widget found`。
   - P1：弹层高度固定且标题/列表过重，和 Web 紧凑私信弹层不一致。
   - P1：移动端会话页只有松散列表或灰色空图标，缺少 Web 的搜索、会话预览层级与明确空状态 CTA。
   - P2：对话页缺少自己的头像、私信副标题和首条消息语义；选择新联系人会先发送空消息。
   - P2：普通底部 popup 不跟随 iOS `viewInsets`，软件键盘可能盖住搜索框。
2. 已实施修复：
   - 为通用 TDesign popup 补 Material 宿主；新建私信使用 keyboard-aware、scroll-controlled sheet。
   - 弹层改为内容感知高度、紧凑标题/搜索/用户行；移除无意义箭头。
   - 会话列表补搜索、清晰空状态与 CTA；对话页补双侧头像、Web 式气泡、私信副标题和首条消息提示。
   - 新联系人先进入本地会话，只有第一条真实消息才调用服务端创建会话。
   - 输入器在空内容时禁用发送，并用 iOS 软件键盘实测面板上移。
3. 复核证据：
   - `/tmp/yourtj-messages-design-qa.png`：Web source 与最新会话列表 golden 同图比较。
   - `apps/mobile/packages/forum_app/test/golden/golden/pages/messages_new_chat.png`：最新新建私信态。
   - `/tmp/yourtj-messages-keyboard.png`：iPhone 17 Pro 软件键盘弹出后搜索框和空状态仍可见，无溢出。
   - messages golden 比对、Messages 行为测试、UI Kit bottom-sheet/输入组件测试全部通过。

**Open Questions**

- 无阻塞问题。桌面 source 与移动端 implementation 的列数差异是明确的响应式设计选择。

**Implementation Checklist**

- [x] 对齐 Web 会话列表视觉层级。
- [x] 对齐 Web 新建私信弹层的密度与组件语义。
- [x] 修复 Material 宿主错误。
- [x] 修复 iOS 键盘遮挡风险。
- [x] 验证真实首条消息创建语义。
- [x] 完成 golden、行为、组件与静态分析检查。

**Follow-up Polish**

- P3：后续接入稳定的测试头像资源后，可让 golden 更接近 Web 截图中的真实头像观感；不影响当前生产路径。

## Home

**对比目标**

- source visual truth path: `/var/folders/tt/nrdb7rh94p3dkvnpvcgwddb80000gn/T/TemporaryItems/NSIRD_screencaptureui_aZ1SyU/Screenshot 2026-08-08 at 18.09.00.png`
- implementation golden paths: `apps/mobile/packages/forum_app/test/golden/golden/pages/home_page.png`、`apps/mobile/packages/forum_app/test/golden/golden/pages/home_page_list.png`
- iOS implementation screenshot path: `/tmp/yourtj-home-aligned.jpeg`
- combined comparison paths: `/tmp/yourtj-home-design-qa-full.png`、`/tmp/yourtj-home-design-qa-focus.png`
- viewport: Flutter golden `390 × 844` CSS px，deviceScaleFactor `1`；实机复核为 iPhone 17 Pro / iOS 26.3。
- source pixels: `3248 × 1932`；implementation pixels: `390 × 844`；iOS simulator window capture: `435 × 929`。
- density normalization: full comparison 将桌面 source 等比缩放至 `780px` 宽并与 `390 × 844` 移动端 golden 放入同一画布；focused comparison 将 Web 内容区与移动端上半屏统一到 `500px` 高。source 是桌面布局、implementation 是移动端布局，因此比较信息层级、视觉 token、密度和响应式映射，不把桌面宽度误当作移动端逐像素约束。
- state: source 与 implementation 均为 Latest + Cards 首页；另对 List 状态、Cards 状态切换和底部 Publish 入口做 iOS 实机复核。

**Findings**

- 当前没有未解决的 P0 / P1 / P2 差异。
- 字体与排版：Latest/Hot/Popular 保持 Web 的主筛选层级，卡片标题、摘要、作者和 meta 信息使用现有 Gf/TDesign 字体 token；移动端缩短内容宽度但不改变语义顺序。
- 间距与布局节奏：筛选 tabs 与 List/Cards 胶囊合并为单行，胶囊靠右；外壳总高实测 `32px`、按钮高 `28px`，与 Web 紧凑控制区的密度一致。顶部工具栏垂直 padding 收紧为 `7px`，不会挤压首张卡片。
- 色彩与 token：帖子卡片内容面明确使用 `base100` 纯白，页面背景保留浅灰层次；边线、阴影、primary 蓝色与 Web 同源 token，不再用更深的卡片底色制造无必要层级。
- 图像与资产：顶部继续使用项目真实 YourTJHub logo、搜索/月亮/用户图标库和 API 头像；未新增 emoji、手绘 SVG 或替代资产。
- 文案与入口：移除工具栏内重复的 New topic；发布入口只保留底部中央 Publish，并由 pencil 改为加号，符合“新建”而非“编辑”的动作语义。
- 交互与可用性：Latest/Hot/Popular 与 List/Cards 都保持可点击；视图模式持久化测试通过，iOS 实机已在 List/Cards 间切换并最终停留在 Cards 状态。

**Focused Region Comparison**

- `/tmp/yourtj-home-design-qa-focus.png` 同图检查了 Web 主筛选/视图切换/首张帖子与移动端顶部/工具栏/首张卡片。移动端按用户指定把胶囊移到右侧，并删去 Web 桌面端的 New topic，因为移动端已有底部 Publish 主入口。
- 卡片白色内容面、12px 左右内容边距、紧凑筛选行和浅灰页面画布在 focused comparison 中层次清楚；没有文字裁切、错误圆角、异常边框或横向溢出。
- `/tmp/yourtj-home-aligned.jpeg` 复核了真实 iOS 安全区、顶部品牌栏、三枚筛选、右侧胶囊、连续帖子卡片和底部导航的完整纵向关系。

**Comparison History**

1. 用户指出的差异：
   - P1：帖子卡片底色比 Web 深，内容面显脏且层级过重。
   - P1：List/Cards 胶囊外壳过高，并与 Latest/Hot/Popular 分成两行。
   - P2：工具栏 New topic 与底部 Publish 重复，占用首页空间。
   - P2：Publish 使用 pencil，更像编辑已有内容而不是新建内容。
2. 已实施修复：
   - emphasized `GfCard` 显式使用 `base100` 纯白；新增 token 回归断言。
   - 胶囊改为 `32px` 总高、`28px` 按钮高，并移到筛选行右侧。
   - 删除首页工具栏 New topic，只保留底部 Publish。
   - Publish 的普通和激活图标都改为 `Icons.add`。
3. 复核证据：
   - full/focused combined comparison 均为 source 与最新 implementation 同图检查。
   - iPhone 17 Pro 实机检查 Cards 与 List 两态；最终 Cards 截图见 `/tmp/yourtj-home-aligned.jpeg`。
   - Home golden、首页行为测试、bottom shell 测试、UI Kit card 测试、全 workspace analyze/test 均通过。

**Open Questions**

- 无阻塞问题。Web 的 New topic 与移动端底部 Publish 是平台导航模式差异；本轮按用户明确要求只保留移动端底部入口。

**Implementation Checklist**

- [x] 帖子卡片恢复纯白内容面。
- [x] Latest/Hot/Popular 与 List/Cards 合并为单行。
- [x] 收紧 List/Cards 胶囊高度并加入尺寸回归测试。
- [x] 移除重复 New topic。
- [x] Publish 图标改为加号。
- [x] 完成 Web/移动端同图 QA、golden、行为测试、静态分析和 iOS 实机复核。

**Follow-up Polish**

- P3：待 API 测试数据包含稳定封面图后，可继续补充多图帖子卡片的移动端 golden；不影响本轮首页层级收口。

final result: passed
