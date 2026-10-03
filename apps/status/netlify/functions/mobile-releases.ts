import type { Config } from '@netlify/functions'
import { serveMobileReleases } from '../../server/mobile-releases'

export default (request: Request) => serveMobileReleases(request)
export const config: Config = { path: '/mobile/releases.json' }
