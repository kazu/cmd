# OpenAI TTS 音声番号と再生速度の設定機能

## 依頼内容
> OpenAI TTS に投げる 音声番号と再生速度を設定で変更できるように。

## 問題点
- OpenAI TTS の音声番号と速度が固定値（voice: "1", speed: 1）で変更できなかった
- 異なる音声で試したい場合や再生速度を調整したい場合に不便

## 実装内容

### 変更ファイル
- `openai_tts_realtime.html`

### 修正内容

#### 【変更1: HTML - 設定フィールドを追加】

設定セクション内に2つの新しい入力欄を追加：

```html
<div class="mb-3">
  <label class="block mb-1 text-sm text-gray-600">音声番号:</label>
  <input
    type="number"
    id="voiceNumber"
    class="block w-32 p-2 border border-gray-300 rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500"
    value="1"
    min="0"
    max="999"
  />
  <p class="text-xs text-gray-500 mt-1">OpenAI TTS に投げる音声番号（0～999）</p>
</div>
<div class="mb-3">
  <label class="block mb-1 text-sm text-gray-600">再生速度:</label>
  <input
    type="number"
    id="playbackSpeed"
    class="block w-32 p-2 border border-gray-300 rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500"
    value="1.0"
    min="0.1"
    max="3.0"
    step="0.1"
  />
  <p class="text-xs text-gray-500 mt-1">再生速度（0.1～3.0）</p>
</div>
```

#### 【変更2: JavaScript - 設定値を取得する関数】

```javascript
function getVoiceNumber() {
  return document.getElementById("voiceNumber").value || "1";
}

function getPlaybackSpeed() {
  return parseFloat(document.getElementById("playbackSpeed").value) || 1.0;
}
```

#### 【変更3: APIリクエストで動的に設定値を使用】

```javascript
fetch("http://10.2.1.15:19999/run2", {
  method: "POST",
  headers: {
    "Content-Type": "application/json"
  },
  body: JSON.stringify({
    model: model,
    input: text,
    voice: getVoiceNumber(),        // ← 動的に取得
    speed: getPlaybackSpeed(),      // ← 動的に取得
    response_format: responseFormat
  })
})
```

#### 【変更4: ページロード時にクエリパラメータから設定値を読み込む】

```javascript
window.addEventListener("load", () => {
  updateTextDisplay();
  loadTemplateFiles();
  
  const urlParams = new URLSearchParams(window.location.search);
  
  // voiceNumberを読み込む
  const voiceNum = urlParams.get('voiceNumber');
  if (voiceNum !== null) {
    document.getElementById("voiceNumber").value = voiceNum;
    addLog(`🎤 音声番号を設定: ${voiceNum}`);
  }
  
  // playbackSpeedを読み込む
  const speed = urlParams.get('playbackSpeed');
  if (speed !== null) {
    document.getElementById("playbackSpeed").value = speed;
    addLog(`⏱️ 再生速度を設定: ${speed}`);
  }
  
  // 設定値の変更イベントをリッスン
  document.getElementById("voiceNumber").addEventListener("change", (e) => {
    addLog(`🎤 音声番号を変更: ${e.target.value}`);
  });
  
  document.getElementById("playbackSpeed").addEventListener("change", (e) => {
    addLog(`⏱️ 再生速度を変更: ${e.target.value}`);
  });
});
```

### 使用方法

1. **UIで直接設定**
   - 「音声番号」フィールドに0～999の値を入力
   - 「再生速度」フィールドに0.1～3.0の値を入力
   - 再生開始ボタンで新しい設定値でAPIリクエスト

2. **URLパラメータで指定**
   - `?voiceNumber=5&playbackSpeed=1.5` を付加して開く
   - ページロード時に自動的に設定値が適用される
   - 例: `http://localhost:19999/test?message=sample&voiceNumber=3&playbackSpeed=0.8`

## 結果
- OpenAI TTS の音声番号をUIから変更可能 ✅
- 再生速度をUIから変更可能 ✅
- URLパラメータで設定値をプリセット可能 ✅
- 設定値の変更時にログに記録される ✅
- APIリクエストに動的に反映される ✅
