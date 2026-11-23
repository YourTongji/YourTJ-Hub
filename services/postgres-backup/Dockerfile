# PostgreSQL 16 + pgBackRest image for continuous WAL archiving and PITR to Cloudflare R2.
# Keeps the same Alpine musl base as postgres:16-alpine so existing pgdata volumes mount unchanged.
FROM postgres:16-alpine

RUN set -eux; \
    alpine_branch="v$(cut -d. -f1,2 /etc/alpine-release)"; \
    if ! grep -Eq "^https?://.*/${alpine_branch}/community" /etc/apk/repositories; then \
        echo "https://dl-cdn.alpinelinux.org/alpine/${alpine_branch}/community" >> /etc/apk/repositories; \
    fi; \
    apk add --no-cache ca-certificates pgbackrest; \
    mkdir -p /etc/pgbackrest /var/log/pgbackrest /var/spool/pgbackrest; \
    chown -R postgres:postgres /etc/pgbackrest /var/log/pgbackrest /var/spool/pgbackrest; \
    chmod 0750 /etc/pgbackrest /var/log/pgbackrest /var/spool/pgbackrest; \
    pgbackrest version

COPY --chmod=0755 pgbackrest-archive-push.sh /usr/local/bin/pgbackrest-archive-push.sh
