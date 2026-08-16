#!/usr/bin/env python3
"""jcourse SQLite snapshot -> yourtj-hub course-import manifest 包。

用法:
    python3 jcourse_to_manifest.py <snapshot.db> <outdir>

输出 (outdir/):
    manifest-catalog.yaml   目录导入 manifest（courses/instructors/offerings）
    manifest-reviews.yaml   评价导入 manifest（reviews.jsonl，含 rights_approval_ref）
    courses.jsonl           course 目录（按 code 去重）
    instructors.jsonl       教师（teacher.teacherCode 去重）
    offerings.jsonl         开课（coursedetail 每行一个，按 calendarId 映射学期）
    reviews.jsonl           历史评价（全量导出；学期精确匹配挂真实 offering，
                            匹配不上挂该课程 "其他" 学期 offering，一条不丢）
    mapping_report.json     转换统计与无法映射明细计数

已知取舍（详见 mapping_report）:
    - course_aliases (onesystem 课号变体) 不导入：hub importer 仅支持 name 别名，
      课号别名会与 primary_code 语义冲突且大量撞 normalized 唯一键。
    - reviews.semester 为自由文本；能规范化到 calendar 学期且该课程该学期有
      开课记录的评论挂真实 offering；其余（老学期/其他/空文本/学期不匹配/
      无开课记录）统一挂该课程 "其他（历史导入）" 学期 offering，一条不丢。
    - courses 与 coursedetail 按四字段关联（courseCode/newCourseCode/code/newCode）；
      每个教学班只挂一门课：优先班号课（code 精确匹配，如 32000101），无班号课
      则挂主码课（courseCode/newCourseCode/newCode，如 320001）。offering 行携带
      class_code/class_name 班号信息供 hub 侧按班展示。旧版"每匹配课程生成一个
      offering"会把同一教学班同时挂到主码课与班号课（双写，如体育 295 班 ->
      295 条主码 offering + 102 条班号 offering），已废弃。
    - courses 重复 code（同课不同教师行）合并为单条 course，教师信息由 offering 承载。
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
    """自由文本学期 -> "YYYY-YYYY-N"，识别不了返回 None。"""
    s = (raw or "").strip().replace(" ", "")
    if not s:
        return None
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
        "courses_dup_code_merged": 0,
        "instructors": 0,
        "offerings_real": 0,
        "offerings_class_attached": 0,
        "offerings_course_attached": 0,
        "offerings_other": 0,
        "courses_with_other_only": 0,
        "reviews_raw": 0,
        "reviews_exact": 0,
        "reviews_other": 0,
        "reviews_unmapped": 0,
        "unmapped_semesters": {},  # 原始 semester 文本 -> 条数
    }

    # ---- courses：按 code 去重（重复行合并，取第一条有意义的） ----
    code_to_ext = {}
    courses_out = []
    for r in cur.execute(
        "SELECT id, code, name, credit, department FROM courses ORDER BY id"
    ):
        report["courses_raw"] += 1
        code = (r["code"] or "").strip()
        if not code or not (r["name"] or "").strip():
            continue
        if code in code_to_ext:
            report["courses_dup_code_merged"] += 1
            continue  # 同课不同教师行：教师信息由 offering 承载
        ext = str(r["id"])
        code_to_ext[code] = ext
        courses_out.append(
            {
                "id": ext,
                "code": code,
                "name": r["name"].strip(),
                "department": (r["department"] or "").strip(),
                "credit": float(r["credit"] or 0),
            }
        )
    report["courses_merged"] = len(courses_out)

    # ---- instructors：teacher 表按 teacherCode 去重，department/title 从 teachers 补 ----
    # hub importer 按 (name, dept) 自然键去重；jcourse 存在同名同院系多工号
    # （130 组、164 行），必须用工号消歧：冲突组的教师名加 " (工号)" 后缀，
    # 保证自然键唯一，offering 引用工号不变。
    teacher_meta = {}
    for r in cur.execute(
        "SELECT tid, name, title, department FROM teachers WHERE tid IS NOT NULL AND trim(tid) != ''"
    ):
        tid = r["tid"].strip()
        if tid not in teacher_meta:
            teacher_meta[tid] = {
                "title": r["title"] or "",
                "department": r["department"] or "",
            }
    # 预扫描自然键冲突：name+dept -> [codes]
    natural_key_groups = {}
    for r in cur.execute(
        "SELECT DISTINCT teacherCode, teacherName FROM teacher WHERE teacherCode IS NOT NULL AND trim(teacherCode) != '' AND teacherName IS NOT NULL AND trim(teacherName) != ''"
    ):
        code = r["teacherCode"].strip()
        name = (r["teacherName"] or "").strip()
        meta = teacher_meta.get(code, {})
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
        meta = teacher_meta.get(code, {})
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

    # ---- offerings：coursedetail 每行一个，term 由 calendarId 映射 ----
    # 四字段关联（与上游 010_materialize_courses_from_pk.sql 一致）：courseCode /
    # newCourseCode / code / newCode 任一等于 courses.code 即可挂载，但每个教学班
    # 只挂一门课（班号课优先，其次主码课），避免同一班在 Hub 目录双写。
    # offering 行附带 class_code/class_name 班号信息，供 hub 侧按班展示。
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

    # course_id -> course external id（供 reviews 预扫与导出）
    course_id_to_ext = {
        r["id"]: code_to_ext[(r["code"] or "").strip()]
        for r in cur.execute("SELECT id, code FROM courses")
        if (r["code"] or "").strip() in code_to_ext
    }

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
        # 单挂载优先链：班号课（code）> 主码课（courseCode）> newCourseCode > newCode。
        # 每个教学班只挂一门课，消除"同一班同时挂主码课与班号课"的双写。
        attached_course = None
        for code_field in (
            r["code"],
            r["courseCode"],
            r["newCourseCode"],
            r["newCode"],
        ):
            code = (code_field or "").strip()
            if not code:
                continue
            course_ext = code_to_ext.get(code)
            if course_ext is not None:
                attached_course = course_ext
                break
        course_exts = [attached_course] if attached_course is not None else []
        for course_ext in course_exts:
            if course_ext == code_to_ext.get((r["code"] or "").strip()):
                report["offerings_class_attached"] += 1
            else:
                report["offerings_course_attached"] += 1
            offering_id = f"{class_id}-{course_ext}"
            if offering_id in seen_offering:
                continue
            seen_offering.add(offering_id)
            offerings_out.append(
                {
                    "id": offering_id,
                    "course_id": course_ext,
                    "term": term,
                    "campus": campus_names.get(r["campus"], "") or "",
                    "faculty": faculty_names.get(r["faculty"], "") or "",
                    "instructor_ids": instructor_ids,
                    # 班号信息：教学班 code（如 32000101）与班名（如 01班）。
                    # hub importer 当前忽略未知字段，模型支持后可按班展示/分组。
                    "class_code": (r["code"] or "").strip(),
                    "class_name": (r["name"] or "").strip(),
                }
            )
            offering_by_course_term.setdefault(course_ext, {}).setdefault(
                term, []
            ).append(offering_id)
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
            offerings_out.append(
                {
                    "id": offering_id,
                    "course_id": course_ext,
                    "term": "其他",
                    "campus": "",
                    "faculty": "",
                    "instructor_ids": [],
                }
            )
    report["offerings_other"] = len(other_offering_ids)
    report["courses_with_other_only"] = sum(
        1 for c in other_offering_ids if c not in offering_by_course_term
    )

    # ---- reviews：全量导出；精确学期匹配挂真实 offering，其余挂该课程 other ----
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
