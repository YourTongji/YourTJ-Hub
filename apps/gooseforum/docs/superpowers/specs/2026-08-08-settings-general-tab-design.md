# 设置页「通用」选项卡 — 设计规格

- 日期：2026-08-08
- 分支：`feat/settings-general-tab`
- 关联 Issue：#48（自定义字体）、#49（鼠标点击动画）
- 状态：待评审

## 1. 背景与目标

在用户设置页新增 **「通用」** 选项卡，提供两类设备级偏好设置，按用户确认的范围：

1. **自定义字体**（Issue #48）：字体大小（全局缩放）+ 字体样式选择（预设 + 自定义输入）。
2. **鼠标点击动画**（Issue #49）：全局涟漪效果开关。

所有设置**即时生效**（无需刷新）、**持久化到 localStorage**（设备级，与现有隐私设置、主题一致），范围限定在**站点前端**，不涉及 admin 后台。

## 2. 非目标（明确不做）

- 不做账号级/跨设备同步（数据库模型 + 迁移 + API），留作后续。
- 不引入 webfont 嵌入（go:embed），仅使用系统本地字体。
- 涟漪不覆盖 admin 后台（独立 Vue 应用 + 独立样式体系）。
- 不改变现有设置页其它选项卡行为。

## 3. 需求要点

| # | 需求 | 验收标准 |
|---|---|---|
| R1 | 设置页顶部出现「通用」选项卡 | URL `/settings?tab=general` 可直达；导航高亮正确 |
| R2 | 字体大小可调（14–20px，步进 1，默认 16） | 调滑块即改 `html` 根字号，全站文本+间距即时缩放 |
| R3 | 字体样式可选预设（系统默认/宋体/楷体/黑体） | 选择后 `body` 字族立即切换 |
| R4 | 支持输入自定义本地字体 | 下拉选「自定义…」出现文本输入，任意字体名立即生效 |
| R5 | 鼠标点击动画可开关（默认关） | 开启后点击可交互元素出现涟漪；关闭后无 |
| R6 | 「恢复默认」按钮 | 一键重置字号/字族/点击动画为默认值 |
| R7 | 设置持久化 | 刷新后保持；所有改动即时生效 |
| R8 | 尊重 `prefers-reduced-motion` | 系统减少动效时，即使开关开启也不生成涟漪 |
| R9 | 四语言 i18n（zh/en/ja/it） | 新增文案在四种语言下均有键值 |

## 4. 设计

### 4.1 通用选项卡接入（前后端同步）

现有 tab 由「服务端 payload 渲染导航 + 客户端 `tabKeys`/`v-show` 区块」两段构成，需同步修改：

**Go 服务端**
- `apps/gooseforum/app/http/controllers/forum/payload.go` → `buildSettingsPageProps`（约 2490–2496 行）的 `Tabs` 数组追加：
  ```go
  {Key: "general", URL: "/settings?tab=general"},
  ```
- 设置页 props 不在 openapi.yaml 契约覆盖内，无需契约/代码生成改动。

**Vue 客户端**（`resource/src/site/pages/SettingsPage.vue`）
- `tabKeys`（约 61 行）追加 `'general'`。
- `settingsTabLabel()`（约 273–280 行）追加 `if (key === 'general') return t('settings.tabs.general')`。
- 模板在现有各 tab `<section>` 后追加 `<section v-show="activeTab === 'general'">`，内部含「字体」「点击动画」两个区块，布局仿隐私区块（约 1123–1148 行：`SectionHeader` + `max-w-2xl divide-y` 列表）。

**i18n**（`resource/src/locales/{zh,en,ja,it}.ts`）
- `settings.tabs.general`：通用 / General / 一般 / Generale。
- 新增 `settings.general.*` 文案（见 §4.4）。

### 4.2 运行时模块 `runtime/appearance-settings.ts`（新建）

镜像 `runtime/site-theme.ts` 的模块化模式，集中管理设备级外观偏好。

```ts
export type FontFamilyPreset = 'system' | 'serif' | 'kai' | 'hei' | 'custom'

export interface AppearanceSettings {
  fontSize: number            // 14–20，默认 16
  fontFamilyPreset: FontFamilyPreset // 默认 'system'
  customFontFamily: string    // preset === 'custom' 时的原始输入，默认 ''
  clickAnimation: boolean     // 默认 false
}
```

