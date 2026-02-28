(function () {
  var i18nEl = document.getElementById("i18n-strings");
  if (!i18nEl) {
    return;
  }

  var i18n = i18nEl.dataset || {};
  var loginStartPath = i18n.loginStartPath || "";
  var loginFinishPath = i18n.loginFinishPath || "";
  var registerStartPath = i18n.registerStartPath || "";
  var registerFinishPath = i18n.registerFinishPath || "";
  var jsLoginStartError = i18n.loginStartError || "failed to start passkey login";
  var jsLoginFinishError = i18n.loginFinishError || "failed to finish passkey login";
  var jsRegisterStartError = i18n.registerStartError || "failed to start passkey registration";
  var jsRegisterFinishError = i18n.registerFinishError || "failed to finish passkey registration";
  var jsPasskeyFailed = i18n.passkeyFailed || "failed to sign in with passkey";
  var jsEmailRequired = i18n.emailRequired || "email is required";
  var jsPasskeyCreated = i18n.passkeyCreated || "Passkey created; signing you in";
  var jsRegisterFailed = i18n.registerFailed || "failed to create passkey";

  var passkeyButton = document.getElementById("passkey-login");
  var passkeyError = document.getElementById("passkey-error");
  var registerButton = document.getElementById("passkey-register");
  var registerEmail = document.getElementById("email");
  var registerError = document.getElementById("register-error");
  var registerSuccess = document.getElementById("register-success");

  function show(el, message) {
    if (!el) {
      return;
    }
    el.textContent = message;
    el.hidden = false;
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
      // Use fallback if payload is not JSON.
    }
    return fallback;
  }

  async function startPasskeyLogin() {
    var response = await fetch(loginStartPath, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({})
    });
    if (!response.ok) {
      throw new Error(await readErrorMessage(response, jsLoginStartError));
    }
    return response.json();
  }

  async function finishPasskeyLogin(sessionID, credential) {
    var response = await fetch(loginFinishPath, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ session_id: sessionID, credential: credential })
    });
    if (!response.ok) {
      throw new Error(await readErrorMessage(response, jsLoginFinishError));
    }
    return response.json();
  }

  async function startPasskeyRegister(email) {
    var response = await fetch(registerStartPath, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ email: email })
    });
    if (!response.ok) {
      throw new Error(await readErrorMessage(response, jsRegisterStartError));
    }
    return response.json();
  }

  async function finishPasskeyRegister(sessionID, credential) {
    var response = await fetch(registerFinishPath, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ session_id: sessionID, credential: credential })
    });
    if (!response.ok) {
      throw new Error(await readErrorMessage(response, jsRegisterFinishError));
    }
    return response.json();
  }

  async function performPasskeyLogin() {
    var start = await startPasskeyLogin();
    var publicKey = normalizeRequestOptions(start.public_key.publicKey);
    var assertion = await navigator.credentials.get({ publicKey: publicKey });
    var credentialJSON = credentialToJSON(assertion);
    var finish = await finishPasskeyLogin(start.session_id, credentialJSON);
    if (finish.redirect_url) {
      window.location = finish.redirect_url;
    }
  }

  if (passkeyButton && window.PublicKeyCredential) {
    passkeyButton.addEventListener("click", async function () {
      if (passkeyError) {
        passkeyError.hidden = true;
      }
      try {
        await performPasskeyLogin();
      } catch (err) {
        show(passkeyError, err.message || jsPasskeyFailed);
      }
    });
  }

  if (registerButton && window.PublicKeyCredential) {
    registerButton.addEventListener("click", async function () {
      if (registerError) {
        registerError.hidden = true;
      }
      if (registerSuccess) {
        registerSuccess.hidden = true;
      }
      var email = registerEmail ? registerEmail.value.trim() : "";
      if (!email) {
        show(registerError, jsEmailRequired);
        return;
      }
      try {
        var start = await startPasskeyRegister(email);
        var publicKey = normalizeCreationOptions(start.public_key.publicKey);
        var credential = await navigator.credentials.create({ publicKey: publicKey });
        await finishPasskeyRegister(start.session_id, credentialToJSON(credential));
        show(registerSuccess, jsPasskeyCreated);
        await performPasskeyLogin();
      } catch (err) {
        show(registerError, err.message || jsRegisterFailed);
      }
    });
  }
})();
