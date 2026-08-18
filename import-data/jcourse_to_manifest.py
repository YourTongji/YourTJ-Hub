#!/usr/bin/env python3
"""jcourse SQLite snapshot -> yourtj-hub course-import manifest 包。

用法:
    python3 jcourse_to_manifest.py <snapshot.db> <outdir>

输出 (outdir/):
    manifest-catalog.yaml   目录导入 manifest（courses/instructors/offerings）
    manifest-reviews.yaml   评价导入 manifest（reviews.jsonl，含 rights_approval_ref）
    courses.jsonl           course 目录（行级导出：每行 = 上游 courses 行 = 一个
                            (code, teacher) 卡，携带 teacher_code）
    instructors.jsonl       教师（teacher.teacherCode 去重 + 无法用工号表达的
                            历史教师合成 "syn-{teacher_id}" 行）
    offerings.jsonl         开课（coursedetail 每行一个，按 calendarId 映射学期）
    reviews.jsonl           历史评价（全量导出；学期精确匹配挂真实 offering，
                            匹配不上挂该课程 "其他" 学期 offering，一条不丢）
    mapping_report.json     转换统计与无法映射明细计数

身份模型（对齐 Serverless GROUP BY c.code, c.name, COALESCE(t.name,'')）:
    - course 行 = 一门课的一个 (code, teacher) 身份；卡片/详情/评价按行归属。
      teacher_code 解析优先级：teachers.tid（须在 teacherCode 集合内）>
      teacherName 匹配的 teacherCode（取最小）> 合成 "syn-{teacher_id}"。
    - 同 (code, teacher_code) 多行合并为一行（保留 is_icu=0 优先、id 小），
      被合并行的评价经 reviews.course_id 重定向到保留行，一条不丢。
    - 每个教学班只挂一行（互斥）：(code, teacher_code) 精确 > (courseCode,
      teacher_code) 精确 > code 任意行（id 最小）> courseCode 任意行 >
      newCourseCode/newCode 兜底（当前数据无命中）；offering 行携带
      class_code/class_name 班号信息供 hub 侧按班展示。

已知取舍（详见 mapping_report）:
    - course_aliases (onesystem 课号变体) 不导入：hub importer 仅支持 name 别名，
      课号别名会与 primary_code 语义冲突且大量撞 normalized 唯一键。
    - reviews.semester 为自由文本；能规范化到 calendar 学期且该课程该学期有
      开课记录的评论挂真实 offering；其余（老学期/其他/空文本/学期不匹配/
      无开课记录）统一挂该课程 "其他（历史导入）" 学期 offering，一条不丢。
    - 历史教师（teachers 行无 tid 或 tid 不在 teacherCode 集合）用名字匹配
      teacherCode；匹配不上则合成 "syn-{teacher_id}" 教师行（85 名，
      含 ADMIN/马洪宽/古晞等），保证 course.teacher_code 与 instructors 一一对应。
    - catalog（courses/instructors/offerings）与 reviews 拆分为两个 manifest：
      hub 的 catalog importer 遇到 reviews 文件会拒绝，二者必须分开导入
      （先 catalog 后 reviews，source 一致保证 offering 映射可解析）。
"""

import hashlib
import json
import re
import sqlite3
import sys
from datetime import datetime, timezone
from pathlib import Path

# calendarId -> term code "YYYY-YYYY-N"，与 hub course_term.code 对应
CALENDAR_TERM = {
    118: "2024-2025-1",
    119: "2024-2025-2",
    120: "2025-2026-1",
    121: "2025-2026-2",
    122: "2026-2027-1",
}
_CN_NUM = {
    "一": "1",
    "二": "2",
    "三": "3",
    "四": "4",
    "五": "5",
    "六": "6",
    "七": "7",
    "八": "8",
    "九": "9",
}

_SEM_PATTERNS = [
    (re.compile(r"^(\d{4})-(\d{4})学年第(\d)学期$"), None),
    (re.compile(r"^(\d{4})-(\d{4})第(\d)学期$"), None),
    (re.compile(r"^(\d{4})-(\d{4})学年第([一二三四])学期$"), None),
    (re.compile(r"^(\d{4})-(\d{4})第([一二三四])学期$"), None),
    (re.compile(r"^(\d{4})-(\d{4})-(\d)$"), None),
    (re.compile(r"^(\d{4})-(\d{4}) (\d)$"), None),
    (re.compile(r"^(\d{4})-(\d{4})(学年第)?第二学期$"), 2),
    (re.compile(r"^(\d{4})-(\d{4}) w$"), 2),
    (re.compile(r"^(\d{2})-(\d{2})-(\d)$"), None),
    (re.compile(r"^（(\d{4})-(\d{4})-(\d)）$"), None),
    (re.compile(r"^(\d{2})(\d{2})(\d)$"), None),  # 25262 -> 25-26-2
]


