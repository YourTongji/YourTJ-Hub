/**
 * img-fx 风格高性能 Gradient Sweep (梯度扫掠消融) 引擎 (Canvas 2D 实现)
 *
 * 深度复刻 Jakub Antalik 的 img-fx (https://image.jakubantalik.com / https://github.com/Jakubantalik/img-fx)
 * sweep-gradient Light 预设实现：
 * 1. 平淡克制低调的极简灰度色调：
 *    采用与官网 Light 预设一致的纯净银白/极简中性灰度（#f5f5f5, #ededed, #eaeaea, #d2d2d2, #fafafa），
 *    彻底去除鲜艳蓝杂色，呈现如文具便签与毛玻璃光晕般的高级素雅质感；
 * 2. 舒缓平滑的呼吸感节奏：
 *    默认 2.4s 舒展优雅的 easeOutCubic 扫掠周期，与官网 demo 节奏严丝合缝；
 * 3. 边缘局域微闪 (Edge-localized Sweep Band)：
 *    扫掠过渡带边缘伴随柔和的单格明暗光晕，自左上向右下单向平滑消融，露出高清底图。
 */

export interface ImgFxOptions {
  cellSize?: number
  gap?: number
  palette?: string[]
  holdDurationMs?: number
  sweepDurationMs?: number
  onRevealed?: () => void
}

/** 严格对标 img-fx sweep-gradient light 预设调色板：纯净克制、平淡低调的中性银白与浅灰 */
const DEFAULT_PALETTE = [
  '#f5f5f5',
  '#ededed',
  '#eaeaea',
  '#d2d2d2',
  '#e5e5e5',
  '#f0f0f0',
  '#fafafa',
]

function pseudoRandom(x: number, y: number, seed: number): number {
  const n = Math.sin(x * 12.9898 + y * 78.233 + seed * 37.719) * 43758.5453
  return n - Math.floor(n)
}

function hexToRgba(hex: string, alpha: number): string {
  const clean = hex.replace('#', '')
  const r = parseInt(clean.substring(0, 2), 16)
  const g = parseInt(clean.substring(2, 4), 16)
  const b = parseInt(clean.substring(4, 6), 16)
  return `rgba(${r}, ${g}, ${b}, ${alpha})`
}

function easeOutCubic(x: number): number {
  return 1 - Math.pow(1 - x, 3)
}

export class ImgFxController {
  private canvas: HTMLCanvasElement
  private ctx: CanvasRenderingContext2D | null = null
  private width = 0
  private height = 0
  private dpr = 1

  private cellSize: number
  private gap: number
  private palette: string[]
  private holdDurationMs: number
  private sweepDurationMs: number
  private onRevealed?: () => void

  private state: 'idle' | 'playing' | 'completed' = 'idle'
  private startTime = 0
  private rafId: number | null = null
  private destroyed = false

  constructor(canvas: HTMLCanvasElement, options: ImgFxOptions = {}) {
    this.canvas = canvas
    this.ctx = canvas.getContext('2d')
    this.cellSize = options.cellSize ?? 22
    this.gap = options.gap ?? 0.5
    this.palette = options.palette ?? DEFAULT_PALETTE
    this.holdDurationMs = options.holdDurationMs ?? 1100
    this.sweepDurationMs = options.sweepDurationMs ?? 2400
    this.onRevealed = options.onRevealed
  }

  public resize(cssWidth: number, cssHeight: number) {
    if (!this.canvas || !this.ctx || cssWidth <= 0 || cssHeight <= 0) return
    this.dpr = typeof window !== 'undefined' ? Math.min(window.devicePixelRatio || 1, 2) : 1
    this.width = cssWidth
    this.height = cssHeight

    this.canvas.width = Math.round(cssWidth * this.dpr)
    this.canvas.height = Math.round(cssHeight * this.dpr)
    this.canvas.style.width = `${cssWidth}px`
    this.canvas.style.height = `${cssHeight}px`
  }

  public play(callback?: () => void) {
    if (this.destroyed || !this.ctx) return
    if (callback) this.onRevealed = callback

    // 无障碍减弱动画模式：直接瞬间完成
    if (
      typeof window !== 'undefined' &&
      window.matchMedia &&
      window.matchMedia('(prefers-reduced-motion: reduce)').matches
    ) {
      this.complete()
      return
    }

    this.canvas.style.opacity = '1'
    this.state = 'playing'
    this.startTime = performance.now()
    // 立即同步绘制第 0 帧，确保首帧瞬间完全遮挡底图，杜绝任何闪烁
    this.renderFrame(this.startTime)
    this.startLoop()
  }

  private complete() {
    this.state = 'completed'
    this.stopLoop()
    if (this.ctx && this.canvas) {
      this.ctx.clearRect(0, 0, this.canvas.width, this.canvas.height)
      this.canvas.style.opacity = '0'
    }
    if (this.onRevealed) {
      this.onRevealed()
    }
  }

