# 同济 EnquiryOfCourses 选课数据与抓取脚本

同济 `1.tongji.edu.cn`「课程查询(EnquiryOfCourses)」页面的课程数据抓取脚本（Node / Python 双版本）。仓库不提交真实会话凭证，也不提交未经脱敏的上游快照。

## 数据

脚本默认输出 `courses-<calendarId>-<timestamp>-sanitized.json` 和 CSV。输出字段采用明确白名单，并递归拒绝 `student`、`selection`、联系方式、凭据等字段；CSV 同时中和公式前缀。原始全量响应不会写入磁盘。

主要字段：课程编码 / 课程名称 / 学分 / 课程性质 / 开课学院 / 校区 / 培养层次 / 学习形式 / 教师 / 教学班 / 上课时间地点 / 结构化排课 `timeTableList` / 考核方式 / 教学语言等。如需其它学期，修改脚本 `FILTERS.calendarId` 即可。

## 脚本用法（Node / Python 语义一致）

```bash
# 1) 先准备鉴权：登录 1.tongji.edu.cn → F12 → Console → sessionStorage.getItem('sessionid')
#    通过环境变量提供，不要填入脚本或提交到 Git
export ONESYSTEM_X_TOKEN='从 sessionStorage 读取的 sessionid'
# 可选：export ONESYSTEM_COOKIE='整段 Cookie'

# 2) 查看所有可选真实选项（学期 id / 开课学院 deptCode / 课程性质 / 校区 / 培养层次 / 学习形式）
node fetch_courses.js options        # 或 python fetch_courses.py options

# 3) 编辑脚本顶部 FILTERS（留空 = 不过滤），然后抓取 → JSON + CSV
node fetch_courses.js                # 或 python fetch_courses.py
```

两种脚本使用相同的 camelCase `FILTERS` 键：`calendarId`(学期)、`newCourseCode`(课程编码)、`teachClassCode`(教学班号)、`courseName`(课程名)、`teacherName`(教师)、`faculty`(开课学院 deptCode)、`nature`(课程性质)、`campu`(校区)、`trainingLevel`(培养层次)、`formLearning`(学习形式)。

## 鉴权与隐私

- 接口鉴权用请求头 `X-Token`（值为浏览器 `sessionStorage` 里的 `sessionid`），**不是 Cookie**。
- 凭证只从 `ONESYSTEM_X_TOKEN` / `ONESYSTEM_COOKIE` 读取；请勿提交真实 token、Cookie 或抓包 HAR。
- 脚本遇到禁止导出的个人字段会失败并不生成输出；如需保留原始响应，必须在受控的本地环境按组织隐私规范处理。

## 声明

本项目仅供学习与数据分析交流，禁止商用。使用请遵守同济大学相关规定与网站服务条款。
