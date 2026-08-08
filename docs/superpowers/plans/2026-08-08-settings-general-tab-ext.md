# 通用选项卡扩展 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development or execute task-by-task. Steps use checkbox (`- [ ]`) syntax.

**Goal:** 在既有「通用」选项卡上新增：① 自定义 CSS 导入（粘贴+文件），② 字号改数字输入框，③ 界面/正文/代码三区字号+字族分区调控。

**Architecture:** 重构 `runtime/appearance-settings.ts` 为分区模型 `zones{ui,body,code}` + `customCss`；经 CSS 变量注入作用于 `.gf-prose`（正文）、`pre,code`（代码）、`html` 根字号+body 字族（界面）；设置页 UI 改为三区行 + 自定义 CSS 区块。仍 localStorage 持久化、即时生效。

**Tech Stack:** Vue 3.5 / Vite 8 / Tailwind 4 / TS ~6 / Vitest 4。

## Global Constraints

- 范围限定站点前端 `apps/gooseforum/resource/src/**`；admin 不动。
- localStorage 键不变：`goose-appearance-settings`，新形状 `{ zones: {ui,body,code}, clickAnimation, customCss }`；**兼容旧 `{fontSize, fontFamilyPreset, customFontFamily, clickAnimation}` 迁移**。
- 字号 clamp **12–24**；`customFamily` 截断 200 字符；`customCss` 截断 `256*1024` 字符。
- 分区：`ui`=html 根字号+body 字族；`body`=`.gf-prose`；`code`=`pre,code`（含 `.gf-prose` 内）。
- 三区全默认（pristine）时**移除全部覆盖**（尊重浏览器默认）；任一分区定制后全部显式应用。
- 字族预设：`system | serif | kai | hei | mono | custom`（新增 `mono`）。
- i18n 四语言 zh/en/ja/it 全部补齐。
- 无新依赖；CI 门禁 `pnpm typecheck`；测试在 `resource/test/` 用相对导入 `../src/...`。
- 提交信息用 conventional types。

---

### Task 1: appearance-settings.ts 重构（分区模型 + 自定义 CSS + 单测）

**Files:**
- Modify: `apps/gooseforum/resource/src/runtime/appearance-settings.ts`（整体重写）
- Modify: `apps/gooseforum/resource/test/appearance-settings.test.ts`（更新为新模型）

**Interfaces:**
- Produces: `FontZone = 'ui'|'body'|'code'`；`FontFamilyPreset = 'system'|'serif'|'kai'|'hei'|'mono'|'custom'`；`ZoneFont {size,familyPreset,customFamily}`；`AppearanceSettings {zones, clickAnimation, customCss}`；`FONT_STACKS`；`FONT_ZONES`；`DEFAULT_APPEARANCE_SETTINGS`；`normalizeAppearanceSettings(raw)`（含旧数据迁移）；`resolveFontFamily(zone)`；`isFontPristine(settings)`；`isClickAnimationEnabled()`；`loadAppearanceSettings`；`applyAppearanceSettings`；`applyStoredAppearanceSettings`；`saveAppearanceSettings`；`resetAppearanceSettings`。

- [ ] **Step 1: 重写模块**，完整代码如下（务必逐字实现）：

