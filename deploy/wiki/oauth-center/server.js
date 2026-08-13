// YourTJ-Hub: 自托管入口（上游 index.js 只导出 Koa callback，不 listen）
const http = require('http');
const handler = require('./index.js');
const port = Number(process.env.PORT) || 8300;
// 容器内监听 0.0.0.0: docker -p 的流量到达容器 eth0 而非容器内 loopback,
// 监听 127.0.0.1 会让宿主机回环绑定(compose "127.0.0.1:8300:8300")拒收。
// 公网暴露仍由 compose 的宿主侧 127.0.0.1 绑定保证。
http.createServer(handler).listen(port, '0.0.0.0', () => {
  console.log(`oauth-center listening on 0.0.0.0:${port}`);
});
