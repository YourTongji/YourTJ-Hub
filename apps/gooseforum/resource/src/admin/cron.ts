// 5 段标准 cron 表达式实时校验（issue #569 定时同步）。
// 与后端 robfig/cron v3 标准解析器对齐的常用子集：分 时 日 月 周 各五段，
// 支持 *、n、n-m、*/step、n-m/step、逗号列表与 JAN..DEC/SUN..SAT 名称（含
// 名称范围 MON-FRI）；另接受 robfig 的 @ 描述符（@daily / @hourly / @every
// <时长> 等）。权威校验仍在服务端保存时进行（SavePkSyncScheduleSettings 用
// ParseStandard），此处仅做输入实时反馈。

const MONTH_NAMES = new Set(['JAN', 'FEB', 'MAR', 'APR', 'MAY', 'JUN', 'JUL', 'AUG', 'SEP', 'OCT', 'NOV', 'DEC'])
const DOW_NAMES = new Set(['SUN', 'MON', 'TUE', 'WED', 'THU', 'FRI', 'SAT'])
// 分钟/小时/日/月/周 各字段数值范围；日和月从 1 开始。
const FIELD_LIMITS: number[] = [59, 23, 31, 12, 6]
const FIELD_MINIMUMS: number[] = [0, 0, 1, 1, 0]

// atDescriptor 校验 robfig 的 @ 描述符（@yearly/@annually/@monthly/@weekly/
// @daily/@midnight/@hourly/@every <dur>）。@every 的时长形如 1h30m / 30m / 500ms；Go duration 不支持天（d）。
function isAtDescriptor(spec: string): boolean {
  const trimmed = spec.trim()
  if (/^@(yearly|annually|monthly|weekly|daily|midnight|hourly)$/.test(trimmed)) return true
  return /^@every\s+(?:\d+(?:\.\d+)?(?:ns|us|µs|μs|ms|s|m|h))+$/.test(trimmed)
}

// isValidField 校验单个 cron 字段；names 非空时（月/周）额外接受全名与全名范围。
function isValidField(field: string, minimum: number, limit: number, names: Set<string> | null): boolean {
  for (const part of field.split(',')) {
    if (names && names.has(part.toUpperCase())) continue
    if (names) {
      const nameRange = /^([A-Za-z]{3})-([A-Za-z]{3})(?:\/(\d+))?$/.exec(part)
      if (nameRange) {
        const nameList = [...names]
        const start = nameList.indexOf(nameRange[1].toUpperCase())
        const end = nameList.indexOf(nameRange[2].toUpperCase())
        if (start < 0 || end < 0 || start > end) return false
        if (nameRange[3] !== undefined && Number(nameRange[3]) < 1) return false
        continue
      }
    }
    if (/^\*\/\d+$/.test(part)) {
      const step = Number(part.slice(2))
      if (!Number.isInteger(step) || step < 1) return false
      continue
    }
    const m = /^(\d+)(?:-(\d+))?(\/\d+)?$/.exec(part)
    if (!m) return false
    const start = Number(m[1])
    const end = m[2] !== undefined ? Number(m[2]) : start
    const step = m[3] !== undefined ? Number(m[3].slice(1)) : 1
    if (step < 1 || start < minimum || start > limit || end > limit || start > end) return false
  }
  return true
}

export function isValidCron5Field(spec: string): boolean {
  const trimmed = spec.trim()
  if (!trimmed) return false
  if (isAtDescriptor(trimmed)) return true
  const fields = trimmed.split(/\s+/)
  if (fields.length !== 5) return false
  for (let i = 0; i < 5; i++) {
    if (fields[i] === '*') continue
    // 月/周 段允许名称（robfig ParseMonth/ParseDow）；周 7 等同周日但后端标准
    // 解析器不接收（对齐后端，不额外放行）。
    const names = i === 3 ? MONTH_NAMES : i === 4 ? DOW_NAMES : null
    if (!isValidField(fields[i], FIELD_MINIMUMS[i], FIELD_LIMITS[i], names)) return false
  }
  return true
}