```ts
export type FontZone = 'ui' | 'body' | 'code'
export type FontFamilyPreset = 'system' | 'serif' | 'kai' | 'hei' | 'mono' | 'custom'

export interface ZoneFont {
  size: number
  familyPreset: FontFamilyPreset
  customFamily: string
}

export interface AppearanceSettings {
  zones: Record<FontZone, ZoneFont>
  clickAnimation: boolean
  customCss: string
}

const STORAGE_KEY = 'goose-appearance-settings'
export const FONT_SIZE_MIN = 12
export const FONT_SIZE_MAX = 24
export const MAX_CUSTOM_FONT_LENGTH = 200
export const MAX_CUSTOM_CSS_LENGTH = 256 * 1024
export const CUSTOM_CSS_STYLE_ID = 'goose-custom-css'

const DEFAULT_SIZE: Record<FontZone, number> = { ui: 16, body: 16, code: 14 }
const DEFAULT_FAMILY: Record<FontZone, FontFamilyPreset> = { ui: 'system', body: 'system', code: 'mono' }

export const FONT_ZONES: readonly FontZone[] = ['ui', 'body', 'code']

export const FONT_FAMILY_PRESETS: readonly FontFamilyPreset[] = [
  'system',
  'serif',
  'kai',
  'hei',
  'mono',
  'custom',
]

export const FONT_STACKS: Record<Exclude<FontFamilyPreset, 'custom'>, string> = {
  system: '-apple-system, BlinkMacSystemFont, "SF Pro Text", "PingFang SC", "Hiragino Sans GB", "Segoe UI", "Microsoft YaHei", "Noto Sans CJK SC", Arial, sans-serif',
  serif: '"Songti SC", "SimSun", serif',
  kai: '"Kaiti SC", "KaiTi", "STKaiti", serif',
  hei: '"Heiti SC", "SimHei", "Noto Sans CJK SC", sans-serif',
  mono: 'ui-monospace, SFMono-Regular, Menlo, Consolas, "Courier New", monospace',
}

export const DEFAULT_APPEARANCE_SETTINGS: AppearanceSettings = Object.freeze({
  zones: Object.freeze({
    ui: Object.freeze({ size: 16, familyPreset: 'system', customFamily: '' }),
    body: Object.freeze({ size: 16, familyPreset: 'system', customFamily: '' }),
    code: Object.freeze({ size: 14, familyPreset: 'mono', customFamily: '' }),
  }),
  clickAnimation: false,
  customCss: '',
})

export function normalizeAppearanceSettings(raw: unknown): AppearanceSettings {
  const source = isRecord(raw) ? raw : {}
  const zones = normalizeZones(source.zones, source)
  const clickAnimation = source.clickAnimation === true
  const customCss = typeof source.customCss === 'string'
    ? source.customCss.slice(0, MAX_CUSTOM_CSS_LENGTH)
    : ''
  return { zones, clickAnimation, customCss }
}

function normalizeZones(rawZones: unknown, legacy: Record<string, unknown>): Record<FontZone, ZoneFont> {
  if (isRecord(rawZones)) {
    return {
      ui: normalizeZone(rawZones.ui, 'ui'),
      body: normalizeZone(rawZones.body, 'body'),
      code: normalizeZone(rawZones.code, 'code'),
    }
  }
  const legacySize = clampNumber(legacy.fontSize, FONT_SIZE_MIN, FONT_SIZE_MAX, 16)
  const legacyFamily = legacy.fontFamilyPreset
  const legacyCustom = legacy.customFontFamily
  return {
    ui: normalizeZone({ size: legacySize, familyPreset: legacyFamily, customFontFamily: legacyCustom }, 'ui'),
    body: normalizeZone({ size: legacySize, familyPreset: legacyFamily, customFontFamily: legacyCustom }, 'body'),
    code: normalizeZone({ size: legacySize, familyPreset: legacyFamily, customFontFamily: legacyCustom }, 'code'),
  }
}

function normalizeZone(raw: unknown, zone: FontZone): ZoneFont {
  const source = isRecord(raw) ? raw : {}
  const customFamily = (
    typeof source.customFamily === 'string'
      ? source.customFamily
      : typeof source.customFontFamily === 'string'
        ? source.customFontFamily
        : ''
  ).slice(0, MAX_CUSTOM_FONT_LENGTH)
  const size = clampNumber(source.size, FONT_SIZE_MIN, FONT_SIZE_MAX, DEFAULT_SIZE[zone])
  let familyPreset = FONT_FAMILY_PRESETS.includes(source.familyPreset as FontFamilyPreset)
    ? (source.familyPreset as FontFamilyPreset)
    : DEFAULT_FAMILY[zone]
  if (familyPreset === 'custom' && customFamily.trim() === '') familyPreset = 'system'
  return { size, familyPreset, customFamily }
}

export function resolveFontFamily(zone: ZoneFont): string {
  if (zone.familyPreset === 'custom') return zone.customFamily.trim() || FONT_STACKS.system
  return FONT_STACKS[zone.familyPreset]
}

export function isFontPristine(settings: AppearanceSettings): boolean {
  return FONT_ZONES.every(zone => {
    const zf = settings.zones[zone]
    return zf.size === DEFAULT_SIZE[zone] && zf.familyPreset === DEFAULT_FAMILY[zone]
  })
}

let clickAnimationEnabled = false
export function isClickAnimationEnabled(): boolean {
  return clickAnimationEnabled
}

export function loadAppearanceSettings(): AppearanceSettings {
  try {
    const raw = window.localStorage.getItem(STORAGE_KEY)
    if (raw === null) return copySettings(DEFAULT_APPEARANCE_SETTINGS)
    return normalizeAppearanceSettings(JSON.parse(raw))
  } catch {
    return copySettings(DEFAULT_APPEARANCE_SETTINGS)
  }
}

export function applyAppearanceSettings(settings: AppearanceSettings) {
  clickAnimationEnabled = settings.clickAnimation
  applyCustomCss(settings.customCss)
  if (isFontPristine(settings)) {
    removeFontOverrides()
    return
  }
  applyFontZones(settings.zones)
}

function applyFontZones(zones: Record<FontZone, ZoneFont>) {
  document.documentElement.style.fontSize = `${zones.ui.size}px`
  setFontVar('--gf-font-family-ui', zones.ui)
  setFontVar('--gf-font-family-body', zones.body)
  setFontVar('--gf-font-family-code', zones.code)
  document.documentElement.style.setProperty('--gf-font-size-body', `${zones.body.size}px`)
  document.documentElement.style.setProperty('--gf-font-size-code', `${zones.code.size}px`)
}

function setFontVar(name: string, zone: ZoneFont) {
  const family = resolveFontFamily(zone)
  if (family) document.documentElement.style.setProperty(name, family)
  else document.documentElement.style.removeProperty(name)
}

function removeFontOverrides() {
  const root = document.documentElement.style
  root.removeProperty('font-size')
  root.removeProperty('--gf-font-family-ui')
  root.removeProperty('--gf-font-family-body')
  root.removeProperty('--gf-font-family-code')
  root.removeProperty('--gf-font-size-body')
  root.removeProperty('--gf-font-size-code')
}

function applyCustomCss(css: string) {
  let el = document.getElementById(CUSTOM_CSS_STYLE_ID) as HTMLStyleElement | null
  if (!css) {
    el?.remove()
    return
  }
  if (!el) {
    el = document.createElement('style')
    el.id = CUSTOM_CSS_STYLE_ID
    document.head.appendChild(el)
  }
  el.textContent = css
}

export function applyStoredAppearanceSettings() {
  applyAppearanceSettings(loadAppearanceSettings())
}

export function saveAppearanceSettings(settings: AppearanceSettings) {
  applyAppearanceSettings(settings)
  try {
    window.localStorage.setItem(STORAGE_KEY, JSON.stringify(settings))
  } catch {
    // Ignore storage failures in private or restricted browsing contexts.
  }
}

export function resetAppearanceSettings() {
  applyAppearanceSettings(DEFAULT_APPEARANCE_SETTINGS)
  try {
    window.localStorage.removeItem(STORAGE_KEY)
  } catch {
    // Ignore storage failures in private or restricted browsing contexts.
  }
}

function copySettings(settings: AppearanceSettings): AppearanceSettings {
  return {
    zones: {
      ui: { ...settings.zones.ui },
      body: { ...settings.zones.body },
      code: { ...settings.zones.code },
    },
    clickAnimation: settings.clickAnimation,
    customCss: settings.customCss,
  }
}

function isRecord(value: unknown): value is Record<string, unknown> {
  return typeof value === 'object' && value !== null && !Array.isArray(value)
}

function clampNumber(value: unknown, min: number, max: number, fallback: number): number {
  if (typeof value !== 'number' || Number.isNaN(value)) return fallback
  return Math.min(max, Math.max(min, Math.round(value)))
}
```

