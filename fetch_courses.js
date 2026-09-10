/* ============================================================
 * 同济 1.tongji.edu.cn EnquiryOfCourses 选课信息抓取（Node 版）
 *
 * 用法：
 *   node fetch_courses.js options    # 打印所有可选真实选项（学期/开课学院/课程性质/校区/培养层次/学习形式）
 *   node fetch_courses.js            # 按下方 FILTERS 抓取 -> JSON + CSV
 *
 * 鉴权：通过环境变量 ONESYSTEM_X_TOKEN 提供 sessionStorage 中的 sessionid；
 *       可选用 ONESYSTEM_COOKIE 提供整段 Cookie。不要把密钥写入源码。
 * ============================================================ */
'use strict';

const fs = require('fs');

const CONFIG = {
  HOST: 'https://1.tongji.edu.cn',

  X_TOKEN: process.env.ONESYSTEM_X_TOKEN || '',

  /* 可选：整段 Cookie；一般有 X_TOKEN 即可，留空也行。 */
  COOKIE: process.env.ONESYSTEM_COOKIE || '',

  /* 筛选条件：全部留空 = 拉全部。真实选项值用 node fetch_courses.js options 查询后回填。 */
  FILTERS: {
    calendarId: 'current',   // 学期：'current'=当前学期，或填 options 打印的 id，如 '122'
    newCourseCode: '',       // 课程编码
    teachClassCode: '',      // 课程代码/教学班号
    courseName: '',          // 课程名称（模糊）
    teacherName: '',         // 教师姓名（模糊）
    faculty: '',             // 开课学院 deptCode，如 '000014'（研究生院）
    nature: '',              // 课程性质：1公共课 2专业课 3必修环节
    campu: '',               // 校区：1四平路 2沪北 3嘉定 4沪西 5其他（接口拼写即 campu）
    trainingLevel: '',       // 培养层次：4硕士 6博士
    formLearning: '',        // 学习形式：1全日制 2非全日制 4全日制&非全日制
  },

  PAGE_SIZE: 1000,   // 单页条数（接口上限 1000）
  DELAY_MS: 200,     // 每页间隔，避免压测
};

/* ---------------- 请求封装 ---------------- */

async function api(pathname, opts = {}) {
  const headers = {
    'X-Token': CONFIG.X_TOKEN,
    'X-Requested-With': 'XMLHttpRequest',
    'Accept': 'application/json, text/plain, */*',
    'Referer': CONFIG.HOST + '/EnquiryOfCourses',
  };
  if (CONFIG.COOKIE) headers['Cookie'] = CONFIG.COOKIE;
  if (opts.body) headers['Content-Type'] = 'application/json;charset=UTF-8';

  const res = await fetch(CONFIG.HOST + pathname, {
    method: opts.method || (opts.body ? 'POST' : 'GET'),
    headers,
    body: opts.body ? JSON.stringify(opts.body) : undefined,
  });
  const text = await res.text();
  let json;
  try { json = JSON.parse(text); } catch (e) { throw new Error('非 JSON 响应 HTTP ' + res.status); }
  if (res.status === 401 || /sessionid is not exist/i.test(text)) {
    throw new Error('401 未授权：X-Token 无效或已过期，请重新获取 sessionid');
  }
  if (json.code != null && json.code !== 200) {
    throw new Error('接口返回 code=' + json.code);
  }
  return json;
}

/* ---------------- 选项（学期 / 学院 / 字典） ---------------- */

async function fetchCalendars() {
  const j = await api('/api/baseresservice/schoolCalendar/list');
  return (j.data || []).map(c => ({ id: String(c.id), name: c.fullName || String(c.id) }));
}

async function currentTerm() {
  const j = await api('/api/baseresservice/schoolCalendar/currentTermCalendar?flag=0');
  const d = (j.data && j.data.schoolCalendar) || j.data || {};
  return { id: String(d.id), name: d.simpleName || d.name || String(d.id) };
}

async function fetchFaculties() {
  const j = await api('/api/electionservice/elcMutualCourses/findDept?manageDept=0&type=1&virtualDept=0');
  const data = Array.isArray(j.data) ? j.data : Object.values(j.data || {});
  return data.map(x => ({ code: String(x.deptCode), name: x.deptName }));
}

async function fetchDicts() {
  const j = await api('/api/commonservice/dictionary/query', {
    body: { lang: 'cn', type: 'allChild', keys: ['X_KCXZ', 'X_XQ', 'X_PYCC', 'K_XXXS'] },
  });
  return j.data || {};
}

async function cmdOptions() {
  const cur = await currentTerm();
  console.log('== 当前学期 ==\n  ' + cur.id + '  ' + cur.name + '\n');
  console.log('== 全部学期（FILTERS.calendarId 填 id 或 current）==');
  for (const c of await fetchCalendars()) console.log('  ' + c.id + (c.id === cur.id ? '  <- 当前' : '') + '  ' + c.name);
  console.log('\n== 开课学院（FILTERS.faculty 填 deptCode）==');
  for (const f of await fetchFaculties()) console.log('  ' + f.code + '  ' + f.name);
  const label = { X_KCXZ: '课程性质 nature', X_XQ: '校区 campu', X_PYCC: '培养层次 trainingLevel', K_XXXS: '学习形式 formLearning' };
  const dicts = await fetchDicts();
  for (const k of Object.keys(label)) {
    console.log('\n-- ' + label[k] + ' --');
    for (const [code, name] of Object.entries(dicts[k] || {})) console.log('    ' + code + '  ' + name);
  }
}

