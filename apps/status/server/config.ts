import { createHash } from 'node:crypto'
import { uuid } from './http'
export type Provider = 'uptime' | 'komari' | 'umami'
export type Config = { enabled: boolean; uptime: { url: string; id: string }; komari: { url: string; id: string }; umami: { url: string; id: string } }
export function loadConfig(env: Record<string, string | undefined> = process.env): Config {
  return { enabled: env.STATUS_ENABLED === 'true', uptime: { url: env.UPTIME_URL ?? '', id: env.UPTIME_SLUG ?? '' }, komari: { url: env.KOMARI_URL ?? '', id: env.KOMARI_NODE_ID ?? '' }, umami: { url: env.UMAMI_URL ?? '', id: env.UMAMI_SHARE_ID ?? '' } }
}
export function configured(config: Config, provider: Provider) { return config.enabled && !!(config[provider].url || config[provider].id) }
export function origin(config: Config, provider: Provider): string {
  const { url, id } = config[provider]
  const parsed = new URL(url)
  if (parsed.protocol !== 'https:' || parsed.username || parsed.password || parsed.search || parsed.hash || (parsed.pathname !== '/' && parsed.pathname !== '') ||
      (provider === 'komari' ? !uuid(id) : provider === 'umami' ? !/^[a-zA-Z0-9]{1,100}$/.test(id) : !/^[a-z0-9_-]{1,100}$/.test(id))) throw new Error('Invalid public source configuration')
  return parsed.origin
}
export function cacheKey(config: Config, provider: Provider, part: string) {
  // Changing or revoking a source cannot expose the previous source's retained data.
  const fingerprint = createHash('sha256').update(JSON.stringify(config[provider])).digest('hex').slice(0, 24)
  return `${provider}/${fingerprint}/${part}`
}
