# 行番号クリックでURL生成・再生機能

## 依頼内容
> 再生状態の現在の行番号の番号をクリックしたらその行から再生できるURLを開く

## 問題点
- 特定の行から再生したい場合、毎回手動でURLパラメータを構築する必要があった
- 再生中に別の行から再生したくても不便

## 実装内容

### 変更ファイル
- `openai_tts_realtime.html`

### 修正内容

#### 【変更1: HTML - 行番号を クリック可能に】

「現在の行番号」表示をクリック可能にして、ホバー時に視覚的フィードバック：

```html
<div class="mb-4">
  <p class="text-sm text-gray-600 mb-1">現在の行番号</p>
  <p class="text-3xl font-bold text-blue-600 cursor-pointer hover:text-blue-800" 
     id="curIdx" 
     onclick="generateLineURLAndOpen()" 
     title="クリックするとこの行から再生するURLを開きます">0</p>
  <p class="text-xs text-gray-500 mt-1">クリックで指定行から再生するURLを開く</p>
</div>
```

#### 【変更2: JavaScript - URL生成・オープン関数】

```javascript
/**
 * 現在の行番号をクリックして、その行から再生するURLを生成して開く
 */
function generateLineURLAndOpen() {
  const curLineNum = parseInt(document.getElementById("curIdx").textContent) || 0;
  const voiceNum = getVoiceNumber();
  const speed = getPlaybackSpeed();
  const selectedFile = document.getElementById("templateFileSelect").value;
  
  // 現在のページのURLを基に、パラメータを追加したURLを生成
  const currentURL = window.location.href;
  const baseURL = currentURL.split('?')[0]; // クエリ文字列を削除
  
  let url = `${baseURL}?startLineNumber=${curLineNum}`;
  
  if (selectedFile) {
    url += `&message=${encodeURIComponent(selectedFile)}`;
  }
  
  url += `&voiceNumber=${voiceNum}&playbackSpeed=${speed}`;
  
  addLog(`🔗 URLを開く: ${url}`);
  window.open(url, '_blank');
}
```

### 含まれるパラメータ

生成されるURLには以下のパラメータが含まれます：

| パラメータ | 値 | 用途 |
|-----------|-----|------|
| `startLineNumber` | 現在の行番号 | ここから再生開始 |
| `message` | 選択されたファイル名 | テンプレートファイル |
| `voiceNumber` | 現在の音声番号 | OpenAI TTS の音声設定 |
| `playbackSpeed` | 現在の再生速度 | 再生速度設定 |

### 使用方法

1. 再生中に「現在の行番号」をクリック
2. その行から再生するプリセット済みURLが新しいタブで開く
3. 自動的に：
   - 指定行から再生開始
   - 同じテンプレートファイルが読み込まれる
   - 同じ音声番号・再生速度が適用される

### 例

再生中に行20で止めたい場合：
1. 行番号「20」をクリック
2. 以下のようなURLが開く：
   ```
   http://localhost:19999/test?startLineNumber=20&message=sample&voiceNumber=3&playbackSpeed=0.8
   ```
3. 新しいタブで自動的に20行目から再生開始

## 結果
- 行番号クリックでURL自動生成 ✅
- 現在の設定を保持したまま新規開始 ✅
- テンプレートファイルも自動復元 ✅
- ログに生成されたURLが記録される ✅
- 複数の別ウィンドウでの並列再生が可能 ✅
