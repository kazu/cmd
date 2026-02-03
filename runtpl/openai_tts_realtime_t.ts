// TypeScript conversion of openai_tts_realtime_t.html inline script
// Behavioral parity with original JavaScript version.

// ========== パネル折りたたみ機能 ==========
function togglePanel(contentId: string, iconId: string): void {
  const content = document.getElementById(contentId) as HTMLElement;
  const icon = document.getElementById(iconId) as HTMLElement;

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

// ========== ロギング機能 ==========
function addLog(message: string): void {
  const logContainer = document.getElementById('logContainer') as HTMLElement;
  const timestamp = new Date().toLocaleTimeString('ja-JP');
  const logEntry = document.createElement('div');
  logEntry.textContent = `[${timestamp}] ${message}`;
  logContainer.appendChild(logEntry);
  logContainer.scrollTop = logContainer.scrollHeight;
}

// ========== AudioContext の初期化 ==========
const audioContext = new (window.AudioContext || (window as any).webkitAudioContext)({
  sampleRate: 24000
});

// ========== 受信データ管理 ==========
let pendingBytes: Uint8Array = new Uint8Array(0);
let pendingBytesQueue: Uint8Array[] = [];
let leftover: Uint8Array = new Uint8Array(0);

// ========== 再生制御フラグ ==========
let isPlaying = false;
let sourceNode: AudioBufferSourceNode | null = null;
let idxForceStop = -1;
let nextPlayIndex = 0; // 次に再生予定の行番号

// ========== 多重再生検知 ==========
type ActiveInfo = { startTime: number; text: string; sourceNode: AudioBufferSourceNode | null };
const activePlayingSources: Map<number, ActiveInfo> = new Map(); // { myCnt: { startTime, text, sourceNode } }
let concurrentPlayDetected = false;
let playbackQueue: number[] = []; // 再生待ちのキュー（行番号を格納）

// ========== 並行リクエスト管理 ==========
let MAX_CONCURRENT_REQUESTS = 3; // 同時にリクエストする音声の数（UIから変更可能）
const activeRequests: Set<number> = new Set(); // 現在リクエスト中の行番号

/**
 * 前の音声が完全に終了するまで待機する関数
 */
async function waitForPreviousPlayback(myCnt: number): Promise<void> {
  const waitingNotice = document.getElementById('waitingNotice') as HTMLElement;
  const waitingInfo = document.getElementById('waitingInfo') as HTMLElement;
  let hasLogged = false;

  while (activePlayingSources.size > 0) {
    const otherPlaying = Array.from(activePlayingSources.keys()).filter(idx => idx !== myCnt);
    if (otherPlaying.length === 0) break;

    waitingNotice.classList.remove('hidden');
    waitingInfo.textContent = `行${myCnt}が待機中 (再生中: [${otherPlaying.join(', ')}])`;

    if (!hasLogged) {
      addLog(`行${myCnt}: 他の音声が再生中のため待機... (再生中: [${otherPlaying.join(', ')}])`);
      hasLogged = true;
    }

    await new Promise(resolve => setTimeout(resolve, 100));
  }

  waitingNotice.classList.add('hidden');
}

// ========== AnalyserNode（波形可視化用） ==========
const analyserNode = audioContext.createAnalyser();
analyserNode.fftSize = 1024;
const dataArray = new Uint8Array(analyserNode.frequencyBinCount);

// Canvas の描画コンテキスト
const waveCanvas = document.getElementById('waveCanvas') as HTMLCanvasElement;
const waveCtx = waveCanvas.getContext('2d') as CanvasRenderingContext2D;

// ========== 時間管理 ==========
const playTimeout = 15 * 1000; // 15秒
const intervalPerLine = 1.0; // 行間の時間（秒）

// 各アイテムの実際の再生時間を記録する辞書
const actualDurations: { [k: number]: number } = {}; // { myCnt: duration }

/** 波形を描画する関数 */
function drawWaveform(): void {
  requestAnimationFrame(drawWaveform);
  waveCtx.fillStyle = '#667eea';
  waveCtx.fillRect(0, 0, waveCanvas.width, waveCanvas.height);

  analyserNode.getByteTimeDomainData(dataArray);

  waveCtx.lineWidth = 2;
  waveCtx.strokeStyle = '#fff';
  waveCtx.beginPath();

  const sliceWidth = waveCanvas.width * 1.0 / dataArray.length;
  let x = 0;

  for (let i = 0; i < dataArray.length; i++) {
    const v = dataArray[i] / 128.0;
    const y = (v * waveCanvas.height) / 2;
    if (i === 0) waveCtx.moveTo(x, y);
    else waveCtx.lineTo(x, y);
    x += sliceWidth;
  }

  waveCtx.lineTo(waveCanvas.width, waveCanvas.height / 2);
  waveCtx.stroke();
}

let playTexts: string[] = [];
let playStart: number = performance.now();
let statuses: number[] = [];
let playStarts: number[] = [];

const STATUS_START = 0;
const STATUS_PUSHED = 1;
const STATUS_END = 2;

let curPlaying = 0;

// ========== ステータス更新表示 ==========
function updateStatusDisplay(): void {
  const container = document.getElementById('statusContainer') as HTMLElement;
  container.innerHTML = '';

  const activeItems: number[] = [];
  for (let i = 0; i < statuses.length; i++) {
    if (statuses[i] !== STATUS_END && !activePlayingSources.has(i) && i > curPlaying) {
      activeItems.push(i);
    }
  }

  const displayItems = activeItems.slice(Math.max(0, activeItems.length - 5));

  for (const i of displayItems) {
    const statusText = ['準備中', '再生待ち', '完了'][statuses[i]] || '不明';
    const statusClass = ['status-start', 'status-pushed', 'status-end'][statuses[i]] || 'status-start';

    const div = document.createElement('div');
    div.className = `text-xs truncate`;
    div.innerHTML = `<span class=\"status-badge ${statusClass}\">${statusText}</span> 行${i}`;
    container.appendChild(div);
  }

  updateBufferStatus();
}

function updateBufferStatus(): void {
  const requesting = document.getElementById('requestingCount') as HTMLElement;
  const queued = document.getElementById('queuedCount') as HTMLElement;
  requesting.textContent = String(activeRequests.size);
  queued.textContent = String(pendingBytesQueue.length);
}

// ========== 多重再生検知と表示 ==========
function updateConcurrentPlayingDisplay(): void {
  const playingList = document.getElementById('playingList') as HTMLElement;
  const warningDiv = document.getElementById('concurrentWarning') as HTMLElement;
  const countSpan = document.getElementById('concurrentCount') as HTMLElement;

  if (activePlayingSources.size === 0) {
    playingList.innerHTML = '<p class=\"text-gray-500\">なし</p>';
    warningDiv.classList.add('hidden');
    concurrentPlayDetected = false;
  } else {
    playingList.innerHTML = '';
    const entries = Array.from(activePlayingSources.entries());

    entries.forEach(([idx, info]) => {
      const div = document.createElement('div');
      const elapsed = (performance.now() - info.startTime) / 1000;
      const isConcurrent = activePlayingSources.size > 1;

      div.className = `p-2 rounded ${isConcurrent ? 'bg-red-100 border border-red-300' : 'bg-blue-100'}`;
      div.innerHTML = `
        <div class=\"font-bold ${isConcurrent ? 'text-red-700' : 'text-blue-700'}\">\n          ${isConcurrent ? '⚠️ ' : '▶️ '}行${idx}\n        </div>\n        <div class=\"text-xs text-gray-600 truncate\">${info.text.substring(0, 30)}...</div>\n        <div class=\"text-xs text-gray-500\">経過: ${elapsed.toFixed(1)}秒</div>\n      `;
      playingList.appendChild(div);
    });

    if (activePlayingSources.size > 1) {
      warningDiv.classList.remove('hidden');
      countSpan.textContent = `${activePlayingSources.size}個の音声が同時再生中`;
      if (!concurrentPlayDetected) {
        concurrentPlayDetected = true;
        addLog(`⚠️ 多重再生検知: ${activePlayingSources.size}個の音声が同時再生されています`);
      }
    } else {
      warningDiv.classList.add('hidden');
      concurrentPlayDetected = false;
    }
  }
}

function updateTimerDisplay(): void {
  const elapsedElem = document.getElementById('elapsedTime') as HTMLElement;
  const remainingElem = document.getElementById('remainingTime') as HTMLElement;

  if (curPlaying < 0 || !playStarts[curPlaying]) {
    elapsedElem.textContent = '0.0';
    remainingElem.textContent = '0.0';
    updateBufferStatus();
    return;
  }

  if (isPlaying && sourceNode && sourceNode.buffer) {
    const elapsed = audioContext.currentTime - playStarts[curPlaying];
    const remaining = sourceNode.buffer.duration - elapsed;
    elapsedElem.textContent = elapsed.toFixed(1);
    remainingElem.textContent = Math.max(0, remaining).toFixed(1);
  }

  updateConcurrentPlayingDisplay();
  updateBufferStatus();
}

setInterval(updateTimerDisplay, 100);

function EndStatuses(cur: number): void {
  if (cur - 5 < 0) return;
  for (let i = Math.max(0, cur - 5); i <= cur; i++) {
    if (statuses[i] !== STATUS_END) statuses[i] = STATUS_END;
  }
  updateStatusDisplay();
}

function getRemainingTime(myCnt: number): number {
  if (actualDurations[myCnt] !== undefined) {
    return actualDurations[myCnt] * 1000 + 500;
  }
  if (sourceNode == null) return playTimeout;
  const totalDuration = sourceNode.buffer!.duration + intervalPerLine;
  return Math.max(playTimeout, totalDuration * 1000);
}

// 再生をスケジューリングする関数
async function schedulePlayback(myCnt: number): Promise<void> {
  if (myCnt !== nextPlayIndex) {
    playbackQueue.push(myCnt);
    addLog(`行${myCnt}: キューに登録（次は${nextPlayIndex}を待機）`);
    return;
  }

  if (isPlaying && performance.now() - playStart > getRemainingTime(myCnt)) {
    isPlaying = false;
    EndStatuses(curPlaying);
    idxForceStop = curPlaying;
    addLog(`強制停止: ${curPlaying} (タイムアウト)`);
    console.log('force stop: ', curPlaying);
    return;
  }

  if (activePlayingSources.size > 0) {
    const waitingFor = Array.from(activePlayingSources.keys());
    addLog(`行${myCnt}: 待機開始 (再生中: [${waitingFor.join(', ')}])`);
    await waitForPreviousPlayback(myCnt);
    addLog(`行${myCnt}: 待機終了、再生を開始します`);
  }

  if (isPlaying) {
    addLog(`行${myCnt}: 既に他の音声が再生中のためスキップ`);
    return;
  }

  if (pendingBytesQueue.length === 0) {
    addLog(`行${myCnt}: キューが空のため再生できません`);
    return;
  }

  if (playTexts.length === 0) {
    addLog(`行${myCnt}: テキストが空のため再生できません`);
    return;
  }

  curPlaying = myCnt;
  isPlaying = true;
  nextPlayIndex++;

  playStart = performance.now();
  const ctext = playTexts.shift()!;
  (document.getElementById('prevText') as HTMLElement).innerText = (document.getElementById('currentText') as HTMLElement).innerText;
  (document.getElementById('currentText') as HTMLElement).innerText = ctext;
  (document.getElementById('curIdx') as HTMLElement).innerText = String(curPlaying);

  addLog(`再生開始: 行${curPlaying} "${ctext.substring(0, 20)}..."`);

  const dataToPlay = pendingBytesQueue.shift()!;

  const int16Data = new Int16Array(dataToPlay.buffer);
  const float32Data = new Float32Array(int16Data.length);
  for (let i = 0; i < int16Data.length; i++) float32Data[i] = int16Data[i] / 32768.0;

  const fadeLength = Math.min(480, Math.floor(float32Data.length * 0.02));
  for (let i = 0; i < fadeLength; i++) {
    const fadeMultiplier = i / fadeLength;
    float32Data[i] *= fadeMultiplier;
  }
  for (let i = 0; i < fadeLength; i++) {
    const fadeMultiplier = i / fadeLength;
    float32Data[float32Data.length - 1 - i] *= fadeMultiplier;
  }

  const audioBuffer = audioContext.createBuffer(1, float32Data.length, audioContext.sampleRate);
  audioBuffer.copyToChannel(float32Data, 0);

  const actualDuration = audioBuffer.duration;
  actualDurations[myCnt] = actualDuration;
  addLog(`音声データ長: 行${myCnt} ${actualDuration.toFixed(2)}秒`);

  sourceNode = audioContext.createBufferSource();
  sourceNode.buffer = audioBuffer;

  activePlayingSources.set(myCnt, {
    startTime: performance.now(),
    text: ctext,
    sourceNode
  });

  if (activePlayingSources.size > 1) {
    const activeIndices = Array.from(activePlayingSources.keys()).join(', ');
    addLog(`⚠️ 警告: 多重再生検知！ 再生中の行: [${activeIndices}]`);
  }

  updateConcurrentPlayingDisplay();

  sourceNode.connect(analyserNode);
  analyserNode.connect(audioContext.destination);

  sourceNode.onended = () => {
    isPlaying = false;
    activePlayingSources.delete(myCnt);
    updateConcurrentPlayingDisplay();
    highlightActiveLine();

    if (idxForceStop > -1 && idxForceStop !== curPlaying) {
      EndStatuses(curPlaying - 1);
      idxForceStop = -1;
      return;
    }
    EndStatuses(curPlaying);
    addLog(`再生完了: 行${curPlaying}`);
    idxForceStop = -1;

    if (playbackQueue.length > 0) {
      const nextIdxPosition = playbackQueue.indexOf(nextPlayIndex);
      if (nextIdxPosition >= 0) {
        playbackQueue.splice(nextIdxPosition, 1);
        addLog(`キューから取り出し: 行${nextPlayIndex}を${intervalPerLine}秒後に再生開始`);
        setTimeout(() => {
          schedulePlayback(nextPlayIndex);
        }, intervalPerLine * 1000);
      } else {
        addLog(`警告: キューに次の行${nextPlayIndex}が見つかりません（キュー: [${playbackQueue.join(', ')}]）`);
      }
    }
  };

  sourceNode.start();
  playStarts[curPlaying] = audioContext.currentTime;
}

// 波形描画開始
drawWaveform();

let isPause = false;
let isStop = false;

(document.getElementById('pause') as HTMLButtonElement).addEventListener('click', () => {
  if (isPause) {
    audioContext.resume().then(() => {
      isPause = false;
      (document.getElementById('pause') as HTMLButtonElement).innerText = '⏸️ 一時停止';
      addLog('再開しました');
    });
  } else {
    audioContext.suspend().then(() => {
      isPause = true;
      (document.getElementById('pause') as HTMLButtonElement).innerText = '▶️ 再開';
      addLog('一時停止しました');
    });
  }
});

// 再生停止ボタンのクリックイベント
(document.getElementById('stop') as HTMLButtonElement).addEventListener('click', () => {
  isStop = true;
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

  if (sourceNode) {
    try { sourceNode.stop(); } catch {}
    try { sourceNode.disconnect(); } catch {}
    isPlaying = false;
    pendingBytes = new Uint8Array(0);
    leftover = new Uint8Array(0);
  }

  activePlayingSources.clear();
  updateConcurrentPlayingDisplay();
  addLog('再生を停止しました');
});

(document.getElementById('clearLog') as HTMLButtonElement).addEventListener('click', () => {
  (document.getElementById('logContainer') as HTMLElement).innerHTML = '';
});

// ========== テンプレートファイル選択機能 ==========
async function loadTemplateFiles(): Promise<void> {
  try {
    const response = await fetch('http://10.2.1.15:19999/listfiles');
    const data = await response.json();
    const select = document.getElementById('templateFileSelect') as HTMLSelectElement;

    select.innerHTML = '<option value="">-- ファイルを選択 --</option>';
    data.files.forEach((file: string) => {
      if (!file.endsWith('.html')) {
        const option = document.createElement('option');
        option.value = file;
        option.textContent = file;
        select.appendChild(option);
      }
    });
  } catch (error) {
    console.error('テンプレートファイル一覧の読み込みエラー:', error);
  }
}

async function loadTemplateFileContent(selectedFile: string): Promise<void> {
  if (!selectedFile) return;
  try {
    const response = await fetch(`http://10.2.1.15:19999/test?message=${encodeURIComponent(selectedFile)}`);
    const html = await response.text();
    const parser = new DOMParser();
    const doc = parser.parseFromString(html, 'text/html');
    const textarea = doc.querySelector('#textInput') as HTMLTextAreaElement | null;
    if (textarea) {
      (document.getElementById('textInput') as HTMLTextAreaElement).value = textarea.textContent || '';
      addLog(`テンプレート読み込み: ${selectedFile}`);
      updateTextDisplay();
    }
  } catch (error) {
    console.error('テンプレート読み込みエラー:', error);
    addLog(`エラー: テンプレート「${selectedFile}」を読み込めません`);
  }
}

(document.getElementById('templateFileSelect') as HTMLSelectElement).addEventListener('change', async (e) => {
  const selectedFile = (e.target as HTMLSelectElement).value;
  if (!selectedFile) return;
  await loadTemplateFileContent(selectedFile);
});

window.addEventListener('load', async () => {
  updateTextDisplay();
  const urlParams = new URLSearchParams(window.location.search);
  const startLine = urlParams.get('l');
  if (startLine !== null) {
    (document.getElementById('startLineNumber') as HTMLInputElement).value = startLine;
    addLog(`📍 開始行番号を設定: ${startLine}`);
  }
  const voiceNum = urlParams.get('voiceNumber');
  if (voiceNum !== null) {
    (document.getElementById('voiceNumber') as HTMLInputElement).value = voiceNum;
    addLog(`🎤 音声番号を設定: ${voiceNum}`);
  }
  const speed = urlParams.get('playbackSpeed');
  if (speed !== null) {
    (document.getElementById('playbackSpeed') as HTMLInputElement).value = speed;
    addLog(`⏱️ 再生速度を設定: ${speed}`);
  }

  await loadTemplateFiles();

  const message = urlParams.get('message');
  if (message !== null) {
    (document.getElementById('templateFileSelect') as HTMLSelectElement).value = message;
    addLog(`📄 テンプレートファイルを設定: ${message}`);
    await loadTemplateFileContent(message);
  }

  (document.getElementById('voiceNumber') as HTMLInputElement).addEventListener('change', (e) => {
    addLog(`🎤 音声番号を変更: ${(e.target as HTMLInputElement).value}`);
  });
  (document.getElementById('playbackSpeed') as HTMLInputElement).addEventListener('change', (e) => {
    addLog(`⏱️ 再生速度を変更: ${(e.target as HTMLInputElement).value}`);
  });
  (document.getElementById('concurrentRequests') as HTMLInputElement).addEventListener('change', (e) => {
    MAX_CONCURRENT_REQUESTS = parseInt((e.target as HTMLInputElement).value) || 3;
    addLog(`📊 並行リクエスト数を変更: ${(e.target as HTMLInputElement).value}`);
  });
});

let model = 'voicevox-v1';
let responseFormat = 'pcm';

function getVoiceNumber(): string {
  return (document.getElementById('voiceNumber') as HTMLInputElement).value || '1';
}

function getPlaybackSpeed(): number {
  return parseFloat((document.getElementById('playbackSpeed') as HTMLInputElement).value) || 1.0;
}

let cnt = 0;

async function processStream(reader: ReadableStreamDefaultReader<Uint8Array>, myCnt: number): Promise<void> {
  try {
    while (true) {
      const { done, value } = await reader.read();
      if (isStop) return;
      if (done) {
        if (leftover.length >= 2) {
          const leftoverUseBytes = leftover.length - (leftover.length % 2);
          if (leftoverUseBytes > 0) {
            const leftoverChunk = leftover.slice(0, leftoverUseBytes);
            const newPending = new Uint8Array(pendingBytes.length + leftoverChunk.length);
            newPending.set(pendingBytes, 0);
            newPending.set(leftoverChunk, pendingBytes.length);
            pendingBytes = newPending;
          }
        }
        leftover = new Uint8Array(0);

        addLog(`データ受信完了: 行${myCnt} (${pendingBytes.length}バイト)`);

        if (pendingBytes.length > 0) {
          pendingBytesQueue.push(pendingBytes);
          statuses[myCnt] = STATUS_PUSHED;
          updateStatusDisplay();
          addLog(`キューに追加: 行${myCnt} (キュー内: ${pendingBytesQueue.length})`);
          schedulePlayback(myCnt);
          highlightActiveLine();
        }
        pendingBytes = new Uint8Array(0);
        return;
      }

      const chunk = value || new Uint8Array(0);
      const combined = new Uint8Array(leftover.length + chunk.length);
      combined.set(leftover, 0);
      combined.set(chunk, leftover.length);
      leftover = new Uint8Array(0);

      const useBytes = combined.length - (combined.length % 2);
      if (useBytes > 0) {
        const validChunk = combined.slice(0, useBytes);
        const newPending = new Uint8Array(pendingBytes.length + validChunk.length);
        newPending.set(pendingBytes, 0);
        newPending.set(validChunk, pendingBytes.length);
        pendingBytes = newPending;
      }

      if (useBytes < combined.length) leftover = combined.slice(useBytes);
    }
  } catch (e) {
    addLog(`ストリーム処理中のエラー: 行${myCnt} - ${(e as Error).message}`);
    console.error(e);
  }
}

const f = (text: string): Promise<void> => {
  return new Promise((resolve) => {
    playTexts.push(text);
    const myCnt = cnt++;
    statuses[myCnt] = STATUS_START;
    updateStatusDisplay();
    addLog(`APIリクエスト送信: 行${myCnt} "${text.substring(0, 30)}..."`);
    activeRequests.add(myCnt);

    fetch('http://10.2.1.15:19999/run2', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ model, input: text, voice: getVoiceNumber(), speed: getPlaybackSpeed(), response_format: responseFormat })
    }).then(async (response) => {
      if (!response.ok) throw new Error(`OpenAI API error: ${response.status}`);
      addLog(`APIレスポンス受信: 行${myCnt}`);
      const reader = response.body!.getReader();

      while (myCnt > 0 && statuses[myCnt - 1] < STATUS_PUSHED) {
        await new Promise(r => setTimeout(r, 10));
      }
      resolve();
      await processStream(reader, myCnt);
    }).catch((error) => {
      addLog(`エラー: 行${myCnt} - ${error.message}`);
      console.error('エラー:', error);
    }).finally(() => {
      activeRequests.delete(myCnt);
    });
  });
};

