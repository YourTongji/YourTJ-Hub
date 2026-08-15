# yourtj-hub 单二进制运行镜像(前端已 go:embed 进二进制)
FROM alpine:3.20

RUN apk add --no-cache ca-certificates tzdata git \
    && adduser -D -u 1000 app

WORKDIR /app
COPY --chmod=0755 yourtj-hub /app/yourtj-hub

# 挂载卷:
#   config.toml -> /app/config.toml (只读)
#   storage/    -> /app/storage    (SQLite 数据, uid 1000)
VOLUME ["/app/storage"]

USER app
EXPOSE 5234
ENTRYPOINT ["/app/yourtj-hub"]
CMD ["serve"]
