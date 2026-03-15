(function () {
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

  function credentialToJSON(credential) {
    if (!credential) {
      return null;
    }
    var response = credential.response;
    var data = {
      id: credential.id,
      rawId: bufferToBase64Url(credential.rawId),
      type: credential.type
    };
    if (response) {
      data.response = {
        clientDataJSON: bufferToBase64Url(response.clientDataJSON),
        attestationObject: response.attestationObject ? bufferToBase64Url(response.attestationObject) : undefined,
        authenticatorData: response.authenticatorData ? bufferToBase64Url(response.authenticatorData) : undefined,
        signature: response.signature ? bufferToBase64Url(response.signature) : undefined,
        userHandle: response.userHandle ? bufferToBase64Url(response.userHandle) : undefined
      };
    }
    return data;
  }

  function normalizeRequestOptions(options) {
    var publicKey = options;
    publicKey.challenge = base64UrlToBuffer(publicKey.challenge);
    if (Array.isArray(publicKey.allowCredentials)) {
      publicKey.allowCredentials = publicKey.allowCredentials.map(function (cred) {
        return {
          id: base64UrlToBuffer(cred.id),
          transports: cred.transports,
          type: cred.type
        };
      });
    }
    return publicKey;
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

  async function readErrorMessage(response, fallback) {
    try {
      var payload = await response.json();
      if (payload && typeof payload.error === "string" && payload.error.trim() !== "") {
        return payload.error;
      }
    } catch (_) {
      // Ignore malformed JSON and fall back to the generic message.
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

  function setupLoginAndRegister() {
    var i18nEl = document.getElementById("i18n-strings");
    if (!i18nEl) {
      return;
    }

    var i18n = i18nEl.dataset || {};
    var pendingID = i18n.pendingId || "";
    var nextPath = i18n.next || "";
    var loginStartPath = i18n.loginStartPath || "";
    var loginFinishPath = i18n.loginFinishPath || "";
    var registerStartPath = i18n.registerStartPath || "";
    var registerFinishPath = i18n.registerFinishPath || "";
    var jsLoginStartError = i18n.loginStartError || "Unable to start passkey login.";
    var jsLoginFinishError = i18n.loginFinishError || "Unable to finish passkey login.";
    var jsRegisterStartError = i18n.registerStartError || "Unable to start passkey registration.";
    var jsRegisterFinishError = i18n.registerFinishError || "Unable to finish passkey registration.";
    var jsPasskeyFailed = i18n.passkeyFailed || "Passkey login failed.";
    var jsLoginUsernameRequired = i18n.loginUsernameRequired || "Username is required to log in.";
    var jsRegisterUsernameRequired = i18n.registerUsernameRequired || "Username is required to create an account.";
    var jsRegisterFailed = i18n.registerFailed || "Passkey registration failed.";

    var passkeyButton = document.getElementById("passkey-login");
    var passkeyError = document.getElementById("passkey-error");
    var loginForm = document.getElementById("login-form");
    var loginUsername = document.getElementById("login-username");
    var registerButton = document.getElementById("passkey-register");
    var registerForm = document.getElementById("register-form");
    var registerUsername = document.getElementById("register-username");
    var registerError = document.getElementById("register-error");

    async function startPasskeyLogin(username) {
      return postJSON(loginStartPath, { username: username }, jsLoginStartError);
    }

    async function finishPasskeyLogin(sessionID, credential) {
      return postJSON(loginFinishPath, {
        session_id: sessionID,
        pending_id: pendingID,
        next: nextPath,
        credential: credential
      }, jsLoginFinishError);
    }

    async function startPasskeyRegister(username) {
      return postJSON(registerStartPath, { username: username }, jsRegisterStartError);
    }

    async function finishPasskeyRegister(sessionID, credential) {
      return postJSON(registerFinishPath, {
        session_id: sessionID,
        pending_id: pendingID,
        next: nextPath,
        credential: credential
      }, jsRegisterFinishError);
    }

    async function performPasskeyLogin(username) {
      if (!username) {
        throw new Error(jsLoginUsernameRequired);
      }
      var start = await startPasskeyLogin(username);
      var publicKey = normalizeRequestOptions(start.public_key.publicKey);
      var assertion = await navigator.credentials.get({ publicKey: publicKey });
      var finish = await finishPasskeyLogin(start.session_id, credentialToJSON(assertion));
      if (finish.redirect_url) {
        window.location = finish.redirect_url;
      }
    }

    if (loginForm && passkeyButton && window.PublicKeyCredential) {
      passkeyButton.addEventListener("click", async function () {
        hide(passkeyError);
        try {
          var username = loginUsername ? loginUsername.value.trim() : "";
          await performPasskeyLogin(username);
        } catch (err) {
          show(passkeyError, err.message || jsPasskeyFailed);
        }
      });
    }

    if (registerForm && registerButton && window.PublicKeyCredential) {
      registerButton.addEventListener("click", async function () {
        hide(registerError);
        var username = registerUsername ? registerUsername.value.trim() : "";
        if (!username) {
          show(registerError, jsRegisterUsernameRequired);
          return;
        }
        try {
          if (registerButton.disabled) {
            return;
          }
      var start = await startPasskeyRegister(username);
      var publicKey = normalizeCreationOptions(start.public_key.publicKey);
      var credential = await navigator.credentials.create({ publicKey: publicKey });
      var finish = await finishPasskeyRegister(start.session_id, credentialToJSON(credential));
          if (finish.redirect_url) {
            window.location = finish.redirect_url;
          }
        } catch (err) {
          show(registerError, err.message || jsRegisterFailed);
        }
      });
    }
  }

  function setupRecovery() {
    var i18nEl = document.getElementById("recovery-i18n-strings");
    if (!i18nEl) {
      return;
    }

    var i18n = i18nEl.dataset || {};
    var pendingID = i18n.pendingId || "";
    var nextPath = i18n.next || "";
    var recoveryStartPath = i18n.recoveryStartPath || "";
    var recoveryFinishPath = i18n.recoveryFinishPath || "";
    var jsRecoveryStartError = i18n.recoveryStartError || "Unable to start account recovery.";
    var jsRecoveryFinishError = i18n.recoveryFinishError || "Unable to finish account recovery.";
    var jsRecoveryUsernameRequired = i18n.recoveryUsernameRequired || "Username is required to recover an account.";
    var jsRecoveryCodeRequired = i18n.recoveryCodeRequired || "Recovery code is required to recover an account.";
    var jsRecoveryFailed = i18n.recoveryFailed || "Account recovery failed.";

    var recoveryButton = document.getElementById("recovery-submit");
    var recoveryError = document.getElementById("recovery-error");
    var recoveryForm = document.getElementById("recovery-form");
    var recoveryUsername = document.getElementById("recovery-username");
    var recoveryCode = document.getElementById("recovery-code");

    async function startRecovery(username, code) {
      return postJSON(recoveryStartPath, {
        username: username,
        recovery_code: code
      }, jsRecoveryStartError);
    }

    async function finishRecovery(recoverySessionID, sessionID, credential) {
      return postJSON(recoveryFinishPath, {
        recovery_session_id: recoverySessionID,
        session_id: sessionID,
        pending_id: pendingID,
        next: nextPath,
        credential: credential
      }, jsRecoveryFinishError);
    }

    if (recoveryForm && recoveryButton && window.PublicKeyCredential) {
      recoveryButton.addEventListener("click", async function () {
        hide(recoveryError);
        var username = recoveryUsername ? recoveryUsername.value.trim() : "";
        var code = recoveryCode ? recoveryCode.value.trim() : "";
        if (!username) {
          show(recoveryError, jsRecoveryUsernameRequired);
          return;
        }
        if (!code) {
          show(recoveryError, jsRecoveryCodeRequired);
          return;
        }
        try {
          var start = await startRecovery(username, code);
          var publicKey = normalizeCreationOptions(start.public_key.publicKey);
          var credential = await navigator.credentials.create({ publicKey: publicKey });
          var finish = await finishRecovery(start.recovery_session_id, start.session_id, credentialToJSON(credential));
          if (finish.redirect_url) {
            window.location = finish.redirect_url;
          }
        } catch (err) {
          show(recoveryError, err.message || jsRecoveryFailed);
        }
      });
    }
  }

  function setupRecoveryCodePage() {
    var i18nEl = document.getElementById("recovery-code-i18n");
    if (!i18nEl) {
      return;
    }

    var downloadButton = document.getElementById("recovery-code-download");
    var ackCheckbox = document.getElementById("recovery-code-ack");
    var continueButton = document.getElementById("recovery-code-continue");
    var codeEl = document.querySelector("[data-recovery-code]");
    var code = codeEl ? codeEl.getAttribute("data-recovery-code") || "" : "";
    var filename = (i18nEl.dataset || {}).downloadFilename || "fracturing-space-recovery-code.txt";

    if (ackCheckbox && continueButton) {
      ackCheckbox.addEventListener("change", function () {
        continueButton.disabled = !ackCheckbox.checked;
      });
    }

    if (downloadButton && code) {
      downloadButton.addEventListener("click", function () {
        var blob = new Blob([code + "\n"], { type: "text/plain;charset=utf-8" });
        var objectURL = window.URL.createObjectURL(blob);
        var link = document.createElement("a");
        link.href = objectURL;
        link.download = filename;
        document.body.appendChild(link);
        link.click();
        document.body.removeChild(link);
        window.URL.revokeObjectURL(objectURL);
      });
    }
  }

  setupLoginAndRegister();
  setupRecovery();
  setupRecoveryCodePage();
})();
