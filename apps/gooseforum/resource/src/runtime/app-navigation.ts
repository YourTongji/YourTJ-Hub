// SPA 导航桥：runtime 模块（browser-notification 等）不依赖 vue-router，
// 由应用入口（site/main.ts）注册 push 回调；未注册时调用方回退整页跳转。
type AppNavigator = (path: string) => void
let navigator: AppNavigator | null = null
export function registerAppNavigator(fn: AppNavigator) {
  navigator = fn
}
export function navigateAppTo(path: string): boolean {
  if (navigator) {
    navigator(path)
    return true
  }
  return false
}
