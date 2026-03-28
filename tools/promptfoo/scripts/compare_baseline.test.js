const test = require('node:test');
const assert = require('node:assert/strict');

const { compareResults, renderMarkdown, classify } = require('./compare_baseline');

const baselineEntry = (scenario, model, profile, pass, score, failureClass) => ({
  scenario,
  model,
  prompt_profile: profile,
  pass,
  score,
  failure_class: failureClass,
  dominant_reason: pass ? 'gm contract satisfied' : failureClass,
  recorded_at: '2026-03-27T00:00:00Z',
});

const resultRow = (scenario, model, profile, success, score) => ({
  success,
  score,
  provider: { label: model },
  prompt: { label: profile },
  vars: { label: scenario, scenario: `ai_gm_${scenario}` },
  gradingResult: { reason: success ? 'gm contract satisfied' : 'some failure' },
  response: {
    output: JSON.stringify({
      label: scenario,
      prompt_profile: profile,
      metric_status: success ? 'pass' : 'fail',
      artifacts: {},
    }),
  },
});

test('classify detects regression when pass becomes fail', () => {
  const entry = baselineEntry('Bootstrap', 'gpt-5.4-mini', 'baseline', true, 1.0, 'pass');
  const group = { passed: 0, validRuns: 1 };
  assert.equal(classify(entry, group), 'regression');
});

test('classify detects improvement when fail becomes pass', () => {
  const entry = baselineEntry('Hope', 'gpt-5.4-mini', 'baseline', false, 0.5, 'resource_accounting');
  const group = { passed: 1, validRuns: 1 };
  assert.equal(classify(entry, group), 'improvement');
});

test('classify detects unchanged when score is same', () => {
  const entry = baselineEntry('Bootstrap', 'gpt-5.4-mini', 'baseline', true, 1.0, 'pass');
  const group = { passed: 1, validRuns: 1 };
  assert.equal(classify(entry, group), 'unchanged');
});

test('classify detects new when no baseline', () => {
  const group = { passed: 1, validRuns: 1 };
  assert.equal(classify(null, group), 'new');
});

test('classify detects score regression without pass/fail change', () => {
  const entry = baselineEntry('Hope', 'gpt-5.4-mini', 'baseline', false, 0.92, 'resource_accounting');
  const group = { passed: 0, validRuns: 1 };
  assert.equal(classify(entry, group), 'regression');
});

test('compareResults groups results against baseline', () => {
  const baseline = {
    entries: [
      baselineEntry('Bootstrap', 'gpt-5.4-mini', 'baseline', true, 1.0, 'pass'),
      baselineEntry('HopeExperience', 'gpt-5.4-mini', 'baseline', false, 0.92, 'resource_accounting'),
    ],
  };
  const results = [
    resultRow('Bootstrap', 'gpt-5.4-mini', 'baseline', true, 1),
    resultRow('HopeExperience', 'gpt-5.4-mini', 'baseline', true, 1),
  ];
  const classified = compareResults(baseline, results);
  assert.equal(classified.unchanged.length, 1);
  assert.equal(classified.improvement.length, 1);
  assert.equal(classified.regression.length, 0);
  assert.equal(classified.new.length, 0);
});

test('renderMarkdown produces valid markdown', () => {
  const classified = {
    regression: [{ group: { scenario: 'A', model: 'm', promptProfile: 'p', passed: 0, validRuns: 1, dominantFailure: 'fail' }, baseline: { score: 1.0, pass: true }, currentScore: 0 }],
    improvement: [],
    unchanged: [{ group: { scenario: 'B', model: 'm', promptProfile: 'p', passed: 1, validRuns: 1, dominantFailure: 'pass' }, baseline: { score: 1.0, pass: true }, currentScore: 1 }],
    new: [],
  };
  const md = renderMarkdown(classified);
  assert.match(md, /Regression Report/);
  assert.match(md, /1 regression/);
  assert.match(md, /1 unchanged/);
  assert.match(md, /0 new/);
});
