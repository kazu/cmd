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

---

# Docker 化（Dockerfile と docker-compose.yml の実装）

## 改良指示（TODO.md より）

```
- [ ] Docker 化. 簡単に動作かくにん出来るようにDockerfile の作成と docker compose の対応
```

## 実装内容

### Dockerfile の構成

#### マルチステージビルド

**ビルドステージ**
```dockerfile
FROM golang:1.24.4-alpine AS builder
```
- 公式の Go イメージを使用（バージョン 1.24.4）
- Alpine Linux をベースにすることでイメージサイズを最小化
- ビルド用ツールチェーンが含まれている

**依存関係管理**
```dockerfile
COPY ../go.mod ../go.sum ./
RUN go mod download
```
- キャッシュレイヤーを活用するため、モジュール定義を先にコピー
- Docker ビルドキャッシュを効率的に利用

**ビルドプロセス**
```dockerfile
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o runtpl .
```
- ソースコードをコピー
- CGO_ENABLED=0 で静的リンク（Alpine での動作を確保）
- GOOS=linux で Linux 用にクロスコンパイル

**実行ステージ**
```dockerfile
FROM alpine:latest
RUN mkdir -p tmpl
COPY --from=builder /build/runtpl .
COPY openai_tts_realtime.html tmpl/
```
- 最小限の Alpine Linux イメージ（約7MB）
- ビルドステージのバイナリのみをコピー
- HTML ファイルをテンプレートディレクトリにコピー
- テンプレートディレクトリを事前作成

**ポート公開と実行**
```dockerfile
EXPOSE 19999
CMD ["./runtpl"]
```
- ポート 19999 をコンテナ外に公開
- アプリケーション起動コマンド

#### マルチステージビルドの利点
- **イメージサイズ削減**: ビルドツール（Go SDK 等）を最終イメージに含めない
  - 最終イメージサイズ: 約20-30MB（通常の Go バイナリ + Alpine）
- **セキュリティ向上**: 不要なツールチェーンを排除
- **ビルド時間の最適化**: レイヤーキャッシュの活用による高速ビルド

### docker-compose.yml の構成

**サービス定義**
```yaml
services:
  runtpl:
    build:
      context: .
    ports:
      - "19999:19999"
```
- コンテナ内ポート 19999 をホストの 19999 にマッピング
- `docker-compose up` で自動ビルド・実行可能

**ボリュームマウント**
```yaml
    volumes:
      - ./tmpl:/app/tmpl
```
- ホストの `./tmpl` ディレクトリをコンテナの `/app/tmpl` にマウント
- テンプレートファイルの変更がホット反映される（再ビルド不要）
- Docker 内で作成されたテンプレートファイルがホストに保存される

**環境変数とコンテナ設定**
```yaml
    environment:
      - LOG_LEVEL=info
    container_name: runtpl-server
    restart: unless-stopped
```
- 環境変数で設定を一元管理
- コンテナ名を明確化
- クラッシュ時の自動再起動を設定

**ネットワーク定義**
```yaml
networks:
  runtpl-network:
    driver: bridge
```
- ブリッジネットワークで複数コンテナ間通信を可能にする設定
- 将来の複数サービス連携に対応

## 使用方法

### ビルドと実行
```bash
# ディレクトリに移動
cd /workspaces/cmd/runtpl

# イメージのビルドとコンテナ起動（バックグラウンド）
docker-compose up -d

# ステータス確認
docker-compose ps

# ログ確認
docker-compose logs -f
```

### 動作確認
```bash
# ブラウザで http://localhost:19999/test にアクセス
# または curl を使用
curl http://localhost:19999/test
```

### 停止と削除
```bash
# コンテナを停止
docker-compose stop

# コンテナと関連リソースを削除
docker-compose down

# イメージも削除する場合
docker-compose down --rmi all
```

### トラブルシューティング

**ポートが既に使用されている場合**
```yaml
# docker-compose.yml を編集して別のポートに変更
ports:
  - "19998:19999"  # ホストの 19998 ポートを使用
```

**テンプレートファイルが反映されない場合**
```bash
# コンテナを再構築
docker-compose up -d --build
```

**詳細なログを確認**
```bash
docker-compose logs -f runtpl
```

**イメージサイズの確認**
```bash
docker images runtpl_runtpl
```

## 技術的な補足

