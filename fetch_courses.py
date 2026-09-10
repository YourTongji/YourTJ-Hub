#!/usr/bin/env python3
# -*- coding: utf-8 -*-
"""
同济 1.tongji.edu.cn EnquiryOfCourses 选课数据抓取（按页面筛选条件）

用法：
    python fetch_courses.py options    # 打印所有可选真实选项（学期/开课学院/课程性质/校区/培养层次/学习形式）
    python fetch_courses.py            # 按下方 FILTERS 拉数据 -> JSON + CSV

鉴权：
    通过环境变量 ONESYSTEM_X_TOKEN 提供 sessionStorage 中的 sessionid；
    可选用 ONESYSTEM_COOKIE 提供整段 Cookie。不要把密钥写入源码。
"""

import json
import time
import sys
import os
import csv
import datetime
import re
import urllib.request
import urllib.error

# ======================= 需要你改的部分 =======================

X_TOKEN = os.environ.get("ONESYSTEM_X_TOKEN", "")
COOKIE = os.environ.get("ONESYSTEM_COOKIE", "")

# 筛选条件：全部留空 = 拉全部。只填你想筛的项，其余留空字符串。
# 每个选项的"真实值"可用  python fetch_courses.py options  查出来再回填。
FILTERS = {
    # 学期/学年（必填其一）：'current'=当前学期；或写 options 打印的学期 id，如 '122'
    "calendarId": "current",

    # 课程编码（页面“课程编码”输入框，对应接口 newCourseCode）
    "newCourseCode": "",
    # 课程代码/教学班号（页面可选显示的 courseCode，对应接口 teachClassCode）
    "teachClassCode": "",
    # 课程名称（模糊匹配）
    "courseName": "",
    # 教师姓名（模糊匹配）
    "teacherName": "",
    # 开课学院（填 deptCode，如研究生院='000014'，数学科学学院='...'，用 options 查）
    "faculty": "",
    # 课程性质（X_KCXZ：'1'公共课 / '2'专业课 / '3'必修环节）
    "nature": "",
    # 校区（X_XQ：'1'四平路 / '2'沪北 / '3'嘉定 / '4'沪西 / '5'其他）
    "campu": "",
    # 培养层次（X_PYCC：'4'硕士 / '6'博士）
    "trainingLevel": "",
    # 学习形式（K_XXXS：'1'全日制 / '2'非全日制 / '4'全日制&非全日制）
    "formLearning": "",
}

# ======================= 以下一般不用改 =======================

BASE = "https://1.tongji.edu.cn"
PAGE_SIZE = 1000
DELAY_MS = 0.2
OUT_DIR = "."

# 页面 queryForm 的键名 -> 本脚本 FILTERS 键名（campus 在接口里拼作 campu，照抄页面）
_API_COND = {
    "calendarId": "calendarId",
    "newCourseCode": "newCourseCode",
    "teachClassCode": "teachClassCode",
    "courseName": "courseName",
    "teacherName": "teacherName",
    "faculty": "faculty",
    "nature": "nature",
    "campu": "campu",
    "trainingLevel": "trainingLevel",
    "formLearning": "formLearning",
}
_DICT_KEYS = ["X_KCXZ", "X_XQ", "X_PYCC", "K_XXXS"]
_DICT_LABEL = {"X_KCXZ": "课程性质(nature)", "X_XQ": "校区(campu)",
               "X_PYCC": "培养层次(trainingLevel)", "K_XXXS": "学习形式(formLearning)"}
_SAFE_FIELDS = {
    "calendarId", "id", "code", "name", "courseLabelId", "courseLabelName",
    "assessmentMode", "assessmentModeI18n", "period", "weekHour", "campus",
    "campusI18n", "number", "elcNumber", "startWeek", "endWeek", "courseCode",
    "courseName", "credits", "credit", "teachingLanguage", "teachingLanguageI18n",
    "faculty", "facultyI18n", "newCourseCode", "newCode", "arrangeInfo",
    "teacherList", "majorList", "timeTableList", "trainingLevel", "formLearning",
    "nature", "campu", "teachClassCode", "capacity", "selectedNumber", "totalNumber",
}
_PRIVATE_FIELD_PARTS = ("student", "selection", "contact", "phone", "mobile", "email",
                       "identity", "idcard", "password", "token", "cookie")


def http_json(path, payload=None):
    headers = {
        "X-Token": X_TOKEN,
        "Accept": "application/json, text/plain, */*",
        "Referer": BASE + "/EnquiryOfCourses",
        "User-Agent": "Mozilla/5.0 (Windows NT 10.0; Win64; x64)",
    }
    if COOKIE:
        headers["Cookie"] = COOKIE
    data = None
    if payload is not None:
        headers["Content-Type"] = "application/json;charset=UTF-8"
        data = json.dumps(payload, ensure_ascii=False).encode("utf-8")
    req = urllib.request.Request(BASE + path, data=data, headers=headers, method="POST" if payload is not None else "GET")
    try:
        with urllib.request.urlopen(req, timeout=60) as resp:
            return json.loads(resp.read().decode("utf-8"))
    except urllib.error.HTTPError as e:
        body = e.read().decode("utf-8", "ignore")
        if e.code == 401 or "sessionid is not exist" in body:
            raise RuntimeError("401 未授权：X-Token 无效或已过期，请重新获取 sessionid") from None
        raise RuntimeError("HTTP %s" % e.code) from None


# ----------------------- 选项查询 -----------------------

def current_term():
    j = http_json("/api/baseresservice/schoolCalendar/currentTermCalendar?flag=0")
    d = (j.get("data") or {}).get("schoolCalendar") or j.get("data") or {}
    return str(d.get("id")), d.get("simpleName") or d.get("name") or d.get("id")


