# ポップノイズの修正（フェード処理）

## 依頼内容
> １行再生ごとにブツという音が入るのですが修正をお願いします

## 問題点
- 音声の開始・終了時に「ブツ」という不快なクリック音（ポップノイズ）が発生
- 波形が急激に変化することが原因
- 特に行の切り替わり時に目立つ

## 実装内容

### 変更ファイル
- `openai_tts_realtime.html`

### 修正内容

**schedulePlayback関数内でFloat32Array変換後にフェード処理を追加**

```javascript
async function schedulePlayback(myCnt) {
  // ...（APIリクエスト処理）
  
  // PCM16からFloat32に変換
  const int16 = new Int16Array(arrayBuffer);
  const float32Data = new Float32Array(int16.length);
  for (let i = 0; i < int16.length; i++) {
    float32Data[i] = int16[i] / 32768.0;
  }
  
  // 【フェード処理を追加】ポップノイズ対策
  // フェードの長さは全体の2%または480サンプル（20ms）の短い方
  const fadeLength = Math.min(480, Math.floor(float32Data.length * 0.02));
  
  // フェードイン（開始部分）
  for (let i = 0; i < fadeLength; i++) {
    float32Data[i] *= i / fadeLength;
  }
  
  // フェードアウト（終了部分）
  for (let i = 0; i < fadeLength; i++) {
    float32Data[float32Data.length - 1 - i] *= i / fadeLength;
  }
  
  // AudioBufferを作成して再生
  const audioBuffer = audioContext.createBuffer(1, float32Data.length, 24000);
  audioBuffer.copyToChannel(float32Data, 0);
  // ...（再生処理）
}
```

### パラメータ
- **フェード長**: `Math.min(480, Math.floor(float32Data.length * 0.02))`
  - 480サンプル = 約20ms（24kHzの場合）
  - または音声全体の2%の短い方
- **フェードイン**: 音声開始時に0から徐々に音量を上げる
- **フェードアウト**: 音声終了時に徐々に音量を下げる0まで

### 計算式
```
フェードイン:  sample[i] *= (i / fadeLength)           // 0.0 → 1.0
フェードアウト: sample[i] *= (i / fadeLength)           // 1.0 → 0.0（末尾から）
```

## 結果
- ポップノイズ（ブツ音）が消えた ✅
- 滑らかな音声の開始・終了 ✅
- 行の切り替わりが自然になった ✅
- 20msのフェードは知覚されない程度の短さで、音声の内容に影響しない ✅