- [ ] **Step 2: 重写测试** `resource/test/appearance-settings.test.ts`，覆盖：

```ts
import { describe, expect, test } from 'vitest'
import {
  DEFAULT_APPEARANCE_SETTINGS,
  isFontPristine,
  MAX_CUSTOM_CSS_LENGTH,
  normalizeAppearanceSettings,
  resolveFontFamily,
} from '../src/runtime/appearance-settings'

const def = () => ({ ...DEFAULT_APPEARANCE_SETTINGS })

describe('normalizeAppearanceSettings', () => {
  test('returns defaults for garbage / null / empty', () => {
    for (const raw of [null, undefined, 'x', [], {}, 42]) {
      expect(normalizeAppearanceSettings(raw)).toEqual(def())
    }
  })

  test('normalizes zones with clamping 12..24 and per-zone defaults', () => {
    const s = normalizeAppearanceSettings({ zones: {} })
    expect(s.zones.ui.size).toBe(16)
    expect(s.zones.body.size).toBe(16)
    expect(s.zones.code.size).toBe(14)
    expect(s.zones.ui.familyPreset).toBe('system')
    expect(s.zones.code.familyPreset).toBe('mono')
    const s2 = normalizeAppearanceSettings({ zones: { ui: { size: 30, familyPreset: 'kai' } } })
    expect(s2.zones.ui.size).toBe(24)
    expect(s2.zones.ui.familyPreset).toBe('kai')
    expect(s2.zones.body.size).toBe(16)
  })

  test('blank custom preset falls back to system', () => {
    const s = normalizeAppearanceSettings({ zones: { body: { familyPreset: 'custom', customFamily: '   ' } } })
    expect(s.zones.body.familyPreset).toBe('system')
  })

  test('migrates legacy single font shape to all zones', () => {
    const s = normalizeAppearanceSettings({ fontSize: 18, fontFamilyPreset: 'serif', customFontFamily: '' })
    expect(s.zones.ui.size).toBe(18)
    expect(s.zones.body.size).toBe(18)
    expect(s.zones.code.size).toBe(18)
    expect(s.zones.ui.familyPreset).toBe('serif')
    expect(s.zones.code.familyPreset).toBe('serif')
  })

  test('truncates customFamily and customCss', () => {
    const s = normalizeAppearanceSettings({
      zones: { ui: { customFamily: 'x'.repeat(500) } },
      customCss: 'y'.repeat(MAX_CUSTOM_CSS_LENGTH + 10),
    })
    expect(s.zones.ui.customFamily).toHaveLength(200)
    expect(s.customCss).toHaveLength(256 * 1024)
  })

  test('clickAnimation must be exactly true', () => {
    expect(normalizeAppearanceSettings({ clickAnimation: true }).clickAnimation).toBe(true)
    expect(normalizeAppearanceSettings({ clickAnimation: 1 }).clickAnimation).toBe(false)
  })
})

describe('resolveFontFamily', () => {
  test('returns preset stacks for each family', () => {
    for (const p of ['system', 'serif', 'kai', 'hei', 'mono'] as const) {
      expect(resolveFontFamily({ size: 16, familyPreset: p, customFamily: '' })).toMatch(/^["a-z-]/)
    }
    expect(resolveFontFamily({ size: 16, familyPreset: 'serif', customFamily: '' })).toContain('Songti SC')
    expect(resolveFontFamily({ size: 16, familyPreset: 'mono', customFamily: '' })).toContain('ui-monospace')
  })

  test('custom uses trimmed value, blank custom falls back to system stack', () => {
    expect(resolveFontFamily({ size: 16, familyPreset: 'custom', customFamily: '  KaiTi  ' })).toBe('KaiTi')
    expect(resolveFontFamily({ size: 16, familyPreset: 'custom', customFamily: '   ' })).toContain('-apple-system')
  })
})

describe('isFontPristine', () => {
  test('true when all zones at defaults', () => {
    expect(isFontPristine(def())).toBe(true)
  })
  test('false when any zone differs', () => {
    expect(isFontPristine({ ...def(), zones: { ...def().zones, ui: { size: 18, familyPreset: 'system', customFamily: '' } } })).toBe(false)
    expect(isFontPristine({ ...def(), zones: { ...def().zones, code: { size: 14, familyPreset: 'serif', customFamily: '' } } })).toBe(false)
  })
})
```

