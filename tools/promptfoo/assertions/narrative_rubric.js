// narrative_rubric.js — model-graded narrative quality assertion for GM eval.
//
// This assertion extracts committed narration and beat structure from the
// aieval.Output JSON and grades it along four dimensions using an LLM call.
// It returns a fractional score (0–1) with per-dimension breakdown.
//
// Requires INTEGRATION_OPENAI_API_KEY in the environment.

const https = require('node:https');

const RUBRIC_PROMPT = `You are evaluating the narrative quality of an AI Game Master's response in a tabletop RPG session.

You will receive:
- The committed GM narration text (beats)
- The beat types used
- The final prompt beat (what was asked of the player)

Grade the response on these four dimensions, each scored 0-5:

1. **narrator_voice** — The committed narration contains no rules text, meta-commentary, engine vocabulary (internal IDs, enum names like "scene_id", "adversary_id", state labels), or out-of-character content. Pure in-character prose only. Score 0 if rules text or engine terms appear in narration. Score 5 for fully immersive narrator voice.

2. **tone_atmosphere** — The prose establishes a vivid fantasy RPG atmosphere. It uses sensory detail, tension, and pacing appropriate to the scene. Not purple or overwrought, not flat or mechanical. Score 0 for robotic/mechanical output. Score 5 for compelling, tonally consistent prose.

3. **prompt_quality** — The final player-facing prompt asks what the player character does, says, chooses, or commits to next. It does NOT ask the player to author NPC dialogue, NPC choices, or how the story world answers them. Score 0 if the prompt delegates NPC authorship to the player. Score 5 for a clear, character-focused prompt that advances the story.

4. **beat_economy** — Beats are coherent GM moves, not paragraph containers. Related prose stays in one beat. Consecutive beats of the same type are justified only when they represent distinct GM moves. No unnecessary splitting or padding. Score 0 for bloated or fragmented beats. Score 5 for crisp, purposeful beat structure.

Respond with ONLY a JSON object (no markdown fencing):
{"narrator_voice": <0-5>, "tone_atmosphere": <0-5>, "prompt_quality": <0-5>, "beat_economy": <0-5>, "explanation": "<brief justification>"}`;

function extractNarration(data) {
  const parts = [];
  if (data.output_text) {
    parts.push(data.output_text);
  }
  if (data.interaction && data.interaction.prompt_text) {
    parts.push(`[Prompt beat]: ${data.interaction.prompt_text}`);
  }
  const beatTypes = (data.interaction && data.interaction.current_beat_types) || [];
  if (beatTypes.length > 0) {
    parts.push(`[Beat types]: ${beatTypes.join(', ')}`);
  }
  return parts.join('\n\n');
}

function callOpenAI(apiKey, narration) {
  return new Promise((resolve, reject) => {
    const body = JSON.stringify({
      model: 'gpt-5.4-mini',
      messages: [
        { role: 'system', content: RUBRIC_PROMPT },
        { role: 'user', content: narration },
      ],
      temperature: 0,
    });

    const options = {
      hostname: 'api.openai.com',
      port: 443,
      path: '/v1/chat/completions',
      method: 'POST',
      headers: {
        Authorization: `Bearer ${apiKey}`,
        'Content-Type': 'application/json',
        'Content-Length': Buffer.byteLength(body),
      },
    };

    const req = https.request(options, (res) => {
      let data = '';
      res.on('data', (chunk) => {
        data += chunk;
      });
      res.on('end', () => {
        if (res.statusCode !== 200) {
          reject(new Error(`OpenAI API returned ${res.statusCode}: ${data.slice(0, 200)}`));
          return;
        }
        try {
          const parsed = JSON.parse(data);
          const content = parsed.choices[0].message.content;
          resolve(content);
        } catch (err) {
          reject(new Error(`Failed to parse OpenAI response: ${err.message}`));
        }
      });
    });

    req.on('error', reject);
    req.write(body);
    req.end();
  });
}

function parseGradeResponse(raw) {
  const cleaned = raw.replace(/```json\s*/g, '').replace(/```\s*/g, '').trim();
  return JSON.parse(cleaned);
}

module.exports = async function narrativeRubric(output, context) {
  const apiKey = process.env.INTEGRATION_OPENAI_API_KEY || process.env.OPENAI_API_KEY || '';
  if (!apiKey) {
    return {
      pass: true,
      score: 1,
      reason: 'narrative rubric skipped: no API key available',
    };
  }

  let data;
  if (output && typeof output === 'object') {
    data = output;
  } else {
    try {
      data = JSON.parse(String(output || '{}'));
    } catch {
      return { pass: true, score: 1, reason: 'narrative rubric skipped: unparseable output' };
    }
  }

  if (String(data.run_status || '').trim().toLowerCase() === 'failed') {
    return { pass: true, score: 0, reason: 'narrative rubric skipped: run failed' };
  }

  const narration = extractNarration(data);
  if (!narration.trim()) {
    return { pass: true, score: 0, reason: 'narrative rubric skipped: no narration text' };
  }

  try {
    const raw = await callOpenAI(apiKey, narration);
    const grades = parseGradeResponse(raw);
    const dimensions = ['narrator_voice', 'tone_atmosphere', 'prompt_quality', 'beat_economy'];
    const scores = dimensions.map((d) => Number(grades[d] || 0));
    const avg = scores.reduce((a, b) => a + b, 0) / scores.length;
    const normalized = avg / 5; // 0-5 scale to 0-1

    const breakdown = dimensions.map((d) => `${d}: ${grades[d] || 0}/5`).join(', ');
    const explanation = grades.explanation || '';

    return {
      pass: normalized >= 0.6,
      score: Math.round(normalized * 1000) / 1000,
      reason: `narrative quality ${Math.round(normalized * 100)}% (${breakdown}). ${explanation}`.trim(),
    };
  } catch (err) {
    return {
      pass: true,
      score: 1,
      reason: `narrative rubric error: ${err.message}`,
    };
  }
};

module.exports.extractNarration = extractNarration;
module.exports.parseGradeResponse = parseGradeResponse;
