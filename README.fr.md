# Moteur Tor6IA : un modèle, plusieurs machines

**Fais tourner un modèle d'IA trop gros pour ta machine, en le partageant avec d'autres machines.** Le moteur découpe un modèle en blocs de couches. Chaque machine porte les blocs que sa mémoire permet, et chaque token les traverse dans l'ordre.

C'est le moteur de [Tor6IA](https://github.com/tomDeprez/Tor6IA), le réseau pair à pair pour les modèles ouverts. [English](README.md)

<p align="center"><img src="docs/blocks.svg" alt="Les 49 couches d'un modèle sont découpées en trois blocs, un par machine ; chaque token traverse les blocs dans l'ordre" width="100%"></p>

## Essayer en une commande

Il te faut un fichier de modèle : un `.gguf`, ou n'importe quel modèle déjà téléchargé par [Ollama](https://ollama.com).

**Windows** : télécharge [tor6ia-engine-windows-amd64.exe](https://github.com/tomDeprez/Tor6IA-engine/releases/latest/download/tor6ia-engine-windows-amd64.exe), puis :

```powershell
.\tor6ia-engine-windows-amd64.exe demo --model gemma4:12b
```

**Linux et macOS** :

```sh
curl -fsSL https://raw.githubusercontent.com/tomDeprez/Tor6IA-engine/main/install.sh | sh
~/.local/bin/tor6ia-engine demo --model gemma4:12b
```

La démo lance trois serveurs de blocs sur ta machine, répartit le modèle entre eux, affiche qui porte quelles couches, et pose une question :

```
BLOCK   MACHINE           LAYERS     SHARE OF THE MODEL
1       127.0.0.1:54470   0 to 16    34 %
2       127.0.0.1:54471   17 to 32   32 %
3       127.0.0.1:54472   33 to 48   32 %

3.3 tokens per second, with every layer computed by a block server.
```

Rien d'autre à installer : le moteur télécharge le programme qui fait les calculs, [llama.cpp](https://github.com/ggml-org/llama.cpp), et vérifie son empreinte.

## Est-ce que ça tient à travers Internet ?

Mesuré, pas supposé. Gemma 4 12B, un portable sans carte graphique (Intel Core Ultra 7 265U), le modèle coupé entre deux serveurs de blocs, et un relais qui ajoute du délai réseau dans chaque sens ([proof/delayproxy](proof/delayproxy)) :

| Montage | Vitesse |
|---|---|
| Non découpé : un processus, 6 processeurs | 4,5 tokens par seconde |
| Coupé en deux, même machine | 3,3 |
| Coupé en deux, 10 ms dans chaque sens (même ville) | 2,9 |
| Coupé en deux, 25 ms dans chaque sens (même pays) | 2,5 |
| Coupé en deux, 50 ms dans chaque sens (un continent) | 1,9 |

Le découpage coûte peu, et la distance moins que prévu.

## L'utiliser sur plusieurs machines

Un modèle est découpé en **blocs** d'environ un gigaoctet. Chaque machine le découpe de la même façon, à partir du seul en-tête du modèle :

```sh
tor6ia-engine plan --model gemma4:31b
```

```
gemma4:31b: 18.67 GB, 60 layers, and 1102 MB outside the layers
Plan read from the public registry, model not downloaded: 15.8 MB read, 0.08 % of the file.

16 blocks: ...
This machine has 17.2 GB of free memory: 11 of the 16 blocks advised.
```

Sur chaque machine qui prête sa mémoire, choisis les blocs à porter. La machine télécharge ces blocs et rien d'autre, directement depuis le registre public :

```sh
tor6ia-engine block --model gemma4:31b --blocks 4-9 --listen 0.0.0.0:50052
```

Elle affiche la ligne à donner à la tête, par exemple `192.168.1.20:50052=4-9#5f1c…`. La fin de la ligne est un secret : qui a la ligne peut faire calculer cette machine, personne d'autre. Sur la machine qui tient la conversation, qui doit avoir le fichier entier du modèle sur son disque :

```sh
tor6ia-engine head --model gemma4:31b --blocks "192.168.1.20:50052=1-3#5f1c…,192.168.1.21:50052=4-9#a07e…,192.168.1.22:50052=10-16#c3d9…"
```

La tête place chaque couche sur la machine qui l'a annoncée, refuse un groupe qui ne couvre pas tout le modèle en disant quels blocs manquent, puis sert une API compatible OpenAI sur `http://127.0.0.1:8090/v1`.

**Les serveurs de blocs sont enfermés.** Un serveur de blocs obéit aux ordres de calcul d'une autre machine. Il tourne dans une boîte où il ne peut ni lire tes fichiers, ni se connecter ailleurs, ni lancer un autre programme : AppContainer sous Windows, Landlock et seccomp sous Linux, profil de bac à sable sous macOS. Sa mémoire est plafonnée à ce que ses blocs demandent, et il s'arrête avec le moteur, même tué de force. `tor6ia-engine jail-test` montre tout cela sur ta machine. `--unlocked` retire la boîte.

**Entre personnes que tu as choisies.** Le serveur de blocs lui-même n'écoute que sur sa propre machine. Les autres machines joignent le moteur, par un lien chiffré en TLS 1.3 où chaque côté prouve une clé issue du secret de la ligne. Une machine sans la ligne n'obtient rien, et personne entre les deux ne peut lire ce qui passe. Les machines doivent quand même pouvoir se joindre : même réseau, VPN, ou port ouvert.

**Sur Mac**, les serveurs de blocs sont testés sur le processeur : ajoute `--device CPU`. La carte graphique d'un vrai Mac n'a pas été testée.

## Où on en est

Construit et testé, sur Windows, Linux et macOS à chaque version :

- Un modèle réparti entre des serveurs de blocs, chacun enfermé.
- Les blocs : le même découpage sur toutes les machines, le conseil pour cette machine, le téléchargement des seuls blocs choisis.
- Une tête qui assemble les machines qui ont annoncé leurs blocs.
- Branché dans Tor6IA depuis sa version 0.2.0.

Mesuré sur un portable sans carte graphique, Gemma 4 12B :

| Quoi | Résultat |
|---|---|
| Vitesse, trois serveurs de blocs enfermés | 3,2 tokens par seconde |
| Mémoire d'un serveur de blocs | 1,3 fois la taille de ses blocs |
| Envoyé par la tête quand les blocs sont déjà téléchargés | 15 % du modèle, au lieu de tout |
| Trafic par token, de deux à huit machines | 350 à 690 Ko |

Pas encore construit. Le plan détaillé, tâche par tâche, est dans [ROADMAP.md](ROADMAP.md) et il est mis à jour à chaque étape. En résumé :

1. **Relais.** Les machines derrière une box ne se joignent pas directement. Le coordinateur Tor6IA portera le trafic entre la tête et les blocs.
2. **Plusieurs machines par bloc.** Quand l'une part, la suivante qui porte le même bloc prend le relais. Aujourd'hui un serveur de blocs qui s'arrête coupe la réponse.
3. **Des groupes formés par le réseau**, avec les blocs choisis dans l'interface de Tor6IA.
4. **Paiement par bloc.** Chaque machine payée pour la part du modèle qu'elle calcule.

## Licence

[MIT](LICENSE)
