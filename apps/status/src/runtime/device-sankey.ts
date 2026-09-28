import { sankey, sankeyLinkHorizontal, type SankeyNode, type SankeyLink } from 'd3-sankey'
import type { StatusDevices } from '@/types'

export type Dimension = 'device' | 'os' | 'browser'
type NodeData = { id: string; dimension: Dimension; category: string }
type LinkData = { id: string }
export type DeviceNode = SankeyNode<NodeData, LinkData>
export type DeviceLink = SankeyLink<NodeData, LinkData>
export const linkPath = sankeyLinkHorizontal<NodeData, LinkData>()

export function deviceGraph(data: StatusDevices, width: number, dimensions: Dimension[]) {
  const nodes = new Map<string, NodeData>()
  const links = new Map<string, { id: string; source: string; target: string; value: number }>()
  // Fold small OS/browser categories into Other before creating both sides of every link.
  // This preserves flow conservation without losing visitors or inventing relationships.
  const keep = new Map<Dimension, Set<string>>()
  for (const dimension of dimensions) {
    const totals = new Map<string, number>()
    for (const row of data.rows) totals.set(row[dimension], (totals.get(row[dimension]) ?? 0) + row.visitors)
    keep.set(dimension, new Set([...totals].sort((a, b) => b[1] - a[1]).filter(([name, n], i) => dimension === 'device' || name === 'unknown' || name === 'yourtj-app' || (i < 5 && n >= data.visitors * .015)).map(([name]) => name)))
  }
  for (const row of data.rows) {
    if (!row.visitors) continue
    const ids = dimensions.map(dimension => {
      const category = keep.get(dimension)!.has(row[dimension]) ? row[dimension] : 'other'
      const id = `${dimension}:${category}`
      nodes.set(id, { id, dimension, category })
      return id
    })
    for (let i = 1; i < ids.length; i++) {
      const source = ids[i - 1], target = ids[i], id = `${source}/${target}`
      links.set(id, { id, source, target, value: (links.get(id)?.value ?? 0) + row.visitors })
    }
  }
  if (!links.size) return { nodes: [] as DeviceNode[], links: [] as DeviceLink[] }
  const compact = dimensions.length === 2
  return sankey<NodeData, LinkData>()
    .nodeId(d => d.id).nodeWidth(8).nodePadding(30)
    .nodeSort((a, b) => (b.value ?? 0) - (a.value ?? 0) || a.id.localeCompare(b.id))
    .extent([[compact ? 2 : 102, 14], [width - (compact ? 2 : 114), 306]])
    ({ nodes: [...nodes.values()], links: [...links.values()] })
}
