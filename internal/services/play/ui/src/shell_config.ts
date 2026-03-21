export type PlayShellConfig = {
  campaignId: string;
  bootstrapPath: string;
  realtimePath: string;
  backURL: string;
};

type DocumentLike = Pick<Document, "getElementById">;
type ScriptLike = Pick<HTMLScriptElement, "textContent">;

const shellConfigElementID = "play-shell-config";

export function readShellConfig(documentLike: DocumentLike = document): PlayShellConfig | null {
  const element = documentLike.getElementById(shellConfigElementID);
  if (!(element instanceof HTMLScriptElement)) {
    return null;
  }
  return parseShellConfigScript(element);
}

export function parseShellConfigScript(script: ScriptLike): PlayShellConfig | null {
  const raw = script.textContent?.trim();
  if (!raw) {
    return null;
  }

  let parsed: unknown;
  try {
    parsed = JSON.parse(raw);
  } catch {
    return null;
  }

  if (!isRecord(parsed)) {
    return null;
  }

  const campaignId = asTrimmedString(parsed.campaign_id);
  const bootstrapPath = asTrimmedString(parsed.bootstrap_path);
  const realtimePath = asTrimmedString(parsed.realtime_path);
  const backURL = asTrimmedString(parsed.back_url);

  return {
    campaignId,
    bootstrapPath,
    realtimePath,
    backURL,
  };
}

function isRecord(value: unknown): value is Record<string, unknown> {
  return typeof value === "object" && value !== null;
}

function asTrimmedString(value: unknown): string {
  return typeof value === "string" ? value.trim() : "";
}
