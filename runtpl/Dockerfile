# マルチステージビルド: ビルドステージ
FROM golang:1.24.4-alpine AS builder

WORKDIR /build

# キャッシュを活用するため、go.mod と go.sum を先にコピー
COPY go.mod go.sum ./

# 依存関係をダウンロード
RUN go mod download

# runtpl ディレクトリのソースコードをコピー
COPY runtpl/ ./runtpl/
WORKDIR /build/runtpl

# Go アプリケーションをビルド
RUN CGO_ENABLED=0 GOOS=linux go build -o runtpl .

# マルチステージビルド: 実行ステージ
FROM alpine:latest

WORKDIR /app

# 必要なユーティリティをインストール
RUN apk --no-cache add ca-certificates

# ビルドステージからバイナリをコピー
COPY --from=builder /build/runtpl/runtpl .

# テンプレートファイルディレクトリを作成
RUN mkdir -p tmpl

# HTML ファイルをコピー
COPY runtpl/openai_tts_realtime.html tmpl/

# ポート19999を公開
EXPOSE 19999

# アプリケーション起動
CMD ["./runtpl"]
