#!/usr/bin/env python3
"""render_config.py — 把 deploy/config.toml.tmpl 渲染成实例 config.toml（配置治理）。

模板占位符 {{TOKEN}} 的值来源：
  - 实例差异（非敏感）: deploy/instances/<env>.json 提供 SERVER_URL / TRUSTED_PROXIES；
  - 敏感值: 同名环境变量（GitHub Actions 里由 Environments secrets 注入）。

安全契约（fail-closed）:
  - 值经 json.dumps 编码为 TOML basic string / inline array（引号、反斜杠、换行安全）;
  - 渲染后仍残留 {{ 或 }}（未知 token）→ 拒绝输出;
  - 必需 token 为空 → 拒绝输出。allow-empty 按环境收紧:
      main: AI_API_KEY + Google OAuth 凭据允许为空（可选功能）;
      dev:  AI_API_KEY + GH_CLIENT_ID/GH_CLIENT_SECRET + Google OAuth 凭据允许为空
            （dev 因 DB 站点设置无环境隔离保持 GitHub 登录关闭）;
  - PG_DSN 的 dbname 必须等于实例期望 pg_dbname（防跨库连错: dev 配到 yourtj_main 即拒绝）;
  - 产物必须能被 tomllib 解析;
  - 诊断信息只输出键名/来源/长度，绝不打印 secret 值或前缀；stdout 模式（--out -）下
    stdout 只含 TOML 文档，诊断一律走 stderr。

用法:
  SIGNING_KEY=... PG_DSN=... python3 deploy/render_config.py --env dev \
      --out deploy/rendered/config-dev.toml
  python3 deploy/render_config.py --compare-example   # example 键集 ⊇ tmpl 键集断言

仅依赖 python3.11+ 标准库（tomllib），无第三方。
"""

from __future__ import annotations

import argparse
import json
import os
import re
import sys
import tomllib
import urllib.parse

TOKEN_RE = re.compile(r"\{\{\s*([A-Z][A-Z0-9_]*)\s*\}\}")

# 由实例 JSON 提供的非敏感 token；其余 token 一律视为 secret（从环境变量读取）。
INSTANCE_TOKENS = {"SERVER_URL", "TRUSTED_PROXIES"}

# 全部环境都允许为空的 token（可选功能，未配置即关闭）。
BASE_OPTIONAL_TOKENS = {
    "AI_API_KEY",
    "GOOGLE_CLIENT_ID",
    "GOOGLE_CLIENT_SECRET",
    # Web Push VAPID：未配置即通道关闭（dev 必须保持空——快照同步的
    # 订阅/任务行绝不能从 dev 外发推送）。
    "VAPID_PUBLIC_KEY",
    "VAPID_PRIVATE_KEY",
}

DEFAULT_TMPL = os.path.join(
    os.path.dirname(os.path.abspath(__file__)), "config.toml.tmpl"
)
DEFAULT_INSTANCES_DIR = os.path.join(
    os.path.dirname(os.path.abspath(__file__)), "instances"
)
DEFAULT_EXAMPLE = os.path.join(
    os.path.dirname(os.path.abspath(__file__)), "config.toml.example"
)


def optional_tokens(env):
    """按环境返回允许为空的 token 集合。"""
    s = set(BASE_OPTIONAL_TOKENS)
    if env == "dev":
        # dev GitHub 登录关闭（DB siteUrl 无环境隔离，回调会指回生产）。
        s |= {"GH_CLIENT_ID", "GH_CLIENT_SECRET"}
    return s


def flatten_keys(data, prefix=""):
    """把 tomllib 解析结果转成 {'sec.key'} 键集；[[数组]] 内 dict 以其键并入。"""
    keys = set()
    if isinstance(data, dict):
        for k, v in data.items():
            key = f"{prefix}.{k}" if prefix else k
            keys.add(key)
            if isinstance(v, dict):
                keys |= flatten_keys(v, key)
            elif isinstance(v, list):
                for item in v:
                    if isinstance(item, dict):
                        keys |= flatten_keys(item, key)
    return keys


