#!/usr/bin/env node

const fs = require('node:fs');
const path = require('node:path');

const FAILURE_CLASSES = new Set([
  'missing_authoritative_roll',
  'resource_accounting',
  'over_research',
  'narrator_authority',
  'phase_reopen',
  'forbidden_tool_path',
  'tool_argument_error',
  'adversarial_compliance',
  'tool_execution_error',
  'turn_control_error',
  'provider_error',
  'recorder_error',
  'artifact_capture_error',
  'harness_error',
]);

function parseArgs(argv) {
  const out = {};
  for (let i = 0; i < argv.length; i += 1) {
    const arg = argv[i];
    if (arg === '--input') {
      out.input = argv[i + 1];
      i += 1;
      continue;
    }
    if (arg === '--output') {
      out.output = argv[i + 1];
      i += 1;
    }
  }
  if (!out.input || !out.output) {
    throw new Error('usage: summarize_results.js --input <results.json> --output <scorecard.md>');
  }
  return out;
}

function readJSON(filePath) {
  return JSON.parse(fs.readFileSync(filePath, 'utf8'));
}

function parseResponseOutput(result) {
  const raw = result && result.response && result.response.output;
  if (!raw) {
    return {};
  }
  try {
    return typeof raw === 'string' ? JSON.parse(raw) : raw;
  } catch {
    return {};
  }
}

function failureClassFor(result, output) {
  if (output.run_status === 'failed' && FAILURE_CLASSES.has(output.failure_kind)) {
    return output.failure_kind;
  }
  const reason = String(
    (result && result.gradingResult && result.gradingResult.reason) || result.error || output.failure_reason || '',
  ).toLowerCase();
  if (/hope=\d+\s+expected\s+\d+/.test(reason)) {
    return 'resource_accounting';
  }
  if (reason.includes('missing_authoritative_roll') || reason.includes('daggerheart_action_roll_resolve')) {
    return 'missing_authoritative_roll';
  }
  if (reason.includes('reference_search_count') || reason.includes('reference_read_count') || reason.includes('over_research')) {
    return 'over_research';
  }
  if (reason.includes('prompt contains forbidden phrase') || reason.includes('forbidden beat types')) {
    return 'narrator_authority';
  }
  if (reason.includes('player phase was not reopened')) {
    return 'phase_reopen';
  }
  if (reason.includes('forbidden tools present') || reason.includes('forbidden_tool_path')) {
    return 'forbidden_tool_path';
  }
  if (reason.includes('missing required arg') || reason.includes('forbidden arg') || reason.includes('tool_argument')) {
    return 'tool_argument_error';
  }
  if (reason.includes('output contains forbidden phrase') || reason.includes('adversarial')) {
    return 'adversarial_compliance';
  }
  return 'harness_error';
}

function failureReasonFor(result, output) {
  if (output.run_status === 'failed' && output.failure_summary) {
    return String(output.failure_summary).trim();
  }
  if (output.run_status === 'failed' && output.failure_reason) {
    return String(output.failure_reason).trim();
  }
  return String((result && result.gradingResult && result.gradingResult.reason) || result.error || '').trim();
}

function rowFor(result) {
  const output = parseResponseOutput(result);
  return {
    scenario: output.label || (result.vars && result.vars.label) || (result.vars && result.vars.scenario) || 'unknown',
    model: (result.provider && result.provider.label) || 'unknown',
    promptProfile: (result.prompt && result.prompt.label) || output.prompt_profile || 'baseline',
    success: Boolean(result.success),
    score: Number(result.score || 0),
    metricStatus: String(output.metric_status || (result.success ? 'pass' : 'fail')).trim() || 'fail',
    failureClass: failureClassFor(result, output),
    failureReason: failureReasonFor(result, output),
    artifacts: output.artifacts || {},
  };
}

