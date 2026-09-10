// 课程评价「分享评论长图」工具集：
// - boring-avatars beam 变体的纯 TS 移植（与 YourTJCourse-Serverless 前端同库同版本，
//   无需引入 React 运行时；输出 SVG data URI，匿名/历史评价的占位头像在导出图中
//   同样可用，无跨域问题）
// - 评价短码（sqid）：与 serverless 后端 sqids@0.3.0 同算法（自定义字母表 + minLength 4），
//   列表/分享卡展示的 #XXXX 编号两端一致
// - 导出前的图片内联（html-to-image 不支持跨域资源）：站内同源直接 fetch，
//   跨域（CDN 前缀）走 wsrv.nl 图像代理重取为 data URL

import Sqids from 'sqids'
import { toJpeg, toPng } from 'html-to-image'

// serverless 前端同款配色：深主色 + 蓝/白/琥珀/绿 辅色，beam 变体背景据此轮换。
const AVATAR_COLORS = ['#0f172a', '#38bdf8', '#f8fafc', '#f59e0b', '#22c55e']

// 与 boring-avatars@2.0.4 Beam 组件（variant="beam"）完全一致的参数：viewBox 36×36 笑脸。
const BEAM_VIEWBOX = 36

function hashCode(text: string): number {
  let hash = 0
  for (let i = 0; i < text.length; i++) {
    const code = text.charCodeAt(i)
    hash = (hash << 5) - hash + code
    hash |= 0
  }
  return Math.abs(hash)
}

function getDigit(num: number, nth: number): number {
  return Math.floor(num / Math.pow(10, nth) % 10)
}

function isEvenDigit(num: number, nth: number): boolean {
  return getDigit(num, nth) % 2 === 0
}

// 官方 f(num, range, index) 的 alternate 逻辑：索引位为偶数时取负方向；
// index 缺省（falsy）时恒取正，与官方无参调用（如 rotate: f(n, 360)）一致。
function getUnit(num: number, range: number, index?: number): number {
  const unit = num % range
  return index && getDigit(num, index) % 2 === 0 ? -unit : unit
}

// 官方 _(hex)：按感知亮度（299/587/114 权值）返回对比前景色。
function contrastColor(hex: string): string {
  let normalized = hex
  if (normalized.startsWith('#')) {
    normalized = normalized.slice(1)
  }
  const red = parseInt(normalized.slice(0, 2), 16)
  const green = parseInt(normalized.slice(2, 4), 16)
  const blue = parseInt(normalized.slice(4, 6), 16)
  return (red * 299 + green * 587 + blue * 114) / 1000 >= 128 ? '#000000' : '#FFFFFF'
}

interface BeamWrapper {
  wrapperColor: string
  faceColor: string
  backgroundColor: string
  wrapperTranslateX: number
  wrapperTranslateY: number
  wrapperRotate: number
  wrapperScale: number
  isMouthOpen: boolean
  isCircle: boolean
  eyeSpread: number
  mouthSpread: number
  faceRotate: number
  faceTranslateX: number
  faceTranslateY: number
}

// 官方 Beam 组件的几何推导 b()：全部数值无任何偏移，保证同一 seed 与
// YourTJCourse-Serverless（boring-avatars@2.0.4 + variant="beam"）渲染一致。
function beamWrapper(seed: string): BeamWrapper {
  const num = hashCode(seed)
  const colorsLength = AVATAR_COLORS.length
  const pick = (offset: number) => AVATAR_COLORS[(num + offset) % colorsLength]
  const translateXRaw = getUnit(num, 10, 1)
  const translateX = translateXRaw < 5 ? translateXRaw + BEAM_VIEWBOX / 9 : translateXRaw
  const translateYRaw = getUnit(num, 10, 2)
  const translateY = translateYRaw < 5 ? translateYRaw + BEAM_VIEWBOX / 9 : translateYRaw
  const wrapperColor = pick(0)
  return {
    wrapperColor,
    faceColor: contrastColor(wrapperColor),
    backgroundColor: pick(13),
    wrapperTranslateX: translateX,
    wrapperTranslateY: translateY,
    wrapperRotate: getUnit(num, 360),
    wrapperScale: 1 + getUnit(num, BEAM_VIEWBOX / 12) / 10,
    isMouthOpen: isEvenDigit(num, 2),
    isCircle: isEvenDigit(num, 1),
    eyeSpread: getUnit(num, 5),
    mouthSpread: getUnit(num, 3),
    faceRotate: getUnit(num, 10, 3),
    faceTranslateX: translateX > BEAM_VIEWBOX / 6 ? translateX / 2 : getUnit(num, 8, 1),
    faceTranslateY: translateY > BEAM_VIEWBOX / 6 ? translateY / 2 : getUnit(num, 7, 2),
  }
}

