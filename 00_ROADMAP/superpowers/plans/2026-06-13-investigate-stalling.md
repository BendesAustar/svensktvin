# Investigate & Fix: LLM Tool Call "Stalling"

## Problem
When the LLM emits a tool call (e.g., `read`, `bash`, `edit`), text streaming to the user appears to stall — no new tokens are displayed while the tool executes. This gives the impression the model has frozen.

## Observation
Tool calls interrupt the streaming of generated tokens. The user sees nothing happen during the tool round-trip.

## Hypotheses
1. **Inference halts at tool boundary**: GPU idle while tool executes in the MCP host, then resumes after result returns. No speculative pre-computation.
2. **Batching delay**: The framework batches inference steps with the tool round-trip, causing the GPU to wait on the slowest part (I/O).
3. **Streaming layer drops tokens**: Tokens may be generated but not flushed to the transport until the tool call resolves.

## Investigation Plan
1. **Measure token timing**: Record timestamps for each token emitted before and after a tool call. Check if generation actually pauses or if only streaming is delayed.
2. **Profile GPU utilization**: Use `nvidia-smi` or equivalent during a tool-call-heavy session. Check for GPU idle periods vs. steady inference.
3. **Inspect streaming transport**: Check if the MCP/agent framework buffers tokens until tool resolution, or flushes them incrementally.
4. **Compare with and without tool calls**: Run a pure generation session vs. one with interleaved tool calls. Compare time-to-first-token and tokens-per-second.

## Potential Fixes
- **Streaming tool results**: Stream tool output inline as it's generated (e.g., `tail -f log`, `dd if=/dev/urandom bs=1024 count=1`)
- **Speculative pre-computation**: If the framework supports it, pre-compute the next tokens while the tool executes
- **Better UX feedback**: Show a visible "running tool: ..." indicator so the user knows work is in progress
- **Hybrid generation**: Allow the model to stream partial thoughts while a tool is in progress (risky for correctness)

## Status
Uninvestigated. Needs profiling during an actual session.