function updateTextDisplay(): void {
  const textInput = (document.getElementById('textInput') as HTMLTextAreaElement).value.trim();
  const texts = textInput.split('。').filter(t => t.length > 0);
  const textDisplay = document.getElementById('textDisplay') as HTMLElement;
  textDisplay.innerHTML = '';

  texts.forEach((text, index) => {
    const span = document.createElement('span');
    span.id = `text-line-${index}`;
    span.className = 'px-2 py-1 rounded transition-colors';
    span.textContent = text + '。';
    textDisplay.appendChild(span);
    if (index < texts.length - 1) textDisplay.appendChild(document.createTextNode(' '));
  });
}

function highlightActiveLine(): void {
  document.querySelectorAll('#textDisplay span').forEach(span => {
    span.classList.remove('bg-yellow-300', 'font-bold');
  });
  Array.from(activePlayingSources.keys()).forEach(idx => {
    const span = document.getElementById(`text-line-${idx}`) as HTMLElement | null;
    if (span) span.classList.add('bg-yellow-300', 'font-bold');
  });
}

(document.getElementById('textInput') as HTMLTextAreaElement).addEventListener('change', updateTextDisplay);
(document.getElementById('textInput') as HTMLTextAreaElement).addEventListener('input', updateTextDisplay);

(document.getElementById('startButton') as HTMLButtonElement).addEventListener('click', async () => {
  const apiKey = (document.getElementById('apiKeyInput') as HTMLInputElement).value.trim();
  const textInputVal = (document.getElementById('textInput') as HTMLTextAreaElement).value.trim();
  const allTexts = textInputVal.split('。').filter(t => t.length > 0);
  if (allTexts.length === 0) { alert('テキストを入力してください。'); return; }

  const startLineInput = document.getElementById('startLineNumber') as HTMLInputElement;
  const startLineNumber = Math.max(0, parseInt(startLineInput.value) || 0);
  if (startLineNumber >= allTexts.length) {
    alert(`開始行番号が大きすぎます。0～${allTexts.length - 1}の範囲で指定してください。`);
    return;
  }

  const texts = allTexts.slice(startLineNumber);
  addLog(`開始行: ${startLineNumber}行目から（全${texts.length}行を処理）`);

  const concurrentInput = document.getElementById('concurrentRequests') as HTMLInputElement;
  MAX_CONCURRENT_REQUESTS = Math.max(1, Math.min(10, parseInt(concurrentInput.value) || 3));

  cnt = startLineNumber;
  nextPlayIndex = startLineNumber;
  statuses = [];
  playTexts = [];
  pendingBytes = new Uint8Array(0);
  pendingBytesQueue = [];
  leftover = new Uint8Array(0);
  isPlaying = false;
  isStop = false;
  for (const k in actualDurations) delete actualDurations[Number(k)];
  activePlayingSources.clear();
  playbackQueue = [];
  activeRequests.clear();
  concurrentPlayDetected = false;
  updateConcurrentPlayingDisplay();
  (document.getElementById('waitingNotice') as HTMLElement).classList.add('hidden');
  updateTextDisplay();

  addLog(`再生開始: ${texts.length}行のテキストを処理します`);
  addLog(`並行リクエスト数: ${MAX_CONCURRENT_REQUESTS} 行`);
  (document.getElementById('currentText') as HTMLElement).innerText = texts[0].trim();
  updateStatusDisplay();

  try {
    let currentIndex = 0;
    while (currentIndex < texts.length) {
      let availableSlots = MAX_CONCURRENT_REQUESTS - pendingBytesQueue.length;
      let waitLoggedOnce = false;
      while (availableSlots <= 0) {
        if (!waitLoggedOnce) {
          addLog(`⏸️ 再生待ち行数が多いため待機中（待ち：${pendingBytesQueue.length}行、最大：${MAX_CONCURRENT_REQUESTS}行）`);
          waitLoggedOnce = true;
        }
        await new Promise(r => setTimeout(r, 500));
        availableSlots = MAX_CONCURRENT_REQUESTS - pendingBytesQueue.length;
      }

      const batchSize = Math.min(availableSlots, texts.length - currentIndex);
      const batch: Promise<void>[] = [];
      for (let i = 0; i < batchSize; i++) batch.push(f(texts[currentIndex + i].trim()));

      addLog(`バッチ処理: 行${currentIndex}～${currentIndex + batchSize - 1} (${batchSize}行、有効スロット：${availableSlots}/${MAX_CONCURRENT_REQUESTS})`);
      await Promise.all(batch);
      currentIndex += batchSize;
      if (currentIndex < texts.length) await new Promise(r => setTimeout(r, 200));
    }
    addLog('すべてのテキストの処理が完了しました');
  } catch (error: any) {
    addLog(`エラーが発生しました: ${error.message}`);
    console.error('エラー:', error);
    alert('エラーが発生しました。コンソールをご確認ください。');
  }
});

function generateLineURLAndOpen(): void {
  const curLineNum = parseInt((document.getElementById('curIdx') as HTMLElement).textContent || '0') || 0;
  const voiceNum = getVoiceNumber();
  const speed = getPlaybackSpeed();
  const selectedFile = (document.getElementById('templateFileSelect') as HTMLSelectElement).value;
  let url = `http://10.2.1.15:19999/test?`;
  url += `l=${curLineNum}`;
  if (selectedFile) url += `&message=${encodeURIComponent(selectedFile)}`;
  url += `&voiceNumber=${encodeURIComponent(voiceNum)}&playbackSpeed=${encodeURIComponent(String(speed))}`;
  addLog(`🔗 URLを開く: ${url}`);
  window.open(url, '_blank');
}

// Expose some functions used by inline attributes (togglePanel/generateLineURLAndOpen)
(window as any).togglePanel = togglePanel;
(window as any).generateLineURLAndOpen = generateLineURLAndOpen;