（测试需 import `MAX_CUSTOM_CSS_LENGTH`；上面的 `def()` 用 spread 生成可变副本。）

- [ ] **Step 3: 验证** `cd apps/gooseforum/resource && pnpm exec vitest run test/appearance-settings.test.ts && pnpm typecheck`
- [ ] **Step 4: 提交** `feat: rework appearance-settings into per-zone font model with custom css`

---

### Task 2: CSS 分区变量与规则

**Files:**
- Modify: `apps/gooseforum/resource/src/styles/base.css`（body 字族变量）
- Modify: `apps/gooseforum/resource/src/styles/prose.css`（`.gf-prose` 分区规则）
- Modify: `apps/gooseforum/resource/src/styles/code-highlighting.css`（`pre,code` 分区规则）

- [ ] **Step 1: base.css** 第 8–10 行 `body { font-family: <原栈> }` 改为：
```css
body {
  font-family: var(--gf-font-family-ui, -apple-system, BlinkMacSystemFont, "SF Pro Text", "PingFang SC", "Hiragino Sans GB", "Segoe UI", "Microsoft YaHei", "Noto Sans CJK SC", Arial, sans-serif);
}
```
- [ ] **Step 2: prose.css** 末尾追加：
```css
.gf-prose {
  font-size: var(--gf-font-size-body);
  font-family: var(--gf-font-family-body);
}
```
- [ ] **Step 3: code-highlighting.css** 末尾追加：
```css
pre,
code,
.gf-prose pre,
.gf-prose code {
  font-size: var(--gf-font-size-code);
  font-family: var(--gf-font-family-code, ui-monospace, SFMono-Regular, Menlo, Consolas, "Courier New", monospace);
}
```
- [ ] **Step 4: 验证** `cd apps/gooseforum/resource && pnpm typecheck && pnpm build`
- [ ] **Step 5: 提交** `style: add per-zone font variables for body/code/ui`

