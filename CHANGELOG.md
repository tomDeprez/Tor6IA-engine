# Changelog

Each `## vX.Y.Z` section becomes the notes of that release.

## v0.2.4

- `relay`: a small server that joins two machines unable to reach each other, as most machines behind a home router are. A holder started with `block --relay host:port` keeps a connection open towards it, and its line gains a second route. The head tries the direct address first and the relay after. The relay carries the link's own TLS: it sees who talks to whom and how much, never what. It caps the speed of each pair, its total speed and what it carries in a day.
- `head --json` says by which route each holder was reached.

## v0.2.3

- A block server listens on its own machine only. Other machines reach the engine through a link encrypted with TLS 1.3, where each side proves a key. The line a holder gives now ends with a secret (`192.168.1.20:50052=4-9#5f1c…`): only who has the line can use the blocks. `head --blocks` takes those lines.
- The memory of a block server is capped on Linux and macOS too, not only on Windows: past what its blocks need, it is stopped.
- On macOS a block server now stops with the engine, even when the engine is killed by force.
- `jail-test` also proves the memory cap, and that a block server does not outlive a killed engine.
- A block server that stops is started again. `abuse-test` plays a hostile head: noise, absurd sizes, more memory than allowed, a connection cut in the middle. After each attempt the block server must serve again.
- The block cache holds the blocks this machine holds and nothing else: blocks it held before are removed. Blocks already there are not read again at the next start. `clean` empties the cache, `clean --all` removes everything the engine installed.
- The demo and a plain `block` no longer write to the cache.

## v0.2.2

- A block server announces the address this machine really uses on the network. It could announce a dead interface (169.254.x.x).

## v0.2.1

- `block --json --exit-with-stdin`: a program that drives a block server gets clean JSON lines, and the block server stops when that program goes away.

## v0.2.0

- Block servers are locked in by default: no personal file, no outgoing connection, no other program. An AppContainer on Windows, Landlock and seccomp on Linux, a sandbox profile on macOS. `jail-test` proves it on your machine; `--unlocked` removes the lock.
- Blocks: `plan` cuts a model into blocks of about 1 GB and says how many this machine can hold, without downloading the model. `block --model M --blocks 4-9` downloads those blocks only and holds them. `head --blocks ADDR=1-3,ADDR=4-9` assembles the machines and refuses an incomplete group.
- The word table stays on the head: traffic per token drops from 1.3 MB to 0.35 MB.
- The head reserves memory for 8192 tokens of conversation instead of the model's maximum (`--ctx`): a block server now uses 1.3 times the size of its blocks.
- Programs the engine starts stop with it, even when it is killed by force.

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
