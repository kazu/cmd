# 開始行番号の設定機能追加

## 依頼内容
> 設定で開始行を指定できるようにしてください

## 問題点
- 常に1行目から再生が始まるため、途中から再生したい場合に不便
- デバッグやテスト時に毎回最初から聞く必要があった

## 実装内容

### 変更ファイル
- `openai_tts_realtime.html`

### 修正内容

#### 【変更1: HTML - 開始行番号入力欄を追加】
```html
<div class="mb-4">
  <label class="block text-gray-200 text-sm font-bold mb-2">
    開始行番号
  </label>
  <input 
    type="number" 
    id="start-line" 
    min="1" 
    value="1"
    class="w-full px-4 py-2 bg-gray-800 text-white rounded-lg border border-gray-700 focus:ring-2 focus:ring-blue-500 focus:border-transparent"
    placeholder="開始行番号（デフォルト: 1）"
  >
</div>
```

#### 【変更2: JavaScript - 開始行番号の取得と処理】
```javascript
startButton.addEventListener('click', async function() {
  // 開始行番号を取得（UIから）
  let startLineNumber = parseInt(document.getElementById('start-line').value);
  if (isNaN(startLineNumber) || startLineNumber < 1) {
    startLineNumber = 1;
  }
  
  // テキストを行分割
  const allTexts = textInput.value.trim().split('\n').filter(t => t);
  
  // 開始行番号が全行数を超える場合は最終行から開始
  if (startLineNumber > allTexts.length) {
    startLineNumber = allTexts.length;
  }
  
  // 指定行から末尾までの行を取得（配列は0始まりなので -1）
  const texts = allTexts.slice(startLineNumber - 1);
  
  // nextPlayIndexを開始行番号に設定
  nextPlayIndex = startLineNumber - 1;
  
  addLog(`📖 再生開始: ${startLineNumber}行目から（全${allTexts.length}行中）`);
  
  // ...（以降の処理）
});
```

### 処理フロー
1. ユーザーが開始行番号を入力（例: 5）
2. 全テキストを行分割
3. 入力値のバリデーション
   - NaNまたは1未満 → 1に修正
   - 全行数超過 → 最終行に修正
4. `slice(startLineNumber - 1)`で指定行以降を抽出
5. `nextPlayIndex = startLineNumber - 1`で再生開始位置を設定
6. 指定行から再生開始

### 例
```
全テキスト:
1: Hello
2: World
3: Test
4: Message
5: End

開始行番号 = 3 の場合:
→ texts = ["Test", "Message", "End"]
→ nextPlayIndex = 2（配列の0始まり）
→ 3行目から再生開始
```

## 結果
- 任意の行から再生開始可能 ✅
- デバッグやテスト時に便利 ✅
- 入力値のバリデーション完備 ✅
- ログに開始行番号が表示される ✅
