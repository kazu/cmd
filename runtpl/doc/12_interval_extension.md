# 行間インターバルの延長

## 依頼内容
> 行の再生の音声をもう少しインターバルを開けてください

## 問題点
- 行と行の間隔が短く、連続再生が早すぎて聞き取りにくい
- `intervalPerLine = 0.5秒`では不十分

## 実装内容

### 変更ファイル
- `openai_tts_realtime.html`

### 修正内容

#### 【変更1: intervalPerLine定数の変更】
```javascript
// 修正前
const intervalPerLine = 0.5; // 秒

// 修正後
const intervalPerLine = 1.0; // 秒
```

#### 【変更2: onendedハンドラでのsetTimeout使用】
```javascript
sourceNode.onended = () => {
  activePlayingSources.delete(myCnt);
  
  if (playbackQueue.length > 0) {
    const nextIdxPosition = playbackQueue.indexOf(nextPlayIndex);
    if (nextIdxPosition >= 0) {
      playbackQueue.splice(nextIdxPosition, 1);
      // setTimeoutで遅延を挿入
      setTimeout(() => {
        schedulePlayback(nextPlayIndex);
      }, intervalPerLine * 1000); // 1000ms = 1秒
    }
  }
  
  updateCurPlaying();
};
```

### 修正ポイント
1. `intervalPerLine`を0.5秒から1.0秒に延長
2. `setTimeout`で次の行の再生前に明示的に待機
3. `intervalPerLine * 1000`ミリ秒（= 1000ms = 1秒）の遅延

## タイミング図
```
行1: [====音声再生====]           
              ↓(onended)
              [1秒待機]
                  ↓
行2:            [====音声再生====]
                      ↓(onended)
                      [1秒待機]
                          ↓
行3:                    [====音声再生====]
```

## 結果
- 行と行の間に1秒の間隔が空く ✅
- 聞き取りやすさが向上 ✅
- 音声が途切れることなく、適度な間で再生される ✅
