# UIパネルの折りたたみ機能

## 依頼内容
> これだと復活できないので、消すときはタイトルだけ出るように縮めてください。

## 問題点
- 各ビューパネルを×ボタンでクリックすると完全に非表示になり、復活できなかった

## 実装内容

### 変更ファイル
- `openai_tts_realtime.html`

### 修正内容

1. **togglePanel関数の追加**
```javascript
function togglePanel(contentId, iconId) {
  const content = document.getElementById(contentId);
  const icon = document.getElementById(iconId);
  
  if (content.style.display === 'none') {
    content.style.display = '';
    icon.textContent = '▼';
    icon.style.transform = 'rotate(0deg)';
  } else {
    content.style.display = 'none';
    icon.textContent = '▶';
    icon.style.transform = 'rotate(-90deg)';
  }
}
```

2. **各パネルのHTML構造変更**
- ステータスパネル（📊 再生状態）
- 波形表示パネル（🎵 波形表示）
- ログパネル（📋 デバッグログ）

3. **ヘッダー部分の修正例**
```html
<div class="flex justify-between items-center mb-4 cursor-pointer" 
     onclick="togglePanel('statusContent', 'statusIcon')">
  <h2 class="text-lg font-bold text-gray-800">📊 再生状態</h2>
  <span id="statusIcon" class="text-gray-500 text-xl transition-transform">▼</span>
</div>
<div id="statusContent">
  <!-- パネルの内容 -->
</div>
```

## 結果
- パネルのタイトルをクリックすると内容を折りたたむ
- タイトル部分は常に表示されているため、いつでも展開可能
- 折りたたみ時はアイコンが ▶ に変わる