  private startLoop() {
    if (this.rafId !== null) return
    const step = (timestamp: number) => {
      this.renderFrame(timestamp)
      if (this.state === 'playing') {
        this.rafId = requestAnimationFrame(step)
      } else {
        this.rafId = null
      }
    }
    this.rafId = requestAnimationFrame(step)
  }

  private stopLoop() {
    if (this.rafId !== null) {
      cancelAnimationFrame(this.rafId)
      this.rafId = null
    }
  }

  private renderFrame(timestamp: number) {
    const ctx = this.ctx
    if (!ctx || this.width <= 0 || this.height <= 0) return

    const now = timestamp || performance.now()
    const elapsed = now - this.startTime
    const elapsedSec = elapsed / 1000

    let sweepProgress = 0
    let isSweepActive = false

    if (elapsed > this.holdDurationMs) {
      isSweepActive = true
      const sweepElapsed = elapsed - this.holdDurationMs
      const linearP = Math.min(1, Math.max(0, sweepElapsed / this.sweepDurationMs))
      sweepProgress = easeOutCubic(linearP)
    }

    ctx.save()
    ctx.scale(this.dpr, this.dpr)
    ctx.clearRect(0, 0, this.width, this.height)

    const step = this.cellSize + this.gap
    const cols = Math.ceil(this.width / step)
    const rows = Math.ceil(this.height / step)

    // 对标 img-fx sweep-gradient 算法：
    // waveWindow 为过渡波宽，progress 经 easeOutCubic 驱动自左上至右下舒展扫掠
    const waveWindow = 0.38
    const wavePos = sweepProgress * (1 + waveWindow)

    let activeCount = 0

    for (let c = 0; c < cols; c++) {
      for (let r = 0; r < rows; r++) {
        // 对角归一化坐标：0（左上）至 1（右下）
        const nx = c / cols
        const ny = r / rows
        const dd = nx * 0.65 + ny * 0.35
        const jitter = (pseudoRandom(c, r, 97) - 0.5) * 0.08
        const cellPos = Math.max(0, Math.min(1, dd + jitter))

        let cellOpacity = 1
        let cellScale = 1
        let edge = 0

        if (!isSweepActive) {
          // 阶段一：纯粹的生成加载微光态（Hold / Churn 阶段），完全遮蔽底图，流光生动
          cellOpacity = 1
          cellScale = 1
        } else {
          // 阶段二：Gradient Sweep 渐进扫掠消融
          const diff = wavePos - cellPos
          if (diff <= 0) {
            cellOpacity = 1
            cellScale = 1
          } else if (diff >= waveWindow) {
            cellOpacity = 0
            cellScale = 0
          } else {
            const localP = diff / waveWindow
            edge = localP * (1 - localP) * 4 // 在过渡带中心形成柔和微光
            cellOpacity = Math.max(0, (1 - localP) * 0.98)
            cellScale = Math.max(0, 1 - localP * 0.5)
          }
        }

        if (cellOpacity <= 0.01 || cellScale <= 0.01) {
          continue
        }

        activeCount++

        // 纯解析克制低频正弦微光（平缓沉静，低调纯净）
        const wave = Math.sin((c * 0.22 + r * 0.16) - elapsedSec * 2.0) * 0.5 + 0.5
        const phase = pseudoRandom(c, r, 31) * Math.PI * 2
        const cellShimmer = Math.sin(elapsedSec * 2.5 + phase) * 0.08

        const intensity = Math.max(0, Math.min(1, wave * 0.5 + 0.25 + cellShimmer + edge * 0.2))
        const colorIdx = Math.floor(intensity * (this.palette.length - 1))
        const hex = this.palette[colorIdx]

        // 基础不透明度保持在 0.97，彻底遮挡底图；过渡阶段由 cellOpacity 驱动平滑消融
        const baseAlpha = 0.97 + edge * 0.03
        const finalAlpha = Math.max(0, Math.min(1, baseAlpha * cellOpacity))

        ctx.fillStyle = hexToRgba(hex, finalAlpha)

        // 单元格紧密排列，微倒角形成如高档陶瓷/微晶马赛克质感
        const drawSize = (this.cellSize + this.gap) * cellScale
        const offset = ((this.cellSize + this.gap) - drawSize) / 2
        const drawX = c * step + offset
        const drawY = r * step + offset
        const radius = Math.min(3, drawSize / 3)

        if (typeof ctx.roundRect === 'function') {
          ctx.beginPath()
          ctx.roundRect(drawX, drawY, drawSize, drawSize, radius)
          ctx.fill()
        } else {
          ctx.fillRect(drawX, drawY, drawSize, drawSize)
        }
      }
    }

    ctx.restore()

    if ((isSweepActive && elapsed >= this.holdDurationMs + this.sweepDurationMs) || activeCount === 0) {
      this.complete()
    }
  }

  public destroy() {
    this.destroyed = true
    this.stopLoop()
    this.ctx = null
  }
}
