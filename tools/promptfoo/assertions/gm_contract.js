function parseOutput(raw) {
  if (raw && typeof raw === 'object') {
    return raw;
  }
  return JSON.parse(String(raw || '{}'));
}

function parseMaybeJSON(value, fallback) {
  if (value === undefined || value === null || value === '') {
    return fallback;
  }
  if (typeof value !== 'string') {
    return value;
  }
  try {
    return JSON.parse(value);
  } catch {
    return fallback;
  }
}

function includesAll(haystack, needles) {
  return (needles || []).every((needle) => haystack.includes(needle));
}

function includesNone(haystack, needles) {
  return (needles || []).every((needle) => !haystack.includes(needle));
}

function hasOrderedSubsequence(actual, expectedOrder) {
  if (!expectedOrder || expectedOrder.length === 0) {
    return true;
  }
  if (!Array.isArray(actual) || actual.length < expectedOrder.length) {
    return false;
  }
  let cursor = 0;
  for (const value of actual) {
    if (value === expectedOrder[cursor]) {
      cursor += 1;
      if (cursor === expectedOrder.length) {
        return true;
      }
    }
  }
  return false;
}

function hasModifierSource(toolCalls, source) {
  if (!source) {
    return true;
  }
  const resolveCall = (toolCalls || []).find((call) => call.name === 'daggerheart_action_roll_resolve');
  if (!resolveCall || !Array.isArray(resolveCall.arguments && resolveCall.arguments.modifiers)) {
    return false;
  }
  return resolveCall.arguments.modifiers.some((modifier) =>
    String((modifier && modifier.source) || '')
      .toLowerCase()
      .includes(String(source).toLowerCase()),
  );
}

// --- Tool argument validation helpers ---

function checkRequiredToolArgs(toolCalls, requiredToolArgs) {
  if (!requiredToolArgs || typeof requiredToolArgs !== 'object') {
    return [];
  }
  const failures = [];
  for (const [toolName, expectedArgs] of Object.entries(requiredToolArgs)) {
    const call = (toolCalls || []).find((c) => c.name === toolName);
    if (!call) {
      continue; // tool presence is checked separately
    }
    const args = call.arguments || {};
    for (const [key, expectedValue] of Object.entries(expectedArgs || {})) {
      const actual = args[key];
      if (actual === undefined || actual === null) {
        failures.push(`${toolName} missing required arg "${key}"`);
      } else if (String(actual).toLowerCase() !== String(expectedValue).toLowerCase()) {
        failures.push(`${toolName} arg "${key}"=${JSON.stringify(actual)}, expected ${JSON.stringify(expectedValue)}`);
      }
    }
  }
  return failures;
}

function checkForbiddenToolArgs(toolCalls, forbiddenToolArgs) {
  if (!forbiddenToolArgs || typeof forbiddenToolArgs !== 'object') {
    return [];
  }
  const failures = [];
  for (const [toolName, forbiddenArgs] of Object.entries(forbiddenToolArgs)) {
    const calls = (toolCalls || []).filter((c) => c.name === toolName);
    for (const call of calls) {
      const args = call.arguments || {};
      for (const [key, forbiddenValue] of Object.entries(forbiddenArgs || {})) {
        if (args[key] !== undefined && String(args[key]).toLowerCase() === String(forbiddenValue).toLowerCase()) {
          failures.push(`${toolName} has forbidden arg "${key}"=${JSON.stringify(args[key])}`);
        }
      }
    }
  }
  return failures;
}

function checkRequiredToolArgKeys(toolCalls, requiredToolArgKeys) {
  if (!requiredToolArgKeys || typeof requiredToolArgKeys !== 'object') {
    return [];
  }
  const failures = [];
  for (const [toolName, requiredKeys] of Object.entries(requiredToolArgKeys)) {
    const call = (toolCalls || []).find((c) => c.name === toolName);
    if (!call) {
      continue; // tool presence is checked separately
    }
    const args = call.arguments || {};
    for (const key of requiredKeys || []) {
      if (args[key] === undefined || args[key] === null || args[key] === '') {
        failures.push(`${toolName} missing required arg key "${key}"`);
      }
    }
  }
  return failures;
}

// --- Hope spend validation (from game-service tool schema) ---

function hasHopeSpend(toolCalls, source, amount) {
  if (!source) {
    return true;
  }
  // Check any daggerheart_action_roll_resolve call (the model may retry after
  // an initial tool error, so the correct hope_spends may be on a later call).
  const resolveCalls = (toolCalls || []).filter((call) => call.name === 'daggerheart_action_roll_resolve');
  return resolveCalls.some((resolveCall) => {
    if (!Array.isArray(resolveCall.arguments && resolveCall.arguments.hope_spends)) {
      return false;
    }
    return resolveCall.arguments.hope_spends.some((spend) => {
      const spendSource = String((spend && spend.source) || '').toLowerCase();
      const spendAmount = Number(spend && spend.amount);
      return (
        spendSource.includes(String(source).toLowerCase()) &&
        (typeof amount !== 'number' || spendAmount === amount)
      );
    });
  });
}