def tokens_in(text):
    """模板中出现的全部 token 名。"""
    return set(TOKEN_RE.findall(text))


def load_instance(env, instances_dir):
    path = os.path.join(instances_dir, f"{env}.json")
    with open(path, encoding="utf-8") as f:
        data = json.load(f)
    if data.get("instance") != env:
        raise SystemExit(
            f"render: {path} instance 字段 {data.get('instance')!r} != env {env!r}"
        )
    return data


def render(text, values):
    """把模板中的 {{TOKEN}} 全部替换为 TOML 编码后的值；出现未知 token 即失败。"""
    out = text
    for tok in sorted(tokens_in(text)):
        if tok not in values:
            raise SystemExit(
                f"render: token {tok} 无值来源（instance JSON 或 secret 缺失）"
            )
        # json.dumps → TOML basic string（字符串）或 inline array（列表），
        # 天然转义引号/反斜杠/换行/控制字符。
        out = out.replace("{{%s}}" % tok, json.dumps(values[tok]))
    if "{{" in out or "}}" in out:
        raise SystemExit("render: 渲染后仍残留 {{ 或 }}（占位符未全部替换）")
    return out


def dsn_dbname(dsn):
    """从 libpq key=value 或 postgres:// URL DSN 提取 dbname；解析失败返回 None。

    支持两种格式（与 deploy/scripts/pgdsn.sh 语义一致）:
      postgres://user:pass@host:port/dbname?sslmode=disable
      host=postgres user=u password=p dbname=xxx port=5432 sslmode=disable
    """
    dsn = dsn.strip()
    if not dsn:
        return None
    if re.match(r"^[a-z]+://", dsn, re.IGNORECASE):
        rest = dsn.split("://", 1)[1]
        # 去掉 authority（host[:port][@userinfo] 前的部分），取首个 / 后的路径段
        if "/" not in rest:
            return None
        path_query = rest.split("/", 1)[1]
        db = path_query.split("?")[0].split("#")[0]
        return urllib.parse.unquote(db) if db else None
    # key=value 格式（dbname 值不含空格/引号——本仓库 DSN 由 init-server 生成，值无空格）
    for tok in dsn.split():
        if tok.startswith("dbname="):
            v = tok[len("dbname=") :]
            if len(v) >= 2 and v[0] in "\"'" and v[-1] == v[0]:
                v = v[1:-1]
            return v or None
    return None


def validate_pg_dsn(values, instance):
    """fail-closed: PG_DSN 的 dbname 必须与实例期望一致（防跨库连错生产库）。"""
    pg_val = values.get("PG_DSN")
    expect_db = instance.get("pg_dbname")
    if not pg_val or not expect_db:
        return
    actual = dsn_dbname(pg_val)
    if not actual:
        raise SystemExit("render: PG_DSN 无法解析 dbname（fail-closed，拒绝输出）")
    if actual != expect_db:
        raise SystemExit(
            f"render: PG_DSN dbname='{actual}' ≠ 实例期望 '{expect_db}'"
            "（疑似跨库连错，拒绝输出）"
        )


def build_values(env, instance, environ, tokens, allow_empty):
    """组装 token → 值。返回 (values, summary)；空必需值即失败。"""
    instance_token_map = {
        "SERVER_URL": str(instance["server_url"]),
        "TRUSTED_PROXIES": list(instance["trusted_proxies"]),
    }
    values = {}
    summary = []
    for tok in sorted(tokens):
        if tok in instance_token_map:
            values[tok] = instance_token_map[tok]
            summary.append((tok, values[tok], "instance"))
            continue
        raw = environ.get(tok, "")
        if raw == "":
            if tok in allow_empty:
                values[tok] = ""
                summary.append((tok, "", "secret-optional-empty"))
                continue
            raise SystemExit(
                f"render: 必需 secret {tok} 为空（{env} 环境未设置或为必填）"
            )
        values[tok] = raw
        summary.append((tok, raw, "secret"))
    return values, summary


