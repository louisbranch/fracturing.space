(function () {
  function debounce(fn, delayMs) {
    var timeoutID = 0;
    return function () {
      var args = arguments;
      clearTimeout(timeoutID);
      timeoutID = window.setTimeout(function () {
        fn.apply(null, args);
      }, delayMs);
    };
  }

  async function readErrorMessage(response, fallback) {
    try {
      var payload = await response.json();
      if (payload && typeof payload.error === "string" && payload.error.trim() !== "") {
        return payload.error;
      }
    } catch (_) {
      // Fall back to the caller-provided message.
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

  function setHiddenText(el, message) {
    if (!el) {
      return;
    }
    el.textContent = message || "";
    el.hidden = !message;
  }

  function setupSignupUsernameCheck() {
    var i18nEl = document.getElementById("i18n-strings");
    var input = document.getElementById("register-username");
    var hintEl = document.getElementById("register-username-hint");
    var button = document.getElementById("passkey-register");
    if (!i18nEl || !input || !hintEl || !button) {
      return;
    }

    var data = i18nEl.dataset || {};
    var checkPath = data.registerCheckPath || "";
    if (!checkPath) {
      return;
    }

    var messages = {
      invalid: data.registerUsernameInvalid || "Use 3 to 32 lowercase letters, numbers, or underscores.",
      unavailable: data.registerUsernameUnavailable || "That username is already taken.",
      error: data.registerCheckError || "Unable to validate the username right now."
    };
    var requestSequence = 0;

    function setButtonEnabled(enabled) {
      button.disabled = !enabled;
    }

    function resetValidationState() {
      input.classList.remove("validator");
      input.removeAttribute("aria-invalid");
      input.setCustomValidity("");
      setHiddenText(hintEl, "");
    }

    function applyInvalidState(message) {
      input.classList.add("validator");
      input.setAttribute("aria-invalid", "true");
      input.setCustomValidity(message);
      setHiddenText(hintEl, message);
    }

    function applyValidState() {
      input.classList.add("validator");
      input.removeAttribute("aria-invalid");
      input.setCustomValidity("");
      setHiddenText(hintEl, "");
    }

    function applyState(state) {
      if (state === "available") {
        applyValidState();
        setButtonEnabled(true);
        return;
      }
      if (state === "unavailable") {
        applyInvalidState(messages.unavailable);
        setButtonEnabled(false);
        return;
      }
      applyInvalidState(messages.invalid);
      setButtonEnabled(false);
    }

    var checkUsername = debounce(async function (value, sequence) {
      if (value.length < 3) {
        resetValidationState();
        setButtonEnabled(false);
        return;
      }
      try {
        var payload = await postJSON(checkPath, { username: value }, messages.error);
        if (sequence !== requestSequence) {
          return;
        }
        applyState(payload.state || "invalid");
      } catch (err) {
        if (sequence !== requestSequence) {
          return;
        }
        applyInvalidState(err.message || messages.error);
        setButtonEnabled(false);
      }
    }, 200);

    setButtonEnabled(false);
    input.addEventListener("input", function () {
      requestSequence += 1;
      checkUsername(input.value.trim(), requestSequence);
    });
  }

  function setupInviteUsernameSearch() {
    var form = document.querySelector("[data-campaign-invite-create-form='true']");
    if (!form) {
      return;
    }
    var input = form.querySelector("[data-campaign-invite-search-input='true']");
    var resultsEl = form.querySelector("[data-campaign-invite-search-results='true']");
    var emptyEl = form.querySelector("[data-campaign-invite-search-empty='true']");
    if (!input || !resultsEl || !emptyEl) {
      return;
    }

    var searchPath = form.getAttribute("data-campaign-invite-search-path") || "";
    if (!searchPath) {
      return;
    }
    var requestSequence = 0;
    var emptyMessage = emptyEl.textContent || "No users found.";
    var errorMessage = form.getAttribute("data-campaign-invite-search-error") || "Unable to search users right now.";

    function clearResults() {
      resultsEl.innerHTML = "";
      resultsEl.hidden = true;
      emptyEl.hidden = true;
    }

    function renderResults(users) {
      resultsEl.innerHTML = "";
      if (!Array.isArray(users) || users.length === 0) {
        resultsEl.hidden = true;
        emptyEl.textContent = emptyMessage;
        emptyEl.hidden = false;
        return;
      }
      emptyEl.hidden = true;
      users.forEach(function (user) {
        var button = document.createElement("button");
        button.type = "button";
        button.className = "flex w-full items-center justify-between gap-3 rounded-lg border border-base-300 bg-base-100 px-3 py-2 text-left";
        button.addEventListener("click", function () {
          input.value = user.username || "";
          clearResults();
          input.focus();
        });

        var label = document.createElement("div");
        label.className = "grid gap-1";

        var primary = document.createElement("span");
        primary.className = "font-medium";
        primary.textContent = user.name && user.name.trim() !== "" ? user.name : "@" + (user.username || "");
        label.appendChild(primary);

        var secondary = document.createElement("span");
        secondary.className = "text-sm opacity-70";
        secondary.textContent = "@" + (user.username || "");
        label.appendChild(secondary);

        button.appendChild(label);

        if (user.is_contact) {
          var badge = document.createElement("span");
          badge.className = "badge badge-outline";
          badge.textContent = "Contact";
          button.appendChild(badge);
        }

        resultsEl.appendChild(button);
      });
      resultsEl.hidden = false;
    }

    var searchUsers = debounce(async function (value, sequence) {
      if (value.length < 2) {
        clearResults();
        return;
      }
      try {
        var payload = await postJSON(searchPath, { query: value, limit: 8 }, errorMessage);
        if (sequence !== requestSequence) {
          return;
        }
        renderResults(payload.users || []);
      } catch (err) {
        if (sequence !== requestSequence) {
          return;
        }
        resultsEl.innerHTML = "";
        resultsEl.hidden = true;
        emptyEl.textContent = err.message || errorMessage;
        emptyEl.hidden = false;
      }
    }, 200);

    input.addEventListener("input", function () {
      requestSequence += 1;
      searchUsers(input.value.trim(), requestSequence);
    });

    input.addEventListener("blur", function () {
      window.setTimeout(clearResults, 150);
    });
  }

  setupSignupUsernameCheck();
  setupInviteUsernameSearch();
})();
