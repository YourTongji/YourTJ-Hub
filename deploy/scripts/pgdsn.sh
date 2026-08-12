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

# pg_uri_split <uri> <var> — 从 postgres:// URL 提取组件。
# 支持: postgres://user:pass@host:port/db?sslmode=disable&param=v
#       postgresql://user@host/db
# 输出到 <var>_user/_password/_host/_port/_db/_params(空为未提供)。
pg_uri_split() {
  local uri="$1" var="$2" rest userinfo hostport pathquery dbpart
  # 去掉 scheme 前缀
  rest="${uri#*://}"
  # 分离 userinfo(最后一个 @ 之前)与 host 部分
  if [[ "$rest" == *"@"* ]]; then
    userinfo="${rest%%@*}"
    rest="${rest#*@}"
  else
    userinfo=""
  fi
  # userinfo: user[:password]
  if [ -n "$userinfo" ]; then
    if [[ "$userinfo" == *":"* ]]; then
      eval "${var}_user='${userinfo%%:*}'"
      eval "${var}_password='${userinfo#*:}'"
    else
      eval "${var}_user='$userinfo'"
      eval "${var}_password=''"
    fi
  else
    eval "${var}_user=''"
    eval "${var}_password=''"
  fi
  # host[:port] 与 path 的边界是第一个 /
  hostport="${rest%%/*}"
  pathquery="${rest#*/}"
  # 若路径不存在(rest 无 /), pathquery 会等于 rest, 需修正
  if [ "$pathquery" = "$rest" ]; then
    pathquery=""
  fi
  # pathquery: db[?params]
  if [[ "$pathquery" == *"?"* ]]; then
    dbpart="${pathquery%%\?*}"
    eval "${var}_db='$dbpart'"
    eval "${var}_params='${pathquery#*\?}'"
  else
    eval "${var}_db='$pathquery'"
    eval "${var}_params=''"
  fi
  # host[:port]
  if [[ "$hostport" == *":"* ]]; then
    eval "${var}_host='${hostport%%:*}'"
    eval "${var}_port='${hostport#*:}'"
  else
    eval "${var}_host='$hostport'"
    eval "${var}_port=''"
  fi
}

# pg_dsn_dbname <dsn> — 输出 DSN 中的数据库名。
# 支持 key=value(key 可带引号)与 postgres:// URL 两种格式。
# 解析失败输出错误到 stderr 并返回 1。
pg_dsn_dbname() {
  local dsn="$1" key uri db
  [ -n "$dsn" ] || { echo "pg_dsn_dbname: empty DSN" >&2; return 1; }
  # URL 格式: postgres:// 或 postgresql://
  if [[ "$dsn" == postgres://* || "$dsn" == postgresql://* ]]; then
    pg_uri_split "$dsn" uri
    [ -n "$uri_db" ] || { echo "pg_dsn_dbname: URL DSN 缺少数据库名: $dsn" >&2; return 1; }
    [ -n "$uri_host" ] || { echo "pg_dsn_dbname: URL DSN 缺少主机名: $dsn" >&2; return 1; }
    # URL 中的 dbname 参数优先于路径段(libpq 行为: 后者覆盖前者)
    if [ -n "$uri_params" ]; then
      local IFS='&' p
      for p in $uri_params; do
        case "$p" in
          dbname=*) db="${p#dbname=}";;
        esac
      done
    fi
    echo "${db:-$uri_db}"
    return 0
  fi
  # key=value 格式: 取 dbname= 的值(支持引号包裹)
  if [[ "$dsn" == *"dbname="* ]]; then
    key="${dsn#*dbname=}"
    key="${key%%[[:space:]]*}"
    # 去掉包裹引号(单/双)
    if [[ "$key" == \"*\" || "$key" == \'*\' ]]; then
      key="${key:1:${#key}-2}"
    fi
    [ -n "$key" ] || { echo "pg_dsn_dbname: dbname 为空: $dsn" >&2; return 1; }
    echo "$key"
    return 0
  fi
  echo "pg_dsn_dbname: 无法解析 DSN(需为 postgres:// URL 或含 dbname= 的 key=value 格式): $dsn" >&2
  return 1
}

# pg_dsn_normalize <dsn> — 输出归一化 DSN 用于日志/回显:
#   - URL 格式: 脱敏密码(保留 user@host/db)
#   - key=value 格式: 脱敏 password 值
pg_dsn_normalize() {
  local dsn="$1" uri result
  if [[ "$dsn" == postgres://* || "$dsn" == postgresql://* ]]; then
    pg_uri_split "$dsn" uri
    # shellcheck disable=SC2154  # uri_* 由 pg_uri_split 的 eval 赋值
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
  for tok in $dsn; do
    case "$tok" in
      password=*) result="$result password=***";;
      *) result="$result $tok";;
    esac
  done
  echo "${result# }"
}
