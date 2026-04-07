# Agent Bridge Protocol

## Overview

Agent Bridge enables implants with a built-in agent loop (`bridge_agent` module) to execute natural-language tasks autonomously. The implant runs its own agent locally, while the server proxies LLM API calls on its behalf.

This mechanism coexists with the legacy poison/tapping pipeline (which hijacks an external LLM provider session). The two are completely independent; the client dispatches to the correct backend based on which modules the session has loaded.

## Architecture

```
Client                     Server                      Implant
  |                          |                            |
  |-- BridgeAgentChat() ---->|                            |
  |   (text, model,         |-- Spite(BridgeAgentReq) -->|
  |    provider, max_turns)  |                            |
  |                          |   agent loop running...    |
  |                          |                            |
  |                          |<-- Spite(BridgeLlmReq) ---|
  |                          |   (raw OpenAI JSON body)   |
  |                          |                            |
  |                          |-- POST /chat/completions ->| LLM API
  |                          |<-- JSON response ----------|
  |                          |                            |
  |                          |-- Spite(BridgeLlmResp) -->|
  |                          |   {"payload": <raw>}       |
  |                          |                            |
  |                          |   ... repeat per turn ...  |
  |                          |                            |
  |                          |<-- Spite(BridgeAgentResp)-|
  |<-- Task Done ------------|   (text, tool_calls, etc.) |
```

## Proto Messages

All messages are defined in `implant/implantpb/implant.proto` as Spite body variants:

| Field Number | Message | Direction | Description |
|---|---|---|---|
| 164 | `BridgeAgentRequest` | server -> implant | Initial task with text, model, config |
| 165 | `BridgeAgentResponse` | implant -> server | Final result with text, tool calls |
| 166 | `BridgeLlmRequest` | implant -> server | Raw OpenAI-format request body |
| 167 | `BridgeLlmResponse` | server -> implant | Wrapped API response `{"payload": ...}` |

Legacy `llm_event = 160` is preserved for the poison/tapping pipeline.

## RPC

```protobuf
rpc BridgeAgentChat(implantpb.BridgeAgentRequest) returns (clientpb.Task);
```

The handler uses `StreamGenericHandler` for bidirectional streaming (same pattern as `handlePtyStart`). It spawns a `runTaskHandler` goroutine that:

1. Reads from the implant channel (`out`)
2. If `BridgeLlmRequest`: calls `llm.CallProvider()`, sends response back via `in.Send()`
3. If `BridgeAgentResponse`: calls `HandlerSpite()` + `Finish()`, returns

## LLM Provider

`server/internal/llm/provider.go` handles LLM API proxying with a three-level config resolution:

1. **Request provider/model** (from client's `config ai` settings, passed via `BridgeAgentRequest`)
2. **Server config.yaml** (`server.llm`, including per-provider endpoint/api_key/proxy_url/timeout)
3. **Environment variables** (`BRIDGE_<PROVIDER>_BASE_URL`, `BRIDGE_API_KEY`, etc.)
4. **Provider presets** (built-in base URLs for openai, openrouter, deepseek, groq, moonshot)

## Client Commands

### `chat`

```
chat [message]
chat -m gpt-4o "list all files"
chat -p deepseek "scan the network"
```

Sends a message to the implant's self-agent. Reads LLM config from `config ai` automatically. Flags `--model`/`--provider` override the config values.

Requires the `bridge_agent` module on the session.

### `skill` (dual dispatch)

```
skill <name> [arguments...]
skill list
```

Loads a `SKILL.md` file and dispatches based on session capabilities:
- Session has `bridge_agent` module -> `BridgeAgentChat`
- Otherwise -> `Poison` (legacy pipeline)

The `depend` annotation is `bridge_agent,poison`, meaning the command is visible when either module is present.

## Configuration

The `chat` command still reads `provider` / `model` from the existing client-side `config ai` settings:

```
config ai --provider openai --api-key sk-xxx --endpoint https://api.openai.com/v1
```

Server-side LLM credentials and endpoints are resolved from `server.config.yaml`:

```yaml
server:
  llm:
    default_provider: openai
    timeout: 120
    proxy_url: ""
    providers:
      openai:
        endpoint: https://api.openai.com/v1
        api_key: ""
```

If the selected provider is not configured there, the server falls back to environment variables and presets.

## Module Detection

The client now dispatches by session target rather than a separate `bridge_agent` capability:

```go
func hasModule(sess *client.Session, name string) bool
```

This determines which backend to use for `skill` dispatch and whether `chat` is available in the command tree.