---

### Task 3: 设置页 UI 重构 + 四语言 i18n

**Files:**
- Modify: `apps/gooseforum/resource/src/site/pages/SettingsPage.vue`
- Modify: `apps/gooseforum/resource/src/locales/{zh,en,ja,it}.ts`

**Interfaces:**
- Consumes: Task 1 的全部导出（`AppearanceSettings`, `FontZone`, `loadAppearanceSettings`, `applyAppearanceSettings`, `saveAppearanceSettings`, `resetAppearanceSettings`）；`SiteSelect`。

- [ ] **Step 1: 脚本**（`appearance` reactive 与函数区，原 `:656-681` 附近）

```ts
import {
  applyAppearanceSettings,
  loadAppearanceSettings,
  resetAppearanceSettings,
  saveAppearanceSettings,
  type AppearanceSettings,
  type FontZone,
} from '@/runtime/appearance-settings'

const appearance = reactive<AppearanceSettings>(loadAppearanceSettings())

const fontFamilyOptions = computed(() => [
  { value: 'system', label: t('settings.general.fontSystem') },
  { value: 'serif', label: t('settings.general.fontSerif') },
  { value: 'kai', label: t('settings.general.fontKai') },
  { value: 'hei', label: t('settings.general.fontHei') },
  { value: 'mono', label: t('settings.general.fontMono') },
  { value: 'custom', label: t('settings.general.fontCustom') },
])

const fontZones = computed(() => [
  { key: 'ui' as FontZone, label: t('settings.general.zoneUi'), description: t('settings.general.zoneUiDescription') },
  { key: 'body' as FontZone, label: t('settings.general.zoneBody'), description: t('settings.general.zoneBodyDescription') },
  { key: 'code' as FontZone, label: t('settings.general.zoneCode'), description: t('settings.general.zoneCodeDescription') },
])

function previewAppearance() {
  applyAppearanceSettings({ ...appearance })
}

function saveAppearance() {
  saveAppearanceSettings({ ...appearance })
}

function saveCustomFont(zone: FontZone) {
  const zf = appearance.zones[zone]
  if (zf.familyPreset === 'custom' && !zf.customFamily.trim()) zf.familyPreset = 'system'
  saveAppearance()
}

let cssPreviewTimer: number | undefined
function previewCssDebounced() {
  window.clearTimeout(cssPreviewTimer)
  cssPreviewTimer = window.setTimeout(previewAppearance, 300)
}

const cssFileInput = ref<HTMLInputElement | null>(null)
function triggerCssFileImport() {
  cssFileInput.value?.click()
}
function onCssFileChange(event: Event) {
  const file = (event.target as HTMLInputElement).files?.[0]
  ;(event.target as HTMLInputElement).value = ''
  if (!file) return
  const reader = new FileReader()
  reader.onload = () => {
    appearance.customCss = typeof reader.result === 'string' ? reader.result : ''
    saveAppearance()
  }
  reader.readAsText(file)
}
function clearCustomCss() {
  appearance.customCss = ''
  saveAppearance()
}

function confirmResetAppearance() {
  if (!window.confirm(t('settings.general.resetConfirm'))) return
  resetAppearanceSettings()
  Object.assign(appearance, loadAppearanceSettings())
}
```

