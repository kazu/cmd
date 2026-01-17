# 動的並行リクエスト制御

## 依頼内容
> 並列リクエスト最大数は　再生待ち行数 で引いて動的に変えてください。再生待ちのものが並列リクスト最大数と同じか多い場合は再生待ちのもの並列リクエスト最大すより少なくなるのを待ってください。

## 問題点
- 並行リクエスト数が固定で、再生待ちキューが溜まりすぎる可能性があった
- バッファが過剰になり、メモリ使用量が増加

## 実装内容

### 変更ファイル
- `openai_tts_realtime.html`

### 修正内容

**startButtonのクリックイベント内でバッチ処理を修正**

```javascript
while (currentIndex < texts.length) {
  // 【動的並行リクエスト制御】再生待ち行数を考慮して有効リクエスト数を計算
  let availableSlots = MAX_CONCURRENT_REQUESTS - pendingBytesQueue.length;
  
  // 再生待ち行数がMAX_CONCURRENT_REQUESTS以上の場合は待機
  let waitLoggedOnce = false;
  while (availableSlots <= 0) {
    if (!waitLoggedOnce) {
      addLog(`⏸️ 再生待ち行数が多いため待機中（待ち：${pendingBytesQueue.length}行、最大：${MAX_CONCURRENT_REQUESTS}行）`);
      waitLoggedOnce = true;
    }
    await new Promise(resolve => setTimeout(resolve, 500));
    availableSlots = MAX_CONCURRENT_REQUESTS - pendingBytesQueue.length;
  }
  
  // 現在のバッチを取得（最大で有効スロット数まで）
  const batchSize = Math.min(availableSlots, texts.length - currentIndex);
  // ...
}
```

### 計算式
```
availableSlots = MAX_CONCURRENT_REQUESTS - pendingBytesQueue.length
```

- `MAX_CONCURRENT_REQUESTS`: 並行リクエスト最大数（UIから設定、デフォルト3）
- `pendingBytesQueue.length`: 現在の再生待ち行数
- `availableSlots`: 新規リクエスト可能数

## 結果
- 再生待ちキューが溜まりすぎるのを防止
- メモリ使用量の削減
- 効率的なバッファ管理
- 待機時のログ出力（初回のみ）
