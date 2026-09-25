'use strict';

const express = require('express');
const session = require('express-session');
const path = require('path');
const os = require('os');

const PORT = process.env.PORT || 3000;
const COOKIE_NAME = process.env.SESSION_COOKIE_NAME || 'sid';
const INSTANCE_ID = process.env.INSTANCE_ID || os.hostname(); // K8s なら Pod 名になる
const STARTED_AT = new Date().toISOString();

function localIPs() {
  return Object.values(os.networkInterfaces())
    .flat()
    .filter((i) => i && i.family === 'IPv4' && !i.internal)
    .map((i) => i.address);
}

const app = express();
app.set('trust proxy', true); // X-Forwarded-For からクライアントIPを取得

// 全レスポンスに振り分け先を付与
app.use((req, res, next) => {
  res.set('X-Served-By', INSTANCE_ID);
  res.set('Cache-Control', 'no-store');
  next();
});

// ヘルスチェック(セッションを作らないよう session ミドルウェアより前に定義)
app.get('/healthz', (req, res) => res.status(200).send('ok'));

// 静的ファイル(public/)。セッション・カウント対象外、キャッシュ無効
app.use(
  express.static(path.join(__dirname, 'public'), {
    etag: false,          // ETag を付けない(304 応答を防ぐ)
    lastModified: false,  // Last-Modified を付けない
    setHeaders: (res) => {
      res.set('Cache-Control', 'no-store, no-cache, must-revalidate');
      res.set('Pragma', 'no-cache'); // HTTP/1.0 の古いプロキシ向け
      res.set('Expires', '0');
    },
  })
);

// セッション: MemoryStore(インスタンスごとに独立)
app.use(
  session({
    name: COOKIE_NAME,
    secret: process.env.SESSION_SECRET || 'change-me',
    resave: false,
    saveUninitialized: true,
    cookie: { httpOnly: true, maxAge: 30 * 60 * 1000 },
  })
);

// セッション破棄(httpOnly Cookie はブラウザの JS から消せないためサーバー側で処理)
app.post('/api/reset', (req, res) => {
  req.session.destroy(() => {
    res.clearCookie(COOKIE_NAME);
    res.status(204).end();
  });
});

// カウントアップして状態を返す
app.get('/api', (req, res) => {
  const cookieSent = (req.headers.cookie || '').includes(`${COOKIE_NAME}=`);
  const isNew = req.session.count === undefined;

  if (isNew) {
    req.session.count = 0;
    req.session.createdAt = new Date().toISOString();
  }
  req.session.count += 1;

  res.set('Cache-Control', 'no-store');
  res.json({
    instanceId: INSTANCE_ID,
    hostname: os.hostname(),
    serverIPs: localIPs(),
    serverStartedAt: STARTED_AT,
    clientIP: req.ip,
    xForwardedFor: req.headers['x-forwarded-for'] || null,
    sessionId: req.sessionID,
    sessionCreatedAt: req.session.createdAt,
    count: req.session.count,
    newSession: isNew,
    // Cookie を送ってきたのにこのインスタンスにセッションが無い = 別インスタンスに振られた
    persistenceBroken: cookieSent && isNew,
    time: new Date().toISOString(),
  });
});

app.listen(PORT, () => {
  console.log(`session-counter listening on :${PORT} (instance=${INSTANCE_ID})`);
});