> 注意：`previewAppearance` 会把当前 `appearance`（含 customCss）应用；CSS textarea 用防抖预览 + blur 持久化。

- [ ] **Step 2: 模板**——替换原「字体大小/字体样式/自定义 CSS 相关」区块为三区行 + 自定义 CSS 区块（保留点击动画开关与恢复默认；`@input/@change/@blur/@keydown.enter` 语义与上一步函数对应）。

三区行（在 `settings.general` 区块内，`divide-y` 容器中）：
```html
<div v-for="zone in fontZones" :key="zone.key" class="py-4">
  <div class="flex items-center justify-between gap-4">
    <span class="min-w-0">
      <span class="block text-sm font-semibold text-base-content">{{ zone.label }}</span>
      <span class="text-sm text-base-content/55">{{ zone.description }}</span>
    </span>
    <div class="flex shrink-0 items-center gap-2">
      <input
        type="number"
        min="12"
        max="24"
        step="1"
        class="gf-input h-9 w-20 text-center text-sm"
        v-model.number="appearance.zones[zone.key].size"
        :aria-label="zone.label"
        @input="previewAppearance"
        @change="saveAppearance"
      />
      <SiteSelect
        class="w-44 shrink-0"
        :options="fontFamilyOptions"
        v-model="appearance.zones[zone.key].familyPreset"
        @update:model-value="saveAppearance"
      />
    </div>
  </div>
  <div v-if="appearance.zones[zone.key].familyPreset === 'custom'" class="mt-3">
    <input
      v-model="appearance.zones[zone.key].customFamily"
      type="text"
      class="gf-input w-full"
      maxlength="200"
      :placeholder="t('settings.general.customFontPlaceholder')"
      :aria-label="zone.label + ' ' + t('settings.general.fontCustom')"
      @input="previewAppearance"
      @blur="saveCustomFont(zone.key)"
      @keydown.enter="saveCustomFont(zone.key)"
    />
  </div>
</div>
```

