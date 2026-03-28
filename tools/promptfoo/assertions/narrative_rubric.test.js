const test = require('node:test');
const assert = require('node:assert/strict');

const { extractNarration, parseGradeResponse } = require('./narrative_rubric');

test('extractNarration includes output text, prompt text, and beat types', () => {
  const narration = extractNarration({
    output_text: 'The harbor lights flicker as you approach.',
    interaction: {
      prompt_text: 'What does Theron do next?',
      current_beat_types: ['fiction', 'prompt'],
    },
  });
  assert.match(narration, /harbor lights/);
  assert.match(narration, /Theron/);
  assert.match(narration, /fiction, prompt/);
});

test('extractNarration handles missing fields gracefully', () => {
  const narration = extractNarration({});
  assert.equal(narration, '');
});

test('extractNarration handles missing interaction', () => {
  const narration = extractNarration({ output_text: 'Some text' });
  assert.equal(narration, 'Some text');
});

test('parseGradeResponse parses clean JSON', () => {
  const grades = parseGradeResponse(
    '{"narrator_voice": 4, "tone_atmosphere": 5, "prompt_quality": 3, "beat_economy": 4, "explanation": "Good"}',
  );
  assert.equal(grades.narrator_voice, 4);
  assert.equal(grades.tone_atmosphere, 5);
  assert.equal(grades.explanation, 'Good');
});

test('parseGradeResponse strips markdown fencing', () => {
  const grades = parseGradeResponse(
    '```json\n{"narrator_voice": 3, "tone_atmosphere": 4, "prompt_quality": 5, "beat_economy": 3, "explanation": "OK"}\n```',
  );
  assert.equal(grades.narrator_voice, 3);
  assert.equal(grades.prompt_quality, 5);
});

test('narrativeRubric skips when no API key', async () => {
  const saved = process.env.INTEGRATION_OPENAI_API_KEY;
  const savedAlt = process.env.OPENAI_API_KEY;
  delete process.env.INTEGRATION_OPENAI_API_KEY;
  delete process.env.OPENAI_API_KEY;
  try {
    const narrativeRubric = require('./narrative_rubric');
    const result = await narrativeRubric({ output_text: 'test' }, {});
    assert.equal(result.pass, true);
    assert.match(result.reason, /skipped/);
  } finally {
    if (saved !== undefined) {
      process.env.INTEGRATION_OPENAI_API_KEY = saved;
    }
    if (savedAlt !== undefined) {
      process.env.OPENAI_API_KEY = savedAlt;
    }
  }
});

test('narrativeRubric skips on failed run status when API key is set', async () => {
  // Temporarily set a fake API key so the rubric reaches the run_status check.
  // The function will return early before making any HTTP call.
  const saved = process.env.INTEGRATION_OPENAI_API_KEY;
  process.env.INTEGRATION_OPENAI_API_KEY = 'test-fake-key';
  try {
    // Re-require to pick up the env change at call time (env is read per-call).
    const narrativeRubric = require('./narrative_rubric');
    const result = await narrativeRubric({ run_status: 'failed' }, {});
    assert.equal(result.score, 0);
    assert.match(result.reason, /run failed/);
  } finally {
    if (saved !== undefined) {
      process.env.INTEGRATION_OPENAI_API_KEY = saved;
    } else {
      delete process.env.INTEGRATION_OPENAI_API_KEY;
    }
  }
});
