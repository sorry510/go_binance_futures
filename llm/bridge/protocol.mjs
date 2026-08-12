import process from "node:process";

export async function readRequest() {
  const chunks = [];
  for await (const chunk of process.stdin) {
    chunks.push(chunk);
  }

  const payload = Buffer.concat(chunks).toString("utf8").trim();
  if (!payload) {
    throw new Error("empty bridge request");
  }
  return JSON.parse(payload);
}

export function buildPrompt(request, includeSystem = true) {
  const parts = [];
  if (includeSystem && request.system) {
    parts.push(`SYSTEM:\n${request.system}`);
  }
  for (const message of request.messages ?? []) {
    parts.push(`${String(message.role).toUpperCase()}:\n${message.content}`);
  }
  return parts.join("\n\n");
}

export function writeResponse(response) {
  process.stdout.write(`${JSON.stringify(response)}\n`);
}

export function writeError(error) {
  const message = error instanceof Error ? error.message : String(error);
  writeResponse({ error: message });
}
