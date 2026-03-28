#!/usr/bin/env node

const crypto = require('node:crypto');
const fs = require('node:fs');
const path = require('node:path');

function parseArgs(argv) {
  const out = {};
  for (let i = 0; i < argv.length; i += 1) {
    const arg = argv[i];
    if (arg === '--config') {
      out.config = argv[i + 1];
      i += 1;
      continue;
    }
    if (arg === '--capture-dir') {
      out.captureDir = argv[i + 1];
      i += 1;
      continue;
    }
    if (arg === '--output') {
      out.output = argv[i + 1];
      i += 1;
      continue;
    }
    if (arg === '--run-id') {
      out.runId = argv[i + 1];
      i += 1;
    }
  }
  if (!out.config || !out.captureDir || !out.output) {
    throw new Error('usage: synthesize_results.js --config <promptfooconfig.js> --capture-dir <dir> --output <results.json> [--run-id <id>]');
  }
  return out;
}

function readJSON(filePath) {
  return JSON.parse(fs.readFileSync(filePath, 'utf8'));
}

function listCaptureFiles(rootDir) {
  if (!fs.existsSync(rootDir)) {
    return [];
  }
  return fs
    .readdirSync(rootDir)
    .filter((name) => name.endsWith('.json'))
    .sort()
    .map((name) => path.join(rootDir, name));
}

function defaultTokens() {
  return {
    total: 0,
    prompt: 0,
    completion: 0,
    cached: 0,
    numRequests: 0,
  };
}

function defaultPromptMetrics() {
  return {
    score: 0,
    testPassCount: 0,
    testFailCount: 0,
    testErrorCount: 0,
    assertPassCount: 0,
    assertFailCount: 0,
    totalLatencyMs: 0,
    tokenUsage: {
      prompt: 0,
      completion: 0,
      cached: 0,
      total: 0,
      numRequests: 0,
      completionDetails: {
        reasoning: 0,
        acceptedPrediction: 0,
        rejectedPrediction: 0,
      },
      assertions: {
        total: 0,
        prompt: 0,
        completion: 0,
        cached: 0,
        numRequests: 0,
        completionDetails: {
          reasoning: 0,
          acceptedPrediction: 0,
          rejectedPrediction: 0,
        },
      },
    },
    namedScores: {},
    namedScoresCount: {},
    cost: 0,
  };
}

function hashPrompt(prompt) {
  return crypto.createHash('sha256').update(JSON.stringify(prompt)).digest('hex');
}

function normalizePrompt(prompt) {
  if (typeof prompt === 'string') {
    return {
      raw: prompt,
      label: prompt,
      config: {},
    };
  }
  return {
    raw: prompt.raw || prompt.label || prompt.id || '',
    label: prompt.label || prompt.display || prompt.raw || prompt.id || '',
    config: prompt.config || {},
  };
}

function normalizeProvider(provider) {
  if (typeof provider === 'string') {
    return {
      id: provider,
      label: provider,
    };
  }
  return {
    id: provider.id,
    label: provider.label || provider.id,
    ...(provider.config ? { config: provider.config } : {}),
  };
}

function buildPromptEntries(config) {
  const prompts = (config.prompts || []).map(normalizePrompt);
  const providers = (config.providers || []).map(normalizeProvider);
  const entries = [];
  providers.forEach((provider) => {
    prompts.forEach((prompt) => {
      const completedPrompt = {
        id: hashPrompt(prompt),
        raw: prompt.raw,
        label: prompt.label,
        provider: provider.label,
        metrics: defaultPromptMetrics(),
      };
      entries.push({
        completedPrompt,
        prompt,
        provider,
      });
    });
  });
  return entries;
}

function captureKey({ scenario, model, promptProfile, repeatIndex }) {
  return [scenario, model, promptProfile, repeatIndex].join('|');
}

function buildMissingCapture(vars, providerLabel, promptLabel) {
  return {
    scenario: String(vars.scenario || '').trim(),
    label: String(vars.label || '').trim(),
    model: providerLabel,
    prompt_profile: promptLabel,
    run_status: 'failed',
    metric_status: 'invalid',
    failure_kind: 'harness_error',
    failure_summary: 'missing captured provider output',
    failure_reason: 'missing captured provider output',
  };
}

function normalizeGradeResult(rawResult) {
  return {
    pass: Boolean(rawResult.pass),
    score: Number(rawResult.score || 0),
    reason: String(rawResult.reason || '').trim(),
    namedScores: {},
    tokensUsed: defaultTokens(),
    componentResults: [
      {
        pass: Boolean(rawResult.pass),
        score: Number(rawResult.score || 0),
        reason: String(rawResult.reason || '').trim(),
      },
    ],
  };
}

function failureReasonCode(output, grade) {
  if (String(output.run_status || '').trim().toLowerCase() === 'failed') {
    return 2;
  }
  if (!grade.pass) {
    return 1;
  }
  return 0;
}