/* ---------------- 抓取 ---------------- */

async function fetchAll(calendarId) {
  const rows = [];
  let page = 1;
  let total = null;
  for (;;) {
    const condition = { ...CONFIG.FILTERS, calendarId: String(calendarId) };
    const body = { pageNum_: page, pageSize_: CONFIG.PAGE_SIZE, condition };
    const j = await api('/api/electionservice/student/round/allArrangementCourses', { body });
    const data = j.data || {};
    const list = Array.isArray(data.list) ? data.list : [];
    rows.push(...list);
    total = data.total_ != null ? data.total_ : (data.total != null ? data.total : null);
    process.stdout.write(`\r  第 ${page} 页 +${list.length} 条（累计 ${rows.length}${total != null ? ' / ' + total : ''}）   `);
    if (!list.length) break;
    if (total != null && rows.length >= total) break;
    if (list.length < CONFIG.PAGE_SIZE) break;
    page++;
    await new Promise(r => setTimeout(r, CONFIG.DELAY_MS));
  }
  process.stdout.write('\n');
  return rows;
}

/* ---------------- 输出 ---------------- */

// 只导出课程查询所需的非个人字段；上游新增字段默认不落盘，避免快照意外
// 扩大为选课关系/联系方式等个人数据。
const SAFE_FIELDS = new Set([
  'calendarId', 'id', 'code', 'name', 'courseLabelId', 'courseLabelName',
  'assessmentMode', 'assessmentModeI18n', 'period', 'weekHour', 'campus',
  'campusI18n', 'number', 'elcNumber', 'startWeek', 'endWeek', 'courseCode',
  'courseName', 'credits', 'credit', 'teachingLanguage', 'teachingLanguageI18n',
  'faculty', 'facultyI18n', 'newCourseCode', 'newCode', 'arrangeInfo',
  'teacherList', 'majorList', 'timeTableList', 'trainingLevel', 'formLearning',
  'nature', 'campu', 'teachClassCode', 'capacity', 'selectedNumber', 'totalNumber',
]);
const PRIVATE_FIELD = /(?:student|selection|selectedStudent|contact|phone|mobile|email|identity|idCard|password|token|cookie)/i;

function sanitizeValue(value, fieldName) {
  if (Array.isArray(value)) return value.map(item => sanitizeValue(item, fieldName));
  if (!value || typeof value !== 'object') return value;
  const out = {};
  for (const [key, item] of Object.entries(value)) {
    if (PRIVATE_FIELD.test(key)) throw new Error('响应包含禁止导出的隐私字段: ' + key);
    out[key] = sanitizeValue(item, key);
  }
  return out;
}

function sanitizeRows(rows) {
  return rows.map(row => {
    const out = {};
    for (const [key, value] of Object.entries(row)) {
      if (PRIVATE_FIELD.test(key)) throw new Error('响应包含禁止导出的隐私字段: ' + key);
      if (SAFE_FIELDS.has(key)) out[key] = sanitizeValue(value, key);
    }
    return out;
  });
}

function toCSV(rows) {
  const keys = [];
  for (const r of rows) for (const k of Object.keys(r)) if (!keys.includes(k)) keys.push(k);
  const esc = v => {
    if (v == null) return '';
    let s = (typeof v === 'object') ? JSON.stringify(v) : String(v);
    if (/^[=+\-@\t\r\n]/.test(s)) s = "'" + s;
    return /[",\n\r]/.test(s) ? '"' + s.replace(/"/g, '""') + '"' : s;
  };
  const lines = [keys.map(esc).join(',')];
  for (const r of rows) lines.push(keys.map(k => esc(r[k])).join(','));
  return '\uFEFF' + lines.join('\r\n'); // BOM：Excel 直接打开不乱码
}

/* ---------------- 入口 ---------------- */

async function main() {
  if (!CONFIG.X_TOKEN || /^(?:在此粘贴|x{8,})/i.test(CONFIG.X_TOKEN)) {
    throw new Error('请通过 ONESYSTEM_X_TOKEN 环境变量提供 sessionid（不要写入源码）');
  }
  const cmd = process.argv[2];
  if (cmd === 'calendar') { const cur = await currentTerm(); console.log('当前学期: ' + cur.id + '  ' + cur.name); for (const c of await fetchCalendars()) console.log('  ' + c.id + '  ' + c.name); return; }
  if (cmd === 'options') { await cmdOptions(); return; }

  const raw = String(CONFIG.FILTERS.calendarId || 'current');
  const cal = raw === 'current' ? (await currentTerm()).id : raw;
  console.log('抓取学期: ' + cal);
  console.log('筛选条件: ' + JSON.stringify(CONFIG.FILTERS));

  const rows = await fetchAll(cal);
  const ts = new Date().toISOString().replace(/[:T]/g, '-').slice(0, 19);
  const base = `courses-${cal}-${ts}`;
  const safeRows = sanitizeRows(rows);
  fs.writeFileSync(base + '-sanitized.json', JSON.stringify(safeRows, null, 1), 'utf8');
  fs.writeFileSync(base + '.csv', toCSV(safeRows), 'utf8');
  console.log(`完成：${safeRows.length} 条 -> ${base}-sanitized.json / ${base}.csv`);
}

main().catch(e => { console.error('错误: ' + e.message); process.exit(1); });
