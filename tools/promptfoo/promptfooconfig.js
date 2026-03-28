const path = require('node:path');

const defaultPromptProfiles = ['baseline', 'mechanics_hardened'];
const defaultModels = ['gpt-5.4', 'gpt-5.4-mini', 'gpt-5.3-codex'];

function promptSummary(profile) {
  if (profile === 'mechanics_hardened') {
    return 'Mechanics-hardened GM instruction profile with the eval-only instruction override.';
  }
  return 'Baseline GM instruction profile using the default repo instruction bundle.';
}

function parseListEnv(value, fallback) {
  const raw = String(value || '')
    .split(',')
    .map((entry) => entry.trim())
    .filter(Boolean);
  return raw.length > 0 ? raw : fallback;
}

function repeatCount() {
  const parsed = Number.parseInt(process.env.PROMPTFOO_REPEAT || '1', 10);
  if (!Number.isFinite(parsed) || parsed < 1) {
    return 1;
  }
  return parsed;
}

function buildPromptProfilePrompt(profile) {
  return {
    raw: promptSummary(profile),
    label: profile,
    config: {
      profile,
      promptSummary: promptSummary(profile),
    },
  };
}

function selectedScenarioSet() {
  const requested = String(process.env.PROMPTFOO_SCENARIO_SET || 'core').trim().toLowerCase();
  if (requested === 'extended') {
    return 'extended';
  }
  return 'core';
}

function loadScenarios() {
  const scenarioSet = selectedScenarioSet();
  return require(path.join(__dirname, 'scenarios', `${scenarioSet}.json`));
}

function encodeScenarioVars(scenario) {
  const out = {};
  for (const [key, value] of Object.entries(scenario)) {
    if (Array.isArray(value) || (value && typeof value === 'object')) {
      out[key] = JSON.stringify(value);
      continue;
    }
    out[key] = value;
  }
  return out;
}

function buildTests(scenarios, count) {
  const tests = [];
  for (const scenario of scenarios) {
    for (let i = 0; i < count; i += 1) {
      const suffix = count > 1 ? ` [run ${i + 1}]` : '';
      tests.push({
        description: `${scenario.label}${suffix}`,
        vars: encodeScenarioVars({
          ...scenario,
          repeat_index: i + 1,
          repeat_count: count,
          scenario_set: selectedScenarioSet(),
        }),
        assert: [
          {
            type: 'javascript',
            value: 'file://assertions/gm_contract.js',
          },
        ],
      });
    }
  }
  return tests;
}

const promptProfiles = parseListEnv(process.env.PROMPTFOO_PROMPT_PROFILES, defaultPromptProfiles);
const models = parseListEnv(process.env.PROMPTFOO_MODELS, defaultModels);
const scenarios = loadScenarios();
const repeats = repeatCount();

module.exports = {
  description: 'AI GM mechanics-fidelity eval over the live Go orchestration harness',
  prompts: promptProfiles.map(buildPromptProfilePrompt),
  providers: models.map((model) => ({
    id: 'file://providers/gm_scenario_provider.js',
    label: model,
    config: {
      model,
      reasoningEffort: 'medium',
      instructionsRoots: {
        baseline: '',
        mechanics_hardened: './instructions/mechanics_hardened',
      },
    },
  })),
  tests: buildTests(scenarios, repeats),
};