def list_calendars():
    j = http_json("/api/baseresservice/schoolCalendar/list")
    out = []
    for c in j.get("data") or []:
        out.append((str(c["id"]), c.get("fullName") or str(c.get("id"))))
    return out


def list_faculties():
    j = http_json("/api/electionservice/elcMutualCourses/findDept?manageDept=0&type=1&virtualDept=0")
    items = j.get("data") or []
    if isinstance(items, dict):
        items = list(items.values())
    out = []
    for it in items:
        out.append((str(it.get("deptCode")), it.get("deptName")))
    out.sort(key=lambda x: x[1])
    return out


def list_dicts():
    j = http_json("/api/commonservice/dictionary/query",
                  {"lang": "cn", "type": "allChild", "keys": _DICT_KEYS})
    return j.get("data") or {}


def cmd_options():
    cur_id, cur_name = current_term()
    print("== 当前学期 ==")
    print("  %s  %s\n" % (cur_id, cur_name))
    print("== 全部学期（FILTERS['calendarId'] 可填 id 或 'current'）==")
    for cid, name in list_calendars():
        mark = "  <- 当前" if cid == cur_id else ""
        print("  %s  %s%s" % (cid, name, mark))
    print("\n== 开课学院（FILTERS['faculty'] 填第一列 deptCode）==")
    for code, name in list_faculties():
        print("  %s  %s" % (code, name))
    print("\n== 字典选项（value 填进对应 FILTERS 项）==")
    dicts = list_dicts()
    for dk, label in _DICT_LABEL.items():
        print("  -- %s --" % label)
        for code, name in (dicts.get(dk) or {}).items():
            print("    %s  %s" % (code, name))


# ----------------------- 数据抓取 -----------------------

def fetch_all(condition):
    rows, page, total = [], 1, None
    while True:
        body = {
            "pageNum_": page,
            "pageSize_": PAGE_SIZE,
            "condition": condition,
        }
        j = http_json("/api/electionservice/student/round/allArrangementCourses", body)
        data = j.get("data") or {}
        lst = data.get("list") or []
        rows.extend(lst)
        total = data.get("total_", data.get("total"))
        print("\r  第 %d 页 +%d 条（累计 %d%s）    " % (page, len(lst), len(rows), (" / " + str(total)) if total is not None else ""), end="")
        if not lst:
            break
        if total is not None and len(rows) >= int(total):
            break
        if len(lst) < PAGE_SIZE:
            break
        page += 1
        time.sleep(DELAY_MS)
    print()
    return rows


def build_condition():
    cond = {}
    for fk, api_key in _API_COND.items():
        val = str(FILTERS.get(fk) or "").strip()
        if fk == "calendarId":
            if not val or val == "current":
                cid, _ = current_term()
                cond[api_key] = cid
            else:
                cond[api_key] = val
        else:
            cond[api_key] = val  # 与网页一致：空串=不过滤
    return cond


def sanitize_value(value):
    if isinstance(value, list):
        return [sanitize_value(item) for item in value]
    if isinstance(value, dict):
        out = {}
        for key, item in value.items():
            if any(part in str(key).lower() for part in _PRIVATE_FIELD_PARTS):
                raise RuntimeError("响应包含禁止导出的隐私字段: %s" % key)
            out[key] = sanitize_value(item)
        return out
    return value


def sanitize_rows(rows):
    safe_rows = []
    for row in rows:
        safe = {}
        for key, value in row.items():
            if any(part in str(key).lower() for part in _PRIVATE_FIELD_PARTS):
                raise RuntimeError("响应包含禁止导出的隐私字段: %s" % key)
            if key in _SAFE_FIELDS:
                safe[key] = sanitize_value(value)
        safe_rows.append(safe)
    return safe_rows


def csv_value(value):
    if isinstance(value, (dict, list)):
        value = json.dumps(value, ensure_ascii=False)
    text = "" if value is None else str(value)
    if text[:1] in ("=", "+", "-", "@", "\t", "\r", "\n"):
        return "'" + text
    return text


def write_outputs(rows, cal_id, cond):
    ts = datetime.datetime.now().strftime("%Y-%m-%d-%H-%M-%S")
    base = os.path.join(OUT_DIR, "courses-%s-%s" % (cal_id, ts))
    rows = sanitize_rows(rows)
    with open(base + "-sanitized.json", "w", encoding="utf-8") as f:
        json.dump(rows, f, ensure_ascii=False, indent=1)
    keys = []
    for r in rows:
        for k in r.keys():
            if k not in keys:
                keys.append(k)
    with open(base + ".csv", "w", encoding="utf-8-sig", newline="") as f:
        w = csv.writer(f)
        w.writerow(keys)
        for r in rows:
            w.writerow([csv_value(r.get(k)) for k in keys])
    summary = "; ".join("%s=%s" % (k, v) for k, v in cond.items() if v)
    print("完成：%d 条  筛选[%s]" % (len(rows), summary))
    print("输出: %s-sanitized.json / %s.csv" % (base, base))


def main():
    token = X_TOKEN.strip()
    if not token or token.startswith("在此粘贴") or re.fullmatch(r"x{8,}", token, re.IGNORECASE):
        raise SystemExit("请通过 ONESYSTEM_X_TOKEN 环境变量提供 sessionid（不要写入源码）")
    if len(sys.argv) > 1 and sys.argv[1] == "options":
        cmd_options()
        return
    cond = build_condition()
    print("抓取条件:", cond)
    rows = fetch_all(cond)
    write_outputs(rows, cond.get("calendarId"), cond)


if __name__ == "__main__":
    main()
