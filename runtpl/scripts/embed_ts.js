// runtpl/scripts/embed_ts.js
const fs = require('fs');
const path = require('path');

let esbuild;
try {
  esbuild = require('esbuild');
} catch (e) {
  console.error('esbuild が見つかりません。まず `npm install` を実行してください。');
  process.exit(1);
}

const base = path.resolve(__dirname, '..');
const htmlPath = process.argv[2] ? path.resolve(process.argv[2]) : path.join(base, 'openai_tts_realtime_t.html');
const tsPath = process.argv[3] ? path.resolve(process.argv[3]) : path.join(base, 'openai_tts_realtime_t.ts');

if (!fs.existsSync(htmlPath)) { console.error('HTML が見つかりません:', htmlPath); process.exit(2); }
if (!fs.existsSync(tsPath)) { console.error('TS が見つかりません:', tsPath); process.exit(2); }

const result = esbuild.buildSync({
  entryPoints: [tsPath],
  bundle: true,
  platform: 'browser',
  format: 'esm',
  sourcemap: false,
  write: false,
  target: ['es2018'],
});

if (!result.outputFiles || result.outputFiles.length === 0) {
  console.error('esbuild が出力を返しませんでした。');
  process.exit(3);
}

let js = result.outputFiles[0].text;
// エスケープして安全に埋め込む
js = js.replace(/<\/script>/g, '<\\/script>');

const inlineScript = `<script type="module">\n${js}\n</script>`;

const html = fs.readFileSync(htmlPath, 'utf8');
const re = /<script\s+type=(?:'|")module(?:'|")\s+src=(?:'|")\.\/openai_tts_realtime_t\.js(?:'|")\s*><\/script>/i;
let newHtml;
if (re.test(html)) {
  newHtml = html.replace(re, inlineScript);
} else {
  console.warn('外部 script タグが見つかりません。代わりに </body> の直前へインラインスクリプトを挿入します。');
  const bodyCloseRe = /<\/body>/i;
  if (bodyCloseRe.test(html)) {
    newHtml = html.replace(bodyCloseRe, inlineScript + '\n</body>');
  } else {
    // 最後に追記するフォールバック
    newHtml = html + '\n' + inlineScript;
  }
}
// バックアップを残す
fs.copyFileSync(htmlPath, `${htmlPath}.bak`);
fs.writeFileSync(htmlPath, newHtml, 'utf8');
console.log('インライン化完了:', htmlPath, '(バックアップ:', `${htmlPath}.bak`, ')');
