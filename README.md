# Tor6IA engine: one model, several machines

[![CI](https://github.com/tomDeprez/Tor6IA-engine/actions/workflows/ci.yml/badge.svg)](https://github.com/tomDeprez/Tor6IA-engine/actions/workflows/ci.yml)
[![Latest release](https://img.shields.io/github/v/release/tomDeprez/Tor6IA-engine)](https://github.com/tomDeprez/Tor6IA-engine/releases/latest)
[![License: MIT](https://img.shields.io/badge/license-MIT-green)](LICENSE)

**Run a language model that is too big for your machine, by sharing it with other machines.** The engine cuts a model into blocks of layers. Each machine holds the blocks its memory allows, and a token crosses them in order.

It is the engine behind [Tor6IA](https://github.com/tomDeprez/Tor6IA), the peer-to-peer network for open models. [Français](README.fr.md)

<p align="center"><img src="docs/blocks.svg" alt="The 49 layers of a model are cut into three blocks, one per machine; each token crosses the blocks in order" width="100%"></p>

## Try it in one command

You need one model file: a `.gguf`, or any model [Ollama](https://ollama.com) already downloaded.

**Windows**: download [tor6ia-engine-windows-amd64.exe](https://github.com/tomDeprez/Tor6IA-engine/releases/latest/download/tor6ia-engine-windows-amd64.exe), then:

```powershell
.\tor6ia-engine-windows-amd64.exe demo --model gemma4:12b
```

**Linux and macOS**:

```sh
curl -fsSL https://raw.githubusercontent.com/tomDeprez/Tor6IA-engine/main/install.sh | sh
~/.local/bin/tor6ia-engine demo --model gemma4:12b
```

The demo starts three block servers on your machine, splits the model across them, shows who holds which layers, and asks one question:

```
3 block servers started. Loading gemma4:12b across them...

BLOCK   MACHINE           LAYERS     SHARE OF THE MODEL
1       127.0.0.1:54470   0 to 16    34 %
2       127.0.0.1:54471   17 to 32   32 %
3       127.0.0.1:54472   33 to 48   32 %

Question: Explain in one sentence what a peer-to-peer network is.
Answer:   A peer-to-peer (P2P) network is a decentralized computing architecture where each
          computer (node) has equal status and can both share and receive resources or data
          directly with other nodes without the need for a central server.

3.3 tokens per second, with every layer computed by a block server.
```

Nothing else to install: the engine downloads the program that does the math, [llama.cpp](https://github.com/ggml-org/llama.cpp), and checks its checksum.

## Does it hold across the Internet?

Measured, not guessed. Gemma 4 12B, a laptop with no graphics card (Intel Core Ultra 7 265U), the model split between two block servers, and a relay that adds network delay in each direction ([proof/delayproxy](proof/delayproxy)):

| Setup | Speed |
|---|---|
| Not split: one process, 6 processors | 4.5 tokens per second |
| Split in two, same machine | 3.3 |
| Split in two, 10 ms each way (same city) | 2.9 |
| Split in two, 25 ms each way (same country) | 2.5 |
| Split in two, 50 ms each way (across a continent) | 1.9 |

Splitting costs little, and distance costs less than expected: about 40 % of the speed is left to the network at 50 ms each way.

## Use it on several machines

A model is cut into **blocks** of about one gigabyte. Every machine cuts it the same way, from the model's header alone:

```sh
tor6ia-engine plan --model gemma4:31b
```

```
gemma4:31b: 18.67 GB, 60 layers, and 1102 MB outside the layers
Plan read from the public registry, model not downloaded: 15.8 MB read, 0.08 % of the file.

16 blocks: ...
This machine has 17.2 GB of free memory: 11 of the 16 blocks advised.
```

On each machine that lends its memory, choose the blocks to hold. The machine downloads those blocks and nothing else, straight from the public registry:

```sh
tor6ia-engine block --model gemma4:31b --blocks 4-9 --listen 0.0.0.0:50052
```

It prints what to give to the head, like `192.168.1.20:50052=4-9`. On the machine that holds the conversation, which needs the whole model file on its disk:

```sh
tor6ia-engine head --model gemma4:31b --blocks 192.168.1.20:50052=1-3,192.168.1.21:50052=4-9,192.168.1.22:50052=10-16
```

The head places each layer on the machine that announced it, refuses a group that does not cover the whole model and says which blocks are missing, then serves an OpenAI-compatible API on `http://127.0.0.1:8090/v1`.

**Block servers are locked in.** A block server obeys another machine's computing orders. It runs in a lock where it cannot read your files, connect anywhere or start another program: an AppContainer on Windows, Landlock and seccomp on Linux, a sandbox profile on macOS. `tor6ia-engine jail-test` shows it on your machine. `--unlocked` removes the lock.

**Still between machines you trust.** A block server has no login and no encryption yet: any machine that can reach its port can give it work, and sees what passes through. Use a private network or a VPN.

**On a Mac**, block servers are tested on the processor: add `--device CPU`. A real Mac's graphics card has not been tested.

## Where it stands

Built and tested, on Windows, Linux and macOS at every release:

- A model split across block servers, each one locked in.
- Blocks: the same cut on every machine, the advice for this machine, the download of chosen blocks only.
- A head that assembles the machines that announced their blocks.
- Plugged into Tor6IA since its version 0.2.0.

Measured on one laptop with no graphics card, Gemma 4 12B:

| What | Result |
|---|---|
| Speed, three locked-in block servers | 3.2 tokens per second |
| Memory of a block server | 1.3 times the size of its blocks |
| Sent by the head when the blocks are already downloaded | 15 % of the model, instead of all of it |
| Traffic per token, two to eight machines | 350 to 690 KB |

Not built yet. The detailed plan, task by task, is in [ROADMAP.md](ROADMAP.md) (in French) and is updated as the work advances. In short:

1. **Relay.** Home machines cannot reach each other directly. The Tor6IA coordinator will carry the traffic between head and blocks.
2. **Several machines per block.** When one leaves, the next one that holds the same block takes over. Today a block server that stops ends the answer.
3. **Groups formed by the network**, with the blocks chosen in the Tor6IA interface.
4. **Payment per block.** Each machine paid for the share of the model it computes.

## How it works

- **llama.cpp does the math.** It already knows how to place layers on remote machines. The engine starts it, feeds it the right machines and reads back where each layer went. One build is pinned with its checksum: what you download is what was tested.
- **No second download.** A model Ollama has is used where it is.
- **One file.** No dependency outside the Go standard library.

```sh
go test ./...
go build -o dist/tor6ia-engine .
```

| File | Role |
|---|---|
| `llama.go` | pinned llama.cpp download, model lookup |
| `run.go` | block servers, head, demo |
| `blocks.go` | reads which machine holds which layers |
| `proof/delayproxy` | the relay that adds delay, used for the measurements |

## License

[MIT](LICENSE)
