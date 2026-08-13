FROM node:20-alpine
WORKDIR /app
# 同时拷贝 lockfile 并用 npm ci, 保证镜像依赖与 review/冒烟时的依赖图一致
COPY package.json package-lock.json ./
RUN npm ci --omit=dev
COPY index.js server.js ./
COPY src ./src
ENV PORT=8300
EXPOSE 8300
CMD ["node", "server.js"]
