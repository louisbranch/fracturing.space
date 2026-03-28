const test = require('node:test');
const assert = require('node:assert/strict');

const { groupRows, renderMarkdown, rowFor } = require('./summarize_results');

test('rowFor classifies resource accounting failures from grading reasons', () => {
  const row = rowFor({
    success: false,
    provider: { label: 'gpt-5.4' },
    prompt: { label: 'baseline' },
    vars: { label: 'HopeExperience' },
    gradingResult: { reason: 'hope=3 expected 1' },
    response: {
      output: JSON.stringify({
        label: 'HopeExperience',
        prompt_profile: 'baseline',
        artifacts: { summary: '/tmp/hope.summary.json' },
      }),
    },
  });

  assert.equal(row.failureClass, 'resource_accounting');
  assert.equal(row.failureReason, 'hope=3 expected 1');
});

test('rowFor preserves structured invalid failure kinds', () => {
  const row = rowFor({
    success: false,
    provider: { label: 'gpt-5.4' },
    prompt: { label: 'baseline' },
    vars: { label: 'NarratorAuthority' },
    response: {
      output: JSON.stringify({
        label: 'NarratorAuthority',
        prompt_profile: 'baseline',
        run_status: 'failed',
        metric_status: 'invalid',
        failure_kind: 'provider_error',
        failure_summary: 'provider returned neither tool calls nor final output',
        artifacts: { diagnostics: '/tmp/narrator.diagnostics.json' },
      }),
    },
  });

  assert.equal(row.metricStatus, 'invalid');
  assert.equal(row.failureClass, 'provider_error');
  assert.equal(row.failureReason, 'provider returned neither tool calls nor final output');
});

test('groupRows summarizes pass rate and dominant failure', () => {
  const groups = groupRows([
    {
      scenario: 'HopeExperience',
      model: 'gpt-5.4-mini',
      promptProfile: 'baseline',
      success: false,
      metricStatus: 'fail',
      failureClass: 'missing_authoritative_roll',
      failureReason: 'missing required tool "daggerheart_action_roll_resolve"',
      artifacts: { harness_log: '/tmp/failure.log' },
    },
    {
      scenario: 'HopeExperience',
      model: 'gpt-5.4-mini',
      promptProfile: 'baseline',
      success: false,
      metricStatus: 'fail',
      failureClass: 'missing_authoritative_roll',
      failureReason: 'missing required tool "daggerheart_action_roll_resolve"',
      artifacts: { harness_log: '/tmp/failure.log' },
    },
    {
      scenario: 'HopeExperience',
      model: 'gpt-5.4',
      promptProfile: 'baseline',
      success: true,
      metricStatus: 'pass',
      failureClass: 'harness_error',
      failureReason: '',
      artifacts: { summary: '/tmp/pass.summary.json' },
    },
    {
      scenario: 'HopeExperience',
      model: 'gpt-5.4',
      promptProfile: 'baseline',
      success: false,
      metricStatus: 'invalid',
      failureClass: 'provider_error',
      failureReason: 'provider returned neither tool calls nor final output',
      artifacts: { harness_log: '/tmp/invalid.log' },
    },
  ]);

  assert.equal(groups.length, 2);
  const failed = groups.find((group) => group.model === 'gpt-5.4-mini');
  assert.equal(failed.qualityPassRate, '0/2');
  assert.equal(failed.dominantFailure, 'missing_authoritative_roll');

  const mixed = groups.find((group) => group.model === 'gpt-5.4');
  assert.equal(mixed.qualityPassRate, '1/1');
  assert.equal(mixed.invalidRuns, 1);

  const markdown = renderMarkdown(groups, '.tmp/promptfoo/results.json');
  assert.match(markdown, /Promptfoo GM Eval Scorecard/);
  assert.match(markdown, /missing_authoritative_roll/);
  assert.match(markdown, /Invalid Runs/);
  assert.match(markdown, /Diagnostics/);
});
