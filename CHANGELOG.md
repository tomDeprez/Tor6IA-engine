# Changelog

Each `## vX.Y.Z` section becomes the notes of that release.

## v0.1.0

- First release. `demo` splits a model across block servers on one machine and shows who holds which layers. `block` and `head` do it across machines you trust.
- llama.cpp b12810 is downloaded on first use, with its checksum verified.
- Models Ollama already has are used where they are.
- Measured: Gemma 4 12B split in two answers at 3.3 tokens per second on one laptop, 1.9 with 50 ms of network delay each way.
