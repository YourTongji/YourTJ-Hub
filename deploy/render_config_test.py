#!/usr/bin/env python3
"""render_config_test.py — deploy/render_config.py 的单元测试（assert 风格，仿 pgdsn_test.sh）。

运行: python3 deploy/render_config_test.py  （退出码非 0 = 失败，无需外部依赖）
覆盖: token 解析 / 占位替换 / TOML 转义 / 残留拒绝 / 空必需值拒绝 / allow-empty 白名单 /
      PG_DSN dbname 跨库校验 / stdout 纯 TOML / example↔tmpl 键集一致性 / dev 端到端渲染。
"""

from __future__ import annotations

import os
import subprocess
import sys
import tempfile
import tomllib

HERE = os.path.dirname(os.path.abspath(__file__))
sys.path.insert(0, HERE)

import render_config as rc  # noqa: E402

PASS = 0
FAIL = 0


def check(desc, cond):
    global PASS, FAIL
    if cond:
        PASS += 1
        print(f"ok  - {desc}")
    else:
        FAIL += 1
        print(f"FAIL- {desc}")


def check_eq(desc, want, got):
    check(f"{desc}: want {want!r}, got {got!r}", want == got)


def expect_exit(desc, fn):
    try:
        fn()
        check(desc, False)
    except SystemExit:
        check(desc, True)


# --- 1. token 解析与渲染（string / array / 转义） ---
tmpl_src = "url = {{SERVER_URL}}\narr = {{TRUSTED_PROXIES}}\ntxt = {{GH_SECRET}}\n"
check_eq(
    "tokens_in 收集",
    {"SERVER_URL", "TRUSTED_PROXIES", "GH_SECRET"},
    rc.tokens_in(tmpl_src),
)

out = rc.render(
    tmpl_src,
    {
        "SERVER_URL": "https://f.yourtj.de",
        "TRUSTED_PROXIES": ["172.16.0.0/12", "::1"],
        "GH_SECRET": 'p"a\\b\nc',
    },
)
parsed = tomllib.loads(out)
check_eq("string 渲染", "https://f.yourtj.de", parsed["url"])
check_eq("array 渲染", ["172.16.0.0/12", "::1"], parsed["arr"])
check_eq("转义保留(引号/反斜杠/换行)", 'p"a\\b\nc', parsed["txt"])

# --- 2. 残留 / 未知 token 拒绝 ---
expect_exit(
    "未知 token 应失败",
    lambda: rc.render("a = {{SERVER_URL}}\nbad = {{UNKNOWN}}", {"SERVER_URL": "x"}),
)
expect_exit("残留 {{ 应失败", lambda: rc.render('a = "x{{BROKEN"', {}))
expect_exit(
    "渲染后仍含 }} 应失败",
    lambda: rc.render("a = {{SERVER_URL}} }}\n", {"SERVER_URL": "x"}),
)

# --- 2.5 PG_DSN dbname 解析与跨库校验 ---
check_eq(
    "dsn key=value 解析",
    "yourtj_dev",
    rc.dsn_dbname(
        "host=postgres user=yourtj password=x dbname=yourtj_dev port=5432 sslmode=disable"
    ),
)
check_eq(
    "dsn URL 解析",
    "yourtj_main",
    rc.dsn_dbname("postgres://yourtj:x@postgres:5432/yourtj_main?sslmode=disable"),
)
check_eq(
    "dsn URL 空库返回 None", None, rc.dsn_dbname("postgres://yourtj:x@postgres:5432")
)
check_eq("dsn 空串返回 None", None, rc.dsn_dbname(""))
check_eq("dsn 无 dbname 返回 None", None, rc.dsn_dbname("host=postgres user=yourtj"))

inst_dev = {"pg_dbname": "yourtj_dev"}
rc.validate_pg_dsn({"PG_DSN": "host=x dbname=yourtj_dev"}, inst_dev)  # 不抛
expect_exit(
    "PG_DSN dbname 与实例不符应拒绝",
    lambda: rc.validate_pg_dsn({"PG_DSN": "host=x dbname=yourtj_main"}, inst_dev),
)
expect_exit(
    "PG_DSN 无法解析 dbname 应拒绝",
    lambda: rc.validate_pg_dsn({"PG_DSN": "host=x port=5432"}, inst_dev),
)

