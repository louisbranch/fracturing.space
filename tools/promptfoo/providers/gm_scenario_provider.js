const { spawnSync } = require('node:child_process');
const fs = require('node:fs');
const path = require('node:path');

function compactFailureText(text) {
  const lines = String(text || '')
    .split(/\r?\n/)
    .map((line) => line.trim())
    .filter(Boolean)
    .filter((line) => !/^\d{4}\/\d{2}\/\d{2}\s/.test(line))
    .filter((line) => !line.includes('written to '))
    .filter((line) => !line.startsWith('FAIL'))
    .filter((line) => !line.startsWith('--- FAIL:'))
    .filter((line) => !line.startsWith('step='));
  const first = lines[0] || '';
  return first.replace(/^Error:\s*/, '');
}

function safeSegment(value) {
  return String(value || '')
    .trim()
    .toLowerCase()
    .replace(/[^a-z0-9]+/g, '-')
    .replace(/^-+|-+$/g, '') || 'unknown';
}

function buildFailureEnvelope(vars, model, promptProfile, detail) {
  return {
    case_id: buildCaseID(vars, model, promptProfile),
    scenario: String(vars.scenario || '').trim(),
    label: String(vars.label || '').trim(),
    model: String(model || '').trim(),
    prompt_profile: promptProfile,
    run_status: 'failed',
    metric_status: 'invalid',
    failure_kind: 'harness_error',
    failure_summary: detail || 'live eval failed',
    failure_reason: detail || 'live eval failed',
  };
}

function parseJSONMaybe(value) {
  if (!value) {
    return null;
  }
  try {
    return JSON.parse(String(value));
  } catch {
    return null;
  }
}

function writeCapturedCase(vars, model, promptProfile, payload, startedAt) {
  const captureDir = String(process.env.PROMPTFOO_CAPTURE_DIR || '').trim();
  if (!captureDir) {
    return;
  }

  const scenario = String(vars.scenario || '').trim();
  const repeatIndex = Number.parseInt(String(vars.repeat_index || '1'), 10) || 1;
  const fileName = [
    safeSegment(scenario),
    safeSegment(model),
    safeSegment(promptProfile),
    `run-${repeatIndex}`,
  ].join('__');

  const outPath = path.join(captureDir, `${fileName}.json`);
  const record = {
    scenario,
    label: String(vars.label || '').trim(),
    model: String(model || '').trim(),
    prompt_profile: promptProfile,
    repeat_index: repeatIndex,
    recorded_at: new Date().toISOString(),
    latency_ms: Date.now() - startedAt,
    vars,
    output: payload,
  };

  fs.mkdirSync(captureDir, { recursive: true });
  fs.writeFileSync(outPath, `${JSON.stringify(record, null, 2)}\n`);
}

function resolveInstructionsRoot(promptProfile, config) {
  const roots = (config && config.instructionsRoots) || {};
  const selected = roots[promptProfile];
  if (!selected) {
    return '';
  }
  return path.resolve(__dirname, '..', selected);
}

function buildCaseID(vars, model, promptProfile) {
  const runID = String(process.env.PROMPTFOO_RUN_ID || '').trim() || 'manual';
  const repeatIndex = Number.parseInt(String(vars.repeat_index || '1'), 10) || 1;
  return [
    String(vars.scenario || '').trim(),
    String(model || '').trim(),
    String(promptProfile || '').trim(),
    `run-${repeatIndex}`,
    runID,
  ]
    .map(safeSegment)
    .join('__');
}

module.exports = class GMScenarioProvider {
  constructor(options = {}) {
    this.providerId = options.id || 'gm-scenario-provider';
    this.config = options.config || {};
  }

  id() {
    return this.providerId;
  }

  async callApi(prompt, context) {
    const promptMeta = (context && context.prompt) || {};
    const promptConfig = (promptMeta && promptMeta.config) || {};
    const promptProfile = String(promptConfig.profile || promptMeta.label || prompt || 'baseline').trim() || 'baseline';
    const vars = (context && context.vars) || {};
    const repoRoot = path.resolve(__dirname, '..', '..', '..');
    const startedAt = Date.now();

    const args = [
      'run',
      './cmd/aieval',
      '--scenario',
      String(vars.scenario || '').trim(),
      '--model',
      String(this.config.model || '').trim(),
      '--prompt-profile',
      promptProfile,
      '--case-id',
      buildCaseID(vars, this.config.model, promptProfile),
    ];

    const runID = String(process.env.PROMPTFOO_RUN_ID || '').trim();
    if (runID) {
      args.push('--run-id', runID);
    }

    const reasoningEffort = String(this.config.reasoningEffort || '').trim();
    if (reasoningEffort) {
      args.push('--reasoning-effort', reasoningEffort);
    }

    const instructionsRoot = resolveInstructionsRoot(promptProfile, this.config);
    if (instructionsRoot) {
      args.push('--instructions-root', instructionsRoot);
    }

    const result = spawnSync('go', args, {
      cwd: repoRoot,
      env: process.env,
      encoding: 'utf8',
      maxBuffer: 10 * 1024 * 1024,
    });

    const parsedOutput = parseJSONMaybe(String(result.stdout || '').trim());
    if (result.status !== 0) {
      const detail = compactFailureText(result.stderr) || compactFailureText(result.stdout);
      const payload = parsedOutput || buildFailureEnvelope(vars, this.config.model, promptProfile, detail);
      writeCapturedCase(vars, this.config.model, promptProfile, payload, startedAt);
      return {
        output: JSON.stringify(payload),
      };
    }

    writeCapturedCase(vars, this.config.model, promptProfile, parsedOutput || {}, startedAt);
    return {
      output: String(result.stdout || '').trim(),
    };
  }
};
