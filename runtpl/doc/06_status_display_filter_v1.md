# 処理状況表示のフィルタ改善（第1段階）

## 依頼内容
> まだ　再生中、再生済みの行が処理状況の再生待ちに表示されています

## 問題点
- ステータス表示に`STATUS_END`（完了）の行が表示されていた
- 再生済みの行が「再生待ち」として誤表示

## 実装内容

### 変更ファイル
- `openai_tts_realtime.html`

### 修正内容

**updateStatusDisplay関数の修正（第1段階）**

```javascript
function updateStatusDisplay() {
  const statusDisplay = document.getElementById('status-display');
  const activeItems = [];
  
  for (let i = 0; i < status.length; i++) {
    // STATUS_ENDの行を除外
    if (status[i] !== STATUS_END) {
      activeItems.push(i);
    }
  }
  
  // ...表示処理
}
```

### フィルタ条件
- `status[i] !== STATUS_END`: 完了（STATUS_END = 2）でない行のみ表示

## 結果
- 再生済み（STATUS_END）の行が処理状況から消える
- ただし、まだ再生中の行も表示される問題が残った（次の修正で対応）
