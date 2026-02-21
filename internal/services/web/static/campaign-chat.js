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

  function randomID(prefix) {
    if (window.crypto && typeof window.crypto.randomUUID === "function") {
      return prefix + window.crypto.randomUUID();
    }
    return prefix + Date.now().toString(36) + Math.random().toString(36).slice(2);
  }

  function wsURL() {
    var scheme = window.location.protocol === "https:" ? "wss" : "ws";
    var host = window.location.host;
    var wsHost = host.indexOf("chat.") === 0 ? host : "chat." + host;
    return scheme + "://" + wsHost + "/ws";
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

    setStatus(chatText.connecting, false);
    if (sendBtn) {
      sendBtn.disabled = true;
    }

    try {
      socket = new WebSocket(wsURL());
    } catch (err) {
      setStatus(chatText.unavailable, true);
      appendLine("", chatText.wsSetupFailed, true);
      return;
    }

    socket.addEventListener("open", function () {
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
      setStatus(chatText.unavailable, true);
    });

    socket.addEventListener("close", function () {
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

  connect();
})();
