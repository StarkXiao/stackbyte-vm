const $ = (selector) => document.querySelector(selector);
const state = { examples: [], sessionId: null, busy: false };

async function request(path, options = {}) {
  const response = await fetch(path, {
    headers: { "Content-Type": "application/json" },
    ...options,
  });
  const body = response.status === 204 ? null : await response.json();
  if (!response.ok) {
    const error = new Error(body?.message || `HTTP ${response.status}`);
    error.details = body;
    throw error;
  }
  return body;
}

function setBusy(busy, message = "就绪") {
  state.busy = busy;
  ["compileBtn", "runBtn", "stepBtn"].forEach((id) => $(`#${id}`).disabled = busy);
  $("#continueBtn").disabled = busy || !state.sessionId;
  $("#statusText").textContent = message;
  $("#statusDot").style.background = busy ? "#e9b44c" : "#56d68b";
}

function showError(error) {
  const diagnostics = error.details?.diagnostics || [];
  $("#diagnostics").textContent = diagnostics.length
    ? diagnostics.map((item) => `${item.phase} ${item.line}:${item.column} ${item.message}`).join("\n")
    : error.message;
  setBusy(false, "执行失败");
}

function selectTab(name) {
  document.querySelectorAll(".tab").forEach((tab) => tab.classList.toggle("active", tab.dataset.tab === name));
  document.querySelectorAll(".panel").forEach((panel) => panel.classList.remove("active"));
  $(`#${name}Panel`).classList.add("active");
}

function sourceBody() { return JSON.stringify({ source: $("#source").value }); }

async function compile() {
  setBusy(true, "正在编译");
  $("#diagnostics").textContent = "";
  try {
    const data = await request("/api/v1/compile", { method: "POST", body: sourceBody() });
    $("#bytecode").textContent = data.disassembly;
    selectTab("bytecode");
    setBusy(false, `编译完成 · ${data.code_bytes} bytes`);
  } catch (error) { showError(error); }
}

async function run() {
  await resetSession();
  setBusy(true, "正在运行");
  $("#diagnostics").textContent = "";
  try {
    const data = await request("/api/v1/run", { method: "POST", body: sourceBody() });
    $("#output").textContent = data.output || `(返回 ${data.value})`;
    $("#instructions").textContent = data.stats.instructions;
    $("#maxStack").textContent = data.stats.max_stack;
    $("#maxFrames").textContent = data.stats.max_frames;
    $("#duration").textContent = `${data.stats.duration_ms.toFixed(3)} ms`;
    selectTab("output");
    setBusy(false, "运行完成");
  } catch (error) { showError(error); }
}

async function step() {
  setBusy(true, state.sessionId ? "执行单步" : "创建调试会话");
  try {
    let data;
    if (!state.sessionId) {
      data = await request("/api/v1/debug/sessions", { method: "POST", body: sourceBody() });
      state.sessionId = data.id;
    } else {
      data = await request(`/api/v1/debug/sessions/${state.sessionId}/step`, { method: "POST" });
    }
    renderSnapshot(data);
  } catch (error) { showError(error); }
}

async function continueRun() {
  if (!state.sessionId) return;
  setBusy(true, "继续执行");
  try {
    const data = await request(`/api/v1/debug/sessions/${state.sessionId}/continue`, { method: "POST" });
    renderSnapshot(data);
  } catch (error) { showError(error); }
}

function renderSnapshot(data) {
  $("#currentInstruction").textContent = `${String(data.offset).padStart(4, "0")}  ${data.instruction}`;
  $("#currentLine").textContent = `源码行 ${data.line || "—"}`;
  $("#output").textContent = data.output || "等待输出...";
  renderList("#stack", data.stack, (value, i) => `[${i}] ${value}`);
  renderList("#frames", data.frames, (frame) => `${frame.function} · line ${frame.line}`);
  selectTab("state");
  if (data.error) $("#diagnostics").textContent = data.error;
  if (data.done) state.sessionId = null;
  setBusy(false, data.done ? "调试结束" : "调试已暂停");
}

function renderList(selector, values = [], format) {
  const list = $(selector);
  list.replaceChildren(...values.map((value, index) => {
    const item = document.createElement("li");
    item.textContent = format(value, index);
    return item;
  }));
}

async function resetSession() {
  const id = state.sessionId;
  state.sessionId = null;
  if (id) {
    try { await request(`/api/v1/debug/sessions/${id}`, { method: "DELETE" }); } catch (_) { /* expired */ }
  }
  $("#currentInstruction").textContent = "—";
  $("#currentLine").textContent = "源码行 —";
  renderList("#stack", [], String);
  renderList("#frames", [], String);
  setBusy(false, "已重置");
}

async function loadExamples() {
  state.examples = await request("/api/v1/examples");
  const select = $("#exampleSelect");
  state.examples.forEach((example, index) => {
    const option = document.createElement("option");
    option.value = String(index);
    option.textContent = example.name;
    select.append(option);
  });
  loadExample(0);
}

function loadExample(index) {
  $("#source").value = state.examples[index]?.source || "";
  $("#diagnostics").textContent = "";
  resetSession();
}

document.querySelectorAll(".tab").forEach((tab) => tab.addEventListener("click", () => selectTab(tab.dataset.tab)));
$("#compileBtn").addEventListener("click", compile);
$("#runBtn").addEventListener("click", run);
$("#stepBtn").addEventListener("click", step);
$("#continueBtn").addEventListener("click", continueRun);
$("#resetBtn").addEventListener("click", resetSession);
$("#exampleSelect").addEventListener("change", (event) => loadExample(Number(event.target.value)));
loadExamples().catch(showError);
