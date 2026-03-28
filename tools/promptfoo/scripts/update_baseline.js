#!/usr/bin/env node

const fs = require('node:fs');

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
    }
  }
  if (!out.results || !out.baseline) {
    throw new Error('usage: update_baseline.js --results <results.json> --baseline <baseline.json>');
  }
  return out;
}

function baselineKey(entry) {
  return [entry.scenario, entry.model, entry.prompt_profile].join('|');
}

function updateBaseline(existing, results) {
  const entryMap = new Map();
  for (const entry of existing.entries || []) {
    entryMap.set(baselineKey(entry), entry);
  }

  const rows = results.map(rowFor);
  const groups = groupRows(rows);
  let updated = 0;
  let skipped = 0;

  for (const group of groups) {
    if (group.validRuns === 0) {
      skipped += 1;
      continue;
    }
    const key = [group.scenario, group.model, group.promptProfile].join('|');
    const allPassed = group.passed > 0 && group.passed === group.validRuns;
    const score = group.validRuns === 0 ? 0 : group.passed / group.validRuns;
    const avgScore = typeof group.avgScore === 'string' && group.avgScore !== 'n/a'
      ? parseFloat(group.avgScore) / 100
      : score;

    entryMap.set(key, {
      scenario: group.scenario,
      model: group.model,
      prompt_profile: group.promptProfile,
      pass: allPassed,
      score: Math.round((Number.isNaN(avgScore) ? score : avgScore) * 1000) / 1000,
      failure_class: group.dominantFailure,
      dominant_reason: group.representativeReason,
      recorded_at: new Date().toISOString(),
    });
    updated += 1;
  }

  const entries = [...entryMap.values()].sort((a, b) => {
    const cmp = a.scenario.localeCompare(b.scenario);
    if (cmp !== 0) return cmp;
    const mCmp = a.model.localeCompare(b.model);
    if (mCmp !== 0) return mCmp;
    return a.prompt_profile.localeCompare(b.prompt_profile);
  });

  return {
    baseline: { updated_at: new Date().toISOString(), entries },
    updated,
    skipped,
  };
}

function main() {
  const args = parseArgs(process.argv.slice(2));
  const results = JSON.parse(fs.readFileSync(args.results, 'utf8'));
  let existing = { entries: [] };
  if (fs.existsSync(args.baseline)) {
    existing = JSON.parse(fs.readFileSync(args.baseline, 'utf8'));
  }
  const resultsList = (((results || {}).results || {}).results || []);
  const { baseline, updated, skipped } = updateBaseline(existing, resultsList);
  fs.writeFileSync(args.baseline, `${JSON.stringify(baseline, null, 2)}\n`);
  console.log(`Baseline updated: ${updated} entries updated, ${skipped} skipped (invalid). Total: ${baseline.entries.length} entries.`);
}

if (require.main === module) {
  main();
}

module.exports = { updateBaseline, baselineKey };
