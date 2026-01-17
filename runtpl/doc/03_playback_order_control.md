# 再生順序の制御実装

## 依頼内容
> 再生順序がぐちゃぐちゃなのを順番にするように修正してください。

## 問題点
- 並行リクエストにより、APIレスポンスの到着順序がバラバラ
- データが到着した順に再生されていたため、順序が保証されていなかった

## 実装内容

### 変更ファイル
- `openai_tts_realtime.html`

### 修正内容

1. **nextPlayIndex変数の追加**
```javascript
let nextPlayIndex = 0; // 次に再生予定の行番号
```

2. **schedulePlayback関数に順序チェックを追加**
```javascript
async function schedulePlayback(myCnt) {
  // 【順序チェック】この行番号が次に再生される予定の行番号でなければ、キューに追加して待機
  if (myCnt !== nextPlayIndex) {
    playbackQueue.push(myCnt);
    addLog(`行${myCnt}: キューに登録（次は${nextPlayIndex}を待機）`);
    return;
  }
  
  // ... 再生処理 ...
  
  nextPlayIndex++; // 次の再生対象を更新
}
```

3. **再生終了時のキュー処理を修正**
```javascript
sourceNode.onended = () => {
  // ...
  
  // 【順序制御】キューに待機している行があれば、次を再生開始
  // nextPlayIndex と一致する行を探す
  if (playbackQueue.length > 0) {
    const nextIdxPosition = playbackQueue.indexOf(nextPlayIndex);
    if (nextIdxPosition >= 0) {
      playbackQueue.splice(nextIdxPosition, 1);
      addLog(`キューから取り出し: 行${nextPlayIndex}を再生開始`);
      schedulePlayback(nextPlayIndex);
    }
  }
};
```

4. **初期化処理でnextPlayIndexをリセット**
```javascript
nextPlayIndex = 0; // 再生順序をリセット
```

## 結果
- テキストが常に順番通りに再生される
- データが先に到着しても、前の行の再生が終わるまで待機
- キューから正しい順序で行を取り出して再生