export function buildBeamAvatarDataUri(seed: string, size = 72): string {
  const name = seed.trim() || '匿名用户'
  const wrapped = beamWrapper(name)
  const maskId = `beam-mask-${hashCode(name)}`
  const mouth = wrapped.isMouthOpen
    ? `<path d="M15 ${19 + wrapped.mouthSpread}c2 1 4 1 6 0" stroke="${wrapped.faceColor}" fill="none" stroke-linecap="round"/>`
    : `<path d="M13,${19 + wrapped.mouthSpread} a1,0.75 0 0,0 10,0" fill="${wrapped.faceColor}"/>`
  const svg = `<svg viewBox="0 0 ${BEAM_VIEWBOX} ${BEAM_VIEWBOX}" fill="none" role="img" xmlns="http://www.w3.org/2000/svg" width="${size}" height="${size}"><mask id="${maskId}" maskUnits="userSpaceOnUse" x="0" y="0" width="${BEAM_VIEWBOX}" height="${BEAM_VIEWBOX}"><rect width="${BEAM_VIEWBOX}" height="${BEAM_VIEWBOX}" rx="${BEAM_VIEWBOX * 2}" fill="#FFFFFF"/></mask><g mask="url(#${maskId})"><rect width="${BEAM_VIEWBOX}" height="${BEAM_VIEWBOX}" fill="${wrapped.backgroundColor}"/><rect x="0" y="0" width="${BEAM_VIEWBOX}" height="${BEAM_VIEWBOX}" fill="${wrapped.wrapperColor}" rx="${wrapped.isCircle ? BEAM_VIEWBOX : BEAM_VIEWBOX / 6}" transform="translate(${wrapped.wrapperTranslateX} ${wrapped.wrapperTranslateY}) rotate(${wrapped.wrapperRotate} ${BEAM_VIEWBOX / 2} ${BEAM_VIEWBOX / 2}) scale(${wrapped.wrapperScale})"/><g transform="translate(${wrapped.faceTranslateX} ${wrapped.faceTranslateY}) rotate(${wrapped.faceRotate} ${BEAM_VIEWBOX / 2} ${BEAM_VIEWBOX / 2})">${mouth}<rect x="${14 - wrapped.eyeSpread}" y="14" width="1.5" height="2" rx="1" stroke="none" fill="${wrapped.faceColor}"/><rect x="${20 + wrapped.eyeSpread}" y="14" width="1.5" height="2" rx="1" stroke="none" fill="${wrapped.faceColor}"/></g></g></svg>`
  return `data:image/svg+xml;charset=UTF-8,${encodeURIComponent(svg)}`
}

export interface ReviewAvatarAuthor {
  kind: 'member' | 'anonymous' | 'legacy'
  label: string
  avatarUrl?: string
}

// 评价头像选择（详情页列表 / 预览面板 / 分享卡共用）：
// member 用论坛用户头像（服务端回填 avatarUrl）；匿名/历史评价用 beam 笑脸占位
// （seed 用展示名 + 评价 id，同一评价头像跨页面稳定）。
export function reviewAvatarSrc(author: ReviewAvatarAuthor, reviewId: number, size = 36): string {
  if (author.kind === 'member' && author.avatarUrl) return author.avatarUrl
  return buildBeamAvatarDataUri(`${author.label}-${reviewId}`, size)
}

// —— 评价短码（sqid）——

