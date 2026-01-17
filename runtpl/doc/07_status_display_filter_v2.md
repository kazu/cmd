# 処理状況表示のフィルタ改善（第2段階）

## 依頼内容
> まだ　再生中、再生済みの行が処理状況の再生待ちに表示されています

## 問題点（第1段階修正後）
- STATUS_ENDは除外されたが、現在再生中の行（`activePlayingSources`に登録されている行）も表示されていた

## 実装内容

### 変更ファイル
- `openai_tts_realtime.html`

### 修正内容

**updateStatusDisplay関数の修正（第2段階）**

```javascript
function updateStatusDisplay() {
  const statusDisplay = document.getElementById('status-display');
  const activeItems = [];
  
  for (let i = 0; i < status.length; i++) {
    // STATUS_ENDと再生中の行を除外
    if (status[i] !== STATUS_END && !activePlayingSources.has(i)) {
      activeItems.push(i);
    }
  }
  
  // ...表示処理
}
```

### フィルタ条件
1. `status[i] !== STATUS_END`: 完了でない
2. `!activePlayingSources.has(i)`: 再生中でない

## 結果
- 再生済み（STATUS_END）の行が消える
- 現在再生中の行も処理状況から消える
- ただし、過去に完了した行がまだ表示される問題が残った（次の修正で対応）
