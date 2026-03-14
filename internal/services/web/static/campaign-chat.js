(function () {
  var root = document.querySelector("[data-campaign-id]");
  if (!root) {
    return;
  }

  var bootstrapEl = document.getElementById("campaign-game-bootstrap");
  if (!bootstrapEl) {
    return;
  }

  var bootstrap = {};
  try {
    bootstrap = JSON.parse(bootstrapEl.textContent || "{}");
  } catch (err) {
    bootstrap = {};
  }

  var campaignID = String(root.getAttribute("data-campaign-id") || "").trim();
  var statusEl = document.getElementById("chat-status");
  var transcriptEl = document.getElementById("game-transcript");
  var formEl = document.getElementById("game-message-form");
  var inputEl = document.getElementById("game-message-input");
  var sendBtn = document.getElementById("game-send");
  var personaSelectEl = document.getElementById("game-persona-select");
  var controlNoteEl = document.getElementById("game-control-note");
  var openGateTypeEl = document.getElementById("game-open-gate-type");
  var requestHandoffBtn = document.getElementById("game-request-handoff");
  var openGateBtn = document.getElementById("game-open-gate");
  var resolveGateBtn = document.getElementById("game-resolve-gate");
  var abandonGateBtn = document.getElementById("game-abandon-gate");
  var gateSummaryEl = document.getElementById("game-gate-summary");
  var spotlightSummaryEl = document.getElementById("game-spotlight-summary");
  var currentStreamEl = document.getElementById("game-current-stream");
  var streamButtons = Array.prototype.slice.call(document.querySelectorAll("[data-game-stream-button]"));
  var socket = null;
  var lastSequenceID = 0;
  var wsHostCandidates = [];
  var wsHostIndex = 0;
  var state = {
    participant: bootstrap.participant || {},
    sessionID: String(bootstrap.sessionId || "").trim(),
    sessionName: String(bootstrap.sessionName || "").trim(),
    defaultStreamID: String(bootstrap.defaultStreamId || "").trim(),
    defaultPersonaID: String(bootstrap.defaultPersonaId || "").trim(),
    currentStreamID: String(bootstrap.defaultStreamId || "").trim(),
    currentPersonaID: String(bootstrap.defaultPersonaId || "").trim(),
    activeSessionGate: bootstrap.activeSessionGate || null,
    activeSessionSpotlight: bootstrap.activeSessionSpotlight || null,
    streams: Array.isArray(bootstrap.streams) ? bootstrap.streams.slice() : [],
    personas: Array.isArray(bootstrap.personas) ? bootstrap.personas.slice() : [],
    messages: []
  };
  var chatText = {
    invalidId: root.dataset.chatInvalidId || "",
    connecting: root.dataset.chatConnecting || "",
    unavailable: root.dataset.chatChatUnavailable || "",
    wsSetupFailed: root.dataset.chatWsSetupFailed || "",
    connected: root.dataset.chatConnected || "",
    invalidFrame: root.dataset.chatInvalidFrame || "",
    joined: root.dataset.chatJoined || "",
    participant: root.dataset.chatParticipant || "",
    requestFailed: root.dataset.chatRequestFailed || "",
    errorPrefix: root.dataset.chatErrorPrefix || "",
    unableToSend: root.dataset.chatUnableToSend || "",
    disconnectedRetrying: root.dataset.chatDisconnectedRetrying || ""
  };
  var fallbackPort = String(root.dataset.chatFallbackPort || "").trim();

  function randomID(prefix) {
    if (window.crypto && typeof window.crypto.randomUUID === "function") {
      return prefix + window.crypto.randomUUID();
    }
    return prefix + Date.now().toString(36) + Math.random().toString(36).slice(2);
  }

  function isLocalHost(hostname) {
    var normalized = String(hostname || "").trim().toLowerCase();
    if (!normalized) {
      return false;
    }
    return normalized === "localhost" || normalized === "127.0.0.1" || normalized === "::1" || normalized === "[::1]" || normalized.endsWith(".localhost");
  }

  function replaceAppHostPrefix(host) {
    host = String(host || "").trim();
    if (host.indexOf("app.") === 0) {
      return "chat." + host.slice(4);
    }
    return host;
  }

  function addWSHostCandidate(host) {
    if (!host) {
      return;
    }
    var normalized = String(host).trim();
    if (!normalized) {
      return;
    }
    for (var i = 0; i < wsHostCandidates.length; i += 1) {
      if (wsHostCandidates[i] === normalized) {
        return;
      }
    }
    wsHostCandidates.push(normalized);
  }

  function stripChatHostPrefix(hostname) {
    if (String(hostname).indexOf("chat.") === 0) {
      return String(hostname).slice(5);
    }
    return hostname;
  }

  function buildWSHostCandidates() {
    var host = String(window.location.host || "").trim();
    var hostname = String(window.location.hostname || "").trim();
    var chatBaseHostname = stripChatHostPrefix(hostname);
    var pagePort = String(window.location.port || "").trim();
    var canUseLocalFallback = isLocalHost(hostname) || isLocalHost(chatBaseHostname);
    var resolvedHost = host.indexOf("chat.") === 0 ? host : "chat." + host;
    var appHost = replaceAppHostPrefix(host);
    var chatProxyHost = chatBaseHostname;

    if (canUseLocalFallback && !fallbackPort) {
      if (chatProxyHost === "127.0.0.1" || chatProxyHost === "::1" || chatProxyHost === "[::1]") {
        chatProxyHost = "localhost";
      }
      if (pagePort) {
        addWSHostCandidate("chat." + chatProxyHost + ":" + pagePort);
      } else {
        addWSHostCandidate("chat." + chatProxyHost);
      }
    }

    if (fallbackPort) {
      if (canUseLocalFallback) {
        addWSHostCandidate(hostname + ":" + fallbackPort);
        var chatHost = replaceAppHostPrefix(hostname);
        addWSHostCandidate(chatHost + ":" + fallbackPort);
        if (hostname === "localhost" || hostname.endsWith(".localhost")) {
          addWSHostCandidate("127.0.0.1:" + fallbackPort);
          addWSHostCandidate("[::1]:" + fallbackPort);
        } else if (hostname === "127.0.0.1" || hostname === "::1" || hostname === "[::1]") {
          addWSHostCandidate("localhost:" + fallbackPort);
        }
        if (chatBaseHostname !== hostname) {
          addWSHostCandidate(chatBaseHostname + ":" + fallbackPort);
        }
      }
    }

    if (!canUseLocalFallback || !fallbackPort) {
      addWSHostCandidate(resolvedHost);
      if (appHost !== host) {
        addWSHostCandidate(appHost);
      }
    }
  }

  function wsURL() {
    var scheme = window.location.protocol === "https:" ? "wss" : "ws";
    var host = wsHostCandidates[wsHostIndex] || "";
    if (!host) {
      return "";
    }
    return scheme + "://" + host + "/ws";
  }

  function maybeUseNextHost() {
    if (wsHostIndex + 1 >= wsHostCandidates.length) {
      return false;
    }
    wsHostIndex += 1;
    socket = null;
    return true;
  }

  function setStatus(text, isError) {
    if (!statusEl) {
      return;
    }
    statusEl.textContent = text;
    statusEl.classList.toggle("text-error", !!isError);
    statusEl.classList.toggle("opacity-70", !isError);
  }

  function streamByID(streamID) {
    for (var i = 0; i < state.streams.length; i += 1) {
      if (String(state.streams[i].id || "") === String(streamID || "")) {
        return state.streams[i];
      }
    }
    return null;
  }

  function personaByID(personaID) {
    for (var i = 0; i < state.personas.length; i += 1) {
      if (String(state.personas[i].id || "") === String(personaID || "")) {
        return state.personas[i];
      }
    }
    return null;
  }

  function activeGateSummary() {
    var gate = state.activeSessionGate;
    if (!gate) {
      return "No active gate";
    }
    var parts = [];
    if (gate.type) {
      parts.push(String(gate.type));
    }
    if (gate.status) {
      parts.push(String(gate.status));
    }
    if (gate.reason) {
      parts.push(String(gate.reason));
    }
    return parts.join(" · ") || "Active gate";
  }

  function activeSpotlightSummary() {
    var spotlight = state.activeSessionSpotlight;
    if (!spotlight) {
      return "No active spotlight";
    }
    var parts = [];
    if (spotlight.type) {
      parts.push(String(spotlight.type));
    }
    if (spotlight.characterId) {
      parts.push(String(spotlight.characterId));
    }
    return parts.join(" · ") || "Active spotlight";
  }

  function syncJoinedState(payload) {
    payload = payload || {};
    if (payload.session_id) {
      state.sessionID = String(payload.session_id).trim();
    }
    if (Array.isArray(payload.streams)) {
      state.streams = payload.streams.slice();
    }
    if (Array.isArray(payload.personas)) {
      state.personas = payload.personas.slice();
    }
    if (payload.default_stream_id) {
      state.defaultStreamID = String(payload.default_stream_id).trim();
    }
    if (payload.default_persona_id) {
      state.defaultPersonaID = String(payload.default_persona_id).trim();
    }
    if (!state.currentStreamID || !streamByID(state.currentStreamID)) {
      state.currentStreamID = state.defaultStreamID;
    }
    if (!state.currentPersonaID || !personaByID(state.currentPersonaID)) {
      state.currentPersonaID = state.defaultPersonaID;
    }
  }

  function currentStreamLabel() {
    var stream = streamByID(state.currentStreamID);
    if (stream && stream.label) {
      return String(stream.label);
    }
    return state.sessionName || "Transcript";
  }

  function syncControls() {
    if (personaSelectEl) {
      personaSelectEl.value = state.currentPersonaID || state.defaultPersonaID || "";
    }
    if (currentStreamEl) {
      currentStreamEl.textContent = currentStreamLabel();
    }
    if (gateSummaryEl) {
      gateSummaryEl.textContent = activeGateSummary();
    }
    if (spotlightSummaryEl) {
      spotlightSummaryEl.textContent = activeSpotlightSummary();
    }
    for (var i = 0; i < streamButtons.length; i += 1) {
      var button = streamButtons[i];
      var active = String(button.getAttribute("data-stream-id") || "") === String(state.currentStreamID || "");
      button.classList.toggle("btn-primary", active);
      button.classList.toggle("btn-ghost", !active);
      button.setAttribute("aria-pressed", active ? "true" : "false");
    }
    var gateIsOpen = !!(state.activeSessionGate && String(state.activeSessionGate.status || "") === "open");
    if (resolveGateBtn) {
      resolveGateBtn.disabled = !gateIsOpen;
    }
    if (abandonGateBtn) {
      abandonGateBtn.disabled = !gateIsOpen;
    }
  }

  function visibleMessages() {
    var selectedStreamID = String(state.currentStreamID || "");
    if (!selectedStreamID) {
      return state.messages.slice();
    }
    return state.messages.filter(function (message) {
      return String(message.stream_id || "") === selectedStreamID;
    });
  }

  function messageToneClass(message) {
    var kind = String(message.kind || "").toLowerCase();
    if (kind === "system") {
      return "border-info/30 bg-info/5";
    }
    var stream = String(message.stream_id || "");
    if (stream.indexOf(":control") >= 0) {
      return "border-warning/30 bg-warning/5";
    }
    return "border-base-300 bg-base-100";
  }

  function renderTranscript() {
    if (!transcriptEl) {
      return;
    }
    transcriptEl.innerHTML = "";
    var messages = visibleMessages();
    if (messages.length === 0) {
      var empty = document.createElement("div");
      empty.className = "rounded-box border border-dashed border-base-300 p-4 text-sm opacity-60";
      empty.textContent = "No messages in this stream yet.";
      transcriptEl.appendChild(empty);
      return;
    }
    for (var i = 0; i < messages.length; i += 1) {
      var message = messages[i];
      var article = document.createElement("article");
      article.className = "mb-3 rounded-box border p-3 " + messageToneClass(message);

      var meta = document.createElement("div");
      meta.className = "mb-2 flex flex-wrap items-center gap-2 text-xs opacity-70";

      var author = document.createElement("span");
      author.className = "font-semibold";
      author.textContent = String((((message || {}).actor || {}).name) || chatText.participant || "Participant");
      meta.appendChild(author);

      var stream = streamByID(message.stream_id);
      if (stream && stream.label) {
        var streamBadge = document.createElement("span");
        streamBadge.className = "badge badge-ghost badge-xs";
        streamBadge.textContent = String(stream.label);
        meta.appendChild(streamBadge);
      }

      if (message.sent_at) {
        var time = document.createElement("span");
        time.textContent = String(message.sent_at);
        meta.appendChild(time);
      }

      article.appendChild(meta);

      var body = document.createElement("div");
      body.className = "whitespace-pre-wrap break-words text-sm leading-6";
      body.textContent = String(message.body || "");
      article.appendChild(body);

      transcriptEl.appendChild(article);
    }
    transcriptEl.scrollTop = transcriptEl.scrollHeight;
  }

  function sendFrame(type, payload) {
    if (!socket || socket.readyState !== WebSocket.OPEN) {
      return false;
    }
    socket.send(JSON.stringify({
      type: type,
      request_id: randomID("req_"),
      payload: payload
    }));
    return true;
  }

  function pushMessage(message) {
    if (!message) {
      return;
    }
    var sequence = Number(message.sequence_id || 0);
    if (sequence > lastSequenceID) {
      lastSequenceID = sequence;
    }
    state.messages.push(message);
    state.messages.sort(function (left, right) {
      return Number(left.sequence_id || 0) - Number(right.sequence_id || 0);
    });
    renderTranscript();
  }

  function connect() {
    if (!campaignID) {
      setStatus(chatText.invalidId, true);
      return;
    }
    if (socket && (socket.readyState === WebSocket.OPEN || socket.readyState === WebSocket.CONNECTING)) {
      return;
    }
    var wsUrl = wsURL();
    if (!wsUrl) {
      setStatus(chatText.unavailable, true);
      return;
    }

    setStatus(chatText.connecting, false);
    if (sendBtn) {
      sendBtn.disabled = true;
    }

    try {
      socket = new WebSocket(wsUrl);
    } catch (err) {
      if (maybeUseNextHost()) {
        window.setTimeout(connect, 300);
        return;
      }
      setStatus(chatText.unavailable, true);
      return;
    }
    var hasConnected = false;

    socket.addEventListener("open", function () {
      hasConnected = true;
      setStatus(chatText.connected, false);
      if (sendBtn) {
        sendBtn.disabled = false;
      }
      sendFrame("chat.join", {
        campaign_id: campaignID,
        last_sequence_id: lastSequenceID
      });
    });

    socket.addEventListener("message", function (event) {
      var frame = null;
      try {
        frame = JSON.parse(event.data);
      } catch (err) {
        setStatus(chatText.invalidFrame, true);
        return;
      }

      if (frame.type === "chat.joined") {
        var payload = frame.payload || {};
        lastSequenceID = Math.max(lastSequenceID, Number(payload.latest_sequence_id || 0));
        syncJoinedState(payload);
        if (payload.active_session_gate) {
          state.activeSessionGate = payload.active_session_gate;
        }
        if (payload.active_session_spotlight) {
          state.activeSessionSpotlight = payload.active_session_spotlight;
        }
        syncControls();
        renderTranscript();
        return;
      }

      if (frame.type === "chat.message") {
        pushMessage((frame.payload || {}).message || {});
        return;
      }

      if (frame.type === "chat.state") {
        var statePayload = frame.payload || {};
        state.activeSessionGate = statePayload.active_session_gate || null;
        state.activeSessionSpotlight = statePayload.active_session_spotlight || null;
        syncControls();
        return;
      }

      if (frame.type === "chat.error") {
        var errObj = ((frame || {}).payload || {}).error || {};
        var code = errObj.code || "UNKNOWN";
        var text = errObj.message || chatText.requestFailed;
        setStatus(chatText.errorPrefix + ": " + code, true);
        pushMessage({
          kind: "system",
          stream_id: state.currentStreamID,
          actor: { name: "system" },
          body: text + " (" + code + ")",
          sent_at: new Date().toISOString()
        });
      }
    });

    socket.addEventListener("error", function () {
      if (hasConnected) {
        setStatus(chatText.unavailable, true);
      }
    });

    socket.addEventListener("close", function () {
      if (!hasConnected && maybeUseNextHost()) {
        window.setTimeout(connect, 300);
        return;
      }
      setStatus(chatText.disconnectedRetrying, true);
      if (sendBtn) {
        sendBtn.disabled = true;
      }
      window.setTimeout(connect, 2000);
    });
  }

  function currentControlNote() {
    return String((controlNoteEl && controlNoteEl.value) || "").trim();
  }

  function currentResolveDecision() {
    var note = currentControlNote();
    if (note) {
      return note;
    }
    return "resolved";
  }

  if (formEl) {
    formEl.addEventListener("submit", function (event) {
      event.preventDefault();
      var body = String((inputEl && inputEl.value) || "").trim();
      if (!body) {
        return;
      }
      var sent = sendFrame("chat.send", {
        client_message_id: randomID("cli_"),
        body: body,
        stream_id: state.currentStreamID || state.defaultStreamID,
        persona_id: state.currentPersonaID || state.defaultPersonaID
      });
      if (!sent) {
        setStatus(chatText.unableToSend, true);
        return;
      }
      inputEl.value = "";
      inputEl.focus();
    });
  }

  if (personaSelectEl) {
    personaSelectEl.addEventListener("change", function () {
      state.currentPersonaID = String(personaSelectEl.value || "").trim();
      syncControls();
    });
  }

  streamButtons.forEach(function (button) {
    button.addEventListener("click", function () {
      state.currentStreamID = String(button.getAttribute("data-stream-id") || "").trim();
      syncControls();
      renderTranscript();
    });
  });

  if (requestHandoffBtn) {
    requestHandoffBtn.addEventListener("click", function () {
      sendFrame("chat.control", {
        action: "gm_handoff.request",
        reason: currentControlNote()
      });
    });
  }

  if (openGateBtn) {
    openGateBtn.addEventListener("click", function () {
      sendFrame("chat.control", {
        action: "gate.open",
        gate_type: String((openGateTypeEl && openGateTypeEl.value) || "choice").trim(),
        reason: currentControlNote()
      });
    });
  }

  if (resolveGateBtn) {
    resolveGateBtn.addEventListener("click", function () {
      var action = state.activeSessionGate && String(state.activeSessionGate.type || "") === "gm_handoff" ? "gm_handoff.resolve" : "gate.resolve";
      sendFrame("chat.control", {
        action: action,
        decision: currentResolveDecision()
      });
    });
  }

  if (abandonGateBtn) {
    abandonGateBtn.addEventListener("click", function () {
      var action = state.activeSessionGate && String(state.activeSessionGate.type || "") === "gm_handoff" ? "gm_handoff.abandon" : "gate.abandon";
      sendFrame("chat.control", {
        action: action,
        reason: currentControlNote()
      });
    });
  }

  buildWSHostCandidates();
  syncControls();
  renderTranscript();
  connect();
})();