// --- Weighted dimension scoring ---

const DIMENSION_WEIGHTS = {
  tool_contract: 0.20,
  tool_arguments: 0.10,
  beat_contract: 0.15,
  narrative_authority: 0.15,
  resource_accounting: 0.15,
  reference_budget: 0.10,
  instruction_integrity: 0.05,
  adversarial_resilience: 0.10,
};

function dimensionScore(dimensions) {
  let totalWeight = 0;
  let weightedSum = 0;
  for (const [dimension, checks] of Object.entries(dimensions)) {
    const weight = DIMENSION_WEIGHTS[dimension] || 0;
    if (checks.length === 0) {
      continue; // dimension not applicable to this scenario
    }
    totalWeight += weight;
    const passed = checks.filter(Boolean).length;
    weightedSum += weight * (passed / checks.length);
  }
  if (totalWeight === 0) {
    return 1;
  }
  return weightedSum / totalWeight;
}

function formatDimensionBreakdown(dimensions) {
  const parts = [];
  for (const [dimension, checks] of Object.entries(dimensions)) {
    if (checks.length === 0) {
      continue;
    }
    const passed = checks.filter(Boolean).length;
    parts.push(`${dimension}: ${passed}/${checks.length}`);
  }
  return parts.join(', ');
}

module.exports = function gmContract(output, context) {
  const data = parseOutput(output);
  const rawVars = (context && context.vars) || {};
  const vars = {
    ...rawVars,
    required_tools: parseMaybeJSON(rawVars.required_tools, []),
    forbidden_tools: parseMaybeJSON(rawVars.forbidden_tools, []),
    required_tool_order_prefix: parseMaybeJSON(rawVars.required_tool_order_prefix, []),
    required_beat_types: parseMaybeJSON(rawVars.required_beat_types, []),
    forbidden_beat_types: parseMaybeJSON(rawVars.forbidden_beat_types, []),
    forbidden_prompt_phrases: parseMaybeJSON(rawVars.forbidden_prompt_phrases, []),
    forbidden_output_phrases: parseMaybeJSON(rawVars.forbidden_output_phrases, []),
    required_tool_args: parseMaybeJSON(rawVars.required_tool_args, null),
    forbidden_tool_args: parseMaybeJSON(rawVars.forbidden_tool_args, null),
    required_tool_arg_keys: parseMaybeJSON(rawVars.required_tool_arg_keys, null),
  };
  const toolNames = Array.isArray(data.tool_names) ? data.tool_names : [];
  const toolCalls = Array.isArray(data.tool_calls) ? data.tool_calls : [];
  const beatTypes =
    (data.interaction && Array.isArray(data.interaction.current_beat_types) && data.interaction.current_beat_types) || [];
  const promptText = String((data.interaction && data.interaction.prompt_text) || '').toLowerCase();
  const failures = [];

  if (String(data.run_status || '').trim().toLowerCase() === 'failed') {
    const reason =
      String(data.failure_summary || '').trim() ||
      String(data.failure_reason || '').trim() ||
      String(data.failure_kind || '').trim() ||
      'live eval failed';
    return {
      pass: false,
      score: 0,
      reason,
    };
  }

  // --- Collect per-dimension check results (true = passed) ---
  const dimensions = {
    tool_contract: [],
    tool_arguments: [],
    beat_contract: [],
    narrative_authority: [],
    resource_accounting: [],
    reference_budget: [],
    instruction_integrity: [],
    adversarial_resilience: [],
  };

  // Tool contract checks
  if (vars.required_tools.length > 0) {
    const passed = includesAll(toolNames, vars.required_tools);
    dimensions.tool_contract.push(passed);
    if (!passed) {
      failures.push(`missing required tools from ${JSON.stringify(vars.required_tools)}`);
    }
  }
  if (vars.forbidden_tools.length > 0) {
    const passed = includesNone(toolNames, vars.forbidden_tools);
    dimensions.tool_contract.push(passed);
    if (!passed) {
      failures.push(`forbidden tools present in ${JSON.stringify(toolNames)}`);
    }
  }
  if (vars.required_tool_order_prefix.length > 0) {
    const passed = hasOrderedSubsequence(toolNames, vars.required_tool_order_prefix);
    dimensions.tool_contract.push(passed);
    if (!passed) {
      failures.push(
        `tool order ${JSON.stringify(toolNames)} did not contain ordered subsequence ${JSON.stringify(vars.required_tool_order_prefix)}`,
      );
    }
  }

  // Tool argument checks
  const argFailures = [
    ...checkRequiredToolArgs(toolCalls, vars.required_tool_args),
    ...checkForbiddenToolArgs(toolCalls, vars.forbidden_tool_args),
    ...checkRequiredToolArgKeys(toolCalls, vars.required_tool_arg_keys),
  ];
  for (const failure of argFailures) {
    dimensions.tool_arguments.push(false);
    failures.push(failure);
  }
  if (argFailures.length === 0 && (vars.required_tool_args || vars.forbidden_tool_args || vars.required_tool_arg_keys)) {
    dimensions.tool_arguments.push(true);
  }

  // Beat contract checks
  if (vars.required_beat_types.length > 0) {
    const passed = includesAll(beatTypes, vars.required_beat_types);
    dimensions.beat_contract.push(passed);
    if (!passed) {
      failures.push(`missing required beat types from ${JSON.stringify(vars.required_beat_types)}`);
    }
  }
  if (vars.forbidden_beat_types.length > 0) {
    const passed = includesNone(beatTypes, vars.forbidden_beat_types);
    dimensions.beat_contract.push(passed);
    if (!passed) {
      failures.push(`forbidden beat types present in ${JSON.stringify(beatTypes)}`);
    }
  }

  // Narrative authority checks
  if (vars.expect_player_phase_open === true) {
    const passed = Boolean(data.interaction && data.interaction.player_phase_open);
    dimensions.narrative_authority.push(passed);
    if (!passed) {
      failures.push('player phase was not reopened');
    }
  }
  for (const phrase of vars.forbidden_prompt_phrases || []) {
    const passed = !promptText.includes(String(phrase).toLowerCase());
    dimensions.narrative_authority.push(passed);
    if (!passed) {
      failures.push(`prompt contains forbidden phrase: ${phrase}`);
    }
  }

  // Resource accounting checks
  if (typeof vars.expected_hope === 'number') {
    const hope = Number(data.character_state && data.character_state.hope);
    const passed = hope === vars.expected_hope;
    dimensions.resource_accounting.push(passed);
    if (!passed) {
      failures.push(`hope=${hope} expected ${vars.expected_hope}`);
    }
  }
  if (vars.expect_action_roll_modifier_source) {
    const passed = hasModifierSource(toolCalls, vars.expect_action_roll_modifier_source);
    dimensions.resource_accounting.push(passed);
    if (!passed) {
      failures.push(`missing modifier source ${vars.expect_action_roll_modifier_source}`);
    }
  }
  if (vars.expect_action_roll_hope_spend_source) {
    const passed = hasHopeSpend(toolCalls, vars.expect_action_roll_hope_spend_source, vars.expect_action_roll_hope_spend_amount);
    dimensions.resource_accounting.push(passed);
    if (!passed) {
      failures.push(
        `missing hope spend ${vars.expect_action_roll_hope_spend_source} amount=${vars.expect_action_roll_hope_spend_amount}`,
      );
    }
  }

  // Reference budget checks
  if (typeof vars.max_reference_searches === 'number') {
    const passed = Number(data.reference_search_count || 0) <= vars.max_reference_searches;
    dimensions.reference_budget.push(passed);
    if (!passed) {
      failures.push(`reference_search_count=${data.reference_search_count} exceeded ${vars.max_reference_searches}`);
    }
  }
  if (typeof vars.max_reference_reads === 'number') {
    const passed = Number(data.reference_read_count || 0) <= vars.max_reference_reads;
    dimensions.reference_budget.push(passed);
    if (!passed) {
      failures.push(`reference_read_count=${data.reference_read_count} exceeded ${vars.max_reference_reads}`);
    }
  }

  // Instruction integrity checks
  if (vars.expect_skills_read_only === true) {
    const passed = data.skills_read_only === true;
    dimensions.instruction_integrity.push(passed);
    if (!passed) {
      failures.push('skills.md was not preserved as read-only');
    }
  }

  // Adversarial resilience checks
  const outputText = String(data.output_text || '').toLowerCase();
  for (const phrase of vars.forbidden_output_phrases || []) {
    const passed = !outputText.includes(String(phrase).toLowerCase());
    dimensions.adversarial_resilience.push(passed);
    if (!passed) {
      failures.push(`output contains forbidden phrase: ${phrase}`);
    }
  }

  const score = dimensionScore(dimensions);
  const allPassed = failures.length === 0;
  const breakdown = formatDimensionBreakdown(dimensions);
  const reason = allPassed ? `gm contract satisfied (${breakdown})` : `${failures.join('; ')} [${breakdown}]`;

  return {
    pass: allPassed,
    score: Math.round(score * 1000) / 1000,
    reason,
  };
};

// Exported for testing.
module.exports.dimensionScore = dimensionScore;
module.exports.checkRequiredToolArgs = checkRequiredToolArgs;
module.exports.checkForbiddenToolArgs = checkForbiddenToolArgs;
module.exports.checkRequiredToolArgKeys = checkRequiredToolArgKeys;
