import { defineConfig } from 'vite'
// Keep vue and @vue/compiler-dom pinned to one version: the helper Symbols below
// must come from the same compiler-core instance used by @vitejs/plugin-vue.
import vue from '@vitejs/plugin-vue'
import {
  ElementTypes,
  NodeTypes,
  RESOLVE_COMPONENT,
  TO_DISPLAY_STRING,
  createCallExpression,
  createCompoundExpression,
  createObjectExpression,
  createObjectProperty,
  createSimpleExpression,
  createVNodeCall,
} from '@vue/compiler-dom'
import tailwindcss from '@tailwindcss/vite'
import { fileURLToPath, URL } from 'node:url'
import {
  isContentEditableValue,
  TWEMOJI_TEXT_FREE_TAGS,
} from './src/runtime/twemoji-editability'

const resourceRoot = fileURLToPath(new URL('./', import.meta.url))
const twemojiPublicPath = fileURLToPath(
  new URL('../../mobile/packages/ui_kit/assets/twemoji-public', import.meta.url),
)

const staticEmojiPattern =
  /[\p{Extended_Pictographic}\p{Regional_Indicator}]|[#*0-9]\uFE0F?\u20E3/u
const excludedTwemojiSubtrees = new WeakSet<object>()

function isPlainTextNode(node: any) {
  return node.type === NodeTypes.TEXT || node.type === NodeTypes.INTERPOLATION
}

function hasBindingForAttribute(node: any, attribute: string): boolean {
  return node.props.some(
    (prop: any) =>
      prop.type === NodeTypes.DIRECTIVE &&
      prop.name === 'bind' &&
      (!prop.arg ||
        prop.arg.isStatic !== true ||
        prop.arg.content.toLowerCase() === attribute),
  )
}

function isTwemojiExcludedElement(node: any): boolean {
  if (TWEMOJI_TEXT_FREE_TAGS.has(String(node.tag).toLowerCase())) return true
  return (
    node.props.some((prop: any) => {
      if (prop.type === NodeTypes.DIRECTIVE && prop.name === 'model') return true
      if (prop.type !== NodeTypes.ATTRIBUTE) return false

      const name = prop.name.toLowerCase()
      if (name === 'data-twemoji-ignore') return true
      return name === 'contenteditable' && isContentEditableValue(prop.value?.content ?? '')
    }) ||
    hasBindingForAttribute(node, 'contenteditable') ||
    hasBindingForAttribute(node, 'data-twemoji-ignore')
  )
}

function ensureTwemojiHtmlDirective(node: any): boolean {
  const hasVHtml = node.props.some(
    (prop: any) => prop.type === NodeTypes.DIRECTIVE && prop.name === 'html',
  )
  if (!hasVHtml) return false

  const hasEmojiHtmlDirective = node.props.some(
    (prop: any) => prop.type === NodeTypes.DIRECTIVE && prop.name === 'twemoji-html',
  )
  if (!hasEmojiHtmlDirective) {
    const source = node.props.find(
      (prop: any) => prop.type === NodeTypes.DIRECTIVE && prop.name === 'html',
    )
    node.props.push({
      type: NodeTypes.DIRECTIVE,
      name: 'twemoji-html',
      exp: undefined,
      arg: undefined,
      modifiers: [],
      loc: source.loc,
    })
  }
  return true
}

function twemojiNodeTransform(node: any, context: any) {
  if (node.type !== NodeTypes.ELEMENT && node.type !== NodeTypes.ROOT) return

  const isElement = node.type === NodeTypes.ELEMENT
  const excludedByAncestor = isElement && excludedTwemojiSubtrees.has(context.parent)
  const excludedHere = isElement && isTwemojiExcludedElement(node)
  if (isElement && (excludedByAncestor || excludedHere)) {
    // Vue invokes node transforms on enter, before traversing children. Record each
    // excluded element so the same policy is inherited by every nested element.
    excludedTwemojiSubtrees.add(node)
    return () => {
      ensureTwemojiHtmlDirective(node)
    }
  }

  return () => {
    if (isElement) {
      if (ensureTwemojiHtmlDirective(node)) return
    }

    if (!node.children?.length) return

    const output: any[] = []
    let textRun: any[] = []
    const flushTextRun = () => {
      if (!textRun.length) return

      const hasInterpolation = textRun.some(
        (child) => child.type === NodeTypes.INTERPOLATION,
      )
      const staticText = textRun
        .filter((child) => child.type === NodeTypes.TEXT)
        .map((child) => child.content)
        .join('')
      if (!hasInterpolation && !staticEmojiPattern.test(staticText)) {
        output.push(...textRun)
        textRun = []
        return
      }

      const expressionParts = textRun.map((child) =>
        child.type === NodeTypes.TEXT
          ? createSimpleExpression(JSON.stringify(child.content), false, child.loc)
          : createCallExpression(context.helper(TO_DISPLAY_STRING), [child.content]),
      )
      const textExpression =
        expressionParts.length === 1
          ? expressionParts[0]
          : createCompoundExpression(
              expressionParts.flatMap((part, index) =>
                index === 0 ? [part] : [' + ', part],
              ),
              textRun[0].loc,
            )
      const component = createCallExpression(
        context.helper(RESOLVE_COMPONENT),
        [createSimpleExpression('TwemojiText', true)],
      )
      const props = createObjectExpression([
        createObjectProperty(createSimpleExpression('text', true), textExpression),
      ])
      const codegenNode = createVNodeCall(
        context,
        component,
        props,
        undefined,
        8, // Vue PatchFlags.PROPS: re-evaluate only the dynamic `text` prop.
        createSimpleExpression('["text"]', false, textRun[0].loc),
        undefined,
        false,
        false,
        true,
        textRun[0].loc,
      )
      const scopeTrackingInterpolation = textRun.find(
        (child) => child.type === NodeTypes.INTERPOLATION,
      )
      output.push({
        type: NodeTypes.ELEMENT,
        tag: 'TwemojiText',
        tagType: ElementTypes.COMPONENT,
        // Slot analysis runs after this transform. Keep the original expression
        // visible to Vue's scope-ref walker so v-for/v-slot content remains
        // dynamic when its owner updates.
        props: scopeTrackingInterpolation
          ? [{
              type: NodeTypes.DIRECTIVE,
              name: 'bind',
              arg: createSimpleExpression('text', true, textRun[0].loc),
              exp: scopeTrackingInterpolation.content,
              modifiers: [],
              loc: textRun[0].loc,
            }]
          : [],
        children: [],
        loc: textRun[0].loc,
        codegenNode,
      })
      textRun = []
    }

    for (const child of node.children) {
      if (isPlainTextNode(child)) {
        textRun.push(child)
      } else {
        flushTextRun()
        output.push(child)
      }
    }
    flushTextRun()
    node.children = output
  }
}

export default defineConfig({
  base: '/assets/',
  publicDir: twemojiPublicPath,
  plugins: [
    vue({ template: { compilerOptions: { nodeTransforms: [twemojiNodeTransform] } } }),
    tailwindcss(),
  ],
  resolve: {
    alias: {
      '@': fileURLToPath(new URL('./src', import.meta.url)),
    },
  },
  build: {
    outDir: 'static/dist',
    assetsDir: 'assets',
    manifest: true,
    emptyOutDir: true,
    // 页面级 CSP（securityHeaders.go buildPageCSP）的 script-src/style-src 均不放行 data:。
    // 默认 4096B 内联阈值会把 ?url 导入的小资产（vditor 语言包、content-theme/hljs 主题 CSS）
    // 内联成 data:text/javascript|css 而被 CSP 拦截：语言包报错令编辑器无法挂载，主题 CSS 静默丢失
    // （issue #461）。必须保持 0，让所有 ?url 运行时资产落成 /assets 同源文件；dev server 恒不内联，
    // 此约束只在生产构建生效。
    assetsInlineLimit: 0,
    rollupOptions: {
      input: {
        site: 'src/site/main.ts',
        admin: 'src/admin/main.ts',
      },
    },
  },
  server: {
    port: 3010,
    fs: { allow: [resourceRoot, twemojiPublicPath] },
    // 不固定 dev server 的来源地址：页面 CSP 为 script-src 'self'，固定后 `?url` 运行时资源
    // 生成绝对 URL 会被判为跨域脚本静默拦截（issue #453）。保持相对路径经后端 /assets 同源代理。
  },
})
