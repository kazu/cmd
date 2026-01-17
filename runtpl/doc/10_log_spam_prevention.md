# ログの重複出力抑制

## 依頼内容
> 連続で "再生待ち行数が多いため待機中" という行をlog に出力しないようにしてください

## 問題点
- 再生待ち行数が多い場合、while ループ内で500msごとにログ出力されていた
- ログパネルが同じメッセージで埋まり、可読性が低下
- 実質的には同じ待機状態なのに何度もログが出る

## 実装内容

### 変更ファイル
- `openai_tts_realtime.html`

### 修正内容

**startButtonのクリックイベント内のバッチ処理ループを修正**

#### 【修正前】
```javascript
while (availableSlots <= 0) {
  // 問題: ループのたびにログ出力される（500msごと）
  addLog(`⏸️ 再生待ち行数が多いため待機中（待ち：${pendingBytesQueue.length}行、最大：${MAX_CONCURRENT_REQUESTS}行）`);
  await new Promise(resolve => setTimeout(resolve, 500));
  availableSlots = MAX_CONCURRENT_REQUESTS - pendingBytesQueue.length;
}
```

#### 【修正後】
```javascript
let waitLoggedOnce = false; // フラグを追加
while (availableSlots <= 0) {
  // 修正: 初回のみログ出力
  if (!waitLoggedOnce) {
    addLog(`⏸️ 再生待ち行数が多いため待機中（待ち：${pendingBytesQueue.length}行、最大：${MAX_CONCURRENT_REQUESTS}行）`);
    waitLoggedOnce = true; // フラグを立てる
  }
  await new Promise(resolve => setTimeout(resolve, 500));
  availableSlots = MAX_CONCURRENT_REQUESTS - pendingBytesQueue.length;
}
```

### 修正ポイント
- `waitLoggedOnce` フラグを追加
- 初回のみログ出力（`if (!waitLoggedOnce)`）
- ループが続いても重複ログなし

## 結果
- 待機ログが1回のみ出力される ✅
- ログパネルの可読性が向上 ✅
- 重要なメッセージが埋もれなくなる ✅
