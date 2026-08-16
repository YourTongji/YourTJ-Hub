import { useNavigationState } from './navigation-state'
import { resolvePageComponent } from './page-registry'
import { createGooseClient } from '@gooseforum/client'
import type { Component } from 'vue'
import type { PagePayload } from '@gooseforum/client'
import { createRouter, createWebHistory, isNavigationFailure, type Router } from 'vue-router'

const client = createGooseClient()

export interface PreparedPage {
  payload: PagePayload
  component: Component
}

export function installNavigation(initialPage: PreparedPage, routeComponent: Component, onPage: (page: PreparedPage) => void): Router {
  const navigation = useNavigationState()
  let initialNavigation = true
  let loadedPage = initialPage

  const router = createRouter({
    history: createWebHistory(),
    routes: [
      {
        path: '/:pathMatch(.*)*',
        component: routeComponent,
      },
    ],
    scrollBehavior(to, _from, savedPosition) {
      return new Promise((resolve) => {
        requestAnimationFrame(() => {
          if (savedPosition) {
            resolve(savedPosition)
            return
          }
          if (to.hash) {
            resolve({
              el: decodeURIComponent(to.hash),
              top: 0,
            })
            return
          }
          resolve({ top: 0 })
        })
      })
    },
  })

  router.beforeEach(async (to) => {
    if (initialNavigation) {
      initialNavigation = false
      loadedPage = initialPage
      return true
    }

    const url = new URL(to.fullPath, window.location.origin)

    navigation.setNavigating(true)
    try {
      loadedPage = await getPreparedPage(url)
      return true
    } catch {
      window.location.href = url.toString()
      return false
    }
  })

  router.afterEach((_to, _from, failure) => {
    if (failure && isNavigationFailure(failure)) {
      navigation.setNavigating(false)
      return
    }
    onPage(loadedPage)
    navigation.setNavigating(false)
  })

  document.addEventListener('click', async (event) => {
    if (event.defaultPrevented || event.button !== 0) return
    if (event.metaKey || event.ctrlKey || event.shiftKey || event.altKey) return

    const link = (event.target as Element | null)?.closest<HTMLAnchorElement>('a[href]')
    if (!link || link.target === '_blank' || link.hasAttribute('download')) return

    const url = new URL(link.href)
    if (url.origin !== window.location.origin || !isRoutablePath(url.pathname)) return

    event.preventDefault()
    const targetPath = `${url.pathname}${url.search}${url.hash}`
    try {
      await navigateTo(targetPath, window.location.pathname)
    } catch {
      window.location.href = url.toString()
    }
  })

  return router

  // ---- 首页 ↔ Wiki 过场动画（机制对齐 codrops/PageFlipLayout 的 PageTurn）----
  // 侧栏在两种模式间整体切换（导航树 ↔ wiki 树），瞬时跳变突兀；
  // 用 View Transitions API 做「纸页盖入」过场。仅首页↔wiki 导航触发，
  // 其余导航保持原样；不支持 API 时退化为普通跳转。
  async function navigateTo(targetPath: string, fromPath: string) {
    if (!isHomeWikiTransition(fromPath, targetPath) || typeof document.startViewTransition !== 'function') {
      await router.push(targetPath)
      return
    }
    const root = document.documentElement
    const toPathname = new URL(targetPath, window.location.origin).pathname
    // 方向语义：去 wiki = 向前翻（纸页从右缘盖入）；回首页 = 向后翻（从左缘盖入）。
    root.classList.add('gf-page-flip', isWikiPath(toPathname) ? 'gf-page-flip--next' : 'gf-page-flip--prev')
    try {
      const transition = document.startViewTransition(async () => {
        await router.push(targetPath)
      })
      await transition.finished.catch(() => {})
    } finally {
      root.classList.remove('gf-page-flip', 'gf-page-flip--next', 'gf-page-flip--prev')
    }
  }
}

function isHomePath(path: string) {
  return path === '/' || path === ''
}

function isWikiPath(path: string) {
  return path === '/wiki' || path.startsWith('/wiki/')
}

function isHomeWikiTransition(fromPath: string, toPath: string) {
  const to = new URL(toPath, window.location.origin)
  return (isHomePath(fromPath) && isWikiPath(to.pathname)) || (isWikiPath(fromPath) && isHomePath(to.pathname))
}

async function getPreparedPage(url: URL): Promise<PreparedPage> {
  return preparePayload(await fetchPage(url))
}

export async function fetchPage(url: URL): Promise<PagePayload> {
  return client.pages.fetch(url)
}

export async function preparePayload(payload: PagePayload): Promise<PreparedPage> {
  const component = await resolvePageComponent(payload.component)
  if (!component) {
    throw new Error(`Unknown page component: ${payload.component}`)
  }
  return { payload, component }
}

function isRoutablePath(pathname: string) {
  if (
    pathname.startsWith('/api') ||
    pathname.startsWith('/admin') ||
    pathname.startsWith('/assets') ||
    pathname.startsWith('/static') ||
    pathname.startsWith('/file')
  ) {
    return false
  }
  if (pathname === '/robots.txt' || pathname === '/sitemap.xml' || pathname === '/rss.xml') {
    return false
  }
  return !/\.[a-z0-9]+$/i.test(pathname)
}
