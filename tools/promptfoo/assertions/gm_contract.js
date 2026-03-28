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

function hasHopeSpend(toolCalls, source, amount) {
  if (!source) {
    return true;
  }
  const resolveCall = (toolCalls || []).find((call) => call.name === 'daggerheart_action_roll_resolve');
  if (!resolveCall || !Array.isArray(resolveCall.arguments && resolveCall.arguments.hope_spends)) {
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
  };
  const toolNames = Array.isArray(data.tool_names) ? data.tool_names : [];
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

  if (!includesAll(toolNames, vars.required_tools)) {
    failures.push(`missing required tools from ${JSON.stringify(vars.required_tools || [])}`);
  }
  if (!includesNone(toolNames, vars.forbidden_tools)) {
    failures.push(`forbidden tools present in ${JSON.stringify(toolNames)}`);
  }
  if (!hasOrderedSubsequence(toolNames, vars.required_tool_order_prefix)) {
    failures.push(`tool order ${JSON.stringify(toolNames)} did not contain ordered subsequence ${JSON.stringify(vars.required_tool_order_prefix || [])}`);
  }
  if (!includesAll(beatTypes, vars.required_beat_types)) {
    failures.push(`missing required beat types from ${JSON.stringify(vars.required_beat_types || [])}`);
  }
  if (!includesNone(beatTypes, vars.forbidden_beat_types)) {
    failures.push(`forbidden beat types present in ${JSON.stringify(beatTypes)}`);
  }
  if (vars.expect_player_phase_open === true && !(data.interaction && data.interaction.player_phase_open)) {
    failures.push('player phase was not reopened');
  }
  if (vars.expect_skills_read_only === true && data.skills_read_only !== true) {
    failures.push('skills.md was not preserved as read-only');
  }
  for (const phrase of vars.forbidden_prompt_phrases || []) {
    if (promptText.includes(String(phrase).toLowerCase())) {
      failures.push(`prompt contains forbidden phrase: ${phrase}`);
    }
  }
  if (
    typeof vars.max_reference_searches === 'number' &&
    Number(data.reference_search_count || 0) > vars.max_reference_searches
  ) {
    failures.push(`reference_search_count=${data.reference_search_count} exceeded ${vars.max_reference_searches}`);
  }
  if (
    typeof vars.max_reference_reads === 'number' &&
    Number(data.reference_read_count || 0) > vars.max_reference_reads
  ) {
    failures.push(`reference_read_count=${data.reference_read_count} exceeded ${vars.max_reference_reads}`);
  }
  if (typeof vars.expected_hope === 'number') {
    const hope = Number(data.character_state && data.character_state.hope);
    if (hope !== vars.expected_hope) {
      failures.push(`hope=${hope} expected ${vars.expected_hope}`);
    }
  }
  if (!hasModifierSource(data.tool_calls, vars.expect_action_roll_modifier_source)) {
    failures.push(`missing modifier source ${vars.expect_action_roll_modifier_source}`);
  }
  if (!hasHopeSpend(data.tool_calls, vars.expect_action_roll_hope_spend_source, vars.expect_action_roll_hope_spend_amount)) {
    failures.push(
      `missing hope spend ${vars.expect_action_roll_hope_spend_source} amount=${vars.expect_action_roll_hope_spend_amount}`,
    );
  }

  return {
    pass: failures.length === 0,
    score: failures.length === 0 ? 1 : 0,
    reason: failures.length === 0 ? 'gm contract satisfied' : failures.join('; '),
  };
};