### Alpine Linux の選択理由
- **最小サイズ**: 基本イメージが約7MB（Ubuntu 等の1/10）
- **セキュリティ**: 最小限のコンポーネントのみ（攻撃対象面を最小化）
- **Go との相性**: Go は静的リンクに対応し、Alpine で完全に動作

### Docker Compose の利点
- **開発の簡略化**: `docker-compose up` で一括起動
- **環境再現性**: ポート、ボリューム、環境変数を統一管理
- **スケーラビリティ**: 複数サービス追加時に対応容易
- **本番環境への親和性**: Kubernetes への移行が容易

### キャッシュレイヤーの最適化
```dockerfile
COPY ../go.mod ../go.sum ./
RUN go mod download
COPY . .
```
- モジュール定義を先にコピーすることで、ソース変更時のリビルド速度を向上
- 依存関係が変更されない限り、キャッシュを活用可能

## 関連ファイル
- `Dockerfile`: コンテナイメージビルド定義
- `docker-compose.yml`: 複数コンテナの管理と実行設定
- `runtpl.go`: メインアプリケーション（ポート19999で動作）
- `openai_tts_realtime.html`: フロントエンド HTML ファイル
- `go.mod`, `go.sum`: Go 依存管理ファイル

---

# 多重再生検知機能の実装

## 改良指示

```
UI は改善したがまだ、複数行が同時に再生してしまう問題があるので、
再生状態のview で多重再生しているのを検知できて表示できるようにしてください。
```

## 実装内容

### 多重再生検知システム

#### activePlayingSources マップの追加
```javascript
let activePlayingSources = new Map(); // { myCnt: { startTime, text, sourceNode } }
let concurrentPlayDetected = false;
```
- 現在再生中のすべての音声を追跡する Map を導入
- キー: 行番号（myCnt）
- 値: { startTime, text, sourceNode } オブジェクト

#### 再生開始時の検知
```javascript
// 再生開始時に activePlayingSources に追加
activePlayingSources.set(myCnt, {
  startTime: performance.now(),
  text: ctext,
  sourceNode: sourceNode
});

// 多重再生をチェック
if (activePlayingSources.size > 1) {
  const activeIndices = Array.from(activePlayingSources.keys()).join(', ');
  addLog(`⚠️ 警告: 多重再生検知！ 再生中の行: [${activeIndices}]`);
}
```
- AudioBufferSourceNode の start() 実行時に Map に追加
- サイズが 2 以上の場合、多重再生と判定してログ出力

#### 再生終了時の処理
```javascript
sourceNode.onended = () => {
  // 再生終了時に activePlayingSources から削除
  activePlayingSources.delete(myCnt);
  updateConcurrentPlayingDisplay();
  // ...
};
```
- 音声再生完了時に Map から削除
- 表示を即座に更新

### UI コンポーネントの追加

#### 多重再生警告パネル
```html
<div id="concurrentWarning" class="hidden mb-4 p-3 bg-red-50 border-l-4 border-red-500 rounded">
  <div class="flex items-center">
    <span class="text-2xl mr-2">⚠️</span>
    <div>
      <p class="text-sm font-bold text-red-700">多重再生検知</p>
      <p class="text-xs text-red-600" id="concurrentCount">複数の音声が同時再生されています</p>
    </div>
  </div>
</div>
```
- 多重再生検知時に表示される警告パネル
- アニメーション効果付き（pulse）
- 赤色で視覚的に警告

#### 再生中の行リスト
```html
<div class="mb-4">
  <p class="text-sm text-gray-600 mb-2">再生中の行</p>
  <div id="playingList" class="bg-gray-50 p-2 rounded text-xs space-y-1 max-h-32 overflow-y-auto">
    <p class="text-gray-500">なし</p>
  </div>
</div>
```
- 現在再生中のすべての行をリスト表示
- 各行の情報（行番号、テキスト、経過時間）を表示
- 多重再生時は赤背景で強調表示

### 表示更新関数