导出：
- `DEFAULT_APPEARANCE_SETTINGS` — 不可变默认值。
- `FONT_PRESETS` — 预设字族解析表：
  - `serif`：`"Songti SC", "SimSun", serif`（宋体）
  - `kai`：`"Kaiti SC", "KaiTi", "STKaiti", serif`（楷体）
  - `hei`：`"Heiti SC", "SimHei", "Noto Sans CJK SC", sans-serif`（黑体）
- `normalizeAppearanceSettings(raw: unknown): AppearanceSettings` — **纯函数**，解析/校验/钳制任意输入：
  - `fontSize`：数字且 14–20，否则回落 16；非数字→16。
  - `fontFamilyPreset`：必须是预设键之一，否则 `'system'`。
  - `customFontFamily`：字符串，超长（>200 字符）截断，空白字符串视为无效自定义（回落 system）。
  - `clickAnimation`：必须为 boolean，否则 false。
- `resolveFontFamily(settings): string` — **纯函数**：`system`→`''`；`custom`→`customFontFamily.trim()`；预设→`FONT_PRESETS[preset]`。
- `loadAppearanceSettings(): AppearanceSettings` — 读 `localStorage` 键 `goose-appearance-settings`，经 `normalize` 返回；无/损坏返回默认。
- `applyAppearanceSettings(settings)` — 应用副作用：
  - 字号：`document.documentElement.style.fontSize = fontSize + 'px'`（根字号全局缩放，含文本与 rem 间距）。
  - 字族：`resolveFontFamily` 为空则移除 CSS 变量，否则 `document.documentElement.style.setProperty('--gf-font-family', family)`。
  - 点击动画：写入模块级 `let clickAnimationEnabled` 供 ripple 读取（`isClickAnimationEnabled()`）。
- `applyStoredAppearanceSettings()` — `load` + `apply`。
- `saveAppearanceSettings(settings)` — `apply` + `localStorage.setItem`（try/catch 忽略受限浏览模式）。
- `resetAppearanceSettings()` — 以默认值 `apply` + 清除存储。

**入口接线**：`resource/src/site/main.ts` 在 `applyStoredTheme()`（约 22 行）旁调用 `applyStoredAppearanceSettings()`；同时调用 `installClickRipple()`（见 §4.3）。SPA 导航不改动 `html`/`body` 样式，设置自然跨页面保持。

**CSS 字族变量**：`resource/src/styles/base.css` 第 9 行硬编码字体栈改为
```css
body { font-family: var(--gf-font-family, <原默认栈>); }
```

### 4.3 点击涟漪 `runtime/click-ripple.ts`（新建）

采用**全局委托监听**，不采用 `v-ripple` 元素指令——避免为每个目标元素改动 `position/overflow:hidden`（可能裁剪按钮/卡片内的下拉、浮层），且天然适配开关。

`installClickRipple()`：在 `document` 挂载一个 `pointerdown` 监听（捕获阶段），每次点击按序判定：

1. `!isClickAnimationEnabled()` → 返回。
2. `window.matchMedia('(prefers-reduced-motion: reduce)').matches` → 返回（JS 侧必须同步检查，否则虽被 CSS 禁用仍会创建 DOM 节点）。
3. `event.button !== 0`（排除右键/触摸非主键）→ 返回。
4. `event.target` 非 `Element` → 返回。
5. 用 `closest()` 找命中元素，选择器（常量 `RIPPLE_SELECTOR`，可单测）：
   ```ts
   'button, a[href], [role="button"], [role="menuitem"], [role="switch"], [role="tab"], [role="checkbox"], .gf-menu-item, .gf-icon-button, .gf-tab, .gf-segmented-item, [data-ripple]:not([data-ripple="false"])'
   ```
   排除纯表单控件（range/checkbox/text 已有原生反馈），其余元素可用 `data-ripple` 属性显式纳入。
6. 命中元素为 disabled（`disabled`、`[aria-disabled="true"]`、`.disabled`）→ 返回。

