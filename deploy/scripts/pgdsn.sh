#!/usr/bin/env bash
# pgdsn.sh — 共享 PostgreSQL DSN 解析库(source 后使用)。
# 同时支持 libpq key=value 与 postgres:// URL 两种 DSN 格式, 供
# backup-db.sh / sync-db-from-main.sh 等运维脚本提取数据库名。
#
# 用法:
#   source pgdsn.sh
#   pg_dsn_dbname <dsn>        # 输出 dbname; 解析失败返回 1 并输出错误到 stderr
#   pg_dsn_normalize <dsn>     # 输出归一化 DSN(用于回显/日志, 脱敏密码)
#
# 设计约束(issue #134):
#   - 解析失败必须返回非零, 调用方必须在任何 DB/stop 操作之前检查,
#     避免"先停服务再报错"导致服务停留在停止状态。
#   - 不依赖 psql/外部工具, 纯 bash 实现, 便于在无 PG 客户端的主机上测试。
#   - 不使用 eval 拼接 DSN 内容(review P1 命令注入): 赋值一律走 printf -v。

# pg_percent_decode <s> — 将 URL percent-encoding(%XX)解码为字符。
# libpq 对 dbname 路径段做 percent-decode, 解析库保持一致(review LOW1),
# 避免 pg_dump -d "my%20db" 连错库。
pg_percent_decode() {
  local s="$1" out="" ch hex
  while [ -n "$s" ]; do
    case "$s" in
      %[0-9a-fA-F][0-9a-fA-F]*)
        hex="${s:1:2}"
        printf -v ch "\\x$hex"
        out="$out$ch"
        s="${s:3}"
        ;;
      *)
        out="$out${s:0:1}"
        s="${s:1}"
        ;;
    esac
  done
  printf '%s' "$out"
}

# pg_uri_split <uri> <var> — 从 postgres:// URL 提取组件。
# 支持: postgres://user:pass@host:port/db?sslmode=disable&param=v
#       postgresql://user@host/db
# 输出到 <var>_user/_password/_host/_port/_db/_params(空为未提供)。
# 密码中可能含任意字符(含 ' " @ / 等), 全部按字面量处理, 不做任何 shell 求值。
pg_uri_split() {
  local uri="$1" var="$2" rest userinfo hostport pathquery dbpart
  # 去掉 scheme 前缀(大小写不敏感, RFC 3986)
  rest="${uri#*://}"
  # 分离 userinfo 与 host 部分: 按最后一个 @ 切分(libpq 语义,
  # 密码中的裸 @ 不会破坏 host 解析)
  if [[ "$rest" == *"@"* ]]; then
    userinfo="${rest%@*}"
    rest="${rest##*@}"
  else
    userinfo=""
  fi
  # userinfo: user[:password]
  if [ -n "$userinfo" ]; then
    if [[ "$userinfo" == *":"* ]]; then
      printf -v "${var}_user" '%s' "${userinfo%%:*}"
      printf -v "${var}_password" '%s' "${userinfo#*:}"
    else
      printf -v "${var}_user" '%s' "$userinfo"
      printf -v "${var}_password" '%s' ""
    fi
  else
    printf -v "${var}_user" '%s' ""
    printf -v "${var}_password" '%s' ""
  fi
  # host[:port] 与 path 的边界是第一个 /
  hostport="${rest%%/*}"
  pathquery="${rest#*/}"
  # 若路径不存在(rest 无 /), pathquery 会等于 rest, 需修正
  if [ "$pathquery" = "$rest" ]; then
    pathquery=""
  fi
  # pathquery: db[?params]
  # 无路径但有 query 参数(postgres://u@h?dbname=dbq): query 挂在 hostport
  # 末尾, db 段为空, ? 之后全部归 params
  if [ -z "$pathquery" ] && [[ "$hostport" == *"?"* ]]; then
    printf -v "${var}_db" '%s' ""
    printf -v "${var}_params" '%s' "${hostport#*\?}"
    hostport="${hostport%%\?*}"
  elif [[ "$pathquery" == *"?"* ]]; then
    dbpart="${pathquery%%\?*}"
    printf -v "${var}_db" '%s' "$dbpart"
    printf -v "${var}_params" '%s' "${pathquery#*\?}"
  else
    printf -v "${var}_db" '%s' "$pathquery"
    printf -v "${var}_params" '%s' ""
  fi
  # host[:port] — IPv6 字面量 [::1] 或 [::1]:5432 按括号整体识别(review INFO),
  # 避免 host 被错误切分为 "["
  if [[ "$hostport" == \[*\]* ]]; then
    printf -v "${var}_host" '%s' "${hostport%%]*}"
    if [[ "${hostport##*]}" == ":"* ]]; then
      printf -v "${var}_port" '%s' "${hostport##*]:}"
    else
      printf -v "${var}_port" '%s' ""
    fi
  elif [[ "$hostport" == *":"* ]]; then
    printf -v "${var}_host" '%s' "${hostport%%:*}"
    printf -v "${var}_port" '%s' "${hostport#*:}"
  else
    printf -v "${var}_host" '%s' "$hostport"
    printf -v "${var}_port" '%s' ""
  fi
}

