// YourTJ-Hub: 自托管入口（上游 index.js 只导出 Koa callback，不 listen）
const http = require('http');
const handler = require('./index.js');
const port = Number(process.env.PORT) || 8300;
http.createServer(handler).listen(port, '127.0.0.1', () => {
  console.log(`oauth-center listening on 127.0.0.1:${port}`);
});