生成涟漪（纯视觉，不触碰目标元素自身样式）：
- `rect = el.getBoundingClientRect()`，按目标矩形创建 `position: fixed` 的 `.gf-ripple` 层，宽高 = rect，`overflow: hidden`，`border-radius` = `getComputedStyle(el).borderRadius`（覆盖圆角裁剪），`pointer-events: none`。
- 内部 `.gf-ripple__wave` 波纹圆：直径 = 矩形对角线 × 2（保证覆盖整元素），圆心定位到点击坐标（相对 rect），`transform: scale(0)` → `scale(1)` 扩散 + 淡出，约 450ms。
- 追加到 `document.body`，`animationend` 后移除；超时兜底 600ms 移除。
- 并发上限：活跃涟漪 > 8 时移除最旧的，防止点击风暴堆积。
- 模块级退出清理：`installClickRipple()` 返回 `() => void` 卸载监听。

**样式**（`resource/src/styles/motion.css` 追加）：
```css
.gf-ripple { position: fixed; z-index: 1000; pointer-events: none; overflow: hidden; contain: strict; }
.gf-ripple__wave {
  position: absolute; border-radius: 9999px;
  background: color-mix(in oklab, var(--gf-color-primary) 22%, transparent);
  transform: scale(0); opacity: .6;
  animation: gf-ripple .45s cubic-bezier(.22, 1, .36, 1) forwards;
}
@keyframes gf-ripple { to { transform: scale(1); opacity: 0; } }
```
- 颜色用 `color-mix` 半透明 `--gf-color-primary`，与现有 design tokens 惯例一致（如 `components.css` 悬停辉光）。
- 缓动沿用全站 `cubic-bezier(0.22, 1, 0.36, 1)`。
- 在现有 `@media (prefers-reduced-motion: reduce)` 块追加 `.gf-ripple__wave { animation-duration: .01ms; }` 作 CSS 侧兜底。

### 4.4 设置页 UI（`settings.general` 区块）

布局仿隐私区块：`SectionHeader`（图标用 `@lucide/vue` 的 `Settings2`）+ `max-w-2xl divide-y divide-line p-4`。

1. **字体大小** 行：
   - 标题 + 描述文案 + 当前值 `{{ fontSize }}px`。
   - `<input type="range" min="14" max="20" step="1">`，`@input` 实时预览（仅 apply），`@change` 持久化（apply + save）。
2. **字体样式** 行：
   - `SiteSelect`（`modelValue` + `options`），选项：系统默认/宋体/楷体/黑体/自定义…。
   - 选中「自定义…」时其下展开 `gf-input` 文本框，绑定 `customFontFamily`，`@input` 实时预览 + 失焦/回车持久化。placeholder：`例如：Noto Serif SC、楷体`。
3. **鼠标点击动画** 行：原生 checkbox（`h-5 w-5 rounded border-line text-primary`），`@change` 即存。
4. **恢复默认** 按钮：`gf-button gf-button-muted`，点击先 `window.confirm(settings.general.resetConfirm)`，确认后调 `resetAppearanceSettings()` 并刷新本地表单 ref；文案用 `settings.general.reset`。

**「即时生效」语义**：所有控件改动立即 `apply`，无需等待保存按钮；持久化时机为 `@change`（滑块/开关/下拉）或失焦（自定义字体文本框）。设置页无全局「保存」按钮，与隐私区块一致。

**i18n 键**（四个语言文件新增，键名按现有 `settings.*` 命名）：
- `settings.tabs.general`
- `settings.general.title` / `settings.general.description`
- `settings.general.fontSize` / `fontSizeDescription` / `fontSizeHint`（缩放提示：会同时影响文本与界面间距）
- `settings.general.fontFamily` / `fontFamilyDescription`
- `settings.general.fontSystem` / `fontSerif` / `fontKai` / `fontHei` / `fontCustom`
- `settings.general.customFontPlaceholder`
- `settings.general.clickAnimation` / `clickAnimationDescription`
- `settings.general.reset` / `resetConfirm`

## 5. 校验与错误处理