# pg_toml_url <cfg> — 从 config.toml 的 [db.default] 区块提取 url 值。
# 按 TOML 语义剥离行尾内联注释(review W1): 只取双引号字符串内部,
# 引号之后的内容(含 "  # 注释")全部丢弃; 引号值内合法 #(如密码含 #)
# 不受影响。旧 grep 实现([^ ]+)对行尾注释免疫, 新 URL 支持的
# tr -d '"' 会把注释并入 DSN 导致 dbname 带脏值, 此函数修复该回归。
# 输出 url; 未配置或非双引号值(如单引号 TOML 字面量)时输出空/原样。
pg_toml_url() {
  local cfg="$1"
  # 只匹配 [db.default] 区块内的 url 行, 避免误取 [db.file]/[meilisearch] 的 url
  # 注: 用 POSIX 字符类而非 \s, 兼容 BSD sed(macOS); \1 捕获双引号内全部内容
  sed -n '/^\[db\.default\]/,/^\[/p' "$cfg" \
    | grep -E '^[[:space:]]*url[[:space:]]*=' \
    | head -n1 \
    | sed -E 's/^[[:space:]]*url[[:space:]]*=[[:space:]]*"([^"]*)".*$/\1/'
}

# pg_dsn_dbname <dsn> — 输出 DSN 中的数据库名。
# 支持 key=value(key 可带引号)与 postgres:// URL 两种格式。
# 解析失败输出错误到 stderr 并返回 1(错误信息经 pg_dsn_normalize 脱敏,
# 避免明文密码进入 CI/部署日志)。
pg_dsn_dbname() {
  local dsn="$1" key uri db
  [ -n "$dsn" ] || { echo "pg_dsn_dbname: empty DSN" >&2; return 1; }
  # URL 格式: postgres:// 或 postgresql:// (scheme 大小写不敏感, RFC 3986)
  # 注: 用 [pP][oO]... 模式匹配而非 ${dsn,,}(bash 4+ 特性, macOS 默认 bash 3.2 不支持)
  if [[ "$dsn" == [pP][oO][sS][tT][gG][rR][eE][sS]://* || "$dsn" == [pP][oO][sS][tT][gG][rR][eE][sS][qQ][lL]://* ]]; then
    pg_uri_split "$dsn" uri
    if [ -z "$uri_db" ] && [ -z "$uri_params" ]; then
      echo "pg_dsn_dbname: URL DSN 缺少数据库名: $(pg_dsn_normalize "$dsn")" >&2
      return 1
    fi
    [ -n "$uri_host" ] || { echo "pg_dsn_dbname: URL DSN 缺少主机名: $(pg_dsn_normalize "$dsn")" >&2; return 1; }
    # URL 中的 dbname 参数优先于路径段(libpq 行为: 后者覆盖前者)。
    # query 中显式出现 dbname=(即使值为空)即覆盖路径段, 空值报错,
    # 不静默回退(review LOW2)。用 has_query_dbname 哨兵区分
    # "显式空 dbname"与"query 未提供 dbname"(后者回退路径段)。
    db=""
    has_query_dbname=""
    if [ -n "$uri_params" ]; then
      local IFS='&' p
      # set -f: 参数值含 * 等通配符时不触发路径展开
      set -f
      for p in $uri_params; do
        case "$p" in
          dbname=*) has_query_dbname=1; db="${p#dbname=}";;
        esac
      done
      set +f
    fi
    if [ -n "$has_query_dbname" ] && [ -z "$db" ]; then
      echo "pg_dsn_dbname: URL DSN 数据库名为空: $(pg_dsn_normalize "$dsn")" >&2
      return 1
    fi
    [ -n "$db" ] || db="$uri_db"
    if [ -z "$db" ]; then
      echo "pg_dsn_dbname: URL DSN 数据库名为空: $(pg_dsn_normalize "$dsn")" >&2
      return 1
    fi
    # libpq 对 dbname 做 percent-decode(review LOW1), 避免 pg_dump -d 连错库
    echo "$(pg_percent_decode "$db")"
    return 0
  fi
  # key=value 格式: 按空格 token 化逐项解析, 取最后一个 dbname token
  # (libpq 语义: 后者覆盖前者)。按 token 解析可避免密码/其他值中
  # 含 "dbname=" 子串时被第一个匹配误判。引号值含空格时
  # (dbname="my forum") 拼接引号内全部 token(review LOW3)。
  if [[ "$dsn" == *"dbname="* ]]; then
    local -a toks=()
    local tok val i=0
    key=""
    set -f
    # read -a 按 IFS(默认含空格)切分; set -f 防 glob 展开
    read -r -a toks <<< "$dsn"
    set +f
    while [ "$i" -lt "${#toks[@]}" ]; do
      tok="${toks[$i]}"
      case "$tok" in
        dbname=*)
          key="${tok#dbname=}"
          # 引号未闭合 → 拼接后续 token 直至闭合引号
          while [[ "$key" == \"* && "$key" != *\" || "$key" == \'* && "$key" != *\' ]]; do
            i=$((i + 1))
            [ "$i" -lt "${#toks[@]}" ] || break
            key="$key ${toks[$i]}"
          done
          # 去掉包裹引号(单/双)
          if [[ "$key" == \"*\" ]]; then
            key="${key:1:${#key}-2}"
          elif [[ "$key" == \'*\' ]]; then
            key="${key:1:${#key}-2}"
          fi
          ;;
      esac
      i=$((i + 1))
    done
    [ -n "$key" ] || { echo "pg_dsn_dbname: dbname 为空: $(pg_dsn_normalize "$dsn")" >&2; return 1; }
    echo "$key"
    return 0
  fi
  echo "pg_dsn_dbname: 无法解析 DSN(需为 postgres:// URL 或含 dbname= 的 key=value 格式): $(pg_dsn_normalize "$dsn")" >&2
  return 1
}

# pg_dsn_normalize <dsn> — 输出归一化 DSN 用于日志/回显:
#   - URL 格式: 脱敏密码(保留 user@host/db)
#   - key=value 格式: 脱敏 password 值
pg_dsn_normalize() {
  local dsn="$1" uri result
  if [[ "$dsn" == [pP][oO][sS][tT][gG][rR][eE][sS]://* || "$dsn" == [pP][oO][sS][tT][gG][rR][eE][sS][qQ][lL]://* ]]; then
    pg_uri_split "$dsn" uri
    # shellcheck disable=SC2154  # uri_* 由 pg_uri_split 的 printf -v 动态赋值
    result="postgres://${uri_user}"
    if [ -n "$uri_password" ]; then
      result="${result}:***"
    fi
    if [ -n "$uri_port" ]; then
      result="${result}@${uri_host}:${uri_port}"
    else
      result="${result}@${uri_host}"
    fi
    if [ -n "$uri_db" ]; then
      result="${result}/${uri_db}"
    fi
    echo "$result"
    return 0
  fi
  # key=value: 脱敏 password
  result=""
  local IFS=' ' tok
  set -f
  for tok in $dsn; do
    case "$tok" in
      password=*) result="$result password=***";;
      *) result="$result $tok";;
    esac
  done
  set +f
  echo "${result# }"
}
