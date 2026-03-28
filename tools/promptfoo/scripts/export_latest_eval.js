#!/usr/bin/env node

const fs = require('node:fs');
const path = require('node:path');
const { execFileSync } = require('node:child_process');

function parseArgs(argv) {
  const out = {};
  for (let i = 0; i < argv.length; i += 1) {
    const arg = argv[i];
    if (arg === '--output') {
      out.output = argv[i + 1];
      i += 1;
      continue;
    }
    if (arg === '--eval-id') {
      out.evalId = argv[i + 1];
      i += 1;
    }
  }
  if (!out.output) {
    throw new Error('usage: export_latest_eval.js --output <results.json> [--eval-id <eval-id>]');
  }
  return out;
}

function sqliteJSON(dbPath, sql) {
  const raw = execFileSync('sqlite3', ['-json', dbPath, sql], { encoding: 'utf8' });
  return JSON.parse(raw || '[]');
}

function readLatestEvalID() {
  const configDir = process.env.PROMPTFOO_CONFIG_DIR || path.join(process.env.HOME || '', '.promptfoo');
  const markerPath = path.join(configDir, 'evalLastWritten');
  const marker = fs.readFileSync(markerPath, 'utf8').trim();
  const parts = marker.match(/^(.*):(\d{4}-\d{2}-\d{2}T.*Z)$/);
  if (parts) {
    return parts[1];
  }
  return marker.split(':')[0];
}

function maybeJSON(value, fallback = null) {
  if (value === null || value === undefined || value === '') {
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

function main() {
  const args = parseArgs(process.argv.slice(2));
  const configDir = process.env.PROMPTFOO_CONFIG_DIR || path.join(process.env.HOME || '', '.promptfoo');
  const dbPath = path.join(configDir, 'promptfoo.db');
  const evalId = args.evalId || readLatestEvalID();

  const evalRows = sqliteJSON(
    dbPath,
    `select id, created_at, description, results from evals where id = '${evalId.replace(/'/g, "''")}';`,
  );
  if (evalRows.length === 0) {
    throw new Error(`eval ${evalId} not found in ${dbPath}`);
  }

  const resultRows = sqliteJSON(
    dbPath,
    `select id, prompt_idx, test_idx, latency_ms, cost, response, error, success, score, grading_result, named_scores, metadata, prompt, provider, test_case from eval_results where eval_id = '${evalId.replace(/'/g, "''")}' order by test_idx asc, prompt_idx asc;`,
  );

  const exported = {
    evalId,
    results: {
      stats: maybeJSON(evalRows[0].results, {}),
      results: resultRows.map((row) => {
        const testCase = maybeJSON(row.test_case, {});
        return {
          id: row.id,
          promptIdx: row.prompt_idx,
          testIdx: row.test_idx,
          latencyMs: row.latency_ms,
          cost: row.cost,
          response: maybeJSON(row.response, null),
          error: row.error || null,
          success: Boolean(row.success),
          score: row.score,
          gradingResult: maybeJSON(row.grading_result, null),
          namedScores: maybeJSON(row.named_scores, null),
          metadata: maybeJSON(row.metadata, null),
          prompt: maybeJSON(row.prompt, null),
          provider: maybeJSON(row.provider, null),
          testCase,
          description: testCase.description || null,
          vars: testCase.vars || {},
        };
      }),
    },
  };

  fs.mkdirSync(path.dirname(args.output), { recursive: true });
  fs.writeFileSync(args.output, `${JSON.stringify(exported, null, 2)}\n`);
}

main();