- localStorage 读取走 try/catch，损坏 JSON 回落默认值；写入 try/catch 忽略受限浏览模式（与 `site-theme.ts:83-87` 一致）。
- `normalizeAppearanceSettings` 对所有字段钳制/回退，杜绝脏数据进入 DOM。
- 自定义字体字符串做长度上限与 trim，避免注入异常 CSS（CSS 变量值非法时浏览器直接忽略该属性，无安全风险）。
- 涟漪创建全程 try/catch 包住 getComputedStyle/rect，DOM 异常不影响业务。

## 6. 测试计划

项目现状：Vitest 4 已装但未接入 CI（CI 仅 `pnpm typecheck`）；测试文件位于 `resource/test/*.test.ts`，手动 `pnpm exec vitest run`。

### 6.1 单元测试（`resource/test/`，沿用现有风格）
- `appearance-settings.test.ts`：
  - `normalizeAppearanceSettings`：损坏/缺失/越界/非法类型全部回退默认；合法值保持。
  - `resolveFontFamily`：system→''、serif/kai/hei→对应栈、custom 空白→''、custom 有值→trim 结果。
- `click-ripple.test.ts`：
  - `RIPPLE_SELECTOR` 覆盖 `button/a[href]/[role=*]/.gf-*` 关键类。
  - 触发判定纯函数（若抽出）的开关/减少动效/主键/disabled 各分支。
  - 波纹直径计算：`computeWaveDiameter(rect)` 对角线 ×2。

### 6.2 验证命令
```bash
cd apps/gooseforum/resource
pnpm exec vitest run        # 新增单测
pnpm typecheck              # CI 门禁
pnpm build                  # 产出 resource/static/dist
cd apps/gooseforum
go vet ./... && go test ./...
```

### 6.3 手工冒烟（浏览器）
- `/settings?tab=general` 直达通用选项卡；调字号/选字族/开涟漪/恢复默认，均即时生效且刷新保持。
- 开启涟漪后点击按钮/菜单项/导航 tab 出现涟漪；系统减少动效时无涟漪。
- 四语言下文案完整。

## 7. 依赖与风险

- 无新增 npm/Go 依赖（涟漪用原生 DOM + CSS，`@vueuse/core` 可选，优先不引入）。
- 根字号缩放会等比缩放 rem 间距——用户已确认接受该语义（全局缩放）。
- 涟漪层 `z-index: 1000` 需与现有浮层（菜单 `z-30`、弹窗）不冲突——已选用高值；如与全屏遮罩冲突，在实现时复核现有最高 z-index。
- 后端 `payload.go` 改动后需跑 `go vet/test` 确认无契约测试回归。

## 8. 落地顺序

1. `appearance-settings.ts` + `base.css` 字族变量 + `site/main.ts` 接线 + 单测。
2. `click-ripple.ts` + `motion.css` 涟漪样式 + `site/main.ts` 接线 + 单测。
3. 通用选项卡：`payload.go` → `SettingsPage.vue`（tabKeys/label/区块）→ 四语言 i18n。
4. 全量验证 + code review + 用户确认后推送 PR。

---

# 追加：通用选项卡功能扩展（2026-08-08，用户确认）

## 9. 新需求

在既有「通用」选项卡基础上新增（沿用站点前端范围、localStorage 持久化、即时生效）：

1. **自定义 CSS 导入**：粘贴文本域 + 文件上传（本地 .css）两种方式。
2. **字号改为输入框**：去掉滑块，改 `<input type="number">`。
3. **字体分区调控**：界面 / 正文 / 代码三个分区，各自可调字号与字族。

用户已确认：导入方式两者兼顾；字号与字族都分区；字号范围放宽到 **12–24px**。

## 10. 数据模型（appearance-settings.ts 重构）

```ts
type FontZone = 'ui' | 'body' | 'code'
type FontFamilyPreset = 'system' | 'serif' | 'kai' | 'hei' | 'mono' | 'custom'

interface ZoneFont {
  size: number              // 12–24，默认 ui/body=16、code=14
  familyPreset: FontFamilyPreset
  customFamily: string      // preset === 'custom' 时的输入
}

interface AppearanceSettings {
  zones: Record<FontZone, ZoneFont>
  clickAnimation: boolean
  customCss: string         // 上限 256KB（MAX_CUSTOM_CSS_LENGTH）
}
```