function buildResultRecord(testCase, vars, promptEntry, capture, promptIdx, testIdx, gmContract) {
  const output = capture ? capture.output : buildMissingCapture(vars, promptEntry.provider.label, promptEntry.prompt.label);
  const grade = gmContract(output, { vars });
  const normalizedGrade = normalizeGradeResult(grade);
  const failureReason = failureReasonCode(output, normalizedGrade);
  const isError = failureReason === 2;
  const latencyMs = Number(capture && capture.latency_ms) || 0;

  const metrics = promptEntry.completedPrompt.metrics;
  metrics.score += normalizedGrade.score;
  metrics.totalLatencyMs += latencyMs;
  metrics.cost += 0;
  if (normalizedGrade.pass) {
    metrics.testPassCount += 1;
    metrics.assertPassCount += 1;
    metrics.tokenUsage.numRequests += 1;
  } else if (isError) {
    metrics.testErrorCount += 1;
  } else {
    metrics.testFailCount += 1;
    metrics.assertFailCount += 1;
    metrics.tokenUsage.numRequests += 1;
    metrics.tokenUsage.assertions.numRequests += 1;
  }

  return {
    cost: 0,
    error: normalizedGrade.pass ? null : normalizedGrade.reason,
    gradingResult: isError ? null : normalizedGrade,
    latencyMs,
    namedScores: {},
    prompt: {
      raw: promptEntry.prompt.raw,
      label: promptEntry.prompt.label,
      config: {
        ...(promptEntry.prompt.config || {}),
        ...(output.prompt_context ? { promptContext: output.prompt_context } : {}),
      },
    },
    promptId: promptEntry.completedPrompt.id,
    promptIdx,
    provider: {
      id: promptEntry.provider.id,
      label: promptEntry.provider.label,
      ...(promptEntry.provider.config ? { config: promptEntry.provider.config } : {}),
    },
    response: {
      output: JSON.stringify(output),
    },
    score: normalizedGrade.score,
    success: normalizedGrade.pass,
    testCase,
    testIdx,
    vars,
    metadata: {
      capturePath: capture ? capture.__path : null,
      importedFrom: 'provider-capture',
      ...(output.prompt_context ? { promptContext: output.prompt_context } : {}),
    },
    failureReason,
  };
}

function synthesizeResults(config, capturesByKey, gmContract, runId) {
  const promptEntries = buildPromptEntries(config);
  const results = [];

  config.tests.forEach((testCase, testIdx) => {
    const vars = testCase.vars || {};
    const repeatIndex = Number.parseInt(String(vars.repeat_index || '1'), 10) || 1;
    promptEntries.forEach((promptEntry, promptIdx) => {
      const key = captureKey({
        scenario: String(vars.scenario || '').trim(),
        model: promptEntry.provider.label,
        promptProfile: promptEntry.prompt.label,
        repeatIndex,
      });
      const capture = capturesByKey.get(key) || null;
      results.push(buildResultRecord(testCase, vars, promptEntry, capture, promptIdx, testIdx, gmContract));
    });
  });

  return {
    evalId: `eval-${String(runId || Date.now())}`,
    metadata: {
      evaluationCreatedAt: new Date().toISOString(),
      author: 'repo-fallback',
    },
    description: config.description,
    config,
    results: {
      version: 3,
      timestamp: new Date().toISOString(),
      prompts: promptEntries.map((entry) => entry.completedPrompt),
      results,
    },
  };
}

function loadCaptures(captureDir) {
  const captures = new Map();
  for (const filePath of listCaptureFiles(captureDir)) {
    const capture = readJSON(filePath);
    capture.__path = filePath;
    captures.set(
      captureKey({
        scenario: String(capture.scenario || '').trim(),
        model: String(capture.model || '').trim(),
        promptProfile: String(capture.prompt_profile || '').trim(),
        repeatIndex: Number.parseInt(String(capture.repeat_index || '1'), 10) || 1,
      }),
      capture,
    );
  }
  return captures;
}

function main() {
  const args = parseArgs(process.argv.slice(2));
  const configPath = path.resolve(args.config);
  const config = require(configPath);
  const gmContract = require(path.resolve(path.dirname(configPath), 'assertions', 'gm_contract.js'));
  const captures = loadCaptures(path.resolve(args.captureDir));
  const synthesized = synthesizeResults(config, captures, gmContract, args.runId);

  fs.mkdirSync(path.dirname(args.output), { recursive: true });
  fs.writeFileSync(args.output, `${JSON.stringify(synthesized, null, 2)}\n`);
}

if (require.main === module) {
  main();
}

module.exports = {
  buildPromptEntries,
  buildResultRecord,
  captureKey,
  normalizePrompt,
  normalizeProvider,
  synthesizeResults,
};
