const express = require('express');
const fs = require('fs');
const path = require('path');
const os = require('os');

const app = express();
const port = 3000;

// サーバーのIPアドレス（IPv4）を取得する関数
function getLocalIPAddress() {
  const interfaces = os.networkInterfaces();
  for (const name of Object.keys(interfaces)) {
    for (const net of interfaces[name]) {
      if (net.family === 'IPv4' && !net.internal) {
        return net.address;
      }
    }
  }
  return '127.0.0.1';
}

app.get('/', (req, res) => {
  // ブラウザやプロキシにキャッシュさせない
  res.set('Cache-Control', 'no-store, no-cache, must-revalidate, proxy-revalidate');
  res.set('Pragma', 'no-cache');
  res.set('Expires', '0');
  res.set('Surrogate-Control', 'no-store');

  // サーバー側の情報
  const serverHostname = os.hostname();
  const serverIp = getLocalIPAddress();

  // クライアント側の情報
  const clientIp = req.headers['x-forwarded-for'] || req.socket.remoteAddress;

  // public/index.html のパスを指定
  const filePath = path.join(__dirname, 'public', 'index.html');

  // HTMLファイルを読み込む
  fs.readFile(filePath, 'utf8', (err, htmlContent) => {
    if (err) {
      res.status(500).send('内部エラーが発生しました。');
      return;
    }

    // HTML内のプレースホルダーを実際の値に置換
    const renderedHtml = htmlContent
      .replace('__SERVER_HOSTNAME__', serverHostname)
      .replace('__SERVER_IP__', serverIp)
      .replace('__CLIENT_IP__', clientIp);

    // 置換後のHTMLを送信
    res.send(renderedHtml);
  });
});

app.listen(port, () => {
  console.log(`Server is running at http://localhost:${port}`);
});