自定义 CSS 区块：
```html
<div class="py-4">
  <div class="flex items-center justify-between gap-4">
    <span>
      <span class="block text-sm font-semibold text-base-content">{{ t('settings.general.customCss') }}</span>
      <span class="text-sm text-base-content/55">{{ t('settings.general.customCssDescription') }}</span>
    </span>
    <div class="flex shrink-0 items-center gap-2">
      <button type="button" class="gf-button gf-button-sm gf-button-muted" @click="triggerCssFileImport">{{ t('settings.general.importCss') }}</button>
      <button type="button" class="gf-button gf-button-sm gf-button-muted" :disabled="!appearance.customCss" @click="clearCustomCss">{{ t('settings.general.clearCss') }}</button>
      <input ref="cssFileInput" type="file" accept=".css,text/css" class="hidden" @change="onCssFileChange" />
    </div>
  </div>
  <textarea
    v-model="appearance.customCss"
    class="gf-textarea mt-3 h-44 w-full font-mono text-xs"
    :placeholder="t('settings.general.customCssPlaceholder')"
    :aria-label="t('settings.general.customCss')"
    @input="previewCssDebounced"
    @blur="saveAppearance"
  ></textarea>
</div>
```

- [ ] **Step 3: i18n 四语言**：`settings.tabs.general` 已有；新增/调整 `settings.general.*`：

zh: `zoneUi:'界面' zoneUiDescription:'导航、菜单、按钮等界面元素' zoneBody:'正文' zoneBodyDescription:'帖子与文章内容' zoneCode:'代码' zoneCodeDescription:'代码块与行内代码' fontMono:'等宽' customCss:'自定义 CSS' customCssDescription:'粘贴或导入 CSS，即时应用并保存在本机' importCss:'导入文件' clearCss:'清除' customCssPlaceholder:'在此粘贴你的 CSS…'`
en: `zoneUi:'Interface' ... zoneBody:'Body text' ... zoneCode:'Code' ... fontMono:'Monospace' customCss:'Custom CSS' customCssDescription:'Paste or import CSS, applied instantly and saved on this device' importCss:'Import file' clearCss:'Clear' customCssPlaceholder:'Paste your CSS here…'`
ja: `zoneUi:'インターフェース' ... zoneBody:'本文' ... zoneCode:'コード' ... fontMono:'等幅' customCss:'カスタムCSS' customCssDescription:'CSSを貼り付けるかインポートすると、この端末に即時適用・保存されます' importCss:'ファイルを読み込む' clearCss:'クリア' customCssPlaceholder:'ここにCSSを貼り付け…'`
it: `zoneUi:'Interfaccia' ... zoneBody:'Corpo del testo' ... zoneCode:'Codice' ... fontMono:'Monospace' customCss:'CSS personalizzato' customCssDescription:'Incolla o importa CSS, applicato subito e salvato su questo dispositivo' importCss:'Importa file' clearCss:'Svuota' customCssPlaceholder:'Incolla qui il tuo CSS…'`

> 原 `settings.general.fontSize/fontSizeDescription/fontSizeHint/fontFamily/fontFamilyDescription` 若不再被引用可删除；`fontSystem/fontSerif/fontKai/fontHei/fontCustom/customFontPlaceholder/clickAnimation*/reset*/resetConfirm` 保留。

- [ ] **Step 4: 验证** `cd apps/gooseforum/resource && pnpm typecheck && pnpm build && pnpm exec vitest run test/`
- [ ] **Step 5: 提交** `feat: per-zone font controls and custom css in general settings tab`

---

### Task 4: 全量验证 + 对抗式评审

- [ ] vitest 全量、typecheck、build、`go vet ./... && go test ./app/http/controllers/forum/ -run TestSettingsTabs`。
- [ ] 浏览器实测（yourtj-local-login）：三区字号/字族分别生效、自定义 CSS 注入、导入文件、清除、恢复默认、刷新持久化。
- [ ] 对抗式评审（runtime 模块 + 设置页 UI/i18n），处理 CRITICAL/HIGH。
- [ ] 交用户确认后再提交到 PR。
