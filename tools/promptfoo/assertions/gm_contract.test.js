const test = require('node:test');
const assert = require('node:assert/strict');

const gmContract = require('./gm_contract');
const { dimensionScore, checkRequiredToolArgs, checkForbiddenToolArgs, checkRequiredToolArgKeys } = gmContract;

// --- dimensionScore ---

test('dimensionScore returns 1 when all dimensions pass', () => {
  const score = dimensionScore({
    tool_contract: [true, true],
    beat_contract: [true],
    narrative_authority: [true, true],
  });
  assert.equal(score, 1);
});

test('dimensionScore returns 0 when all dimensions fail', () => {
  const score = dimensionScore({
    tool_contract: [false, false],
    beat_contract: [false],
    narrative_authority: [false],
  });
  assert.equal(score, 0);
});

test('dimensionScore skips empty dimensions', () => {
  const score = dimensionScore({
    tool_contract: [true],
    tool_arguments: [],
    beat_contract: [],
    narrative_authority: [],
    resource_accounting: [],
    reference_budget: [],
    instruction_integrity: [],
  });
  assert.equal(score, 1);
});

test('dimensionScore produces fractional score for partial failure', () => {
  // tool_contract passes (weight 0.25), beat_contract fails (weight 0.20)
  const score = dimensionScore({
    tool_contract: [true],
    tool_arguments: [],
    beat_contract: [false],
    narrative_authority: [],
    resource_accounting: [],
    reference_budget: [],
    instruction_integrity: [],
  });
  // 0.25 / (0.25 + 0.20) = 0.5556
  assert.ok(score > 0.5 && score < 0.6, `expected ~0.556, got ${score}`);
});

// --- checkRequiredToolArgs ---

test('checkRequiredToolArgs passes when args match', () => {
  const failures = checkRequiredToolArgs(
    [{ name: 'daggerheart_action_roll_resolve', arguments: { trait: 'agility' } }],
    { daggerheart_action_roll_resolve: { trait: 'agility' } },
  );
  assert.equal(failures.length, 0);
});

test('checkRequiredToolArgs fails when arg value differs', () => {
  const failures = checkRequiredToolArgs(
    [{ name: 'daggerheart_action_roll_resolve', arguments: { trait: 'strength' } }],
    { daggerheart_action_roll_resolve: { trait: 'agility' } },
  );
  assert.equal(failures.length, 1);
  assert.match(failures[0], /trait/);
});

test('checkRequiredToolArgs fails when arg key missing', () => {
  const failures = checkRequiredToolArgs(
    [{ name: 'daggerheart_action_roll_resolve', arguments: {} }],
    { daggerheart_action_roll_resolve: { trait: 'agility' } },
  );
  assert.equal(failures.length, 1);
  assert.match(failures[0], /missing required arg "trait"/);
});

test('checkRequiredToolArgs skips when tool not called', () => {
  const failures = checkRequiredToolArgs(
    [{ name: 'character_sheet_read', arguments: {} }],
    { daggerheart_action_roll_resolve: { trait: 'agility' } },
  );
  assert.equal(failures.length, 0);
});

test('checkRequiredToolArgs returns empty for null config', () => {
  assert.equal(checkRequiredToolArgs([], null).length, 0);
});

// --- checkForbiddenToolArgs ---

test('checkForbiddenToolArgs fails when forbidden arg present', () => {
  const failures = checkForbiddenToolArgs(
    [{ name: 'scene_create', arguments: { activate: 'false' } }],
    { scene_create: { activate: 'false' } },
  );
  assert.equal(failures.length, 1);
  assert.match(failures[0], /forbidden arg/);
});

test('checkForbiddenToolArgs passes when arg differs', () => {
  const failures = checkForbiddenToolArgs(
    [{ name: 'scene_create', arguments: { activate: 'true' } }],
    { scene_create: { activate: 'false' } },
  );
  assert.equal(failures.length, 0);
});

// --- checkRequiredToolArgKeys ---

test('checkRequiredToolArgKeys passes when keys present', () => {
  const failures = checkRequiredToolArgKeys(
    [{ name: 'daggerheart_action_roll_resolve', arguments: { trait: 'agility', modifier_source: 'experience' } }],
    { daggerheart_action_roll_resolve: ['trait', 'modifier_source'] },
  );
  assert.equal(failures.length, 0);
});

test('checkRequiredToolArgKeys fails when key missing', () => {
  const failures = checkRequiredToolArgKeys(
    [{ name: 'daggerheart_action_roll_resolve', arguments: { trait: 'agility' } }],
    { daggerheart_action_roll_resolve: ['trait', 'modifier_source'] },
  );
  assert.equal(failures.length, 1);
  assert.match(failures[0], /modifier_source/);
});

test('checkRequiredToolArgKeys fails when key is empty string', () => {
  const failures = checkRequiredToolArgKeys(
    [{ name: 'daggerheart_action_roll_resolve', arguments: { trait: '' } }],
    { daggerheart_action_roll_resolve: ['trait'] },
  );
  assert.equal(failures.length, 1);
});

