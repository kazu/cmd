# 3行で再生が止まる問題の修正

## 依頼内容
> 再生が3行で止まっています

## 問題点
- 再生が3行目で停止し、それ以降再生されない
- `playbackQueue`からの取り出し方法に問題があった
- `shift()`でキューの先頭を取り出していたが、`nextPlayIndex`と一致しない可能性

## 実装内容

### 変更ファイル
- `openai_tts_realtime.html`

### 修正内容

**schedulePlayback関数内のsourceNode.onendedハンドラを修正**

#### 【修正前】
```javascript
sourceNode.onended = () => {
  activePlayingSources.delete(myCnt);
  if (playbackQueue.length > 0) {
    // 問題: shift()で先頭を取るが、nextPlayIndexと一致しない可能性
    const nextIdx = playbackQueue.shift();
    setTimeout(() => {
      schedulePlayback(nextIdx);
    }, intervalPerLine * 1000);
  }
  updateCurPlaying();
};
```

#### 【修正後】
```javascript
sourceNode.onended = () => {
  activePlayingSources.delete(myCnt);
  if (playbackQueue.length > 0) {
    // 修正: nextPlayIndexの位置を検索して、見つかった場合のみ再生
    const nextIdxPosition = playbackQueue.indexOf(nextPlayIndex);
    if (nextIdxPosition >= 0) {
      playbackQueue.splice(nextIdxPosition, 1);
      setTimeout(() => {
        schedulePlayback(nextPlayIndex);
      }, intervalPerLine * 1000);
    }
  }
  updateCurPlaying();
};
```

### 修正ポイント
- `shift()` → `indexOf(nextPlayIndex)` + `splice()`
- キューから次に再生すべき行（`nextPlayIndex`）を正確に取り出す
- 順序が保証され、3行目以降も正しく再生される

## 結果
- 再生が3行で止まらなくなった ✅
- 全ての行が順序通りに再生される ✅
- キューから正しい行を取り出すことで連続再生を実現 ✅
