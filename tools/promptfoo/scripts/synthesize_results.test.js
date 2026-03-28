const test = require('node:test');
const assert = require('node:assert/strict');

const { captureKey, synthesizeResults } = require('./synthesize_results');

function gmContract(output, context) {
  const vars = context.vars || {};
  if (String(output.run_status || '').toLowerCase() === 'failed') {
    return {
      pass: false,
      score: 0,
      reason: output.failure_summary || `${output.failure_kind}: ${output.failure_reason}`,
    };
  }
  if (typeof vars.expected_hope === 'number' && output.character_state && output.character_state.hope !== vars.expected_hope) {
    return {
      pass: false,
      score: 0,
      reason: `hope=${output.character_state.hope} expected ${vars.expected_hope}`,
    };
  }
  return {
    pass: true,
    score: 1,
    reason: 'gm contract satisfied',
  };
}

test('synthesizeResults classifies harness errors and assertion failures', () => {
  const config = {
    description: 'test eval',
    prompts: [
      {
        raw: 'Baseline GM instruction profile using the default repo instruction bundle.',
        label: 'baseline',
        config: {
          profile: 'baseline',
          promptSummary: 'Baseline GM instruction profile using the default repo instruction bundle.',
        },
      },
    ],
    providers: [{ id: 'file://providers/gm_scenario_provider.js', label: 'gpt-5.4-mini' }],
    tests: [
      {
        description: 'HopeExperience',
        vars: {
          scenario: 'ai_gm_campaign_context_hope_experience',
          label: 'HopeExperience',
          repeat_index: 1,
          expected_hope: 1,
        },
        assert: [],
      },
      {
        description: 'NarratorAuthority',
        vars: {
          scenario: 'ai_gm_campaign_context_narrator_authority',
          label: 'NarratorAuthority',
          repeat_index: 1,
        },
        assert: [],
      },
    ],
  };

  const captures = new Map([
    [
      captureKey({
        scenario: 'ai_gm_campaign_context_hope_experience',
        model: 'gpt-5.4-mini',
        promptProfile: 'baseline',
        repeatIndex: 1,
      }),
      {
        scenario: 'ai_gm_campaign_context_hope_experience',
        model: 'gpt-5.4-mini',
        prompt_profile: 'baseline',
        repeat_index: 1,
        latency_ms: 42,
        __path: '/tmp/hope.json',
        output: {
          label: 'HopeExperience',
          prompt_profile: 'baseline',
          prompt_context: {
            profile: 'baseline',
            summary: 'Baseline GM instruction profile using the default repo instruction bundle.',
          },
          character_state: { hope: 2 },
        },
      },
    ],
    [
      captureKey({
        scenario: 'ai_gm_campaign_context_narrator_authority',
        model: 'gpt-5.4-mini',
        promptProfile: 'baseline',
        repeatIndex: 1,
      }),
      {
        scenario: 'ai_gm_campaign_context_narrator_authority',
        model: 'gpt-5.4-mini',
        prompt_profile: 'baseline',
        repeat_index: 1,
        latency_ms: 17,
        __path: '/tmp/narrator.json',
        output: {
          label: 'NarratorAuthority',
          prompt_profile: 'baseline',
          run_status: 'failed',
          metric_status: 'invalid',
          failure_kind: 'harness_error',
          failure_summary: 'missing captured provider output',
          failure_reason: 'missing captured provider output',
        },
      },
    ],
  ]);

  const synthesized = synthesizeResults(config, captures, gmContract, 'test-run');

  assert.equal(synthesized.evalId, 'eval-test-run');
  assert.equal(synthesized.results.prompts.length, 1);
  assert.equal(synthesized.results.results.length, 2);

  const hopeResult = synthesized.results.results[0];
  assert.equal(hopeResult.success, false);
  assert.equal(hopeResult.failureReason, 1);
  assert.equal(hopeResult.error, 'hope=2 expected 1');
  assert.equal(hopeResult.prompt.raw, 'Baseline GM instruction profile using the default repo instruction bundle.');
  assert.equal(hopeResult.prompt.config.promptContext.profile, 'baseline');

  const narratorResult = synthesized.results.results[1];
  assert.equal(narratorResult.success, false);
  assert.equal(narratorResult.failureReason, 2);
  assert.equal(narratorResult.gradingResult, null);
  assert.equal(narratorResult.error, 'missing captured provider output');
});