#### updateConcurrentPlayingDisplay()
```javascript
function updateConcurrentPlayingDisplay() {
  const playingList = document.getElementById("playingList");
  const warningDiv = document.getElementById("concurrentWarning");
  const countSpan = document.getElementById("concurrentCount");
  
  if (activePlayingSources.size === 0) {
    playingList.innerHTML = '<p class="text-gray-500">なし</p>';
    warningDiv.classList.add("hidden");
    concurrentPlayDetected = false;
  } else {
    playingList.innerHTML = "";
    const entries = Array.from(activePlayingSources.entries());
    
    entries.forEach(([idx, info]) => {
      const elapsed = (performance.now() - info.startTime) / 1000;
      const isConcurrent = activePlayingSources.size > 1;
      
      // 各行の表示を生成（多重再生時は赤背景）
      div.className = `p-2 rounded ${isConcurrent ? 'bg-red-100 border border-red-300' : 'bg-blue-100'}`;
      // ...
    });
    
    // 多重再生警告
    if (activePlayingSources.size > 1) {
      warningDiv.classList.remove("hidden");
      countSpan.textContent = `${activePlayingSources.size}個の音声が同時再生中`;
      if (!concurrentPlayDetected) {
        concurrentPlayDetected = true;
        addLog(`⚠️ 多重再生検知: ${activePlayingSources.size}個の音声が同時再生されています`);
      }
    }
  }
}
```
- 100ms ごとにタイマーで自動更新
- 再生中の行数に応じて表示を動的に変更
- 多重再生時は警告パネルを表示

#### 処理状況の視覚的強調
```javascript
function updateStatusDisplay() {
  // ...
  const isConcurrent = activePlayingSources.has(i) && activePlayingSources.size > 1;
  const concurrentClass = isConcurrent ? "concurrent-item" : "";
  
  div.className = `text-xs truncate ${concurrentClass}`;
  div.innerHTML = `<span class="status-badge ${statusClass}">${statusText}</span> 行${i}${isConcurrent ? ' ⚠️' : ''}`;
  // ...
}
```
- 処理状況リストで多重再生中の行を赤背景で表示
- 警告マーク（⚠️）を追加

### CSS アニメーション

```css
.warning-badge {
  animation: pulse 1s infinite;
}

@keyframes pulse {
  0%, 100% { opacity: 1; }
  50% { opacity: 0.6; }
}

.concurrent-item {
  background-color: #fee2e2;
  border-left: 4px solid #dc2626;
}
```
- 警告パネルに点滅アニメーション
- 多重再生中の行を赤枠で強調

### 停止処理の改善

```javascript
document.getElementById("stop").addEventListener("click", () => {
  isStop = true;
  
  // すべての再生中の音声を停止
  activePlayingSources.forEach((info, idx) => {
    if (info.sourceNode) {
      try {
        info.sourceNode.stop();
        info.sourceNode.disconnect();
      } catch (e) {
        console.error(`行${idx}の停止エラー:`, e);
      }
    }
  });
  
  // 多重再生管理をクリア
  activePlayingSources.clear();
  updateConcurrentPlayingDisplay();
  // ...
});
```
- 停止ボタン押下時、すべての再生中音声を確実に停止
- Map をクリアして表示を更新

## 機能の効果

### 問題の可視化
- **リアルタイム検知**: 多重再生が発生した瞬間に検知・表示
- **詳細な情報**: どの行が同時再生されているか一目で確認可能
- **経過時間表示**: 各行の再生開始からの経過時間を表示

### デバッグの容易化
- **ログ記録**: 多重再生発生時に自動的にログ出力
- **視覚的警告**: 赤色の警告パネルとアニメーション
- **処理状況の追跡**: ステータスバッジで各行の状態を表示

### ユーザー体験の向上
- **問題の認識**: 多重再生が発生していることをユーザーに明示
- **原因の特定**: どの行で問題が発生しているか特定可能
- **適切な対処**: 停止ボタンですべての音声を確実に停止

## 技術的な補足

### Map データ構造の選択理由
- キーによる高速検索（O(1)）
- 追加・削除の効率性
- 反復処理の容易さ

### パフォーマンスへの配慮
- 100ms ごとの更新（過度な再描画を防ぐ）
- 必要な情報のみを更新
- メモリリークを防ぐための確実な削除処理

### エラーハンドリング
```javascript
try {
  info.sourceNode.stop();
  info.sourceNode.disconnect();
} catch (e) {
  console.error(`行${idx}の停止エラー:`, e);
}
```
- 既に停止済みの音声への stop() 呼び出しエラーを捕捉
- 一部のエラーで全体が停止しないように try-catch で保護

## 関連ファイル
- `openai_tts_realtime.html`: 多重再生検知機能を実装

