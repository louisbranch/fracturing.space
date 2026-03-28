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

  function showButtonLoadingSpinner(button) {
    if (!button) {
      return;
    }
    if (button.querySelector("[data-button-loading-spinner='true']")) {
      return;
    }
    var spinner = document.createElement("span");
    spinner.className = "loading loading-spinner";
    spinner.setAttribute("aria-hidden", "true");
    spinner.setAttribute("data-button-loading-spinner", "true");
    button.insertBefore(spinner, button.firstChild);
  }

  function setupSignupUsernameCheck() {
    var i18nEl = document.getElementById("i18n-strings");
    var form = document.getElementById("register-form");
    var input = document.getElementById("register-username");
    var hintEl = document.getElementById("register-username-hint");
    var button = document.getElementById("passkey-register");
    if (!i18nEl || !form || !input || !hintEl || !button) {
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

    function syncButtonEnabledState() {
      var allowed = button.getAttribute("data-passkey-register-allowed") === "true";
      var busy = form.getAttribute("data-passkey-busy") === "true";
      button.disabled = busy || !allowed;
    }

    function setButtonEnabled(enabled) {
      button.setAttribute("data-passkey-register-allowed", enabled ? "true" : "false");
      syncButtonEnabledState();
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

    button.setAttribute("data-passkey-register-allowed", "false");
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
    var resultButtons = [];
    var activeIndex = -1;
    var resultIDPrefix = "campaign-invite-search-result-";

    if (!resultsEl.id) {
      resultsEl.id = "campaign-invite-search-results";
    }
    resultsEl.setAttribute("role", "listbox");
    input.setAttribute("aria-controls", resultsEl.id);
    input.setAttribute("aria-expanded", "false");

    function syncExpandedState(expanded) {
      input.setAttribute("aria-expanded", expanded ? "true" : "false");
      if (!expanded) {
        input.removeAttribute("aria-activedescendant");
      }
    }

    function clearResults() {
      resultsEl.innerHTML = "";
      resultsEl.hidden = true;
      emptyEl.hidden = true;
      resultButtons = [];
      activeIndex = -1;
      syncExpandedState(false);
    }

    function setActiveIndex(index) {
      if (resultButtons.length === 0) {
        activeIndex = -1;
        syncExpandedState(false);
        return;
      }
      if (index < 0) {
        index = resultButtons.length - 1;
      }
      if (index >= resultButtons.length) {
        index = 0;
      }
      activeIndex = index;
      resultButtons.forEach(function (button, buttonIndex) {
        var isActive = buttonIndex === activeIndex;
        button.setAttribute("aria-selected", isActive ? "true" : "false");
        button.setAttribute("data-campaign-invite-search-result-active", isActive ? "true" : "false");
        button.classList.toggle("border-primary", isActive);
        button.classList.toggle("bg-base-200", isActive);
        button.classList.toggle("shadow-sm", isActive);
      });
      syncExpandedState(true);
      input.setAttribute("aria-activedescendant", resultButtons[activeIndex].id);
      resultButtons[activeIndex].scrollIntoView({ block: "nearest" });
    }

    function chooseUser(user) {
      input.value = user && user.username ? user.username : "";
      clearResults();
      input.focus();
    }

    function renderResults(users) {
      resultsEl.innerHTML = "";
      resultButtons = [];
      activeIndex = -1;
      if (!Array.isArray(users) || users.length === 0) {
        resultsEl.hidden = true;
        emptyEl.textContent = emptyMessage;
        emptyEl.hidden = false;
        syncExpandedState(false);
        return;
      }
      emptyEl.hidden = true;
      users.forEach(function (user) {
        var button = document.createElement("button");
        button.type = "button";
        button.id = resultIDPrefix + requestSequence + "-" + resultButtons.length;
        button.className = "flex w-full cursor-pointer items-start justify-between gap-3 rounded-lg border border-base-300 bg-base-100 px-4 py-3 text-left transition hover:border-base-content/20 hover:bg-base-200/40";
        button.setAttribute("role", "option");
        button.setAttribute("aria-selected", "false");
        button.setAttribute("data-campaign-invite-search-username", user.username || "");
        button.addEventListener("mousedown", function (event) {
          event.preventDefault();
          chooseUser(user);
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
        resultButtons.push(button);
      });
      resultsEl.hidden = false;
      setActiveIndex(0);
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

    input.addEventListener("keydown", function (event) {
      if (resultButtons.length === 0 || resultsEl.hidden) {
        if (event.key === "Escape") {
          clearResults();
        }
        return;
      }
      if (event.key === "ArrowDown") {
        event.preventDefault();
        setActiveIndex(activeIndex + 1);
        return;
      }
      if (event.key === "ArrowUp") {
        event.preventDefault();
        setActiveIndex(activeIndex - 1);
        return;
      }
      if (event.key === "Enter" && activeIndex >= 0) {
        event.preventDefault();
        chooseUser({ username: resultButtons[activeIndex].getAttribute("data-campaign-invite-search-username") || "" });
        return;
      }
      if (event.key === "Escape") {
        event.preventDefault();
        clearResults();
      }
    });

    document.addEventListener("mousedown", function (event) {
      if (form.contains(event.target)) {
        return;
      }
      clearResults();
    });
  }

  function setupInviteMutationLoadingState() {
    function markBusy(form, button) {
      if (!form || !button) {
        return;
      }
      form.setAttribute("aria-busy", "true");
      form.setAttribute("data-campaign-submit-busy", "true");
      button.disabled = true;
      showButtonLoadingSpinner(button);
      button.setAttribute("aria-busy", "true");
    }

    function bindInviteSubmit(formSelector, buttonSelector) {
      document.querySelectorAll(formSelector).forEach(function (form) {
        if (form.getAttribute("data-campaign-submit-bound") === "true") {
          return;
        }
        form.setAttribute("data-campaign-submit-bound", "true");
        form.addEventListener("submit", function (event) {
          if (form.getAttribute("data-campaign-submit-busy") === "true") {
            event.preventDefault();
            return;
          }
          var button = event.submitter || form.querySelector(buttonSelector);
          if (!button) {
            return;
          }
          markBusy(form, button);
        });
      });
    }

    bindInviteSubmit("[data-campaign-invite-create-form='true']", "[data-campaign-invite-create-submit='true']");
    bindInviteSubmit("[data-campaign-invite-revoke-form='true']", "[data-campaign-invite-revoke-submit='true']");
  }

  setupSignupUsernameCheck();
  setupInviteUsernameSearch();
  setupInviteMutationLoadingState();
})();