def norm_semester(raw: str):
    """自由文本学期 -> "YYYY-YYYY-N"，识别不了返回 None。

    先按原样匹配（覆盖 "2025-2026 1" / "2025-2026 w" 这类带空格形式），
    匹配不上再去掉所有空格重试（覆盖 "2025-2026第二学期" 等紧凑形式）。
    """
    s0 = (raw or "").strip()
    if not s0:
        return None
    for s in dict.fromkeys((s0, s0.replace(" ", ""))):
        for pat, fixed_n in _SEM_PATTERNS:
            m = pat.match(s)
            if not m:
                continue
            g = m.groups()
            if fixed_n is not None:  # 学年 + 固定学期（无捕获组或固定为 2）
                nums = [x for x in g if x]
                if len(nums) == 2:
                    return f"{nums[0]}-{nums[1]}-{fixed_n}"
                return None
            if len(g) == 2:  # 学年 + 中文数字学期
                return f"{g[0]}-{g[1]}-{_CN_NUM[g[1]]}"
            if len(g) == 3:
                a, b, n = g
                if len(a) == 2:
                    a, b = "20" + a, "20" + b
                if n in _CN_NUM:
                    n = _CN_NUM[n]
                return f"{a}-{b}-{n}"
    return None


def sha256_file(path: Path) -> str:
    h = hashlib.sha256()
    with path.open("rb") as f:
        for chunk in iter(lambda: f.read(1 << 20), b""):
            h.update(chunk)
    return h.hexdigest()


