#!/usr/bin/env node

const fs = require('node:fs');
const path = require('node:path');

const { groupRows, rowFor } = require('./summarize_results');

function parseArgs(argv) {
  const out = {};
  for (let i = 0; i < argv.length; i += 1) {
    const arg = argv[i];
    if (arg === '--results') {
      out.results = argv[i + 1];
      i += 1;
    } else if (arg === '--baseline') {
      out.baseline = argv[i + 1];
      i += 1;
    } else if (arg === '--output') {
      out.output = argv[i + 1];
      i += 1;
    }
  }
  if (!out.results || !out.baseline || !out.output) {
    throw new Error('usage: compare_baseline.js --results <results.json> --baseline <baseline.json> --output <regression.md>');
  }
  return out;
}

function baselineKey(entry) {
  return [entry.scenario, entry.model, entry.prompt_profile].join('|');
}

function groupKey(group) {
  return [group.scenario, group.model, group.promptProfile].join('|');
}

function formatScore(score) {
  if (typeof score !== 'number' || Number.isNaN(score)) {
    return 'n/a';
  }
  return `${Math.round(score * 100)}%`;
}

function formatStatus(pass) {
  return pass ? 'PASS' : 'FAIL';
}

function classify(baselineEntry, currentGroup) {
  if (!baselineEntry) {
    return 'new';
  }
  const currentPass = currentGroup.passed > 0 && currentGroup.passed === currentGroup.validRuns;
  const currentScore = currentGroup.validRuns === 0 ? 0 : currentGroup.passed / currentGroup.validRuns;
  const baselineScore = baselineEntry.score;
  const baselinePass = baselineEntry.pass;

  if (baselinePass && !currentPass) {
    return 'regression';
  }
  if (!baselinePass && currentPass) {
    return 'improvement';
  }
  if (currentScore < baselineScore - 0.01) {
    return 'regression';
  }
  if (currentScore > baselineScore + 0.01) {
    return 'improvement';
  }
  return 'unchanged';
}

function compareResults(baseline, results) {
  const baselineMap = new Map();
  for (const entry of baseline.entries || []) {
    baselineMap.set(baselineKey(entry), entry);
  }

  const rows = results.map(rowFor);
  const groups = groupRows(rows);

  const classified = {
    regression: [],
    improvement: [],
    unchanged: [],
    new: [],
  };

  for (const group of groups) {
    if (group.validRuns === 0) {
      continue; // skip invalid-only runs
    }
    const key = groupKey(group);
    const entry = baselineMap.get(key) || null;
    const category = classify(entry, group);
    const currentScore = group.validRuns === 0 ? 0 : group.passed / group.validRuns;
    classified[category].push({ group, baseline: entry, currentScore });
  }

  return classified;
}

function renderMarkdown(classified) {
  const lines = [];
  lines.push('# Eval Regression Report', '');

  // Regressions
  lines.push('## Regressions', '');
  if (classified.regression.length === 0) {
    lines.push('None.', '');
  } else {
    lines.push('| Scenario | Model | Profile | Baseline | Current | Delta | Reason |');
    lines.push('| --- | --- | --- | --- | --- | --- | --- |');
    for (const { group, baseline, currentScore } of classified.regression) {
      const delta = baseline ? currentScore - baseline.score : 0;
      lines.push(
        `| ${group.scenario} | ${group.model} | ${group.promptProfile} | ${baseline ? `${formatScore(baseline.score)} ${formatStatus(baseline.pass)}` : 'n/a'} | ${formatScore(currentScore)} ${formatStatus(group.passed === group.validRuns)} | ${delta >= 0 ? '+' : ''}${formatScore(delta)} | ${group.dominantFailure} |`,
      );
    }
    lines.push('');
  }

  // Improvements
  lines.push('## Improvements', '');
  if (classified.improvement.length === 0) {
    lines.push('None.', '');
  } else {
    lines.push('| Scenario | Model | Profile | Baseline | Current | Delta | Reason |');
    lines.push('| --- | --- | --- | --- | --- | --- | --- |');
    for (const { group, baseline, currentScore } of classified.improvement) {
      const delta = baseline ? currentScore - baseline.score : currentScore;
      lines.push(
        `| ${group.scenario} | ${group.model} | ${group.promptProfile} | ${baseline ? `${formatScore(baseline.score)} ${formatStatus(baseline.pass)}` : 'n/a'} | ${formatScore(currentScore)} ${formatStatus(group.passed === group.validRuns)} | +${formatScore(Math.abs(delta))} | ${group.dominantFailure} |`,
      );
    }
    lines.push('');
  }

  // Unchanged
  lines.push('## Unchanged', '');
  if (classified.unchanged.length === 0) {
    lines.push('None.', '');
  } else {
    lines.push('| Scenario | Model | Profile | Score | Status |');
    lines.push('| --- | --- | --- | --- | --- |');
    for (const { group, currentScore } of classified.unchanged) {
      lines.push(
        `| ${group.scenario} | ${group.model} | ${group.promptProfile} | ${formatScore(currentScore)} | ${formatStatus(group.passed === group.validRuns)} |`,
      );
    }
    lines.push('');
  }

  // New
  lines.push('## New (no baseline)', '');
  if (classified.new.length === 0) {
    lines.push('None.', '');
  } else {
    lines.push('| Scenario | Model | Profile | Score | Status |');
    lines.push('| --- | --- | --- | --- | --- |');
    for (const { group, currentScore } of classified.new) {
      lines.push(
        `| ${group.scenario} | ${group.model} | ${group.promptProfile} | ${formatScore(currentScore)} | ${formatStatus(group.passed === group.validRuns)} |`,
      );
    }
    lines.push('');
  }

  const total =
    classified.regression.length + classified.improvement.length + classified.unchanged.length + classified.new.length;
  lines.push(
    `Summary: ${classified.regression.length} regression(s), ${classified.improvement.length} improvement(s), ${classified.unchanged.length} unchanged, ${classified.new.length} new. ${total} total.`,
  );

  return `${lines.join('\n')}\n`;
}

function main() {
  const args = parseArgs(process.argv.slice(2));
  const results = JSON.parse(fs.readFileSync(args.results, 'utf8'));
  const baseline = JSON.parse(fs.readFileSync(args.baseline, 'utf8'));
  const resultsList = (((results || {}).results || {}).results || []);
  const classified = compareResults(baseline, resultsList);
  const markdown = renderMarkdown(classified);
  fs.mkdirSync(path.dirname(args.output), { recursive: true });
  fs.writeFileSync(args.output, markdown);
}

if (require.main === module) {
  main();
}

module.exports = { compareResults, renderMarkdown, classify, baselineKey, groupKey };
