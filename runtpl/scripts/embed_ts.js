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

const startMarker = '<!-- OPENAI_TTS_REALTIME_T START -->';
const endMarker = '<!-- OPENAI_TTS_REALTIME_T END -->';
const inlineScript = `${startMarker}\n<script type="module">\n${js}\n</script>\n${endMarker}`;

const html = fs.readFileSync(htmlPath, 'utf8');
let newHtml = html;

// 1) 既存のマーカー付きブロックがあれば置換
const startEsc = startMarker.replace(/[-/\\^$*+?.()|[\]{}]/g, '\\$&');
const endEsc = endMarker.replace(/[-/\\^$*+?.()|[\]{}]/g, '\\$&');
const markerRe = new RegExp(startEsc + '[\s\S]*?' + endEsc, 'i');
if (markerRe.test(newHtml)) {
  newHtml = newHtml.replace(markerRe, inlineScript);
} else {
  // マーカーが不整合で残っている可能性があるため、孤立した開始/終了マーカーを先に除去
  newHtml = newHtml.replace(new RegExp(startEsc, 'g'), '');
  newHtml = newHtml.replace(new RegExp(endEsc, 'g'), '');

  // 2) 外部 script タグがあれば置換
  const re = /<script\s+type=(?:'|")module(?:'|")\s+src=(?:'|")\.\/openai_tts_realtime_t\.js(?:'|")\s*><\/script>/i;
  if (re.test(newHtml)) {
    newHtml = newHtml.replace(re, inlineScript);
  } else {
    // 3) 既存のインライン版で識別子を含むものを置換（過去の埋め込みを掃除）
    const inlineIdentRe = /<script[^>]*type=(?:'|")module(?:'|")[^>]*>[\s\S]*?__openai_tts_realtime_initialized[\s\S]*?<\/script>/i;
    if (inlineIdentRe.test(newHtml)) {
      newHtml = newHtml.replace(inlineIdentRe, inlineScript);
    } else {
      // 4) どれにも該当しなければ </body> の直前に挿入
      console.warn('外部 script タグが見つかりません。代わりに </body> の直前へインラインスクリプトを挿入します。');
      const bodyCloseRe = /<\/body>/i;
      if (bodyCloseRe.test(newHtml)) {
        newHtml = newHtml.replace(bodyCloseRe, inlineScript + '\n</body>');
      } else {
        // 最後に追記するフォールバック
        newHtml = newHtml + '\n' + inlineScript;
      }
    }
  }
}
// バックアップを残す
fs.copyFileSync(htmlPath, `${htmlPath}.bak`);
fs.writeFileSync(htmlPath, newHtml, 'utf8');
console.log('インライン化完了:', htmlPath, '(バックアップ:', `${htmlPath}.bak`, ')');
