# Preserve 1Panel's OpenResty/WAF layout; update its private OpenSSL, not Ubuntu's CLI.
FROM 1panel/openresty@sha256:9eb6915fb7690f7c5d7c3e8aee939e8d403681bc5ee86802472c86d1b8c0a9d1 AS build
RUN apt-get update && apt-get install -y --no-install-recommends build-essential perl curl ca-certificates patch
WORKDIR /tmp/build
# OpenSSL release asset and OpenResty's session callback patch, both pinned and verified.
# https://github.com/openssl/openssl/releases/tag/openssl-3.5.9
# https://github.com/openresty/openresty/tree/0de3defb207c5557f5be2fce7d6f1d696b637ede/patches
RUN curl -fSL --retry 3 --connect-timeout 15 --max-time 180 https://github.com/openssl/openssl/releases/download/openssl-3.5.9/openssl-3.5.9.tar.gz -o openssl.tar.gz \
    && echo '603f5602e2eef00d77fbd429d34dcd5822bb301757a1bc9cdb24c670f1eb859a  openssl.tar.gz' | sha256sum -c - \
    && curl -fSL --retry 3 --connect-timeout 15 --max-time 180 https://raw.githubusercontent.com/openresty/openresty/0de3defb207c5557f5be2fce7d6f1d696b637ede/patches/openssl-3.5.5-sess_set_get_cb_yield.patch -o session.patch \
    && echo '0a30cc762a9d72901e8415a33f7671bb68469d46121061e26afe7b718f47581e  session.patch' | sha256sum -c - \
    && tar xzf openssl.tar.gz \
    && cd openssl-3.5.9 \
    && patch -p1 < ../session.patch \
    && ./config --prefix=/usr/local/openresty/openssl3 --libdir=lib \
        -Wl,-rpath,/usr/local/openresty/openssl3/lib shared \
    && make -j4 \
    && make DESTDIR=/stage install_sw

FROM 1panel/openresty@sha256:9eb6915fb7690f7c5d7c3e8aee939e8d403681bc5ee86802472c86d1b8c0a9d1
RUN apt-get update && apt-get -y --no-remove upgrade && rm -rf /var/lib/apt/lists/*
COPY --from=build /stage/usr/local/openresty/openssl3/ /usr/local/openresty/openssl3/
COPY --from=build /tmp/build/openssl-3.5.9/LICENSE.txt /usr/share/doc/yourtj-openresty-openssl/LICENSE.txt
RUN /usr/local/openresty/openssl3/bin/openssl version | grep -F 'OpenSSL 3.5.9' \
    && nginx -V 2>&1 | grep -F 'running with OpenSSL 3.5.9'
