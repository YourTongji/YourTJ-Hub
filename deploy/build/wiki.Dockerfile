# yourtj-wiki 静态站镜像(nginx + VitePress 构建产物 + Pagefind 索引)
# 构建上下文 = /opt/yourtj/build(与 yourtj-hub 共用), 产物由 CI 打包上传后
# 由 deploy-wiki.sh 解包到 build/wiki-dist/, 此处 COPY 进 nginx 镜像。
FROM nginx:1.27-alpine

# 静态产物(VitePress dist, 含 /pagefind/ 索引)
COPY wiki-dist /usr/share/nginx/html
# SPA 路由回退 + 缓存头 + gzip(覆盖镜像自带 default.conf)
COPY wiki.nginx.conf /etc/nginx/conf.d/default.conf

EXPOSE 80