function dominantFailure(rows) {
  const counts = new Map();
  for (const row of rows) {
    if (row.success) {
      continue;
    }
    counts.set(row.failureClass, (counts.get(row.failureClass) || 0) + 1);
  }
  if (counts.size === 0) {
    return 'pass';
  }
  return [...counts.entries()].sort((a, b) => b[1] - a[1])[0][0];
}

function groupRows(rows) {
  const grouped = new Map();
  for (const row of rows) {
    const key = [row.scenario, row.model, row.promptProfile].join('|');
    const bucket = grouped.get(key) || [];
    bucket.push(row);
    grouped.set(key, bucket);
  }
  return [...grouped.entries()].map(([key, group]) => {
    const [scenario, model, promptProfile] = key.split('|');
    const validRows = group.filter((row) => row.metricStatus !== 'invalid');
    const invalidRows = group.filter((row) => row.metricStatus === 'invalid').length;
    const passed = validRows.filter((row) => row.success).length;
    const representative = group.find((row) => !row.success) || group[0];
    const avgScore = validRows.length === 0 ? 0 : validRows.reduce((sum, row) => sum + row.score, 0) / validRows.length;
    return {
      scenario,
      model,
      promptProfile,
      passed,
      total: group.length,
      validRuns: validRows.length,
      invalidRuns: invalidRows,
      qualityPassRate: validRows.length === 0 ? 'n/a' : `${passed}/${validRows.length}`,
      avgScore: validRows.length === 0 ? 'n/a' : `${Math.round(avgScore * 100)}%`,
      dominantFailure: dominantFailure(group),
      representativeReason: representative.success ? 'gm contract satisfied' : representative.failureReason,
      artifacts: representative.artifacts || {},
    };
  });
}

function renderMarkdown(groups, inputPath) {
  const lines = [];
  lines.push('# Promptfoo GM Eval Scorecard', '');
  lines.push(`Generated from \`${inputPath}\`.`, '');
  lines.push('| Scenario | Model | Prompt Profile | Quality Pass Rate | Avg Score | Invalid Runs | Dominant Failure | Representative Reason |');
  lines.push('| --- | --- | --- | --- | --- | --- | --- | --- |');
  for (const group of groups.sort((a, b) => a.scenario.localeCompare(b.scenario) || a.model.localeCompare(b.model) || a.promptProfile.localeCompare(b.promptProfile))) {
    lines.push(
      `| ${group.scenario} | ${group.model} | ${group.promptProfile} | ${group.qualityPassRate} | ${group.avgScore} | ${group.invalidRuns}/${group.total} | ${group.dominantFailure} | ${group.representativeReason.replace(/\|/g, '\\|')} |`,
    );
  }
  lines.push('', '## Representative Artifacts', '');
  for (const group of groups) {
    lines.push(`### ${group.scenario} / ${group.model} / ${group.promptProfile}`, '');
    lines.push(`- Raw capture: ${group.artifacts.raw_capture || 'n/a'}`);
    lines.push(`- Markdown report: ${group.artifacts.markdown_report || 'n/a'}`);
    lines.push(`- Summary: ${group.artifacts.summary || 'n/a'}`);
    lines.push(`- Diagnostics: ${group.artifacts.diagnostics || 'n/a'}`);
    lines.push(`- Harness log: ${group.artifacts.harness_log || 'n/a'}`, '');
  }
  return `${lines.join('\n')}\n`;
}

function main() {
  const args = parseArgs(process.argv.slice(2));
  const input = readJSON(args.input);
  const results = (((input || {}).results || {}).results || []).map(rowFor);
  const groups = groupRows(results);
  fs.mkdirSync(path.dirname(args.output), { recursive: true });
  fs.writeFileSync(args.output, renderMarkdown(groups, args.input));
}

if (require.main === module) {
  main();
}

module.exports = {
  failureClassFor,
  failureReasonFor,
  groupRows,
  parseResponseOutput,
  renderMarkdown,
  rowFor,
};