# --- 3. 真实模板 token 分类与空值 fail-closed ---
fake_instance = {
    "instance": "dev",
    "server_url": "https://dev.yourtj.de",
    "trusted_proxies": ["172.16.0.0/12"],
    "pg_dbname": "yourtj_dev",
}
with open(rc.DEFAULT_TMPL, encoding="utf-8") as f:
    real_tmpl = f.read()
real_tokens = rc.tokens_in(real_tmpl)
required_nonempty = sorted(real_tokens - rc.INSTANCE_TOKENS - rc.optional_tokens("dev"))
check(
    "模板必需非空 secret 含 SIGNING_KEY/PG_DSN/MEILI/WIKI",
    {"SIGNING_KEY", "PG_DSN", "MEILI_MASTER_KEY", "WIKI_WEBHOOK_SECRET"}
    <= set(required_nonempty),
)
check(
    "main 的 GH_CLIENT_ID 必填（不在 allow-empty）",
    "GH_CLIENT_ID" not in rc.optional_tokens("main"),
)
check("dev 的 GH_CLIENT_ID 允许空", "GH_CLIENT_ID" in rc.optional_tokens("dev"))
check(
    "Google OAuth 凭据在 main/dev 均允许空",
    {"GOOGLE_CLIENT_ID", "GOOGLE_CLIENT_SECRET"}
    <= rc.optional_tokens("main")
    and {"GOOGLE_CLIENT_ID", "GOOGLE_CLIENT_SECRET"}
    <= rc.optional_tokens("dev"),
)

env_empty = {tok: "" for tok in real_tokens}
expect_exit(
    "空必需 secret 应拒绝（dev）",
    lambda: rc.build_values(
        "dev", fake_instance, env_empty, real_tokens, rc.optional_tokens("dev")
    ),
)

# --- 4. allow-empty 白名单生效 + main GH 必填 ---
env_partial = dict(env_empty)
env_partial.update(
    {
        "SIGNING_KEY": "s" * 32,
        "PG_DSN": "host=postgres user=yourtj dbname=yourtj_dev port=5432 sslmode=disable",
        "MEILI_MASTER_KEY": "m" * 32,
        "WIKI_WEBHOOK_SECRET": "w" * 64,
    }
)
values, summary = rc.build_values(
    "dev", fake_instance, env_partial, real_tokens, rc.optional_tokens("dev")
)
check_eq("dev 空 GH_CLIENT_ID 渲染为空串", "", values.get("GH_CLIENT_ID", "<missing>"))
check_eq("dev 空 GOOGLE_CLIENT_ID 渲染为空串", "", values.get("GOOGLE_CLIENT_ID", "<missing>"))
check_eq("dev 空 AI_API_KEY 渲染为空串", "", values.get("AI_API_KEY", "<missing>"))
check_eq("dev 渲染 signingKey 正确", "s" * 32, values["SIGNING_KEY"])
secret_toks = {t for t, _, src in summary if src == "secret"}
check(
    "summary 标记 secret",
    "SIGNING_KEY" in secret_toks and "GH_CLIENT_ID" not in secret_toks,
)
# summarize() 输出（stderr）不得含任何 secret 值或前缀（防日志泄露部分密钥）
import contextlib
import io

buf = io.StringIO()
with contextlib.redirect_stderr(buf):
    rc.summarize(summary)
sum_text = buf.getvalue()
check(
    "summarize 输出不含 secret 值前缀",
    "s" * 4 not in sum_text and "Ov23" not in sum_text,
)
check("summarize 输出含键名与长度", "SIGNING_KEY" in sum_text and "len=32" in sum_text)

# main 场景: GH 凭据未设 → 必须失败（fail-closed 生产）
fake_main = {
    "instance": "main",
    "server_url": "https://f.yourtj.de",
    "trusted_proxies": ["172.16.0.0/12"],
    "pg_dbname": "yourtj_main",
}
expect_exit(
    "main 缺 GH_CLIENT_SECRET 应拒绝",
    lambda: rc.build_values(
        "main", fake_main, env_partial, real_tokens, rc.optional_tokens("main")
    ),
)

# --- 5. example ↔ tmpl 键集一致性（仓库实际文件） ---
rc_ok = rc.main(["compare-example"])
check("compare-example: example 键集覆盖 tmpl", rc_ok == 0)

