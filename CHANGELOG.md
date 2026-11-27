# Changelog

Each `## vX.Y.Z` section becomes the notes of that release.

## v0.1.2

- A block server that does not answer is now an error. llama.cpp was quietly loading its layers on the head, and the model was shown as split when it was not.
- Thinking is really turned off: with the previous setting, Gemma 4 E2B returned its reasoning in place of the answer.

## v0.1.1

- `head --json` prints the address and the blocks as one line of JSON, and `--exit-with-stdin` stops the engine when the program that started it goes away: Tor6IA uses both to drive the engine.
- `head` stops with an error when the model stops, for example when a block server leaves.
- `--device CPU` makes a block server compute on the processor.

## v0.1.0

- First release. `demo` splits a model across block servers on one machine and shows who holds which layers. `block` and `head` do it across machines you trust.
- llama.cpp b12810 is downloaded on first use, with its checksum verified.
- Models Ollama already has are used where they are.
- Measured: Gemma 4 12B split in two answers at 3.3 tokens per second on one laptop, 1.9 with 50 ms of network delay each way.