def main() -> int:
    if len(sys.argv) != 3:
        print(__doc__)
        return 2
    src, outdir = Path(sys.argv[1]), Path(sys.argv[2])
    outdir.mkdir(parents=True, exist_ok=True)

    db = sqlite3.connect(f"file:{src}?mode=ro", uri=True)
    db.execute("PRAGMA busy_timeout=5000")
    db.row_factory = sqlite3.Row
    cur = db.cursor()

    report = {
        "courses_raw": 0,
        "courses_merged": 0,
        "courses_dup_teacher_merged": 0,
        "instructors": 0,
        "instructor_natural_key_conflicts": 0,
        "instructor_synthetic": 0,
        "offerings_real": 0,
        "offerings_class_attached": 0,
        "offerings_course_attached": 0,
        "offerings_unattached": 0,
        "offerings_other": 0,
        "courses_with_other_only": 0,
        "reviews_raw": 0,
        "reviews_exact": 0,
        "reviews_other": 0,
        "reviews_unmapped": 0,
        "unmapped_semesters": {},  # 原始 semester 文本 -> 条数
    }

    # ---- teacher_code 解析基础数据 ----
    # teacherCode 集合（teacher 表）
    teacher_code_set = {
        r["teacherCode"].strip()
        for r in cur.execute(
            "SELECT DISTINCT teacherCode FROM teacher WHERE teacherCode IS NOT NULL AND trim(teacherCode) != ''"
        )
    }
    # teachers 全量元数据：主键 id -> meta（course.teacher_id 解析用）
    teacher_meta = {}
    # 工号 tid -> meta（instructor 导出按 teacherCode 补 department/title 用）。
    # teachers.tid 与 teacher.teacherCode 是同一工号体系：两表以工号互相关联。
    teacher_meta_by_tid = {}
    for r in cur.execute("SELECT id, tid, name, title, department FROM teachers"):
        meta = {
            "tid": (r["tid"] or "").strip(),
            "name": (r["name"] or "").strip(),
            "title": r["title"] or "",
            "department": r["department"] or "",
        }
        teacher_meta[r["id"]] = meta
        if meta["tid"]:
            teacher_meta_by_tid[meta["tid"]] = meta
    # teacherName -> 最小 teacherCode（teacher 表；按 (name, dept) 消歧）。
    # teacher 表无 department 列，经 teacherCode = teachers.tid 关联补全；
    # dept 为空（历史数据大量缺省）时退化为纯 name 键，保证无 tid 历史行仍能归并。
    code_by_teacher_name_dept = {}
    for r in cur.execute(
        "SELECT t.teacherName, COALESCE(g.department, '') AS department, MIN(t.teacherCode) AS tcode "
        "FROM teacher t "
        "LEFT JOIN (SELECT tid, department FROM teachers WHERE tid IS NOT NULL AND trim(tid) != '') g "
        "ON g.tid = t.teacherCode "
        "WHERE t.teacherCode IS NOT NULL AND trim(t.teacherCode) != '' "
        "AND t.teacherName IS NOT NULL AND trim(t.teacherName) != '' "
        "GROUP BY t.teacherName, COALESCE(g.department, '')"
    ):
        code_by_teacher_name_dept[
            (r["teacherName"].strip(), (r["department"] or "").strip())
        ] = r["tcode"].strip()
    # teacherName -> 最小 teacherCode（纯姓名键，跨院系）。
    # 历史 teachers 行的 department 多为课程路径垃圾值（如
    # "乌龙茶/必修课/半导体器件原理"），(name, dept) 精确键必然落空；
    # 纯姓名兜底与 Serverless 的 GROUP BY COALESCE(t.name,'') 一致，
    # 且保证同名教师取最小工号（如 490106 李俊 → 12099，与教学班一致）。
    code_by_teacher_name = {}
    for r in cur.execute(
        "SELECT teacherName, MIN(teacherCode) AS tcode FROM teacher "
        "WHERE teacherCode IS NOT NULL AND trim(teacherCode) != '' "
        "AND teacherName IS NOT NULL AND trim(teacherName) != '' "
        "GROUP BY teacherName"
    ):
        code_by_teacher_name[r["teacherName"].strip()] = r["tcode"].strip()

    def resolve_teacher_code(teacher_id):
        """course 行 teacher_id -> teacher_code（None 表示无教师）。"""
        if teacher_id is None:
            return None
        meta = teacher_meta.get(teacher_id)
        if meta is None or not meta["name"]:
            return None
        tid = meta["tid"]
        if tid and tid in teacher_code_set:
            return tid
        # 无 tid 的历史行：优先 (name, dept) 精确（院系干净时按院系消歧）；
        # 落空（历史行院系多为课程路径垃圾值）退化为纯姓名取最小工号，
        # 对齐 Serverless 按名分组语义。
        code = code_by_teacher_name_dept.get((meta["name"], meta["department"]))
        if code is None:
            code = code_by_teacher_name.get(meta["name"])
        if code:
            return code
        return f"syn-{teacher_id}"

    # ---- courses：行级导出，(code, teacher_code) 合并 ----
    raw_courses = []
    for r in cur.execute(
        "SELECT id, code, name, credit, department, teacher_id, is_icu FROM courses ORDER BY id"
    ):
        report["courses_raw"] += 1
        code = (r["code"] or "").strip()
        name = (r["name"] or "").strip()
        if not code or not name:
            continue
        raw_courses.append(
            {
                "id": r["id"],
                "code": code,
                "name": name,
                "department": (r["department"] or "").strip(),
                "credit": float(r["credit"] or 0),
                "teacher_id": r["teacher_id"],
                "is_icu": int(r["is_icu"] or 0),
            }
        )
    groups = {}
    for row in raw_courses:
        row["tcode"] = resolve_teacher_code(row["teacher_id"])
        groups.setdefault((row["code"], row["tcode"]), []).append(row)

    courses_out = []
    course_ext_by_key = {}  # (code, tcode) -> ext
    course_tcode_by_ext = {}  # ext -> tcode（虚拟 offering 教师用）
    course_id_to_ext = {}  # 上游 course_id -> 保留行 ext（含被合并行重定向）
    for key, rows in groups.items():
        # 保留：is_icu=0 优先，其次 id 小
        best = min(rows, key=lambda x: (x["is_icu"], x["id"]))
        ext = str(best["id"])
        course_ext_by_key[key] = ext
        course_tcode_by_ext[ext] = best["tcode"]
        for row in rows:
            course_id_to_ext[row["id"]] = ext
        courses_out.append(
            {
                "id": ext,
                "code": best["code"],
                "name": best["name"],
                "department": best["department"],
                "credit": best["credit"],
                "teacher_code": best["tcode"],
            }
        )
    report["courses_merged"] = len(courses_out)
    report["courses_dup_teacher_merged"] = sum(
        len(rows) - 1 for rows in groups.values() if len(rows) > 1
    )

    # ---- instructors：teacher 表按 teacherCode 去重，department/title 从 teachers 补 ----
    # hub importer 按 (name, dept) 自然键去重；jcourse 存在同名同院系多工号
    # （130 组、164 行），必须用工号消歧：冲突组的教师名加 " (工号)" 后缀，
    # 保证自然键唯一，offering 引用工号不变。
    natural_key_groups = {}
    for r in cur.execute(
        "SELECT DISTINCT teacherCode, teacherName FROM teacher WHERE teacherCode IS NOT NULL AND trim(teacherCode) != '' AND teacherName IS NOT NULL AND trim(teacherName) != ''"
    ):
        code = r["teacherCode"].strip()
        name = (r["teacherName"] or "").strip()
        meta = teacher_meta_by_tid.get(code, {})
        dept = meta.get("department", "")
        natural_key_groups.setdefault((name, dept), set()).add(code)
    conflicted = {
        code
        for codes in natural_key_groups.values()
        if len(codes) > 1
        for code in codes
    }

    instructors_out = []
    seen_codes = set()
    for r in cur.execute(
        "SELECT DISTINCT teacherCode, teacherName FROM teacher WHERE teacherCode IS NOT NULL AND trim(teacherCode) != '' AND teacherName IS NOT NULL AND trim(teacherName) != ''"
    ):
        code = r["teacherCode"].strip()
        name = (r["teacherName"] or "").strip()
        if not name or code in seen_codes:
            continue
        seen_codes.add(code)
        meta = teacher_meta_by_tid.get(code, {})
        dept = meta.get("department", "")
        display_name = f"{name} ({code})" if code in conflicted else name
        instructors_out.append(
            {
                "id": code,
                "name": display_name,
                "department": dept,
                "title": meta.get("title", ""),
            }
        )
    report["instructors"] = len(instructors_out)
    report["instructor_natural_key_conflicts"] = len(conflicted)

    # 合成教师：无法用工号表达的 course 行教师
    synth_codes = sorted(
        {
            tcode
            for tcode in course_tcode_by_ext.values()
            if tcode and tcode.startswith("syn-")
        }
    )
    for sid in synth_codes:
        teacher_id = int(sid.split("-", 1)[1])
        meta = teacher_meta.get(teacher_id, {})
        instructors_out.append(
            {
                "id": sid,
                "name": meta.get("name") or sid,
                "department": meta.get("department", ""),
                "title": meta.get("title", ""),
            }
        )
    report["instructor_synthetic"] = len(synth_codes)
    report["instructors"] = len(instructors_out)

    # ---- offerings：coursedetail 每行一个，term 由 calendarId 映射 ----
    # 互斥挂载链（每班只挂一行）：
    #   s1 (code, teacher) 精确 > s2 (courseCode, teacher) 精确 >
    #   s3 code 任意行（id 最小）> s4 courseCode 任意行 > newCourseCode/newCode 兜底。
    instructor_of_class = {}
    for r in cur.execute(
        "SELECT teachingClassId, teacherCode FROM teacher WHERE teachingClassId IS NOT NULL AND teacherCode IS NOT NULL AND trim(teacherCode) != ''"
    ):
        instructor_of_class.setdefault(r["teachingClassId"], []).append(
            r["teacherCode"].strip()
        )

    campus_names = {
        r["campus"]: r["campusI18n"]
        for r in cur.execute("SELECT campus, campusI18n FROM campus")
    }
    faculty_names = {
        r["faculty"]: r["facultyI18n"]
        for r in cur.execute("SELECT faculty, facultyI18n FROM faculty")
    }

    course_by_code_teacher = {}
    course_by_code = {}
    for key, ext in course_ext_by_key.items():
        code, _ = key
        course_by_code_teacher[key] = ext
        course_by_code.setdefault(code, []).append(ext)

    # reviews 预扫：每门课需要哪些学期（可规范化的）+ 是否有无法规范化的评价
    course_norm_terms = {}
    course_has_unmapped_sem = {}
    for r in cur.execute("SELECT course_id, semester FROM reviews"):
        course_ext = course_id_to_ext.get(r["course_id"])
        if course_ext is None:
            continue
        sem = norm_semester(r["semester"])
        if sem is not None:
            course_norm_terms.setdefault(course_ext, set()).add(sem)
        else:
            course_has_unmapped_sem[course_ext] = True

    offerings_out = []
    offering_by_course_term = {}  # course_ext -> {term: [offering ids]}
    seen_offering = set()
    for r in cur.execute(
        "SELECT id, courseCode, newCourseCode, code, newCode, name, calendarId, campus, faculty FROM coursedetail ORDER BY id"
    ):
        term = CALENDAR_TERM.get(r["calendarId"])
        if term is None:
            continue
        class_id = r["id"]
        instructor_ids = list(dict.fromkeys(instructor_of_class.get(class_id, [])))
        class_code = (r["code"] or "").strip()
        course_code = (r["courseCode"] or "").strip()

        attached = None
        stage = None
        # s1: (code, teacher) 精确
        for tc in instructor_ids:
            if class_code and (class_code, tc) in course_by_code_teacher:
                attached = course_by_code_teacher[(class_code, tc)]
                stage = "class_teacher"
                break
        # s2: (courseCode, teacher) 精确
        if attached is None:
            for tc in instructor_ids:
                if course_code and (course_code, tc) in course_by_code_teacher:
                    attached = course_by_code_teacher[(course_code, tc)]
                    stage = "course_teacher"
                    break
        # s3: code 任意行
        if attached is None and class_code in course_by_code:
            attached = course_by_code[class_code][0]
            stage = "class_any"
        # s4: courseCode 任意行
        if attached is None and course_code in course_by_code:
            attached = course_by_code[course_code][0]
            stage = "course_any"
        # 兜底: newCourseCode / newCode
        if attached is None:
            for f in ("newCourseCode", "newCode"):
                c = (r[f] or "").strip()
                if c in course_by_code:
                    attached = course_by_code[c][0]
                    stage = "course_any"
                    break
        if attached is None:
            report["offerings_unattached"] += 1
            continue
        if stage in ("class_teacher", "class_any"):
            report["offerings_class_attached"] += 1
        else:
            report["offerings_course_attached"] += 1
        offering_id = f"{class_id}-{attached}"
        if offering_id in seen_offering:
            continue
        seen_offering.add(offering_id)
        offerings_out.append(
            {
                "id": offering_id,
                "course_id": attached,
                "term": term,
                "campus": campus_names.get(r["campus"], "") or "",
                "faculty": faculty_names.get(r["faculty"], "") or "",
                "instructor_ids": instructor_ids,
                # 班号信息：教学班 code（如 32000101）与班名（如 01班）。
                # hub importer 落库 class_code/class_name，详情页按班展示。
                "class_code": class_code,
                "class_name": (r["name"] or "").strip(),
            }
        )
        offering_by_course_term.setdefault(attached, {}).setdefault(term, []).append(
            offering_id
        )
    # 每 (course, term) 保留教学班 id 最小的 offering（评价只挂一个，聚合不分散）
    offering_min_by_course_term = {
        course_ext: {
            term: min(ids, key=lambda x: int(x.split("-")[0]))
            for term, ids in terms.items()
        }
        for course_ext, terms in offering_by_course_term.items()
    }
    report["offerings_raw"] = cur.execute(
        "SELECT COUNT(*) FROM coursedetail"
    ).fetchone()[0]
    report["offerings_real"] = len(offerings_out)

    # ---- "其他（历史导入）"学期 offering：有评价但学期匹配不上的课程 ----
    # 生成条件：课程有评价，且存在 (a) 无法规范化的学期文本，或 (b) 规范化后
    # 不在该课程真实 offering 学期集合里，或 (c) 该课程没有任何真实 offering。
    # 行级身份下 other-{course_ext} 即该行专属虚拟班，offering 教师 = 该行教师。
    real_terms_by_course = {
        course_ext: set(terms.keys())
        for course_ext, terms in offering_by_course_term.items()
    }
    other_offering_ids = {}
    other_needed_courses = set(course_norm_terms.keys()) | set(
        course_has_unmapped_sem.keys()
    )
    for course_ext in sorted(other_needed_courses):
        needed = course_norm_terms.get(course_ext, set())
        real = real_terms_by_course.get(course_ext, set())
        if course_has_unmapped_sem.get(course_ext) or not needed.issubset(real):
            offering_id = f"other-{course_ext}"
            other_offering_ids[course_ext] = offering_id
            tcode = course_tcode_by_ext.get(course_ext)
            offerings_out.append(
                {
                    "id": offering_id,
                    "course_id": course_ext,
                    "term": "其他",
                    "campus": "",
                    "faculty": "",
                    "instructor_ids": [tcode] if tcode else [],
                }
            )
    report["offerings_other"] = len(other_offering_ids)
    report["courses_with_other_only"] = sum(
        1 for c in other_offering_ids if c not in offering_by_course_term
    )

    # ---- reviews：全量导出；精确学期匹配挂真实 offering，其余挂该课程 other ----
    # course_id 行级归因：course_id_to_ext 已含被合并行的重定向。
    reviews_out = []
    unmapped_by_sem = {}
    for r in cur.execute(
        "SELECT id, course_id, semester, rating, comment, created_at, approve_count FROM reviews ORDER BY id"
    ):
        report["reviews_raw"] += 1
        course_ext = course_id_to_ext.get(r["course_id"])
        offering_ext = None
        if course_ext is not None:
            sem = norm_semester(r["semester"])
            if sem is not None:
                offering_ext = offering_min_by_course_term.get(course_ext, {}).get(sem)
        if offering_ext is None and course_ext is not None:
            offering_ext = other_offering_ids.get(course_ext)
        if offering_ext is None:
            report["reviews_unmapped"] += 1
            raw = (r["semester"] or "").strip() or "(empty)"
            unmapped_by_sem[raw] = unmapped_by_sem.get(raw, 0) + 1
            continue
        if offering_ext == other_offering_ids.get(course_ext):
            report["reviews_other"] += 1
        else:
            report["reviews_exact"] += 1
        created = datetime.fromtimestamp(r["created_at"], tz=timezone.utc).isoformat()
        reviews_out.append(
            {
                "id": str(r["id"]),
                "offering_external_id": offering_ext,
                "rating": int(r["rating"] or 0),
                "content": r["comment"] or "",
                "created_at": created,
                "legacy_helpful_count": int(r["approve_count"] or 0),
            }
        )
    report["reviews_written"] = len(reviews_out)
    report["unmapped_semesters"] = dict(
        sorted(unmapped_by_sem.items(), key=lambda kv: -kv[1])
    )

    db.close()

    # ---- 写 JSONL ----
    def write_jsonl(name, rows):
        p = outdir / name
        with p.open("w", encoding="utf-8") as f:
            for row in rows:
                f.write(json.dumps(row, ensure_ascii=False) + "\n")
        return p

    p_courses = write_jsonl("courses.jsonl", courses_out)
    p_instructors = write_jsonl("instructors.jsonl", instructors_out)
    p_offerings = write_jsonl("offerings.jsonl", offerings_out)
    p_reviews = write_jsonl("reviews.jsonl", reviews_out)

    def manifest_text(files, counts, rights_approval_ref=""):
        m = "schema_version: 1\nsource: jcourse-snapshot-20260814\n"
        if rights_approval_ref:
            m += f'rights_approval_ref: "{rights_approval_ref}"\n'
        m += "counts:\n"
        for name, n in counts.items():
            m += f"  {name}: {n}\n"
        m += "files:\n"
        for name, digest in files.items():
            m += f"  {name}: {digest}\n"
        return m

    catalog_files = {}
    catalog_counts = {}
    for p in (p_courses, p_instructors, p_offerings):
        catalog_files[p.name] = sha256_file(p)
        catalog_counts[p.name] = sum(1 for _ in p.open(encoding="utf-8"))
    (outdir / "manifest-catalog.yaml").write_text(
        manifest_text(catalog_files, catalog_counts), encoding="utf-8"
    )

    reviews_files = {p_reviews.name: sha256_file(p_reviews)}
    reviews_counts = {p_reviews.name: sum(1 for _ in p_reviews.open(encoding="utf-8"))}
    (outdir / "manifest-reviews.yaml").write_text(
        manifest_text(
            reviews_files,
            reviews_counts,
            rights_approval_ref="本地导入验证 2026-08-14（数据源: /opt/yourtjcourse/data/jcourse.db 一致性快照）",
        ),
        encoding="utf-8",
    )

    (outdir / "mapping_report.json").write_text(
        json.dumps(report, ensure_ascii=False, indent=2), encoding="utf-8"
    )
    print(json.dumps(report, ensure_ascii=False, indent=2))
    print(f"\ncatalog manifest:  {outdir / 'manifest-catalog.yaml'}")
    print(f"reviews manifest:  {outdir / 'manifest-reviews.yaml'}")
    return 0


if __name__ == "__main__":
    sys.exit(main())
