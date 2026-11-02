/** Keep only failed ZIP entries, copying compressed bytes without decoding images.
 * Central-directory offsets are rewritten; the server still validates the archive.
 * ZIP record layout: https://pkware.cachefly.net/webdocs/casestudies/APPNOTE.TXT (4.3).
 */
export async function failedStickerArchive(file: File, names: string[]): Promise<File> {
  const bytes = new Uint8Array(await file.arrayBuffer())
  const view = new DataView(bytes.buffer)
  const invalid = () => new Error('Invalid or ambiguous retry archive')
  let end = bytes.length - 22
  while (end >= Math.max(0, bytes.length - 65557)) {
    if (view.getUint32(end, true) === 0x06054b50 && end + 22 + view.getUint16(end + 20, true) === bytes.length) break
    end--
  }
  if (end < 0 || end < bytes.length - 65557) throw invalid()
  const count = view.getUint16(end + 10, true)
  const centralStart = view.getUint32(end + 16, true)
  if (view.getUint32(end + 4, true) !== 0 || view.getUint16(end + 8, true) !== count || count === 0xffff || centralStart >= end) throw invalid()
  const records: { name: string, start: number, central: Uint8Array<ArrayBuffer> }[] = []
  let cursor = centralStart
  for (let index = 0; index < count; index++) {
    if (cursor + 46 > end || view.getUint32(cursor, true) !== 0x02014b50) throw invalid()
    const nameLength = view.getUint16(cursor + 28, true)
    const length = 46 + nameLength + view.getUint16(cursor + 30, true) + view.getUint16(cursor + 32, true)
    const start = view.getUint32(cursor + 42, true)
    if (cursor + length > end || start + 30 > centralStart || view.getUint32(start, true) !== 0x04034b50) throw invalid()
    records.push({ name: new TextDecoder('utf-8', { fatal: true }).decode(bytes.subarray(cursor + 46, cursor + 46 + nameLength)), start, central: bytes.slice(cursor, cursor + length) })
    cursor += length
  }
  if (cursor !== end || cursor - centralStart !== view.getUint32(end + 12, true)) throw invalid()
  records.sort((a, b) => a.start - b.start)
  const wanted = new Set(names)
  const locals: Uint8Array<ArrayBuffer>[] = []
  const directory: Uint8Array<ArrayBuffer>[] = []
  let offset = 0
  for (const [index, record] of records.entries()) {
    if (!wanted.has(record.name)) continue
    if (records.filter(item => item.name === record.name).length !== 1) throw invalid()
    const next = records[index + 1]?.start ?? centralStart
    if (next <= record.start) throw invalid()
    const local = bytes.slice(record.start, next)
    new DataView(record.central.buffer).setUint32(42, offset, true)
    locals.push(local)
    directory.push(record.central)
    offset += local.length
    wanted.delete(record.name)
  }
  if (wanted.size || !directory.length) throw invalid()
  const footer = new Uint8Array(22)
  const footerView = new DataView(footer.buffer)
  footerView.setUint32(0, 0x06054b50, true)
  footerView.setUint16(8, directory.length, true)
  footerView.setUint16(10, directory.length, true)
  footerView.setUint32(12, directory.reduce((size, record) => size + record.length, 0), true)
  footerView.setUint32(16, offset, true)
  return new File([...locals, ...directory, footer], file.name, { type: 'application/zip' })
}