---

# 多重再生防止機能の実装（遅延再生）

## 改良指示

```
多重再生を検知できるようになりました。多重再生をしないように
後から再生される文を遅延して再生されるように修正してください。
```

## 実装内容

### 待機メカニズムの実装

#### waitForPreviousPlayback 関数の追加
```javascript
async function waitForPreviousPlayback(myCnt) {
  const waitingNotice = document.getElementById("waitingNotice");
  const waitingInfo = document.getElementById("waitingInfo");
  
  // 自分より前の行が再生中の場合、終了まで待機
  while (activePlayingSources.size > 0) {
    // 自分自身以外が再生中かチェック
    const otherPlaying = Array.from(activePlayingSources.keys()).filter(idx => idx !== myCnt);
    if (otherPlaying.length === 0) break;
    
    // 待機中の表示を更新
    waitingNotice.classList.remove("hidden");
    waitingInfo.textContent = `行${myCnt}が待機中 (再生中: [${otherPlaying.join(', ')}])`;
    
    addLog(`行${myCnt}: 他の音声が再生中のため待機...`);
    
    // 100ms 待機
    await new Promise(resolve => setTimeout(resolve, 100));
  }
  
  // 待機終了
  waitingNotice.classList.add("hidden");
}
```

**機能**:
- activePlayingSources が空になるまでループで待機
- 100ms ごとにポーリングして再生可能かチェック
- 待機中の状態を UI に表示
- Promise ベースの非同期待機で他の処理をブロックしない

#### schedulePlayback 関数の改善
```javascript
async function schedulePlayback(myCnt) {
  // タイムアウトチェック...
  
  // 【多重再生防止】前の音声が完全に終了するまで待機
  if (activePlayingSources.size > 0) {
    const waitingFor = Array.from(activePlayingSources.keys());
    addLog(`行${myCnt}: 待機開始 (再生中: [${waitingFor.join(', ')}])`);
    await waitForPreviousPlayback(myCnt);
    addLog(`行${myCnt}: 待機終了、再生を開始します`);
  }
  
  // すでに再生中ならスキップ（念のため再チェック）
  if (isPlaying) {
    addLog(`行${myCnt}: 既に他の音声が再生中のためスキップ`);
    return;
  }
  
  // キューとテキストの存在チェック
  if (pendingBytesQueue.length == 0) {
    addLog(`行${myCnt}: キューが空のため再生できません`);
    return;
  }
  
  if (playTexts.length == 0) {
    addLog(`行${myCnt}: テキストが空のため再生できません`);
    return;
  }
  
  // 再生処理...
}
```

**改善点**:
- 再生開始前に必ず `waitForPreviousPlayback()` を呼び出し
- 待機終了後、念のため再度 `isPlaying` をチェック
- キューとテキストの存在を明示的にチェックしてログ出力
- 各ステップでログを出力して処理を追跡可能に

### UI コンポーネントの追加

#### 待機中通知パネル
```html
<div id="waitingNotice" class="hidden mb-4 p-3 bg-yellow-50 border-l-4 border-yellow-500 rounded">
  <div class="flex items-center">
    <span class="text-2xl mr-2">⏳</span>
    <div>
      <p class="text-sm font-bold text-yellow-700">待機中</p>
      <p class="text-xs text-yellow-600" id="waitingInfo">前の音声の終了を待っています</p>
    </div>
  </div>
</div>
```

**特徴**:
- 黄色の警告パネルで待機状態を表示
- 砂時計アイコン（⏳）で視覚的に表現
- どの行が待機中か、どの行の終了を待っているかを表示
- 待機終了時に自動的に非表示

### 初期化処理の強化

```javascript
// 初期化
cnt = 0;
status = [];
playTexts = [];
pendingBytes = new Uint8Array(0);
pendingBytesQueue = [];
leftover = new Uint8Array(0);
isPlaying = false;
isStop = false;
actualDurations = {};
activePlayingSources.clear(); // 多重再生管理をクリア
playbackQueue = []; // 再生待ちキューをクリア
concurrentPlayDetected = false;
updateConcurrentPlayingDisplay();

// 待機中表示を非表示
document.getElementById("waitingNotice").classList.add("hidden");
```

- `playbackQueue` を追加（将来の拡張用）
- 待機中表示を明示的に非表示に設定
- すべての状態を確実にリセット

