# 処理状況表示のフィルタ改善（第3段階・最終）

## 依頼内容
> 処理状況が　現在の行番号　より小さい行数のものを消去する処理を追加してください

## 問題点（第2段階修正後）
- 現在再生中の行（`curPlaying`）より小さい行番号の行がまだ表示されていた
- 過去に完了した行が処理状況に残り続ける

## 実装内容

### 変更ファイル
- `openai_tts_realtime.html`

### 修正内容

**updateStatusDisplay関数の修正（第3段階・最終形）**

```javascript
function updateStatusDisplay() {
  const statusDisplay = document.getElementById('status-display');
  const activeItems = [];
  
  for (let i = 0; i < status.length; i++) {
    // 【3つのフィルタを適用】
    // 1. 完了済みでない（STATUS_END）
    // 2. 再生中でない（activePlayingSources）
    // 3. 現在の行より後の行（i > curPlaying）
    if (status[i] !== STATUS_END && !activePlayingSources.has(i) && i > curPlaying) {
      activeItems.push(i);
    }
  }
  
  // ...表示処理
}
```

### フィルタ条件（最終形）
1. `status[i] !== STATUS_END`: 完了でない
2. `!activePlayingSources.has(i)`: 再生中でない
3. `i > curPlaying`: 現在の行より後の行

## 結果
- 再生済み（STATUS_END）の行が消える ✅
- 現在再生中の行も消える ✅
- 過去の行（現在より前）も消える ✅
- **未来の準備済み/再生待ちの行のみ表示される** ✅

処理状況パネルには、これから再生する行のみが正しく表示されるようになった。
