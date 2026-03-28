const test = require('node:test');
const assert = require('node:assert/strict');

const { updateBaseline } = require('./update_baseline');

const resultRow = (scenario, model, profile, success, score, metricStatus) => ({
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
      metric_status: metricStatus || (success ? 'pass' : 'fail'),
      artifacts: {},
    }),
  },
});

test('updateBaseline creates entries from results', () => {
  const { baseline, updated, skipped } = updateBaseline(
    { entries: [] },
    [
      resultRow('Bootstrap', 'gpt-5.4-mini', 'baseline', true, 1, 'pass'),
      resultRow('HopeExperience', 'gpt-5.4-mini', 'baseline', false, 0.92, 'fail'),
    ],
  );
  assert.equal(updated, 2);
  assert.equal(skipped, 0);
  assert.equal(baseline.entries.length, 2);

  const bootstrap = baseline.entries.find((e) => e.scenario === 'Bootstrap');
  assert.equal(bootstrap.pass, true);
  assert.equal(bootstrap.model, 'gpt-5.4-mini');
});

test('updateBaseline skips invalid runs', () => {
  const { baseline, updated, skipped } = updateBaseline(
    { entries: [{ scenario: 'Bootstrap', model: 'gpt-5.4-mini', prompt_profile: 'baseline', pass: true, score: 1.0, failure_class: 'pass', dominant_reason: 'ok', recorded_at: '2026-01-01' }] },
    [
      resultRow('Bootstrap', 'gpt-5.4-mini', 'baseline', false, 0, 'invalid'),
    ],
  );
  assert.equal(skipped, 1);
  assert.equal(updated, 0);
  // Original entry preserved
  assert.equal(baseline.entries.length, 1);
  assert.equal(baseline.entries[0].pass, true);
});

test('updateBaseline merges new entries with existing', () => {
  const { baseline } = updateBaseline(
    { entries: [{ scenario: 'Bootstrap', model: 'gpt-5.4-mini', prompt_profile: 'baseline', pass: true, score: 1.0, failure_class: 'pass', dominant_reason: 'ok', recorded_at: '2026-01-01' }] },
    [
      resultRow('HopeExperience', 'gpt-5.4-mini', 'baseline', false, 0.92, 'fail'),
    ],
  );
  assert.equal(baseline.entries.length, 2);
  assert.ok(baseline.entries.find((e) => e.scenario === 'Bootstrap'));
  assert.ok(baseline.entries.find((e) => e.scenario === 'HopeExperience'));
});

test('updateBaseline sorts entries by scenario, model, profile', () => {
  const { baseline } = updateBaseline(
    { entries: [] },
    [
      resultRow('Z-Scenario', 'gpt-5.4-mini', 'baseline', true, 1, 'pass'),
      resultRow('A-Scenario', 'gpt-5.4-mini', 'baseline', true, 1, 'pass'),
    ],
  );
  assert.equal(baseline.entries[0].scenario, 'A-Scenario');
  assert.equal(baseline.entries[1].scenario, 'Z-Scenario');
});