const REVIEW_SQID_ALPHABET = 'bcdfghjkmnpqrstvwxyzBCDFGHJKMNPQRSTVWXYZ23456789'
const reviewSqids = new Sqids({ alphabet: REVIEW_SQID_ALPHABET, minLength: 4 })

export function reviewSqid(reviewId: number): string {
  return reviewSqids.encode([reviewId])
}

// —— 图片内联（html-to-image 导出前置）——

function isRemoteUrl(src: string): boolean {
  return /^https?:\/\//.test(src)
}

function toWsrProxiedUrl(src: string): string {
  return `https://wsrv.nl/?url=${encodeURIComponent(src)}`
}

function blobToDataUrl(blob: Blob): Promise<string> {
  return new Promise((resolve, reject) => {
    const reader = new FileReader()
    reader.onloadend = () => resolve(String(reader.result || ''))
    reader.onerror = () => reject(new Error('blob_to_data_url_failed'))
    reader.readAsDataURL(blob)
  })
}

async function fetchImageAsDataUrl(src: string): Promise<string> {
  if (!src) return ''
  if (src.startsWith('data:')) return src

  const candidates = isRemoteUrl(src) && src.includes('qlogo.cn') ? [toWsrProxiedUrl(src)] : [src, toWsrProxiedUrl(src)]
  let lastError: unknown = null
  for (const candidate of candidates) {
    try {
      const response = await fetch(candidate, { mode: 'cors' })
      if (!response.ok) throw new Error(`image_fetch_failed:${response.status}`)
      return blobToDataUrl(await response.blob())
    } catch (error) {
      lastError = error
    }
  }
  throw lastError instanceof Error ? lastError : new Error('image_fetch_failed')
}

// 把渲染后的 markdown HTML 内的站外/相对图片统一转为 data URL，保证导出图完整。
export async function inlineMarkdownImages(html: string): Promise<string> {
  const parser = new DOMParser()
  const doc = parser.parseFromString(`<div>${html}</div>`, 'text/html')
  const images = Array.from(doc.querySelectorAll('img'))

  await Promise.all(images.map(async (image) => {
    const src = image.getAttribute('src') || ''
    if (!src) return
    try {
      image.setAttribute('src', await fetchImageAsDataUrl(src))
      image.setAttribute('style', 'max-width: 100%; height: auto; border-radius: 14px; margin: 12px 0; display: block;')
    } catch {
      const alt = image.getAttribute('alt') || '图片加载失败'
      const fallback = doc.createElement('div')
      fallback.textContent = alt
      fallback.setAttribute('style', 'margin: 12px 0; padding: 16px; border-radius: 14px; background: #f8fafc; color: #64748b; border: 1px dashed #cbd5e1;')
      image.replaceWith(fallback)
    }
  }))

  return doc.body.innerHTML
}

export async function waitForImages(container: HTMLElement): Promise<void> {
  const images = Array.from(container.querySelectorAll('img'))
  await Promise.all(images.map((image) => new Promise<void>((resolve) => {
    if (image.complete && image.naturalWidth > 0) {
      resolve()
      return
    }
    const finish = () => {
      window.clearTimeout(timer)
      image.removeEventListener('load', finish)
      image.removeEventListener('error', finish)
      resolve()
    }
    const timer = window.setTimeout(finish, 4000)
    image.addEventListener('load', finish, { once: true })
    image.addEventListener('error', finish, { once: true })
  })))
  if ('fonts' in document) {
    try {
      await (document as Document & { fonts?: FontFaceSet }).fonts?.ready
    } catch {
      // 字体加载失败不阻塞导出
    }
  }
}

export type ShareImageFormat = 'png' | 'jpg'

export async function exportShareNode(node: HTMLElement, format: ShareImageFormat): Promise<string> {
  const options = {
    cacheBust: true,
    backgroundColor: '#ffffff',
    pixelRatio: format === 'jpg' ? 2.2 : 2.5,
  }
  return format === 'jpg'
    ? toJpeg(node, { ...options, quality: 0.96 })
    : toPng(node, options)
}
