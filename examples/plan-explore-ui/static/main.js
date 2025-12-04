(() => {
  const promptEl = document.getElementById("prompt");
  const runBtn = document.getElementById("runBtn");
  const statusEl = document.getElementById("status");
  const phasesEl = document.getElementById("phases");

  const defaultPrompt = `请帮我完成一个两阶段的代码分析任务:
1. 规划(Plan): 分析 agentsdk 仓库里 pkg/tools/builtin 目录下各个工具的职责和测试需求, 给出一个分步骤的实施计划。
2. 探索(Explore): 按计划实际阅读相关文件, 重点关注 TodoWrite / ExitPlanMode / Task / subagent_manager 等实现细节。

要求:
- 在规划阶段, 使用 TodoWrite 或 write_todos 工具创建任务列表, 至少包含 "分析 builtin 工具测试需求" 和 "分析当前工具实现状态" 两个任务。
- 在探索阶段, 使用 Read / Glob / Grep 等工具读取具体文件。
- 规划完成后, 调用 ExitPlanMode 返回一个 Markdown 格式的计划。`;

  promptEl.value = defaultPrompt;

  let currentController = null;

  runBtn.addEventListener("click", () => {
    startRun();
  });

  function setStatus(text) {
    statusEl.textContent = text;
  }

  function appendPhaseLine(text, className) {
    const line = document.createElement("div");
    line.className = `phase-line ${className || ""}`;
    line.textContent = text;
    phasesEl.appendChild(line);
    phasesEl.scrollTop = phasesEl.scrollHeight;
    return line;
  }

  function appendToLastLine(text) {
    if (phasesEl.lastElementChild) {
      phasesEl.lastElementChild.textContent += text;
      phasesEl.scrollTop = phasesEl.scrollHeight;
    }
  }

  function clearPhases() {
    phasesEl.textContent = "";
  }

  function startRun() {
    const prompt = promptEl.value.trim();
    if (!prompt) return;

    clearPhases();
    setStatus("运行中...");
    runBtn.disabled = true;

    if (currentController) {
      currentController.abort();
    }
    currentController = new AbortController();

    fetch("/api/chat/stream", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ input: prompt }),
      signal: currentController.signal,
    })
      .then((response) => {
        if (!response.ok) {
          appendPhaseLine(`请求失败: ${response.status}`, "error");
          setStatus("请求失败");
          runBtn.disabled = false;
          return;
        }

        const reader = response.body.getReader();
        const decoder = new TextDecoder("utf-8");
        let buffer = "";
        const uiState = { currentPhase: null, textStarted: false };

        function pump() {
          reader
            .read()
            .then(({ done, value }) => {
              if (done) {
                setStatus("完成");
                runBtn.disabled = false;
                return;
              }
              buffer += decoder.decode(value, { stream: true });
              let idx;
              while ((idx = buffer.indexOf("\n\n")) >= 0) {
                const raw = buffer.slice(0, idx);
                buffer = buffer.slice(idx + 2);
                handleSSEChunk(raw, uiState);
              }
              pump();
            })
            .catch((err) => {
              if (err.name !== "AbortError") {
                appendPhaseLine(`流式读取出错: ${err}`, "error");
                setStatus("出错");
                runBtn.disabled = false;
              }
            });
        }

        pump();
      })
      .catch((err) => {
        if (err.name !== "AbortError") {
          appendPhaseLine(`网络错误: ${err}`, "error");
          setStatus("网络错误");
          runBtn.disabled = false;
        }
      });
  }

  function handleSSEChunk(chunk, uiState) {
    const lines = chunk.split("\n");
    for (const line of lines) {
      if (!line.startsWith("data:")) continue;
      const json = line.slice(5).trim();
      if (!json) continue;
      let evt;
      try {
        evt = JSON.parse(json);
      } catch (e) {
        console.warn("Failed to parse event JSON", e);
        continue;
      }
      renderUIEvent(evt, uiState);
    }
  }

  function renderUIEvent(ev, uiState) {
    if (!ev || !ev.channel || !ev.type) return;

    if (ev.channel === "progress") {
      switch (ev.type) {
        case "text_chunk_start":
          uiState.currentTextLine = appendPhaseLine("", "tool");
          uiState.textStarted = true;
          break;
        case "text_chunk":
          if (ev.payload && ev.payload.delta) {
            // 如果还没有文本行，创建一个（兼容没有 text_chunk_start 的情况）
            if (!uiState.textStarted) {
              uiState.currentTextLine = appendPhaseLine("", "tool");
              uiState.textStarted = true;
            }
            appendToLastLine(ev.payload.delta);
          }
          break;
        case "tool:start":
          if (ev.payload && ev.payload.call) {
            uiState.textStarted = false; // 重置文本状态
            renderToolStartWeb(uiState, ev.payload.call);
          }
          break;
        case "tool:error":
          if (ev.payload && ev.payload.call) {
            uiState.textStarted = false; // 重置文本状态
            appendPhaseLine(`[Tool Error] ${ev.payload.call.name}: ${ev.payload.error || ""}`, "error");
          }
          break;
        case "done":
          uiState.textStarted = false; // 重置文本状态
          appendPhaseLine(`[Done] ${ev.payload && ev.payload.reason ? ev.payload.reason : ""}`, "tool");
          break;
      }
    } else if (ev.channel === "monitor") {
      if (ev.type === "state_changed" && ev.payload) {
        appendPhaseLine(`[State] ${ev.payload.state}`, "tool");
      }
    }
  }

  function renderToolStartWeb(uiState, call) {
    if (!call || !call.name) return;
    const name = call.name;
    const args = call.arguments || {};

    if (name === "TodoWrite" || name === "write_todos") {
      const title = extractActiveTaskTitleWeb(args) || "任务规划";
      appendPhaseLine(`Plan(${title})`, "plan");
      uiState.currentPhase = { kind: "Plan", title: title, count: 0 };
      return;
    }

    if (name === "Task" || name === "task") {
      const subType = getStringArgWeb(args, "subagent_type");
      const prompt = getStringArgWeb(args, "prompt") || "子代理任务";
      const labelType = subType || "Task";
      const className = labelType === "Explore" ? "explore" : labelType === "Plan" ? "plan" : "tool";
      appendPhaseLine(`${labelType}(${prompt})`, className);
      uiState.currentPhase = { kind: labelType, title: prompt, count: 0 };
      return;
    }

    const label = formatToolCallLabelWeb(call);
    if (!label) return;

    if (uiState.currentPhase) {
      if (uiState.currentPhase.count === 0) {
        appendPhaseLine(`  └─ ${label}`, "tool");
      } else {
        appendPhaseLine(`     ${label}`, "tool");
      }
      uiState.currentPhase.count += 1;
    } else {
      appendPhaseLine(`[Tool] ${label}`, "tool");
    }
  }

  function extractActiveTaskTitleWeb(args) {
    const todos = args.todos;
    if (!Array.isArray(todos)) return "";

    for (const item of todos) {
      if (!item || typeof item !== "object") continue;
      const status = getStringArgWeb(item, "status");
      if (status === "in_progress") {
        const activeForm = getStringArgWeb(item, "activeForm");
        if (activeForm) return activeForm;
        const content = getStringArgWeb(item, "content");
        if (content) return content;
      }
    }
    return "";
  }

  function getStringArgWeb(obj, key) {
    if (!obj || typeof obj !== "object") return "";
    const v = obj[key];
    return typeof v === "string" ? v : "";
  }

  function formatToolCallLabelWeb(call) {
    const name = call.name;
    const args = call.arguments || {};

    switch (name) {
      case "Read": {
        const path = getStringArgWeb(args, "file_path");
        return path ? `Read(${path})` : "Read";
      }
      case "Glob": {
        const pattern = getStringArgWeb(args, "pattern");
        const root = getStringArgWeb(args, "root");
        if (pattern && root) return `Glob(${pattern}, root=${root})`;
        if (pattern) return `Glob(${pattern})`;
        return "Glob";
      }
      case "Grep": {
        const pattern = getStringArgWeb(args, "pattern");
        const path = getStringArgWeb(args, "path");
        if (pattern && path) return `Grep("${pattern}" in ${path})`;
        if (pattern) return `Grep("${pattern}")`;
        return "Grep";
      }
      case "Bash": {
        let cmd = getStringArgWeb(args, "command");
        if (!cmd) return "Bash";
        if (cmd.length > 40) cmd = cmd.slice(0, 37) + "...";
        return `Bash(${cmd})`;
      }
      case "ExitPlanMode":
        return "ExitPlanMode(plan)";
      default:
        return name || "";
    }
  }
})();
