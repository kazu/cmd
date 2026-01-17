# openai_tts_realtime.html 改良指示と実装内容

## 改良指示（TODO.md より）

```
- [ ] UIの見た目がよくないので改良（現在は http://ry1:19999/test?message=gun_virus_steel1）
- [ ] 朗読するテキストを１行ずつ読むが、朗読の時間を正確に測れないので、場合によって同時再生されるのを抑制したい。
```

## 実装内容

### 1. UIの見た目の改良

#### ビジュアルデザイン
- **背景**: グレーの単調な背景からグラデーション背景（slate-900→slate-800）に変更
- **レイアウト**: 従来の上下レイアウトから、レスポンシブな左右2カラムレイアウトに変更
  - 左側（lg:col-span-2）: 入力エリアと波形表示
  - 右側（lg:col-span-1）: ステータスパネル（sticky で固定表示）

#### コンポーネント改良
- **ヘッダー**: タイトルに絵文字（🎙️）を追加し、説明文を追加
- **入力フォーム**: カード型（白背景、シャドウ）でグループ化
  - APIキー入力に注釈を追加
  - テキスト入力に使用方法の説明を追加
  - プレースホルダーテキストを改善

#### ボタンのデザイン
- 各ボタンに絵文字アイコンを追加（▶️ 再生開始、⏸️ 一時停止、⏹️ 再生停止）
- グラデーション背景とホバーエフェクトを適用
- より大きく視認しやすいサイズに（py-3 px-6）

#### ステータスパネル（新規追加）
- **現在の行番号**: 大きく表示（text-3xl）
- **前の文と現在の文**: 異なるスタイルで表示（現在は左ボーダー付き）
- **処理状況**: ステータスバッジで色分け表示
  - 待機中（青）、準備中（黄）、完了（緑）
- **再生時間情報**: 経過時間と残り時間をリアルタイム表示

#### 波形表示
- 背景をグラデーション（紫）に変更
- 波形線を白色に変更
- より大きなサイズ（800×200）に拡大
- border-radius でコーナーを丸く

#### デバッグログパネル（新規追加）
- すべての重要なイベントをタイムスタンプ付きで表示
  - APIリクエスト送信
  - APIレスポンス受信
  - データ受信完了
  - 再生開始/完了
  - 強制停止、エラー
- ログクリアボタン

### 2. 朗読時間の正確な計測と同時再生抑制

#### 実装内容

**actualDurations 辞書の追加**
```javascript
let actualDurations = {}; // { myCnt: duration }
```
- 各テキスト（myCnt）の実際の再生時間を記録
- AudioBuffer の duration から取得した値を保存

**正確な再生時間の取得**
```javascript
const actualDuration = audioBuffer.duration;
actualDurations[myCnt] = actualDuration;
```
- AudioBuffer 作成時に、その buffer.duration を記録
- ストリーム全体ではなく、個別のテキストごとの時間を計測

**タイムアウト判定の改善**
```javascript
function getRemainingTime(myCnt) {
  if (actualDurations[myCnt] !== undefined) {
    return actualDurations[myCnt] * 1000 + 500; // 余裕を持たせる
  }
  // フォールバック処理
}
```
- 実際に記録された再生時間を基準に使用
- 500ms の余裕を追加してタイムアウトを判定

**ステータス追跡の厳密化**
```javascript
const STATUS_START = 0;      // 処理開始
const STATUS_PUSHED = 1;     // キューに追加
const STATUS_END = 2;        // 再生完了
```
- 各テキストの処理状態を3段階で管理
- 前の行が STATUS_END に達するまで次の行を再生しない

**リアルタイム計測表示**
```javascript
function updateTimerDisplay() {
  const elapsed = audioContext.currentTime - playStarts[curPlaying];
  const remaining = sourceNode.buffer.duration - elapsed;
  // UI に表示
}
setInterval(updateTimerDisplay, 100);
```
- 100ms ごとに経過時間と残り時間をリアルタイム更新

**同時再生抑制メカニズム**
```javascript
// 前の行が処理されるまで待つ
while (myCnt > 0 && status[myCnt - 1] < STATUS_PUSHED) { }

// 前の行の再生終了を待つ
while (myCnt > 0 && status[myCnt - 1] < STATUS_END) {
  schedulePlayback(myCnt - 1);
  // 条件チェック...
}
```
- Promise の then チェーンで順序を保証
- 状態チェックで確実に前の行を待つ

### 3. その他の改善

- **ログ機能**: `addLog()` 関数ですべてのイベントをログ記録
- **エラーハンドリング**: `.catch()` で API エラーをログに出力
- **初期化処理**: 「再生開始」ボタン押下時に状態をリセット
- **API遅延の最適化**: リクエスト間の遅延を 1000ms から 500ms に短縮
- **ボタンラベルの改善**: 「一時停止」→「⏸️ 一時停止」、「再開」→「▶️ 再開」
- **パラメータ検証**: テキストを「。」で分割した後に空文字をフィルタ

## 技術的な補足

### 使用したCSS フレームワーク
- Tailwind CSS（responsive design, gradient, shadow）
- カスタム CSS（.status-badge, .waveform-container）

### 使用した JavaScript API
- Web Audio API（AudioContext, AudioBuffer, AnalyserNode）
- Fetch API（streaming response）
- requestAnimationFrame（波形アニメーション）
- setInterval（タイマー更新）

### ブラウザ互換性
- Chrome/Edge: 全機能対応
- Firefox: 全機能対応
- Safari: webkit プレフィックス対応