def summarize(summary):
    """向 stderr 输出键名/来源/长度，绝不打印 secret 值或其前缀。"""
    for tok, val, src in summary:
        if isinstance(val, list):
            print(f"  {tok}: {src} (array len={len(val)})", file=sys.stderr)
        else:
            print(f"  {tok}: {src} len={len(val)}", file=sys.stderr)


def cmd_render(args):
    text = ""
    with open(args.tmpl, encoding="utf-8") as f:
        text = f.read()
    tokens = tokens_in(text)
    instance = load_instance(args.env, args.instances_dir)
    allow = optional_tokens(args.env)
    if args.allow_empty_extra:
        allow |= {x.strip() for x in args.allow_empty_extra.split(",") if x.strip()}
    values, summary = build_values(args.env, instance, os.environ, tokens, allow)
    # 跨库防护: DSN dbname 必须匹配实例
    validate_pg_dsn(values, instance)
    rendered = render(text, values)
    try:
        tomllib.loads(rendered)
    except tomllib.TOMLDecodeError as e:
        raise SystemExit(f"render: 产物 TOML 解析失败（fail-closed，不输出）: {e}")
    print(f"render: {args.env} 渲染校验通过，token 清单:", file=sys.stderr)
    summarize(summary)
    out = args.out
    if out == "-":
        # stdout 是产物通道：只写 TOML，诊断已走 stderr
        sys.stdout.write(rendered)
        return
    os.makedirs(os.path.dirname(os.path.abspath(out)), exist_ok=True)
    with open(out, "w", encoding="utf-8") as f:
        f.write(rendered)
    print(f"render: 已写入 {out}")


def cmd_compare_example(args):
    """断言 example 键集 ⊇ tmpl 键集（tmpl 所有键在 example 都有对应说明）。

    example 允许额外携带可选功能节（如 [cron]、[pk.semester_dates]）作人类参考，
    但 tmpl 新增的任何键必须同步进 example，防止模板漂移。
    """
    with open(args.tmpl, encoding="utf-8") as f:
        text = f.read()
    with open(args.example, encoding="utf-8") as f:
        example_text = f.read()
    # 哑渲染: 所有 token 替换为合法值（tomllib 不做类型语义检查）
    for tok in sorted(tokens_in(text)):
        text = text.replace("{{%s}}" % tok, json.dumps("x"))
    try:
        tmpl_keys = flatten_keys(tomllib.loads(text))
        example_keys = flatten_keys(tomllib.loads(example_text))
    except tomllib.TOMLDecodeError as e:
        raise SystemExit(f"compare: 解析失败: {e}")
    missing = sorted(tmpl_keys - example_keys)
    if missing:
        print(f"compare: FAIL — tmpl 有以下键未在 example 出现: {missing}")
        return 1
    extra = sorted(example_keys - tmpl_keys)
    print(
        f"compare: OK — example 键集覆盖 tmpl ({len(tmpl_keys)} 键)；example 额外可选键 {len(extra)}: {extra}"
    )
    return 0


def main(argv=None):
    p = argparse.ArgumentParser(description="渲染实例 config.toml（配置治理）")
    sub = p.add_subparsers(dest="cmd", required=True)

    pr = sub.add_parser("render", help="渲染实例 config")
    pr.add_argument(
        "--env", required=True, choices=["main", "dev"], help="实例: main|dev"
    )
    pr.add_argument("--tmpl", default=DEFAULT_TMPL)
    pr.add_argument("--instances-dir", default=DEFAULT_INSTANCES_DIR)
    pr.add_argument("--out", default="-", help="输出路径（默认 stdout，纯 TOML）")
    pr.add_argument(
        "--allow-empty-extra", default="", help="额外允许为空的 token（逗号分隔）"
    )
    pr.set_defaults(func=cmd_render)

    pc = sub.add_parser(
        "compare-example", help="断言 config.toml.example 键集覆盖 tmpl"
    )
    pc.add_argument("--tmpl", default=DEFAULT_TMPL)
    pc.add_argument("--example", default=DEFAULT_EXAMPLE)
    pc.set_defaults(func=cmd_compare_example)

    args = p.parse_args(argv)
    return args.func(args)


if __name__ == "__main__":
    raise SystemExit(main())