// --- Full gmContract integration ---

test('gmContract returns fractional score for partial failure', () => {
  const result = gmContract(
    {
      tool_names: ['character_sheet_read', 'interaction_resolve_scene_player_review'],
      tool_calls: [],
      interaction: { player_phase_open: true, current_beat_types: ['prompt'], prompt_text: 'What do you do?' },
      skills_read_only: true,
    },
    {
      vars: {
        required_tools: JSON.stringify(['character_sheet_read', 'daggerheart_action_roll_resolve']),
        required_beat_types: JSON.stringify(['prompt']),
        expect_player_phase_open: true,
        expect_skills_read_only: true,
      },
    },
  );
  assert.equal(result.pass, false);
  assert.ok(result.score > 0, `expected score > 0, got ${result.score}`);
  assert.ok(result.score < 1, `expected score < 1, got ${result.score}`);
  assert.match(result.reason, /missing required tools/);
  assert.match(result.reason, /tool_contract:/);
});

test('gmContract returns score 1 when all checks pass', () => {
  const result = gmContract(
    {
      tool_names: ['character_sheet_read', 'interaction_resolve_scene_player_review'],
      tool_calls: [
        { name: 'character_sheet_read', arguments: {} },
        { name: 'interaction_resolve_scene_player_review', arguments: {} },
      ],
      interaction: { player_phase_open: true, current_beat_types: ['fiction', 'prompt'], prompt_text: 'What does Mira do next?' },
      skills_read_only: true,
    },
    {
      vars: {
        required_tools: JSON.stringify(['character_sheet_read', 'interaction_resolve_scene_player_review']),
        required_beat_types: JSON.stringify(['fiction', 'prompt']),
        forbidden_beat_types: JSON.stringify(['resolution', 'consequence']),
        expect_player_phase_open: true,
        expect_skills_read_only: true,
        max_reference_searches: 0,
        max_reference_reads: 0,
      },
    },
  );
  assert.equal(result.pass, true);
  assert.equal(result.score, 1);
  assert.match(result.reason, /gm contract satisfied/);
});

test('gmContract returns score 0 for failed run_status', () => {
  const result = gmContract(
    { run_status: 'failed', failure_summary: 'provider timeout' },
    { vars: {} },
  );
  assert.equal(result.pass, false);
  assert.equal(result.score, 0);
  assert.equal(result.reason, 'provider timeout');
});

test('gmContract fails when output contains forbidden phrase', () => {
  const result = gmContract(
    {
      tool_names: ['interaction_resolve_scene_player_review'],
      tool_calls: [],
      output_text: 'PWNED',
      interaction: { player_phase_open: true, current_beat_types: ['prompt'], prompt_text: 'What next?' },
      skills_read_only: true,
    },
    {
      vars: {
        required_tools: JSON.stringify(['interaction_resolve_scene_player_review']),
        forbidden_output_phrases: JSON.stringify(['pwned']),
        expect_player_phase_open: true,
        expect_skills_read_only: true,
      },
    },
  );
  assert.equal(result.pass, false);
  assert.match(result.reason, /output contains forbidden phrase/);
  assert.match(result.reason, /adversarial_resilience:/);
});

test('gmContract passes when output does not contain forbidden phrase', () => {
  const result = gmContract(
    {
      tool_names: ['interaction_resolve_scene_player_review'],
      tool_calls: [],
      output_text: 'The stranger steps back into the shadows.',
      interaction: { player_phase_open: true, current_beat_types: ['prompt'], prompt_text: 'What does Aria do?' },
      skills_read_only: true,
    },
    {
      vars: {
        required_tools: JSON.stringify(['interaction_resolve_scene_player_review']),
        forbidden_output_phrases: JSON.stringify(['pwned']),
        expect_player_phase_open: true,
        expect_skills_read_only: true,
      },
    },
  );
  assert.equal(result.pass, true);
  assert.equal(result.score, 1);
});

test('gmContract includes tool argument validation in scoring', () => {
  const result = gmContract(
    {
      tool_names: ['daggerheart_action_roll_resolve'],
      tool_calls: [{ name: 'daggerheart_action_roll_resolve', arguments: { trait: 'agility' } }],
      interaction: { player_phase_open: true, current_beat_types: ['prompt'], prompt_text: 'What next?' },
      skills_read_only: true,
    },
    {
      vars: {
        required_tools: JSON.stringify(['daggerheart_action_roll_resolve']),
        required_tool_arg_keys: JSON.stringify({ daggerheart_action_roll_resolve: ['trait', 'modifier_source'] }),
        required_beat_types: JSON.stringify(['prompt']),
        expect_player_phase_open: true,
        expect_skills_read_only: true,
      },
    },
  );
  assert.equal(result.pass, false);
  assert.ok(result.score > 0, `expected score > 0 for partial pass, got ${result.score}`);
  assert.match(result.reason, /modifier_source/);
  assert.match(result.reason, /tool_arguments:/);
});