- 默认：`ui {16, system}` / `body {16, system}` / `code {14, mono}`。
- `FONT_STACKS` 统一字族栈常量：system（原 base.css 默认栈）、mono（等宽）、serif/kai/hei。
- `resolveFontFamily(zone)`：`custom` 非空→自定义；`custom` 空白→system 栈；否则 `FONT_STACKS[preset]`。
- **兼容迁移**：`normalizeAppearanceSettings` 兼容旧 `{fontSize, fontFamilyPreset, customFontFamily, clickAnimation}`——把旧字号/字族映射到三个分区（保留用户原设置）。
- 字号 clamp 12–24；customFamily 截断 200 字符；customCss 截断 256KB。
- `isFontPristine(settings)`：三区均在默认字号且默认字族 → 移除全部字体覆盖（尊重浏览器默认）。
- `applyCustomCss(css)`：`<style id="goose-custom-css">` 注入/移除（镜像 `site-theme.ts` 的 `applySiteThemeCss`）。

## 11. 分区应用

| 分区 | 作用对象 | 应用 |
|---|---|---|
| ui | 全站界面（rem 布局） | `html` 根字号 `= ui.size`；body 字族 `--gf-font-family-ui` |
| body | `.gf-prose`（帖子/文章） | `--gf-font-size-body` / `--gf-font-family-body` |
| code | `pre, code`（含 `.gf-prose` 内） | `--gf-font-size-code` / `--gf-font-family-code`（默认等宽回退） |

CSS（`base.css` 改 body 字族变量；`prose.css` / `code-highlighting.css` 追加规则）：

```css
body { font-family: var(--gf-font-family-ui, <默认栈>); }
.gf-prose { font-size: var(--gf-font-size-body); font-family: var(--gf-font-family-body); }
pre, code, .gf-prose pre, .gf-prose code {
  font-size: var(--gf-font-size-code);
  font-family: var(--gf-font-family-code, ui-monospace, SFMono-Regular, Menlo, Consolas, monospace);
}
```

- 全默认（pristine）时 `applyAppearanceSettings` 移除全部覆盖 → 浏览器默认渲染。
- 任一分区被定制后，三区全部显式应用（输入框显示的值即最终渲染值）。
- 字号值作用于各分区绝对像素；正文/代码内部 `em` 相对尺寸随容器缩放。

## 12. 自定义 CSS

- 设置页 textarea（`gf-textarea`）+ 「导入文件」（`<input type="file" accept=".css,text/css">` 读取内容）+ 「清除」。
- 应用：`applyCustomCss` 注入/移除 `<style id="goose-custom-css">`；持久化到 localStorage。
- 上限 `MAX_CUSTOM_CSS_LENGTH = 256 * 1024`（字符），超长截断。
- 安全：现代浏览器禁止 CSS `url(javascript:)` 执行脚本；为自身浏览器注入，无 XSS 风险。

## 13. 设置页 UI（通用选项卡重构）

- **界面/正文/代码 三区行**，每行：分区标题 + 字号 `<input type="number" min="12" max="24">` + 字族 `SiteSelect` + （`custom` 时）自定义字族输入框。字号输入 `@change` 持久化（`@input` 实时预览）。
- **自定义 CSS 区块**：`gf-textarea`（`@input` 防抖实时预览 + `@blur` 持久化）+ 「导入文件」+「清除」。
- **点击动画开关**、**恢复默认**（重置三区 + CSS + 开关）。

## 14. 验证

- `appearance-settings.test.ts` 更新：新模型 normalize / 旧数据迁移 / clamp 12–24 / customCss 截断 / resolveFontFamily 各预设 / isFontPristine。
- vitest / typecheck / build / go（无后端改动，回归即可）。
- 浏览器实测：三区字号字族分别生效（`.gf-prose` 与 `pre,code` 字体变化、UI 根字号变化）、自定义 CSS 注入生效、导入文件、清除、恢复默认、刷新持久化。
- 已知说明：代码区 `.gf-prose pre` 需要额外选择器覆盖 typography 插件尺寸（实现时以浏览器实测为准）。
