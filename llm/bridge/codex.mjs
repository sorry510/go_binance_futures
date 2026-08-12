import { Codex } from "@openai/codex-sdk";
import { buildPrompt, readRequest, writeError, writeResponse } from "./protocol.mjs";

async function main() {
  const request = await readRequest();
  const codex = new Codex();
  const threadOptions = {
    workingDirectory: request.working_dir || process.cwd(),
    sandboxMode: "read-only",
    approvalPolicy: "never",
  };
  if (request.model) threadOptions.model = request.model;

  const thread = request.session_id
    ? codex.resumeThread(request.session_id)
    : codex.startThread(threadOptions);
  const result = await thread.run(buildPrompt(request));

  writeResponse({
    session_id: result.threadId || thread.id || request.session_id || "",
    model: request.model || "",
    content: result.finalResponse || "",
    finish_reason: "stop",
  });
}

main().catch(writeError);
