import { readFile, writeFile } from 'node:fs/promises'
import { resolve } from 'node:path'
import { pathToFileURL } from 'node:url'

/** Candidate place recognition only: no GeoJSON, overrides, member inference or time parsing. */
export function matchPlaceText(raw, places, campus) {
  const matches = []
  for (const place of new Set(places)) {
    if (!place) continue
    for (let index = raw.indexOf(place); index !== -1; index = raw.indexOf(place, index + place.length)) {
      matches.push({ place, sourceText: place, index, basis: 'literal-place' })
    }
  }
  if (campus === '四平路校区') {
    for (const match of raw.matchAll(/[南北](\d+[A-Za-z]?(?:室|教室|实验室)?)/gu)) {
      const place = match[0][0] === '南' ? '南教学楼' : '北教学楼'
      if (places.includes(place)) matches.push({ place, sourceText: match[0], room: match[1], index: match.index,
        basis: 'user-confirmed-north-south-numeric-room' })
    }
  }
  // Prefer the longest name at overlapping spans, without discarding distinct named members.
  const selected = []
  for (const match of matches.sort((a, b) => b.sourceText.length - a.sourceText.length || a.index - b.index)) {
    if (!selected.some(other => match.index < other.index + other.sourceText.length && other.index < match.index + match.sourceText.length)) selected.push(match)
  }
  return selected.sort((a, b) => a.index - b.index)
}

export function auditPlaceText(table) {
  const rows = [], campuses = {}
  for (const [campus, entries] of Object.entries(table.dictionary)) {
    const places = [...new Set(Object.values(entries).flatMap(result => result.locations.map(member => member.place).filter(Boolean)))]
    const campusRows = Object.entries(entries).map(([raw, result]) => ({ campus, raw,
      matches: matchPlaceText(raw, places, campus),
      extracted: result.locations.map(member => ({ place: member.place, kind: member.kind, detail: member.detail })),
      needsReview: result.needs_review }))
    rows.push(...campusRows)
    campuses[campus] = { inputs: campusRows.length, placeNames: places.length,
      matched: campusRows.filter(row => row.matches.length).length, unmatched: campusRows.filter(row => !row.matches.length).length }
  }
  return { basis: 'Same-campus full candidate place set plus confirmed Siping 南/北 + numeric room; no map or override lookup. A hit is not semantic approval or complete member/time parsing.',
    inputs: rows.length, matched: rows.filter(row => row.matches.length).length, unmatched: rows.filter(row => !row.matches.length).length, campuses, rows }
}

async function main() {
  const [output, input] = process.argv.slice(2)
  if (!output) throw new Error('Usage: audit-place-text.mjs new-report.json [dictionary.json]')
  const table = JSON.parse(await readFile(input ?? new URL('../../src/site/campus-map/data/locations/2026-2027-1.json', import.meta.url), 'utf8'))
  const report = auditPlaceText(table)
  await writeFile(output, JSON.stringify(report, null, 2) + '\n', { flag: 'wx' })
  const escape = value => String(value).replaceAll('|', '\\|').replaceAll('\n', ' ')
  const lines = ['# 未命中地名候选的原始地点', '', report.basis, '',
    `${report.inputs} 条输入中 ${report.matched} 条至少命中一个地名候选，${report.unmatched} 条未命中。命中不是语义准确率，也不是地图覆盖率。`, '']
  for (const [campus, counts] of Object.entries(report.campuses)) {
    lines.push(`## ${campus}（${counts.unmatched} 条）`, '', '| 原始地点 | 字典提取结果（仅供比较） |', '|---|---|')
    for (const row of report.rows.filter(row => row.campus === campus && !row.matches.length)) {
      lines.push(`| ${escape(row.raw)} | ${escape(row.extracted.map(member => (member.place ?? `[${member.kind}]`) + (member.detail ? ` / ${member.detail}` : '')).join('；'))} |`)
    }
    lines.push('')
  }
  await writeFile(output.replace(/\.json$/u, '') + '-unmatched.md', lines.join('\n') + '\n', { flag: 'wx' })
  console.log({ inputs: report.inputs, matched: report.matched, unmatched: report.unmatched, campuses: report.campuses })
}
if (process.argv[1] && import.meta.url === pathToFileURL(resolve(process.argv[1])).href) {
  main().catch(error => { console.error(error.message); process.exitCode = 1 })
}
