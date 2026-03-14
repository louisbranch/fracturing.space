(function () {
  var i18nEl = document.getElementById("settings-security-i18n");
  if (!i18nEl || !window.PublicKeyCredential) {
    return;
  }

  function show(el, message) {
    if (!el) {
      return;
    }
    el.textContent = message;
    el.hidden = false;
  }

  function hide(el) {
    if (!el) {
      return;
    }
    el.hidden = true;
  }

  function bufferToBase64Url(buffer) {
    var bytes = new Uint8Array(buffer);
    var binary = "";
    bytes.forEach(function (b) {
      binary += String.fromCharCode(b);
    });
    return btoa(binary).replace(/\+/g, "-").replace(/\//g, "_").replace(/=+$/, "");
  }

  function base64UrlToBuffer(base64Url) {
    var base64 = String(base64Url || "").replace(/-/g, "+").replace(/_/g, "/");
    var pad = base64.length % 4 ? "=".repeat(4 - (base64.length % 4)) : "";
    var binary = atob(base64 + pad);
    var bytes = new Uint8Array(binary.length);
    for (var i = 0; i < binary.length; i += 1) {
      bytes[i] = binary.charCodeAt(i);
    }
    return bytes.buffer;
  }

  function normalizeCreationOptions(options) {
    var publicKey = options;
    publicKey.challenge = base64UrlToBuffer(publicKey.challenge);
    if (publicKey.user && publicKey.user.id) {
      publicKey.user.id = base64UrlToBuffer(publicKey.user.id);
    }
    if (Array.isArray(publicKey.excludeCredentials)) {
      publicKey.excludeCredentials = publicKey.excludeCredentials.map(function (cred) {
        return {
          id: base64UrlToBuffer(cred.id),
          transports: cred.transports,
          type: cred.type
        };
      });
    }
    return publicKey;
  }

  function credentialToJSON(credential) {
    if (!credential) {
      return null;
    }
    return {
      id: credential.id,
      rawId: bufferToBase64Url(credential.rawId),
      type: credential.type,
      response: {
        clientDataJSON: bufferToBase64Url(credential.response.clientDataJSON),
        attestationObject: bufferToBase64Url(credential.response.attestationObject)
      }
    };
  }

  async function readErrorMessage(response, fallback) {
    try {
      var payload = await response.json();
      if (payload && typeof payload.error === "string" && payload.error.trim() !== "") {
        return payload.error;
      }
    } catch (_) {
      // Ignore malformed JSON and use fallback text.
    }
    return fallback;
  }

  async function postJSON(url, body, fallback) {
    var response = await fetch(url, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify(body)
    });
    if (!response.ok) {
      throw new Error(await readErrorMessage(response, fallback));
    }
    return response.json();
  }

  var i18n = i18nEl.dataset || {};
  var startPath = i18n.passkeyStartPath || "";
  var finishPath = i18n.passkeyFinishPath || "";
  var startError = i18n.passkeyStartError || "Unable to start passkey registration.";
  var finishError = i18n.passkeyFinishError || "Unable to finish passkey registration.";
  var passkeyFailed = i18n.passkeyFailed || "Unable to add a passkey.";

  var addButton = document.getElementById("settings-passkey-add");
  var errorEl = document.getElementById("settings-security-error");

  if (!addButton) {
    return;
  }

  addButton.addEventListener("click", async function () {
    hide(errorEl);
    try {
      var start = await postJSON(startPath, {}, startError);
      var publicKey = normalizeCreationOptions(start.public_key.publicKey);
      var credential = await navigator.credentials.create({ publicKey: publicKey });
      var finish = await postJSON(finishPath, {
        session_id: start.session_id,
        credential: credentialToJSON(credential)
      }, finishError);
      if (finish.redirect_url) {
        window.location = finish.redirect_url;
      }
    } catch (err) {
      show(errorEl, err.message || passkeyFailed);
    }
  });
})();