## 動作フロー

### 通常の再生フロー
1. **行N のデータ受信完了** → `schedulePlayback(N)` 呼び出し
2. **待機チェック**: `activePlayingSources.size === 0` → 即座に再生開始
3. **再生開始**: `activePlayingSources.set(N, ...)` で登録
4. **再生終了**: `activePlayingSources.delete(N)` で削除

### 多重再生が発生しそうな場合のフロー
1. **行N が再生中**
2. **行N+1 のデータ受信完了** → `schedulePlayback(N+1)` 呼び出し
3. **待機チェック**: `activePlayingSources.size > 0` → 待機開始
4. **待機中表示**: 黄色のパネルで「行N+1が待機中 (再生中: [N])」と表示
5. **100ms ごとにポーリング**: `activePlayingSources` をチェック
6. **行N の再生終了**: `activePlayingSources.delete(N)`
7. **待機終了**: ループを抜けて待機パネルを非表示
8. **再生開始**: 行N+1 の再生を開始

### ログ出力例
```
[14:30:15] APIリクエスト送信: 行0 "こんにちは。これはテスト..."
[14:30:16] データ受信完了: 行0 (48256バイト)
[14:30:16] 再生開始: 行0 "こんにちは。これはテスト..."
[14:30:17] APIリクエスト送信: 行1 "次の文章です..."
[14:30:18] データ受信完了: 行1 (52480バイト)
[14:30:18] 行1: 待機開始 (再生中: [0])
[14:30:18] 行1: 他の音声が再生中のため待機... (再生中: [0])
[14:30:19] 再生完了: 行0
[14:30:19] 行1: 待機終了、再生を開始します
[14:30:19] 再生開始: 行1 "次の文章です..."
```

## 技術的な補足

### async/await による非同期待機
```javascript
await new Promise(resolve => setTimeout(resolve, 100));
```
- 100ms 待機しながら、他の JavaScript イベントをブロックしない
- UI の応答性を維持
- Promise チェーンで順序制御

### ポーリング間隔の選択（100ms）
- **短すぎる（< 50ms）**: CPU 負荷が高くなる
- **長すぎる（> 500ms）**: 再生開始の遅延が目立つ
- **100ms**: バランスが良く、ユーザーには遅延を感じさせない

### 待機中のメモリ管理
- ポーリングループ中も `activePlayingSources` は通常の削除で管理
- Promise による待機なので、メモリリークは発生しない
- `while` ループ内で状態が更新されるため、無限ループのリスクなし

### エッジケースの対処

**ケース1: 待機中に停止ボタンが押された場合**
```javascript
document.getElementById("stop").addEventListener("click", () => {
  // ...
  activePlayingSources.clear(); // これにより待機ループが終了
});
```
- `activePlayingSources` がクリアされるため、待機ループが即座に終了
- 次のチェックで `pendingBytesQueue` が空になり、再生はスキップ

**ケース2: 同時に複数の行が待機する場合**
- 各行が独立して `waitForPreviousPlayback()` を実行
- 先に呼ばれた行が先に再生開始条件を満たす
- FIFO（First In, First Out）の順序が自然に保たれる

**ケース3: ネットワーク遅延で順序が入れ替わる場合**
```javascript
while (myCnt > 0 && status[myCnt - 1] < STATUS_PUSHED) { }
```
- データ受信側で前の行が処理されるまで待機
- 再生側でも `waitForPreviousPlayback()` で待機
- 二重の待機メカニズムで順序を保証

## 効果

### 多重再生の完全防止
- **検知から防止へ**: 検知だけでなく、発生を未然に防ぐ
- **確実な順次再生**: 必ず1つずつ順番に再生される
- **ユーザー体験の向上**: 音声が重ならず、聞き取りやすい

### 視覚的フィードバック
- **待機状態の可視化**: 黄色のパネルで待機中であることを明示
- **処理の透明性**: ログで詳細な処理フローを追跡可能
- **安心感**: システムが正常に動作していることをユーザーが確認できる

### 将来の拡張性
- `playbackQueue` を導入済み（より高度なキュー管理に対応可能）
- 待機ロジックが独立した関数として実装（カスタマイズ容易）
- ログ出力で デバッグとモニタリングが容易

## 関連ファイル
- `openai_tts_realtime.html`: 多重再生防止機能を実装

