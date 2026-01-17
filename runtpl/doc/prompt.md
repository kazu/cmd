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

