// openai_tts_realtime_t.ts
function togglePanel(contentId, iconId) {
  const content = document.getElementById(contentId);
  const icon = document.getElementById(iconId);
  if (content.style.display === "none") {
    content.style.display = "";
    icon.textContent = "\u25BC";
    icon.style.transform = "rotate(0deg)";
  } else {
    content.style.display = "none";
    icon.textContent = "\u25B6";
    icon.style.transform = "rotate(-90deg)";
  }
}
function addLog(message) {
  const logContainer = document.getElementById("logContainer");
  const timestamp = (/* @__PURE__ */ new Date()).toLocaleTimeString("ja-JP");
  const logEntry = document.createElement("div");
  logEntry.textContent = `[${timestamp}] ${message}`;
  logContainer.appendChild(logEntry);
  logContainer.scrollTop = logContainer.scrollHeight;
}
var audioContext = new (window.AudioContext || window.webkitAudioContext)({
  sampleRate: 24e3
});
var pendingBytes = new Uint8Array(0);
var pendingBytesQueue = [];
var leftover = new Uint8Array(0);
var isPlaying = false;
var sourceNode = null;
var idxForceStop = -1;
var nextPlayIndex = 0;
var activePlayingSources = /* @__PURE__ */ new Map();
var concurrentPlayDetected = false;
var playbackQueue = [];
var MAX_CONCURRENT_REQUESTS = 3;
var activeRequests = /* @__PURE__ */ new Set();
async function waitForPreviousPlayback(myCnt) {
  const waitingNotice = document.getElementById("waitingNotice");
  const waitingInfo = document.getElementById("waitingInfo");
  let hasLogged = false;
  while (activePlayingSources.size > 0) {
    const otherPlaying = Array.from(activePlayingSources.keys()).filter((idx) => idx !== myCnt);
    if (otherPlaying.length === 0) break;
    waitingNotice.classList.remove("hidden");
    waitingInfo.textContent = `\u884C${myCnt}\u304C\u5F85\u6A5F\u4E2D (\u518D\u751F\u4E2D: [${otherPlaying.join(", ")}])`;
    if (!hasLogged) {
      addLog(`\u884C${myCnt}: \u4ED6\u306E\u97F3\u58F0\u304C\u518D\u751F\u4E2D\u306E\u305F\u3081\u5F85\u6A5F... (\u518D\u751F\u4E2D: [${otherPlaying.join(", ")}])`);
      hasLogged = true;
    }
    await new Promise((resolve) => setTimeout(resolve, 100));
  }
  waitingNotice.classList.add("hidden");
}
var analyserNode = audioContext.createAnalyser();
analyserNode.fftSize = 1024;
var dataArray = new Uint8Array(analyserNode.frequencyBinCount);
var waveCanvas = document.getElementById("waveCanvas");
var waveCtx = waveCanvas.getContext("2d");
var playTimeout = 15 * 1e3;
var intervalPerLine = 1;
var actualDurations = {};
function drawWaveform() {
  requestAnimationFrame(drawWaveform);
  waveCtx.fillStyle = "#667eea";
  waveCtx.fillRect(0, 0, waveCanvas.width, waveCanvas.height);
  analyserNode.getByteTimeDomainData(dataArray);
  waveCtx.lineWidth = 2;
  waveCtx.strokeStyle = "#fff";
  waveCtx.beginPath();
  const sliceWidth = waveCanvas.width * 1 / dataArray.length;
  let x = 0;
  for (let i = 0; i < dataArray.length; i++) {
    const v = dataArray[i] / 128;
    const y = v * waveCanvas.height / 2;
    if (i === 0) waveCtx.moveTo(x, y);
    else waveCtx.lineTo(x, y);
    x += sliceWidth;
  }
  waveCtx.lineTo(waveCanvas.width, waveCanvas.height / 2);
  waveCtx.stroke();
}
var playTexts = [];
var playStart = performance.now();
var statuses = [];
var playStarts = [];
var STATUS_START = 0;
var STATUS_PUSHED = 1;
var STATUS_END = 2;
var curPlaying = 0;
function updateStatusDisplay() {
  const container = document.getElementById("statusContainer");
  container.innerHTML = "";
  const activeItems = [];
  for (let i = 0; i < statuses.length; i++) {
    if (statuses[i] !== STATUS_END && !activePlayingSources.has(i) && i > curPlaying) {
      activeItems.push(i);
    }
  }
  const displayItems = activeItems.slice(Math.max(0, activeItems.length - 5));
  for (const i of displayItems) {
    const statusText = ["\u6E96\u5099\u4E2D", "\u518D\u751F\u5F85\u3061", "\u5B8C\u4E86"][statuses[i]] || "\u4E0D\u660E";
    const statusClass = ["status-start", "status-pushed", "status-end"][statuses[i]] || "status-start";
    const div = document.createElement("div");
    div.className = `text-xs truncate`;
    div.innerHTML = `<span class="status-badge ${statusClass}">${statusText}</span> \u884C${i}`;
    container.appendChild(div);
  }
  updateBufferStatus();
}
function updateBufferStatus() {
  const requesting = document.getElementById("requestingCount");
  const queued = document.getElementById("queuedCount");
  requesting.textContent = String(activeRequests.size);
  queued.textContent = String(pendingBytesQueue.length);
}
function updateConcurrentPlayingDisplay() {
  const playingList = document.getElementById("playingList");
  const warningDiv = document.getElementById("concurrentWarning");
  const countSpan = document.getElementById("concurrentCount");
  if (activePlayingSources.size === 0) {
    playingList.innerHTML = '<p class="text-gray-500">\u306A\u3057</p>';
    warningDiv.classList.add("hidden");
    concurrentPlayDetected = false;
  } else {
    playingList.innerHTML = "";
    const entries = Array.from(activePlayingSources.entries());
    entries.forEach(([idx, info]) => {
      const div = document.createElement("div");
      const elapsed = (performance.now() - info.startTime) / 1e3;
      const isConcurrent = activePlayingSources.size > 1;
      div.className = `p-2 rounded ${isConcurrent ? "bg-red-100 border border-red-300" : "bg-blue-100"}`;
      div.innerHTML = `
        <div class="font-bold ${isConcurrent ? "text-red-700" : "text-blue-700"}">
          ${isConcurrent ? "\u26A0\uFE0F " : "\u25B6\uFE0F "}\u884C${idx}
        </div>
        <div class="text-xs text-gray-600 truncate">${info.text.substring(0, 30)}...</div>
        <div class="text-xs text-gray-500">\u7D4C\u904E: ${elapsed.toFixed(1)}\u79D2</div>
      `;
      playingList.appendChild(div);
    });
    if (activePlayingSources.size > 1) {
      warningDiv.classList.remove("hidden");
      countSpan.textContent = `${activePlayingSources.size}\u500B\u306E\u97F3\u58F0\u304C\u540C\u6642\u518D\u751F\u4E2D`;
      if (!concurrentPlayDetected) {
        concurrentPlayDetected = true;
        addLog(`\u26A0\uFE0F \u591A\u91CD\u518D\u751F\u691C\u77E5: ${activePlayingSources.size}\u500B\u306E\u97F3\u58F0\u304C\u540C\u6642\u518D\u751F\u3055\u308C\u3066\u3044\u307E\u3059`);
      }
    } else {
      warningDiv.classList.add("hidden");
      concurrentPlayDetected = false;
    }
  }
}
function updateTimerDisplay() {
  const elapsedElem = document.getElementById("elapsedTime");
  const remainingElem = document.getElementById("remainingTime");
  if (curPlaying < 0 || !playStarts[curPlaying]) {
    elapsedElem.textContent = "0.0";
    remainingElem.textContent = "0.0";
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
function EndStatuses(cur) {
  if (cur - 5 < 0) return;
  for (let i = Math.max(0, cur - 5); i <= cur; i++) {
    if (statuses[i] !== STATUS_END) statuses[i] = STATUS_END;
  }
  updateStatusDisplay();
}
function getRemainingTime(myCnt) {
  if (actualDurations[myCnt] !== void 0) {
    return actualDurations[myCnt] * 1e3 + 500;
  }
  if (sourceNode == null) return playTimeout;
  const totalDuration = sourceNode.buffer.duration + intervalPerLine;
  return Math.max(playTimeout, totalDuration * 1e3);
}
async function schedulePlayback(myCnt) {
  if (myCnt !== nextPlayIndex) {
    playbackQueue.push(myCnt);
    addLog(`\u884C${myCnt}: \u30AD\u30E5\u30FC\u306B\u767B\u9332\uFF08\u6B21\u306F${nextPlayIndex}\u3092\u5F85\u6A5F\uFF09`);
    return;
  }
  if (isPlaying && performance.now() - playStart > getRemainingTime(myCnt)) {
    isPlaying = false;
    EndStatuses(curPlaying);
    idxForceStop = curPlaying;
    addLog(`\u5F37\u5236\u505C\u6B62: ${curPlaying} (\u30BF\u30A4\u30E0\u30A2\u30A6\u30C8)`);
    console.log("force stop: ", curPlaying);
    return;
  }
  if (activePlayingSources.size > 0) {
    const waitingFor = Array.from(activePlayingSources.keys());
    addLog(`\u884C${myCnt}: \u5F85\u6A5F\u958B\u59CB (\u518D\u751F\u4E2D: [${waitingFor.join(", ")}])`);
    await waitForPreviousPlayback(myCnt);
    addLog(`\u884C${myCnt}: \u5F85\u6A5F\u7D42\u4E86\u3001\u518D\u751F\u3092\u958B\u59CB\u3057\u307E\u3059`);
  }
  if (isPlaying) {
    addLog(`\u884C${myCnt}: \u65E2\u306B\u4ED6\u306E\u97F3\u58F0\u304C\u518D\u751F\u4E2D\u306E\u305F\u3081\u30B9\u30AD\u30C3\u30D7`);
    return;
  }
  if (pendingBytesQueue.length === 0) {
    addLog(`\u884C${myCnt}: \u30AD\u30E5\u30FC\u304C\u7A7A\u306E\u305F\u3081\u518D\u751F\u3067\u304D\u307E\u305B\u3093`);
    return;
  }
  if (playTexts.length === 0) {
    addLog(`\u884C${myCnt}: \u30C6\u30AD\u30B9\u30C8\u304C\u7A7A\u306E\u305F\u3081\u518D\u751F\u3067\u304D\u307E\u305B\u3093`);
    return;
  }
  curPlaying = myCnt;
  isPlaying = true;
  nextPlayIndex++;
  playStart = performance.now();
  const ctext = playTexts.shift();
  document.getElementById("prevText").innerText = document.getElementById("currentText").innerText;
  document.getElementById("currentText").innerText = ctext;
  document.getElementById("curIdx").innerText = String(curPlaying);
  addLog(`\u518D\u751F\u958B\u59CB: \u884C${curPlaying} "${ctext.substring(0, 20)}..."`);
  const dataToPlay = pendingBytesQueue.shift();
  const int16Data = new Int16Array(dataToPlay.buffer);
  const float32Data = new Float32Array(int16Data.length);
  for (let i = 0; i < int16Data.length; i++) float32Data[i] = int16Data[i] / 32768;
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
  addLog(`\u97F3\u58F0\u30C7\u30FC\u30BF\u9577: \u884C${myCnt} ${actualDuration.toFixed(2)}\u79D2`);
  sourceNode = audioContext.createBufferSource();
  sourceNode.buffer = audioBuffer;
  activePlayingSources.set(myCnt, {
    startTime: performance.now(),
    text: ctext,
    sourceNode
  });
  if (activePlayingSources.size > 1) {
    const activeIndices = Array.from(activePlayingSources.keys()).join(", ");
    addLog(`\u26A0\uFE0F \u8B66\u544A: \u591A\u91CD\u518D\u751F\u691C\u77E5\uFF01 \u518D\u751F\u4E2D\u306E\u884C: [${activeIndices}]`);
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
    addLog(`\u518D\u751F\u5B8C\u4E86: \u884C${curPlaying}`);
    idxForceStop = -1;
    if (playbackQueue.length > 0) {
      const nextIdxPosition = playbackQueue.indexOf(nextPlayIndex);
      if (nextIdxPosition >= 0) {
        playbackQueue.splice(nextIdxPosition, 1);
        addLog(`\u30AD\u30E5\u30FC\u304B\u3089\u53D6\u308A\u51FA\u3057: \u884C${nextPlayIndex}\u3092${intervalPerLine}\u79D2\u5F8C\u306B\u518D\u751F\u958B\u59CB`);
        setTimeout(() => {
          schedulePlayback(nextPlayIndex);
        }, intervalPerLine * 1e3);
      } else {
        addLog(`\u8B66\u544A: \u30AD\u30E5\u30FC\u306B\u6B21\u306E\u884C${nextPlayIndex}\u304C\u898B\u3064\u304B\u308A\u307E\u305B\u3093\uFF08\u30AD\u30E5\u30FC: [${playbackQueue.join(", ")}]\uFF09`);
      }
    }
  };
  sourceNode.start();
  playStarts[curPlaying] = audioContext.currentTime;
}
drawWaveform();
var isPause = false;
var isStop = false;
document.getElementById("pause").addEventListener("click", () => {
  if (isPause) {
    audioContext.resume().then(() => {
      isPause = false;
      document.getElementById("pause").innerText = "\u23F8\uFE0F \u4E00\u6642\u505C\u6B62";
      addLog("\u518D\u958B\u3057\u307E\u3057\u305F");
    });
  } else {
    audioContext.suspend().then(() => {
      isPause = true;
      document.getElementById("pause").innerText = "\u25B6\uFE0F \u518D\u958B";
      addLog("\u4E00\u6642\u505C\u6B62\u3057\u307E\u3057\u305F");
    });
  }
});
document.getElementById("stop").addEventListener("click", () => {
  isStop = true;
  activePlayingSources.forEach((info, idx) => {
    if (info.sourceNode) {
      try {
        info.sourceNode.stop();
        info.sourceNode.disconnect();
      } catch (e) {
        console.error(`\u884C${idx}\u306E\u505C\u6B62\u30A8\u30E9\u30FC:`, e);
      }
    }
  });
  if (sourceNode) {
    try {
      sourceNode.stop();
    } catch {
    }
    try {
      sourceNode.disconnect();
    } catch {
    }
    isPlaying = false;
    pendingBytes = new Uint8Array(0);
    leftover = new Uint8Array(0);
  }
  activePlayingSources.clear();
  updateConcurrentPlayingDisplay();
  addLog("\u518D\u751F\u3092\u505C\u6B62\u3057\u307E\u3057\u305F");
});
document.getElementById("clearLog").addEventListener("click", () => {
  document.getElementById("logContainer").innerHTML = "";
});
async function loadTemplateFiles() {
  try {
    const response = await fetch("http://10.2.1.15:19999/listfiles");
    const data = await response.json();
    const select = document.getElementById("templateFileSelect");
    select.innerHTML = '<option value="">-- \u30D5\u30A1\u30A4\u30EB\u3092\u9078\u629E --</option>';
    data.files.forEach((file) => {
      if (!file.endsWith(".html")) {
        const option = document.createElement("option");
        option.value = file;
        option.textContent = file;
        select.appendChild(option);
      }
    });
  } catch (error) {
    console.error("\u30C6\u30F3\u30D7\u30EC\u30FC\u30C8\u30D5\u30A1\u30A4\u30EB\u4E00\u89A7\u306E\u8AAD\u307F\u8FBC\u307F\u30A8\u30E9\u30FC:", error);
  }
}
async function loadTemplateFileContent(selectedFile) {
  if (!selectedFile) return;
  try {
    const response = await fetch(`http://10.2.1.15:19999/test?message=${encodeURIComponent(selectedFile)}`);
    const html = await response.text();
    const parser = new DOMParser();
    const doc = parser.parseFromString(html, "text/html");
    const textarea = doc.querySelector("#textInput");
    if (textarea) {
      document.getElementById("textInput").value = textarea.textContent || "";
      addLog(`\u30C6\u30F3\u30D7\u30EC\u30FC\u30C8\u8AAD\u307F\u8FBC\u307F: ${selectedFile}`);
      updateTextDisplay();
    }
  } catch (error) {
    console.error("\u30C6\u30F3\u30D7\u30EC\u30FC\u30C8\u8AAD\u307F\u8FBC\u307F\u30A8\u30E9\u30FC:", error);
    addLog(`\u30A8\u30E9\u30FC: \u30C6\u30F3\u30D7\u30EC\u30FC\u30C8\u300C${selectedFile}\u300D\u3092\u8AAD\u307F\u8FBC\u3081\u307E\u305B\u3093`);
  }
}
document.getElementById("templateFileSelect").addEventListener("change", async (e) => {
  const selectedFile = e.target.value;
  if (!selectedFile) return;
  await loadTemplateFileContent(selectedFile);
});
window.addEventListener("load", async () => {
  updateTextDisplay();
  const urlParams = new URLSearchParams(window.location.search);
  const startLine = urlParams.get("l");
  if (startLine !== null) {
    document.getElementById("startLineNumber").value = startLine;
    addLog(`\u{1F4CD} \u958B\u59CB\u884C\u756A\u53F7\u3092\u8A2D\u5B9A: ${startLine}`);
  }
  const voiceNum = urlParams.get("voiceNumber");
  if (voiceNum !== null) {
    document.getElementById("voiceNumber").value = voiceNum;
    addLog(`\u{1F3A4} \u97F3\u58F0\u756A\u53F7\u3092\u8A2D\u5B9A: ${voiceNum}`);
  }
  const speed = urlParams.get("playbackSpeed");
  if (speed !== null) {
    document.getElementById("playbackSpeed").value = speed;
    addLog(`\u23F1\uFE0F \u518D\u751F\u901F\u5EA6\u3092\u8A2D\u5B9A: ${speed}`);
  }
  await loadTemplateFiles();
  const message = urlParams.get("message");
  if (message !== null) {
    document.getElementById("templateFileSelect").value = message;
    addLog(`\u{1F4C4} \u30C6\u30F3\u30D7\u30EC\u30FC\u30C8\u30D5\u30A1\u30A4\u30EB\u3092\u8A2D\u5B9A: ${message}`);
    await loadTemplateFileContent(message);
  }
  document.getElementById("voiceNumber").addEventListener("change", (e) => {
    addLog(`\u{1F3A4} \u97F3\u58F0\u756A\u53F7\u3092\u5909\u66F4: ${e.target.value}`);
  });
  document.getElementById("playbackSpeed").addEventListener("change", (e) => {
    addLog(`\u23F1\uFE0F \u518D\u751F\u901F\u5EA6\u3092\u5909\u66F4: ${e.target.value}`);
  });
  document.getElementById("concurrentRequests").addEventListener("change", (e) => {
    MAX_CONCURRENT_REQUESTS = parseInt(e.target.value) || 3;
    addLog(`\u{1F4CA} \u4E26\u884C\u30EA\u30AF\u30A8\u30B9\u30C8\u6570\u3092\u5909\u66F4: ${e.target.value}`);
  });
});
var model = "voicevox-v1";
var responseFormat = "pcm";
function getVoiceNumber() {
  return document.getElementById("voiceNumber").value || "1";
}
function getPlaybackSpeed() {
  return parseFloat(document.getElementById("playbackSpeed").value) || 1;
}
var cnt = 0;
async function processStream(reader, myCnt) {
  try {
    while (true) {
      const { done, value } = await reader.read();
      if (isStop) return;
      if (done) {
        if (leftover.length >= 2) {
          const leftoverUseBytes = leftover.length - leftover.length % 2;
          if (leftoverUseBytes > 0) {
            const leftoverChunk = leftover.slice(0, leftoverUseBytes);
            const newPending = new Uint8Array(pendingBytes.length + leftoverChunk.length);
            newPending.set(pendingBytes, 0);
            newPending.set(leftoverChunk, pendingBytes.length);
            pendingBytes = newPending;
          }
        }
        leftover = new Uint8Array(0);
        addLog(`\u30C7\u30FC\u30BF\u53D7\u4FE1\u5B8C\u4E86: \u884C${myCnt} (${pendingBytes.length}\u30D0\u30A4\u30C8)`);
        if (pendingBytes.length > 0) {
          pendingBytesQueue.push(pendingBytes);
          statuses[myCnt] = STATUS_PUSHED;
          updateStatusDisplay();
          addLog(`\u30AD\u30E5\u30FC\u306B\u8FFD\u52A0: \u884C${myCnt} (\u30AD\u30E5\u30FC\u5185: ${pendingBytesQueue.length})`);
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
      const useBytes = combined.length - combined.length % 2;
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
    addLog(`\u30B9\u30C8\u30EA\u30FC\u30E0\u51E6\u7406\u4E2D\u306E\u30A8\u30E9\u30FC: \u884C${myCnt} - ${e.message}`);
    console.error(e);
  }
}
var f = (text) => {
  return new Promise((resolve) => {
    playTexts.push(text);
    const myCnt = cnt++;
    statuses[myCnt] = STATUS_START;
    updateStatusDisplay();
    addLog(`API\u30EA\u30AF\u30A8\u30B9\u30C8\u9001\u4FE1: \u884C${myCnt} "${text.substring(0, 30)}..."`);
    activeRequests.add(myCnt);
    fetch("http://10.2.1.15:19999/run2", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ model, input: text, voice: getVoiceNumber(), speed: getPlaybackSpeed(), response_format: responseFormat })
    }).then(async (response) => {
      if (!response.ok) throw new Error(`OpenAI API error: ${response.status}`);
      addLog(`API\u30EC\u30B9\u30DD\u30F3\u30B9\u53D7\u4FE1: \u884C${myCnt}`);
      const reader = response.body.getReader();
      while (myCnt > 0 && statuses[myCnt - 1] < STATUS_PUSHED) {
        await new Promise((r) => setTimeout(r, 10));
      }
      resolve();
      await processStream(reader, myCnt);
    }).catch((error) => {
      addLog(`\u30A8\u30E9\u30FC: \u884C${myCnt} - ${error.message}`);
      console.error("\u30A8\u30E9\u30FC:", error);
    }).finally(() => {
      activeRequests.delete(myCnt);
    });
  });
};
function updateTextDisplay() {
  const textInput = document.getElementById("textInput").value.trim();
  const texts = textInput.split("\u3002").filter((t) => t.length > 0);
  const textDisplay = document.getElementById("textDisplay");
  textDisplay.innerHTML = "";
  texts.forEach((text, index) => {
    const span = document.createElement("span");
    span.id = `text-line-${index}`;
    span.className = "px-2 py-1 rounded transition-colors";
    span.textContent = text + "\u3002";
    textDisplay.appendChild(span);
    if (index < texts.length - 1) textDisplay.appendChild(document.createTextNode(" "));
  });
}
function highlightActiveLine() {
  document.querySelectorAll("#textDisplay span").forEach((span) => {
    span.classList.remove("bg-yellow-300", "font-bold");
  });
  Array.from(activePlayingSources.keys()).forEach((idx) => {
    const span = document.getElementById(`text-line-${idx}`);
    if (span) span.classList.add("bg-yellow-300", "font-bold");
  });
}
document.getElementById("textInput").addEventListener("change", updateTextDisplay);
document.getElementById("textInput").addEventListener("input", updateTextDisplay);
document.getElementById("startButton").addEventListener("click", async () => {
  const apiKey = document.getElementById("apiKeyInput").value.trim();
  const textInputVal = document.getElementById("textInput").value.trim();
  const allTexts = textInputVal.split("\u3002").filter((t) => t.length > 0);
  if (allTexts.length === 0) {
    alert("\u30C6\u30AD\u30B9\u30C8\u3092\u5165\u529B\u3057\u3066\u304F\u3060\u3055\u3044\u3002");
    return;
  }
  const startLineInput = document.getElementById("startLineNumber");
  const startLineNumber = Math.max(0, parseInt(startLineInput.value) || 0);
  if (startLineNumber >= allTexts.length) {
    alert(`\u958B\u59CB\u884C\u756A\u53F7\u304C\u5927\u304D\u3059\u304E\u307E\u3059\u30020\uFF5E${allTexts.length - 1}\u306E\u7BC4\u56F2\u3067\u6307\u5B9A\u3057\u3066\u304F\u3060\u3055\u3044\u3002`);
    return;
  }
  const texts = allTexts.slice(startLineNumber);
  addLog(`\u958B\u59CB\u884C: ${startLineNumber}\u884C\u76EE\u304B\u3089\uFF08\u5168${texts.length}\u884C\u3092\u51E6\u7406\uFF09`);
  const concurrentInput = document.getElementById("concurrentRequests");
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
  document.getElementById("waitingNotice").classList.add("hidden");
  updateTextDisplay();
  addLog(`\u518D\u751F\u958B\u59CB: ${texts.length}\u884C\u306E\u30C6\u30AD\u30B9\u30C8\u3092\u51E6\u7406\u3057\u307E\u3059`);
  addLog(`\u4E26\u884C\u30EA\u30AF\u30A8\u30B9\u30C8\u6570: ${MAX_CONCURRENT_REQUESTS} \u884C`);
  document.getElementById("currentText").innerText = texts[0].trim();
  updateStatusDisplay();
  try {
    let currentIndex = 0;
    while (currentIndex < texts.length) {
      let availableSlots = MAX_CONCURRENT_REQUESTS - pendingBytesQueue.length;
      let waitLoggedOnce = false;
      while (availableSlots <= 0) {
        if (!waitLoggedOnce) {
          addLog(`\u23F8\uFE0F \u518D\u751F\u5F85\u3061\u884C\u6570\u304C\u591A\u3044\u305F\u3081\u5F85\u6A5F\u4E2D\uFF08\u5F85\u3061\uFF1A${pendingBytesQueue.length}\u884C\u3001\u6700\u5927\uFF1A${MAX_CONCURRENT_REQUESTS}\u884C\uFF09`);
          waitLoggedOnce = true;
        }
        await new Promise((r) => setTimeout(r, 500));
        availableSlots = MAX_CONCURRENT_REQUESTS - pendingBytesQueue.length;
      }
      const batchSize = Math.min(availableSlots, texts.length - currentIndex);
      const batch = [];
      for (let i = 0; i < batchSize; i++) batch.push(f(texts[currentIndex + i].trim()));
      addLog(`\u30D0\u30C3\u30C1\u51E6\u7406: \u884C${currentIndex}\uFF5E${currentIndex + batchSize - 1} (${batchSize}\u884C\u3001\u6709\u52B9\u30B9\u30ED\u30C3\u30C8\uFF1A${availableSlots}/${MAX_CONCURRENT_REQUESTS})`);
      await Promise.all(batch);
      currentIndex += batchSize;
      if (currentIndex < texts.length) await new Promise((r) => setTimeout(r, 200));
    }
    addLog("\u3059\u3079\u3066\u306E\u30C6\u30AD\u30B9\u30C8\u306E\u51E6\u7406\u304C\u5B8C\u4E86\u3057\u307E\u3057\u305F");
  } catch (error) {
    addLog(`\u30A8\u30E9\u30FC\u304C\u767A\u751F\u3057\u307E\u3057\u305F: ${error.message}`);
    console.error("\u30A8\u30E9\u30FC:", error);
    alert("\u30A8\u30E9\u30FC\u304C\u767A\u751F\u3057\u307E\u3057\u305F\u3002\u30B3\u30F3\u30BD\u30FC\u30EB\u3092\u3054\u78BA\u8A8D\u304F\u3060\u3055\u3044\u3002");
  }
});
function generateLineURLAndOpen() {
  const curLineNum = parseInt(document.getElementById("curIdx").textContent || "0") || 0;
  const voiceNum = getVoiceNumber();
  const speed = getPlaybackSpeed();
  const selectedFile = document.getElementById("templateFileSelect").value;
  let url = `http://10.2.1.15:19999/test?`;
  url += `l=${curLineNum}`;
  if (selectedFile) url += `&message=${encodeURIComponent(selectedFile)}`;
  url += `&voiceNumber=${encodeURIComponent(voiceNum)}&playbackSpeed=${encodeURIComponent(String(speed))}`;
  addLog(`\u{1F517} URL\u3092\u958B\u304F: ${url}`);
  window.open(url, "_blank");
}
window.togglePanel = togglePanel;
window.generateLineURLAndOpen = generateLineURLAndOpen;
//# sourceMappingURL=openai_tts_realtime_t.js.map
