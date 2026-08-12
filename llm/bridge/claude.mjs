import { query } from "@anthropic-ai/claude-agent-sdk";
import { buildPrompt, readRequest, writeError, writeResponse } from "./protocol.mjs";

async function main() {
  const request = await readRequest();
  const options = {
    allowedTools: [],
    permissionMode: "dontAsk",
    settingSources: [],
    cwd: request.working_dir || process.cwd(),
  };
  if (request.model) options.model = request.model;
  if (request.system) options.systemPrompt = request.system;
  if (request.session_id) options.resume = request.session_id;

  let content = "";
  let sessionID = request.session_id || "";
  let model = request.model || "";
  let usage = {};

  for await (const message of query({
    prompt: buildPrompt(request, false),
    options,
  })) {
    sessionID = message.session_id || message.sessionId || sessionID;
    model = message.model || model;
    if (message.type === "result") {
      content = message.result || content;
      usage = {
        input_tokens: message.usage?.input_tokens || 0,
        output_tokens: message.usage?.output_tokens || 0,
        total_tokens:
          (message.usage?.input_tokens || 0) +
          (message.usage?.output_tokens || 0),
      };
    }
  }

  writeResponse({
    session_id: sessionID,
    model,
    content,
    finish_reason: "stop",
    usage,
  });
}

main().catch(writeError);
