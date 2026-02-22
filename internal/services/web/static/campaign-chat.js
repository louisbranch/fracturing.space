(function () {
  var root = document.querySelector("[data-campaign-id]");
  if (!root) {
    return;
  }

  var campaignID = root.getAttribute("data-campaign-id") || "";
  var statusEl = document.getElementById("chat-status");
  var messagesEl = document.getElementById("chat-messages");
  var formEl = document.getElementById("chat-form");
  var inputEl = document.getElementById("chat-input");
  var sendBtn = document.getElementById("chat-send");
  var socket = null;
  var lastSequenceID = 0;
  var wsHostCandidates = [];
  var wsHostIndex = 0;
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

  function isLocalHost(hostname) {
    return hostname === "localhost" || hostname === "127.0.0.1" || hostname === "::1" || hostname === "[::1]";
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
    var host = window.location.host;
    var hostname = window.location.hostname;
    var chatBaseHostname = stripChatHostPrefix(hostname);
    var pagePort = String(window.location.port || "").trim();
    var canUseLocalFallback = isLocalHost(hostname) || isLocalHost(chatBaseHostname);
    var chatProxyHost = chatBaseHostname;

    if (canUseLocalFallback) {
      if (chatProxyHost === "127.0.0.1" || chatProxyHost === "::1" || chatProxyHost === "[::1]") {
        chatProxyHost = "localhost";
      }
      if (pagePort) {
        addWSHostCandidate("chat." + chatProxyHost + ":" + pagePort);
      } else {
        addWSHostCandidate("chat." + chatProxyHost);
      }
    }

    if (canUseLocalFallback && fallbackPort) {
      addWSHostCandidate(hostname + ":" + fallbackPort);
      if (hostname === "localhost") {
        addWSHostCandidate("127.0.0.1:" + fallbackPort);
        addWSHostCandidate("[::1]:" + fallbackPort);
      } else if (hostname === "127.0.0.1") {
        addWSHostCandidate("localhost:" + fallbackPort);
      } else if (hostname === "::1" || hostname === "[::1]") {
        addWSHostCandidate("localhost:" + fallbackPort);
      }
      if (chatBaseHostname !== hostname) {
        addWSHostCandidate(chatBaseHostname + ":" + fallbackPort);
      }
    }

    if (!canUseLocalFallback) {
      addWSHostCandidate(host.indexOf("chat.") === 0 ? host : "chat." + host);
    }
  }

  function randomID(prefix) {
    if (window.crypto && typeof window.crypto.randomUUID === "function") {
      return prefix + window.crypto.randomUUID();
    }
    return prefix + Date.now().toString(36) + Math.random().toString(36).slice(2);
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

  function appendLine(author, text, isSystem) {
    if (!messagesEl) {
      return;
    }
    var line = document.createElement("article");
    line.className = "rounded border border-base-300 p-2";
    if (isSystem) {
      line.className += " opacity-70";
    }
    if (author) {
      var authorEl = document.createElement("div");
      authorEl.className = "text-xs font-semibold text-primary";
      authorEl.textContent = author;
      line.appendChild(authorEl);
    }
    var bodyEl = document.createElement("div");
    bodyEl.textContent = text;
    line.appendChild(bodyEl);
    messagesEl.appendChild(line);
    messagesEl.scrollTop = messagesEl.scrollHeight;
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
      appendLine("", chatText.wsSetupFailed, true);
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
        setStatus(chatText.connecting, false);
        window.setTimeout(connect, 300);
        return;
      }
      setStatus(chatText.unavailable, true);
      appendLine("", chatText.wsSetupFailed, true);
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
        appendLine("", chatText.invalidFrame, true);
        return;
      }

      if (frame.type === "chat.joined") {
        var latest = Number((((frame || {}).payload || {}).latest_sequence_id) || 0);
        if (latest > lastSequenceID) {
          lastSequenceID = latest;
        }
        appendLine("", chatText.joined, true);
        return;
      }

      if (frame.type === "chat.message") {
        var message = (((frame || {}).payload || {}).message) || {};
        var sequence = Number(message.sequence_id || 0);
        if (sequence > lastSequenceID) {
          lastSequenceID = sequence;
        }
        var author = (((message || {}).actor || {}).name) || chatText.participant;
        appendLine(author, String(message.body || ""), false);
        return;
      }

      if (frame.type === "chat.error") {
        var errObj = ((frame || {}).payload || {}).error || {};
        var code = errObj.code || "UNKNOWN";
        var text = errObj.message || chatText.requestFailed;
        setStatus(chatText.errorPrefix + ": " + code, true);
        appendLine("", text + " (" + code + ")", true);
      }
    });

    socket.addEventListener("error", function () {
      if (hasConnected) {
        setStatus(chatText.unavailable, true);
      }
    });

    socket.addEventListener("close", function () {
      if (!hasConnected && maybeUseNextHost()) {
        setStatus(chatText.connecting, false);
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

  if (formEl) {
    formEl.addEventListener("submit", function (event) {
      event.preventDefault();
      if (!inputEl) {
        return;
      }
      var body = inputEl.value.trim();
      if (!body) {
        return;
      }
      var ok = sendFrame("chat.send", {
        client_message_id: randomID("cli_"),
        body: body
      });
      if (!ok) {
        appendLine("", chatText.unableToSend, true);
        return;
      }
      inputEl.value = "";
      inputEl.focus();
    });
  }

  buildWSHostCandidates();
  connect();
})();
