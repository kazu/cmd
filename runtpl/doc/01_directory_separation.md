# テンプレートファイルとテキストファイルのディレクトリ分離

## 依頼内容
> いやテンプレートファイルは今まで通りにしてください。　再生する音声ファイルのみ tmpl/text 以下にするようにしてください。

## 問題点
- `fileDir` が `./tmpl/text/` に設定されていたため、テンプレートHTMLファイル（`test.html`）も同じディレクトリから読み込まれていた
- テンプレートファイルと音声テキストファイルが同じディレクトリに混在していた

## 実装内容

### 変更ファイル
- `runtpl.go`

### 修正内容

1. **新しい定数を追加**
```go
const (
	tmplDir  = "./tmpl/"      // テンプレートHTMLファイル用
	fileDir = "./tmpl/text/"  // 再生する音声ファイル用
)
```

2. **staticer関数の修正**
```go
func staticer(fname string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// ...
		f, err := os.Open(filepath.Join(tmplDir, fname))  // tmplDir を使用
		// ...
	}
}
```

## 結果
- テンプレートHTMLファイルは `tmpl/` ディレクトリに配置
- 音声テキストファイルは `tmpl/text/` ディレクトリに配置
- `/listfiles` エンドポイントは `tmpl/text/` 配下のファイルを返す
