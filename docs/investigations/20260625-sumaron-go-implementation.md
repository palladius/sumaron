# Investigation: Go Implementation of SUMARON Directory Summarizer
Date: 2026-06-25

## Problem Statement
The user requested a fast, compiled implementation of `sumaron`—an LLM-based directory summarizer tool. Because LLM processing takes time and API tokens are metered, it is vital to avoid redundant API calls when the directory content has not changed.

## Solution Architecture
We implemented the summarizer in Go 1.26 (for native execution speed) using the official `github.com/google/generative-ai-go/genai` library.

### Key Design Elements:
1. **Deterministic Hashing**:
   - Walks the target directory recursively, filtering files by target extensions (defaults to `.md`, `.html`, `.json`).
   - Standard build, cache, and code directories (like `.git`, `node_modules`, `.venv`, `bin`) are ignored.
   - Files are sorted alphabetically by relative path.
   - The SHA-256/MD5 hashing digests the relative path, file size, and file content in a deterministic sequence.
2. **Double-Cache System**:
   - **Local Cache**: A `.sumaron.json` file is written inside the target folder.
   - **Central Cache**: A central record is kept at `~/.sumaron-cache.json` matching absolute path -> cached results.
   - On execution, if either cache has a match for the computed hash, the summarization is skipped, and the cached text is outputted immediately (a "cache hit").
3. **Flexible CLI Arguments**:
   - `-dir`: directory to target (defaults to `.` space).
   - `-key`: overrides the `GEMINI_API_KEY` env variable.
   - `-model`: select a custom model (defaults to `gemini-2.5-flash`, though testing showed `gemini-3.5-flash` is fully supported on the environment).
   - `-force`: forces a cache bypass and re-summarizes.
   - `-extensions`: comma-separated extensions list.

## Verification
- Local unit tests check directory hash calculation logic and cache lookup/saving logic.
- Executing `./bin/sumaron -model gemini-3.5-flash` successfully generated a high-quality Markdown summary of `/projects/sumaron` and saved it to `.sumaron.json`.
- Subsequent runs executed instantly with a `Cache hit!` message.