# --- 6. dev 端到端渲染（CLI 子进程, 注入完整 env）: 产物可解析、值正确、无残留 ---
with tempfile.TemporaryDirectory() as td:
    out_path = os.path.join(td, "config.toml")
    env_full = dict(os.environ)
    env_full.update(
        {
            "SIGNING_KEY": "k" * 32,
            "PG_DSN": "host=postgres user=yourtj password=x dbname=yourtj_dev port=5432 sslmode=disable",
            "MEILI_MASTER_KEY": "m" * 32,
            "WIKI_WEBHOOK_SECRET": "w" * 64,
        }
    )
    proc = subprocess.run(
        [
            sys.executable,
            os.path.join(HERE, "render_config.py"),
            "render",
            "--env",
            "dev",
            "--out",
            out_path,
        ],
        env=env_full,
        capture_output=True,
        text=True,
    )
    check(
        f"dev 渲染 exit 0（stderr: {proc.stderr.strip()[:60]}）", proc.returncode == 0
    )
    if proc.returncode == 0:
        with open(out_path, encoding="utf-8") as f:
            cfg = tomllib.loads(f.read())
        check_eq("dev server_url", "https://dev.yourtj.de", cfg["server"]["url"])
        check_eq("dev github client_id 空", "", cfg["github"]["client_id"])
        check_eq("dev signingKey", "k" * 32, cfg["app"]["signingKey"])
        dbname = cfg["db"]["default"]["url"].split("dbname=")[1].split()[0]
        check_eq("dev dsn dbname", "yourtj_dev", dbname)
        rendered_txt = open(out_path, encoding="utf-8").read()
        check(
            "dev 产物无 {{ 残留", "{{" not in rendered_txt and "}}" not in rendered_txt
        )
    # stderr 诊断不得包含 secret 值前缀
    check(
        "stderr 不含 secret 前缀",
        "k" * 4 not in proc.stderr and "Ov23" not in proc.stderr,
    )

    # 6.5 stdout 模式（--out -）: stdout 只含 TOML, 诊断全在 stderr
    proc_stdout = subprocess.run(
        [
            sys.executable,
            os.path.join(HERE, "render_config.py"),
            "render",
            "--env",
            "dev",
            "--out",
            "-",
        ],
        env=env_full,
        capture_output=True,
        text=True,
    )
    check("stdout 模式 exit 0", proc_stdout.returncode == 0)
    if proc_stdout.returncode == 0:
        stdout_cfg = tomllib.loads(proc_stdout.stdout)
        check(
            "stdout 是纯 TOML（可解析）",
            stdout_cfg["server"]["url"] == "https://dev.yourtj.de",
        )
    check("stdout 模式诊断在 stderr", "render:" in proc_stdout.stderr)

    # 6.6 跨库 DSN（dev 指向 yourtj_main）端到端拒绝
    env_cross = dict(env_full)
    env_cross["PG_DSN"] = (
        "host=postgres user=yourtj password=x dbname=yourtj_main port=5432 sslmode=disable"
    )
    proc_cross = subprocess.run(
        [
            sys.executable,
            os.path.join(HERE, "render_config.py"),
            "render",
            "--env",
            "dev",
            "--out",
            "-",
        ],
        env=env_cross,
        capture_output=True,
        text=True,
    )
    check("跨库 DSN 端到端拒绝（exit != 0）", proc_cross.returncode != 0)
    check("跨库错误信息含 dbname", "dbname" in proc_cross.stderr)

# --- 7. 坏模板端到端拒绝（fail-closed 不产文件） ---
with tempfile.TemporaryDirectory() as td:
    bad_tmpl = os.path.join(td, "bad.tmpl")
    with open(bad_tmpl, "w", encoding="utf-8") as f:
        f.write("url = {{SERVER_URL}}\nleftover = {{STRAY}}\n")
    env_bad = dict(os.environ)
    env_bad.update({"SERVER_URL": "https://x.de"})
    bad_out = os.path.join(td, "out.toml")
    proc = subprocess.run(
        [
            sys.executable,
            os.path.join(HERE, "render_config.py"),
            "render",
            "--env",
            "dev",
            "--tmpl",
            bad_tmpl,
            "--out",
            bad_out,
        ],
        env=env_bad,
        capture_output=True,
        text=True,
    )
    check("坏模板端到端拒绝（exit != 0）", proc.returncode != 0)
    check("坏模板不产出文件", not os.path.exists(bad_out))

print(f"\n{'-' * 40}\nrender_config_test: {PASS} passed, {FAIL} failed")
raise SystemExit(1 if FAIL else 0)
