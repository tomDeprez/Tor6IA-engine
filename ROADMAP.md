# Plan de travail : des blocs choisis par chacun, pour le monde entier

But : chaque personne choisit un modèle et un nombre de blocs dans l'interface de Tor6IA, sa machine ne télécharge que ces blocs, et des inconnus du monde entier portent ensemble des modèles trop gros pour une seule machine.

Ce fichier est la référence. Il est mis à jour à chaque tâche terminée : case cochée, date, résultat mesuré, décision prise.

**État : plan validé par Tom le 2026-11-27. Travail en cours.**

**Prochaine tâche : 3.1**

Dernière mise à jour : 2026-11-27

## 1. Le résultat final, en une scène

Trois personnes qui ne se connaissent pas, dans trois pays, ouvrent Tor6IA sur des portables de 8 à 16 Go.

1. Chacune clique sur Gemma 4 31B. L'interface dit : « Pour ta machine, 3 blocs conseillés sur 20 ».
2. Chacune choisit son nombre de blocs. Sa machine télécharge seulement ces blocs.
3. L'interface affiche « blocs couverts : 14 sur 20, il manque 2 machines », puis « modèle disponible ».
4. Une quatrième personne pose une question à ce modèle. La réponse arrive, calculée par la chaîne de machines.
5. Une machine s'éteint au milieu : la réponse se termine quand même.
6. Chaque machine a reçu des crédits en proportion de ses blocs. Aucune n'a pu lire les fichiers d'une autre.

Le projet est fini quand cette scène est jouée pour de vrai et notée dans le journal, avec les vitesses mesurées.

## 2. Comment lire ce plan

- `[ ]` à faire, `[~]` en cours, `[x]` fait, `[!]` bloqué.
- Chaque tâche dit **Fini quand** : un résultat observé, jamais « le code est écrit ».
- **À vérifier** signale une hypothèse que personne n'a encore mesurée.
- Les durées sont des jours de travail, en fourchette. Elles sont corrigées après les essais de l'étape E.
- Une tâche terminée, c'est : tests écrits, pipeline verte sur Windows, Linux et macOS, ce fichier mis à jour, poussé sur GitHub.

## 3. Décisions

Validées par Tom le 2026-11-27, avec deux changements : le relais tourne sur le VPS existant (D4), et les essais à plusieurs vraies machines sont faits avec la machine de Tom et le VPS, puis par Tom avec des gens une fois tout fini (D10).

| N° | Décision | Ma recommandation | Pourquoi |
|---|---|---|---|
| D1 | Par quoi commencer | Tous les essais d'abord (étape E, 4 jours), puis construire | Chaque essai peut changer la route. Mieux vaut le savoir en 4 jours qu'en 4 semaines. |
| D2 | Premier résultat visible | Les blocs à la carte entre machines de confiance (jalon J1), avant le bac à sable | Tu vois ton idée marcher en une semaine, sans risque puisque ce sont des machines que tu choisis. |
| D3 | Ouverture aux inconnus | Interdite tant que le bac à sable (J2) n'est pas fini et testé | Un serveur de blocs sans protection ouvert au monde mettrait chaque utilisateur en danger. |
| D4 | Serveur du relais | **Changé par Tom : le VPS existant.** Service séparé, mémoire, processeur et débit plafonnés | Le VPS porte 8 sites : le relais ne doit jamais pouvoir les ralentir. Le débit nécessaire est mesuré en E4. |
| D5 | Windows sans bac à sable possible | Accepter que porter des blocs sous Windows passe alors par WSL | Seulement si l'essai E1 échoue. Utiliser un modèle reste possible partout. |
| D6 | Vie privée en mode blocs | Accepter que les machines de la chaîne voient passer les données du calcul, et l'écrire dans l'interface | Le chiffrement de bout en bout actuel ne peut pas couvrir un calcul réparti. Le mode « un modèle entier sur une machine » garde son chiffrement. |
| D7 | Sécurité des réponses | Deux machines au moins par bloc avant d'annoncer un modèle disponible | Sans ça, une machine qui part coupe toutes les réponses. |
| D8 | Premier modèle en blocs | Gemma 4 31B seul au début, puis les autres | Un seul modèle à régler, le plus gros du catalogue, celui que presque personne ne tient seul. |
| D9 | Taille d'un bloc | Environ 1 Go, toujours des couches entières | Assez petit pour une machine de 8 Go, assez gros pour ne pas multiplier les sauts réseau. À confirmer en E4. |
| D10 | Machines d'essai | **Changé par Tom : sa machine et le VPS pour les essais, puis il teste lui-même avec des gens quand tout est fini** | Le VPS est une vraie seconde machine à travers Internet. Les essais à trois machines et plus restent à faire par Tom. |
| D11 | Le cache de 22 Go déjà sur ton disque | Le garder jusqu'à la tâche 2.5, qui lui met une limite | Il vient de mes tests du 2026-11-27 (`%LOCALAPPDATA%\\llama.cpp\\rpc`). Il resservira aux essais. Je le supprime si tu préfères. |

## 4. Les jalons : ce que tu vois, et quand

| Jalon | Ce que tu peux faire à la fin | Version publiée | Durée | Cumul |
|---|---|---|---|---|
| E. Essais | Lire un rapport : chaque hypothèse du plan est confirmée ou remplacée | aucune | **fait** | |
| J1. Blocs à la carte entre amis | Choisir un modèle et un nombre de blocs dans l'interface, ne télécharger que ça, voir les blocs couverts | Tor6IA 0.3.0 | 5 à 7 j | 11,5 j |
| J2. Bac à sable | Porter un bloc pour quelqu'un sans qu'il puisse toucher à ta machine | moteur 0.3.0 | 2 à 3 j restants | 17,5 j |
| J3. Le réseau forme les groupes | Des machines derrière des box, sans se connaître, portent un modèle ensemble | Tor6IA 0.4.0, sur invitation | 7,5 à 11,5 j | 29 j |
| J4. Paiement par bloc | Chaque machine payée pour sa part, une machine qui triche est exclue | Tor6IA 0.5.0 | 4 à 6 j | 35 j |
| J5. Ouverture | La scène du chapitre 1, avec de vrais volontaires | Tor6IA 1.0.0 | 3 à 5 j | 40 j |

Total : 26,5 à 40 jours de travail. Chaque jalon se termine par une démonstration que tu peux refaire toi-même.

## 5. Ce qui existe déjà (phase 0, faite le 2026-11-27)

- [x] 0.1 Un modèle réparti entre plusieurs serveurs de blocs avec llama.cpp. *Gemma 4 12B en 3 blocs, 3,3 tokens/s sur un portable sans carte graphique.*
- [x] 0.2 Coût du délai réseau mesuré. *Coupé en deux : 3,3 tokens/s à 0 ms, 2,9 à 10 ms, 2,5 à 25 ms, 1,9 à 50 ms dans chaque sens.*
- [x] 0.3 Une commande pour essayer, testée sur Windows, Linux et macOS à chaque version.
- [x] 0.4 Branché dans Tor6IA 0.2.0 : groupe réglé à la main, découpage affiché, paiement à la machine de tête.
- [x] 0.5 Tor6IA 0.2.1 : une demande attend une machine, chaque modèle affiche ses demandes en attente.
- [x] 0.6 Le plan d'un modèle se lit sans le télécharger. *Gemma 4 12B, 7,38 Go : l'en-tête de 15,8 Mo donne les 48 couches et leur taille exacte, et le registre public sert n'importe quel morceau du fichier à la demande.*

Faits vérifiés qui servent au plan :

- `llama-gguf-split` est fourni avec llama.cpp.
- llama.cpp sait répartir par parts (`--tensor-split`) et placer des couches précises sur une machine précise (`--override-tensor`).
- Le cache d'un serveur de blocs est un dossier de fichiers nommés par empreinte : `llama.cpp/rpc`.
- Un serveur de blocs injoignable n'est pas signalé par llama.cpp : ses couches sont chargées sur la tête. Le moteur le détecte.
- Sur les Mac virtuels de GitHub, la carte graphique arrête un serveur de blocs (« unsupported op »). Le processeur fonctionne.

## 6. Étape E : les essais

Aucun produit fini ici. Chaque essai répond à une question par une mesure, dans `proof/`. À la fin, les durées et la route des jalons sont corrigées, et tu revalides.

- [x] **E1 Un serveur de blocs peut-il être enfermé, sans droits administrateur, sur chaque système ?** *Fait le 2026-11-27 : oui, sur les trois.*
  Linux : bubblewrap et seccomp. Windows : AppContainer avec jeton restreint. macOS : `sandbox-exec`. Repli commun : serveur de blocs compilé en WebAssembly.
  **Fini quand** : sur chaque système, un serveur de blocs enfermé sert un vrai modèle, et un programme de test lancé à sa place échoue à lire un fichier personnel, à ouvrir une connexion sortante et à lancer un autre programme.
  **Décide** : la technique de J2, et la décision D5.
  **Résultat** : la sonde `tor6ia-engine jail-test` lit un fichier personnel, se connecte à Internet et lance un programme quand elle est libre ; enfermée, les trois sont refusés, sur Windows 11, Windows Server, Debian 12 (noyau 6.1), Ubuntu et macOS, sans droits administrateur. La démo tourne enfermée sur les trois systèmes dans la pipeline.
  - Windows : AppContainer sans aucune capacité, plus un « job » limité à deux processus. Un AppContainer ne peut pas être joint par le réseau, même depuis la machine : un petit pont tourne dedans et fait passer les connexions par son entrée et sa sortie (`mux.go`). WSL n'est pas nécessaire : **D5 est sans objet.**
  - Linux : Landlock pour les fichiers, filtre seccomp pour les connexions, `no_new_privs`. Fonctionne là où les espaces de noms sans privilèges sont interdits.
  - macOS : profil `sandbox-exec`.
  - Pièges rencontrés : sur Windows Server un programme enfermé ne peut pas ouvrir `NUL`, il faut lui donner une entrée explicite. Sur Linux, llama.cpp demande `libgomp1`, absent des systèmes minimaux.
  - Coût mesuré sur Windows, Gemma 4 E2B en deux blocs : 2,1 tokens/s enfermé contre 3,1 libre, soit 32 % de perte, à la limite fixée en 2.6.
- [x] **E2 Un serveur de blocs peut-il charger ses couches depuis son disque ?** *Fait le 2026-11-27 : oui pour 85 % du modèle, pas pour le reste.*
  Remplir le cache à l'avance avec les morceaux pris dans le fichier public, puis charger le modèle.
  **Fini quand** : un modèle se charge sur deux serveurs de blocs avec moins de 1 % de ses octets envoyés par la tête, mesuré avec `proof/delayproxy`. La règle de calcul des noms de fichiers du cache est écrite ici.
  **Si ça échoue** : fichier GGUF partiel reconstitué sur chaque machine avec `llama-gguf-split`.
  **Résultat**, Gemma 4 12B (7,4 Go) sur deux serveurs de blocs, octets mesurés avec `proof/delayproxy` :

  | Cache des serveurs de blocs | Envoyé par la tête | Chargement |
  |---|---|---|
  | vide | 7 025 Mo | 205 s |
  | prérempli avec les couches | 1 851 Mo | 112 s |
  | prérempli avec les couches et ce qui est hors couches | 1 063 Mo | 83 s |

  - Règle du cache, confirmée : un fichier par tenseur dans `<LLAMA_CACHE>/rpc/`, nommé par l'empreinte FNV-1a 64 bits de ses octets, écrite en 16 chiffres hexadécimaux. `tor6ia-engine fetch` la reproduit : les fichiers obtenus depuis le disque et depuis le registre public sont identiques.
  - Le critère « moins de 1 % » n'est **pas** atteint : il reste 15 %. llama.cpp ne prend dans son cache que les tenseurs de plus de 10 Mo, seuil fixé dans son code. Les petits tenseurs sont toujours envoyés par la tête : 1,06 Go pour ce modèle.
  - Le gros tenseur hors couches (826 Mo ici) est placé avec les dernières couches : la machine qui porte la fin du modèle doit le précharger aussi (`fetch --outside`).
  - Décision : on continue avec llama.cpp tel quel. Le seuil devient la tâche 3.9, à faire seulement si le trafic du relais le demande.
  - Depuis le registre public, pour un modèle absent du disque : 3 couches de Gemma 4 31B, 0,84 Go en 65 s, 5 % du modèle. Les requêtes reprennent après une coupure (la première tentative s'était arrêtée à 41 %).
- [x] **E3 Peut-on imposer quelle machine porte quelles couches ?** *Fait le 2026-11-27 : oui.*
  **Résultat**, Gemma 4 12B (49 couches en comptant la sortie) sur trois serveurs de blocs, avec `tor6ia-engine head --shares` (`--tensor-split` de llama.cpp) :

  | Parts demandées | Couches obtenues |
  |---|---|
  | aucune (selon la mémoire libre) | 17, 16, 16 |
  | 1, 3, 2 | 9, 24, 16 |
  | 5, 1, 6 | 21, 4, 24 |

  Les couches suivent les parts, en tranches contiguës, dans l'ordre des machines. Donner comme parts le nombre de couches voulu par machine fixe donc le découpage.
- [x] **E4 Combien d'octets et d'allers-retours par token entre deux blocs ?** *Fait le 2026-11-27.*
  **Résultat**, Gemma 4 12B, octets par token généré, tête et blocs confondus :

  | Réglage | 2 blocs | 4 blocs | 8 blocs |
  |---|---|---|---|
  | llama.cpp tel quel | 1 300 Ko | 1 410 Ko | |
  | table des mots gardée sur la tête | 346 Ko | 460 Ko | 688 Ko |

  - Tel quel, le dernier bloc renvoie un score par mot du vocabulaire : 262 144 mots, 1,05 Mo par token. Le moteur garde maintenant la table des mots sur la tête (`--override-tensor`), ce qui divise le trafic par 3 à 4.
  - Reste environ 230 Ko par token, plus 57 Ko par bloc. À 2 tokens par seconde, une conversation fait passer 0,7 à 1,4 Mo par seconde.
  - Allers-retours : environ un par machine et par token (déduit de 0.2 : 100 ms de délai aller-retour ajoutent 209 ms par token pour deux blocs).
  - **Conséquence pour le relais (D4)** : sur le VPS partagé, quelques conversations en même temps au plus. Le relais aura un débit plafonné, et la connexion directe (3.3) devient prioritaire.
  - **Conséquence pour D9** : ce qui ralentit, c'est le nombre de machines dans la chaîne, pas le nombre de blocs. Une machine qui porte plusieurs blocs voisins ne compte que pour un saut. Limite visée : 6 machines par chaîne.
- [x] **E5 De quoi la machine de tête a-t-elle besoin ?** *Fait le 2026-11-27.*
  **Résultat**, Gemma 4 12B (fichier de 7,4 Go) sur deux serveurs de blocs :
  - La tête lit chaque tenseur pour en calculer l'empreinte : il lui faut **le fichier entier sur son disque**. Sa mémoire propre est faible (la table des mots, 0,8 Go) ; le reste de ses 7,2 Go mesurés est le fichier gardé en cache par le système, qui le reprend au besoin.
  - Chaque serveur de blocs occupe **1,8 fois la taille de ses couches** : 6,1 et 5,8 Go pour deux moitiés de 3,3 Go.
  - Règle pour le conseil (1.4) : nombre de blocs = (mémoire libre moins 2 Go) divisé par 1,8 fois la taille d'un bloc.
- [x] **E6 Rapport des essais.** *Fait le 2026-11-27.*

  | Hypothèse du plan | Verdict | Suite |
  |---|---|---|
  | Un serveur de blocs peut être enfermé sans droits administrateur | vrai sur les trois systèmes | fait (E1, 2.1) |
  | Chaque machine peut ne télécharger que sa part | vrai : 5 % du modèle pour 3 couches sur 60 | `plan` et `fetch` existent |
  | La tête n'envoie alors presque plus rien | faux : il reste 15 % | 3.9, si le relais l'exige |
  | On peut choisir quelle machine porte quoi | vrai | `head --shares` |
  | Le trafic par token est faible | faux tel quel (1,3 Mo), acceptable une fois corrigé (0,35 à 0,7 Mo) | corrigé dans le moteur |
  | La tête peut être une petite machine | à moitié : peu de mémoire, mais tout le fichier sur le disque | à dire dans l'interface |

  Durées corrigées : J2 passe à 2 à 3 jours restants (le bac à sable est fait). Les autres jalons gardent leur fourchette. Le plan n'a pas besoin de changer de route.

## 7. Jalon J1 : les blocs à la carte, entre machines de confiance

Tout ce qui se voit dans l'interface, sans encore ouvrir aux inconnus. Dépend de E2, E3, E5.

- [x] **1.1 Le plan d'un modèle dans le moteur.** *Blocs ajoutés le 2026-11-27 : Gemma 4 12B = 6 blocs de 972 à 1 139 Mo, Gemma 4 31B = 16 blocs.* *Fait le 2026-11-27 : `tor6ia-engine plan --model gemma4:31b` décrit un modèle jamais téléchargé (18,67 Go, 60 couches de 257 à 314 Mo, 1 102 Mo hors couches) en lisant 15,8 Mo, soit 0,08 % du fichier. Plan identique depuis le disque et depuis le registre. Reste : le regroupement en blocs d'environ 1 Go et le test sur les 5 modèles du catalogue.*
  `tor6ia-engine plan --model gemma4:31b` lit l'en-tête par morceaux depuis le registre et donne les couches, leur taille, et le découpage en blocs d'environ 1 Go. Sortie lisible et sortie JSON.
  Fichiers : `gguf.go` (lecture de l'en-tête), `plan.go`.
  **Fini quand** : le plan d'un modèle jamais téléchargé s'affiche en ayant reçu moins de 20 Mo, identique à celui lu dans le fichier complet. Testé sur les 5 modèles du catalogue.
- [x] **1.2 Télécharger seulement ses blocs.** *Adressage par blocs ajouté le 2026-11-27 (`fetch --blocks`, et `block --model M --blocks 4-9` qui télécharge avant de servir).* *Fait le 2026-11-27 : `tor6ia-engine fetch --model M --layers 3-5` prend les morceaux sur le disque si le modèle y est, sinon au registre, avec reprise. Reste : l'adressage par blocs et la progression pour l'interface.*
  `tor6ia-engine fetch --model gemma4:31b --blocks 3-5` prend les morceaux du fichier public, avec reprise après coupure et progression.
  À vérifier : le registre ne donne qu'une empreinte pour le fichier entier. L'intégrité d'un morceau repose sur HTTPS vers le registre officiel, sauf si le cache vérifie l'empreinte de chaque couche (E2).
  **Fini quand** : une machine qui porte 3 blocs sur 20 a téléchargé 3 vingtièmes du modèle, pas plus, et sert ces blocs.
- [x] **1.3 Un serveur de blocs qui annonce et tient un nombre de blocs.** *Fait le 2026-11-27. Mesuré avec un contexte de 8 192 tokens : 1,4 Go pour 1 bloc de 1,07 Go, 2,6 Go pour 2, 3,9 Go pour 3, soit 1,3 fois la taille. Plafond de mémoire posé à 1,5 fois la taille plus 0,5 Go (appliqué sous Windows). Piège : par défaut llama.cpp réservait la mémoire pour 262 144 tokens de contexte, d'où les 1,8 de E5 et un premier échec « failed to allocate compute buffers ».*
  `tor6ia-engine block --model gemma4:31b --blocks 3-5` : il ne charge que ça, avec un plafond de mémoire.
  **Fini quand** : la mémoire utilisée correspond aux blocs choisis à 15 % près, mesurée.
- [~] **1.4 Le conseil.** *Fait le 2026-11-27 : `plan` donne le nombre de blocs conseillé d'après la mémoire libre (11 sur 16 de Gemma 4 31B sur un portable avec 17 Go libres), et l'interface de Tor6IA l'affiche avec la taille à télécharger et la mémoire utilisée. Reste : la vérification sur des machines de 8 et 16 Go, à faire par Tom pendant son test.*
  Nombre de blocs conseillé à partir de la mémoire libre mesurée et de la carte graphique, avec la prudence déjà utilisée dans Tor6IA (1,5 fois la taille, 2 Go laissés au système).
  **Fini quand** : sur trois machines de mémoires différentes (8, 16 et 32 Go), le nombre conseillé se charge sans que la machine se mette à ramer.
- [x] **1.5 La tête assemble selon les blocs annoncés.** *Fait le 2026-11-27 : annonces 1, 2-3 et 4-6 données dans le désordre, couches placées 0-7, 8-23, 24-48. Groupe incomplet refusé : « blocks 2 to 3 of 6 are held by no machine ».*
  Elle reçoit « machine A : blocs 1 à 3, machine B : blocs 4 à 9 » et place les couches exactement ainsi (E3). Elle refuse de démarrer s'il manque un bloc et dit lequel.
  **Fini quand** : le découpage affiché est celui demandé, et un bloc manquant donne « il manque les blocs 10 à 12 ».
- [x] **1.6 L'écran « Porter des blocs » dans Tor6IA.** *Fait le 2026-11-27, Tor6IA 0.3.0. L'interface lit le plan sans rien télécharger (« 16 blocs dans ce modèle. Pour ta machine : 8 conseillés »), télécharge les blocs choisis avec une barre de progression, puis affiche « Gemma 4 12B : 2 blocs sur 6, blocs 2 à 3 : chargés et en service », une barre des blocs du modèle et la ligne à donner au groupe. Testé en vrai : blocs 2-3 portés par Tor6IA, deux autres porteurs, groupe formé par l'interface, réponse reçue ; nœud tué de force, aucun processus du moteur ne reste ; redémarrage, les blocs sont repris. Deux défauts trouvés et corrigés : l'adresse annoncée était celle d'une carte réseau morte (169.254.x.x), et la reprise au démarrage lançait un second serveur de blocs.*
  Choix du modèle, « pour ta machine : 3 blocs conseillés sur 20 », choix de 1 au maximum possible, téléchargement bloc par bloc, puis « 3 blocs chargés, 3 en service ». La barre du modèle montre les blocs couverts et ceux qui manquent.
  Fichiers : `ui/index.html`, `split.go` dans Tor6IA.
  **Fini quand** : une personne qui n'a jamais vu le projet porte des blocs sans aide, et l'affichage correspond à ce que le moteur rapporte.
- [~] **1.7 Démonstration J1.** *Faite le 2026-11-27 avec Gemma 4 12B : trois porteurs enfermés (1, 2 et 3 blocs) reliés par des liens chiffrés répondent à 3,3 tokens/s sur une seule machine, et le groupe se forme depuis l'interface de Tor6IA. Tor6IA 0.3.0 publiée. La démonstration avec Gemma 4 31B est impossible sur une machine seule (les trois porteurs y partagent la même mémoire) : elle fait partie du test final de Tom.*
  **Fini quand** : sur ta machine, trois porteurs de blocs choisis à la main (2, 5 et 13 blocs) font répondre Gemma 4 31B, modèle qu'elle ne tient pas seule en mémoire. Vitesse notée ici. Tor6IA 0.3.0 publiée.

## 8. Jalon J2 : le bac à sable

Le verrou de l'ouverture. Dépend de E1.

- [x] **2.1 Le serveur de blocs enfermé, sur les trois systèmes.** *Fait le 2026-11-27 avec E1 : `tor6ia-engine block` et la démo enferment par défaut, `--unlocked` rend l'ancien comportement. Les évasions sont testées dans la pipeline sur les trois systèmes.*
  Avec la technique choisie en E1. Il ne voit que son dossier de blocs. Aucun autre fichier, aucun autre programme, aucune connexion sortante.
  **Fini quand** : les trois tentatives d'évasion de E1 échouent dans la pipeline, sur Windows, Linux et macOS.
- [x] **2.2 Limites de ressources et arrêt garanti.** *Fait le 2026-11-27, moteur 0.2.3. Mémoire plafonnée à 1,5 fois la taille des blocs portés plus 1 Go, sur les trois systèmes : limite du job sous Windows, `RLIMIT_DATA` sous Linux (mesuré dans WSL avec Llama 3.2 3B : 1 321 Mo utilisés pour un bloc de 757 Mo, limite 1 647 Mo), gardien sous macOS. Processeur : la moitié des cœurs. Le serveur de blocs s'arrête avec le moteur même tué de force, sur les trois systèmes (sous macOS par le même gardien). `jail-test` le prouve dans la pipeline : « use 1 GB of memory : refused » ou « stopped », « kill the engine by force : block server stopped with it ».*
  Plafond de mémoire et de processeur. Le serveur de blocs s'arrête avec le moteur, même tué de force (Job Object sous Windows, `PR_SET_PDEATHSIG` sous Linux).
  **Fini quand** : moteur tué de force, aucun de ses processus ne reste. Corrige la limite connue du README.
- [x] **2.3 Aucune écoute sur le réseau.** *Fait le 2026-11-27, moteur 0.2.3. Le serveur de blocs n'écoute plus que sur 127.0.0.1. Les autres machines joignent le moteur par un lien TLS 1.3 où chaque côté prouve une clé issue du secret de la ligne (`192.168.1.20:50052=4-9#secret`). Vérifié : depuis l'adresse réseau les trois ports des serveurs de blocs sont fermés ; un inconnu qui envoie un ordre sur le port du lien n'obtient rien ; une tête avec un mauvais secret est refusée (« the other side does not hold the key of this group ») ; le groupe répond toujours, à la même vitesse (3,3 tokens/s avant et après). Le port du lien reste ouvert pour les connexions directes : il disparaît pour les machines qui passent par le relais (3.2).*
  Le serveur de blocs n'écoute que sur la machine elle-même. Le moteur parle au reste du monde et vérifie la clé de celui qui est en face.
  **Fini quand** : un balayage de ports depuis une autre machine ne trouve rien, et le groupe fonctionne toujours.
- [x] **2.4 Essais d'abus.** *Fait le 2026-11-27, moteur 0.2.3 : `tor6ia-engine abuse-test` joue une tête hostile contre un serveur de blocs enfermé avec 512 Mo. Neuf attaques : bruit, tailles absurdes, ordre inconnu, calcul fait de bruit, calcul plus gros que sa mémoire, bloc de mémoire géant, plus de mémoire que permis, tenseur coupé au milieu. Après chacune le serveur doit répondre à nouveau. Windows et Linux : tout refusé, le serveur ne tombe pas. macOS : une attaque le fait arrêter par le plafond de mémoire, il est relancé. Le moteur relance un serveur de blocs qui s'arrête et le signale ; au-delà de 20 arrêts par minute il abandonne. Dans la pipeline sur les trois systèmes.*
  Ordres malformés, tailles absurdes, demande de mémoire géante, connexion coupée au milieu.
  **Fini quand** : le serveur de blocs plante ou refuse, le bac à sable tient, le moteur le signale et le relance. Ces essais tournent dans la pipeline.
- [x] **2.5 Le cache de blocs sous contrôle.** *Fait le 2026-11-27, moteur 0.2.3. Au démarrage d'un porteur, tout ce qui n'est pas ses blocs est supprimé du cache. Dix changements de modèle et de blocs d'affilée : le cache n'a jamais dépassé la taille des blocs portés (par exemple 2 625 Mo pour 3 174 Mo de blocs). Un index évite de relire des blocs déjà là : redémarrage prêt en 1 s. `clean` vide le cache, `clean --all` retire tout ce que le moteur a installé ; Tor6IA les appelle quand on arrête de porter et à la désinstallation. La démo n'écrit plus dans le cache. L'ancien cache orphelin de 22 Go a été supprimé de la machine d'essai.*
  Taille limitée, blocs inutilisés supprimés, tout effacé quand on arrête de porter un modèle ou qu'on désinstalle. Aujourd'hui il grossit sans limite (22 Go après une journée d'essais).
  **Fini quand** : le dossier ne dépasse jamais la taille des blocs portés, vérifié après dix changements de modèle.
- [x] **2.6 Perte de vitesse mesurée.** *Fait le 2026-11-27, trois mesures par cas. Windows, Gemma 4 12B sur 3 serveurs de blocs : enfermé 3,0 / 2,5 / 3,2 tokens/s, libre 2,8 / 3,5 / 3,0, soit 6 % d'écart, dans le bruit de la mesure. Linux (WSL), Llama 3.2 3B sur 2 serveurs : enfermé 14,2 / 14,0 / 14,4, libre 14,5 / 14,5 / 14,2, soit 1,4 %. Les 32 % mesurés plus tôt sous Windows venaient de la table des mots envoyée à chaque token, corrigée depuis (E4). macOS : pas de machine pour une vraie mesure ; à relever pendant le test de Tom. Le choix de E1 est confirmé.*
  **Fini quand** : tokens par seconde avec et sans bac à sable, sur les trois systèmes, écrits ici. Plus de 30 % de perte rouvre le choix de E1.

## 9. Jalon J3 : le réseau forme les groupes

Dépend de J1, J2 et E4. Rien ici n'est activé pour le public : sur invitation seulement.

- [ ] **3.1 Le serveur du relais.** 0,5 j.
  Choisi d'après E4, loué, déployé avec `deploy.ps1` adapté. Pas le VPS actuel.
  **Fini quand** : coût mensuel et débit garanti écrits ici, relais joignable en HTTPS.
- [ ] **3.2 Le relais.** 2 à 3 j.
  Chaque machine garde une connexion sortante vers le relais, qui fait passer les données entre la tête et les blocs. Identité par la clé du compte, débit plafonné par compte, aucune donnée gardée.
  Fichiers : `relay.go` dans le moteur, un service `tor6ia-relay` sur le serveur.
  **Fini quand** : un groupe de trois machines derrière trois box différentes répond, sans aucun port ouvert.
- [ ] **3.3 Connexion directe quand c'est possible.** 1 à 2 j.
  Même réseau local ou port joignable : les machines se parlent sans passer par le relais.
  **Fini quand** : deux machines du même réseau échangent en direct, vérifié par le compteur d'octets du relais qui reste à zéro.
- [ ] **3.4 Registre des blocs dans le coordinateur.** 1 à 1,5 j.
  Qui porte quels blocs de quel modèle. Le coordinateur lit lui-même le plan de chaque modèle du catalogue (16 Mo) : il ne stocke aucun modèle, et un nouveau modèle ne demande aucune mise à jour des programmes.
  **Fini quand** : l'interface affiche « blocs couverts : 14 sur 20, il manque 2 machines » en direct, et le compte reste juste quand des machines arrivent et partent.
- [ ] **3.5 Le réseau propose à chacun les blocs qui manquent.** 0,5 j.
  Quand tu choisis de porter 3 blocs, le coordinateur te donne les 3 les moins couverts.
  **Fini quand** : dix machines qui choisissent chacune 2 blocs couvrent les 20 blocs une fois, pas dix fois les deux premiers.
- [ ] **3.6 Assemblage d'une chaîne.** 1 à 1,5 j.
  Pour une demande, le coordinateur choisit une machine par bloc, la plus rapide disponible, et une tête (E5).
  **Fini quand** : une demande sur un modèle que personne ne porte en entier reçoit sa réponse, calculée par une chaîne que le réseau a formée seul.
- [ ] **3.7 Reprise quand une machine part.** 1 à 2 j.
  Un modèle n'est annoncé disponible que si chaque bloc a deux porteurs (D7). Une machine part : la chaîne se reforme avec l'autre porteur.
  **Fini quand** : machine débranchée au milieu d'une réponse, la réponse se termine, délai de reprise mesuré.
- [ ] **3.9 Faire accepter les petits tenseurs par le cache.** 1 à 2 j, seulement si 3.2 montre que c'est nécessaire.
  llama.cpp envoie toujours les tenseurs de moins de 10 Mo (15 % de Gemma 4 12B, voir E2). Le corriger demande notre propre compilation de llama.cpp avec un seuil plus bas, pour chaque système.
  **Fini quand** : le chargement d'un modèle entièrement préchargé fait passer moins de 1 % de ses octets par le relais.
- [ ] **3.8 Démonstration J3.** 0,5 j.
  **Fini quand** : au moins trois vraies machines (D10), sur au moins deux connexions Internet, portent un modèle qu'aucune ne tient seule. Vitesse mesurée et écrite ici. Tor6IA 0.4.0 publiée, blocs publics sur invitation.

## 10. Jalon J4 : paiement et triche, par bloc

Dépend de J3.

- [ ] **4.1 Paiement partagé dans le registre.** 1 à 2 j.
  Une réponse paie plusieurs comptes, chacun selon ses couches et les tokens calculés. Toujours signé par celui qui paie, toujours à somme nulle.
  Fichiers : `ledger.go`, `proto.go` dans Tor6IA.
  **Fini quand** : `tor6ia audit` vérifie un registre contenant des paiements partagés, et refuse une part modifiée.
- [ ] **4.2 Détecter une machine qui renvoie n'importe quoi.** 2 à 3 j.
  Deux porteurs du même bloc reçoivent la même entrée : leurs sorties doivent concorder. Tests à réponse connue passés par une chaîne précise.
  **Fini quand** : une machine modifiée pour renvoyer du bruit est exclue en moins de dix demandes, sans exclure une machine honnête sur mille demandes.
- [ ] **4.3 Ce que voit chaque machine, écrit clairement.** 0,5 j.
  Avant de poser une question à un modèle en blocs, l'interface dit que les machines de la chaîne voient passer les données du calcul (D6).
  **Fini quand** : le message est dans l'interface et la documentation, et le mode « groupe de confiance » reste disponible.
- [ ] **4.4 Démonstration J4.** 0,5 j.
  **Fini quand** : une réponse sur un modèle en blocs, les crédits de chaque porteur vérifiés à la main contre ses couches. Tor6IA 0.5.0 publiée.

## 11. Jalon J5 : ouverture

Dépend de tout ce qui précède.

- [ ] **5.1 Essai avec des volontaires.** 2 à 3 j.
  **Fini quand** : dix machines de volontaires (D10) portent un modèle pendant une semaine. Pannes, vitesses et débit du relais notés ici.
- [ ] **5.2 Limites et réglages par défaut.** 0,5 à 1 j.
  **Fini quand** : débit, mémoire et nombre de blocs par défaut sont fixés à partir des mesures de 5.1.
- [ ] **5.3 Activé pour tous.** 0,5 à 1 j.
  **Fini quand** : une installation neuve de Tor6IA propose de porter des blocs, la pipeline le teste sur les trois systèmes, et la scène du chapitre 1 est jouée et notée dans le journal. Tor6IA 1.0.0 publiée.

## 12. Règles de travail

- **Un essai avant chaque construction.** Aucune phase ne démarre sur une hypothèse non mesurée.
- **Rien d'exposé au public avant J2.** Jusque-là, les blocs ne tournent qu'entre machines choisies à la main.
- **Le réseau actuel ne casse jamais.** Les blocs sont un second moteur à côté d'Ollama. Un modèle entier sur une machine continue de marcher tel quel, avec son chiffrement.
- **Chaque version du moteur est épinglée par empreinte dans Tor6IA.** Une nouvelle version n'atteint les utilisateurs qu'après ses propres tests.
- **Ce fichier d'abord.** Une tâche finie est cochée ici avec sa mesure avant de passer à la suivante.
- **Les mauvaises nouvelles tout de suite.** Un essai qui échoue est écrit dans le journal le jour même, avec la route de repli.

## 13. Ce dont j'ai besoin de toi

| Quand | Quoi |
|---|---|
| Quand tout est fini | Tester avec des gens et dire ce qui marche ou non (D10) |
| N'importe quand | Changer le mot de passe root du VPS, exposé dans une conversation |

## 14. Ce qui peut faire dérailler le plan

| Risque | Signe | Réponse prévue |
|---|---|---|
| Pas de bac à sable sans droits administrateur sous Windows | E1 échoue sous Windows | Porter des blocs sous Windows passe par WSL (D5) |
| Le cache ne permet pas de précharger les couches | E2 échoue | Fichier GGUF partiel reconstitué par `llama-gguf-split` |
| On ne peut pas imposer le placement des couches | E3 échoue | Blocs de taille égale, attribués dans l'ordre d'arrivée des machines |
| Trop d'octets par token pour un relais abordable | E4 donne un chiffre trop haut | Connexion directe obligatoire, le relais ne sert qu'à se trouver |
| La tête doit avoir tout le fichier | E5 le confirme | Seules les machines avec assez de disque lancent une conversation, les autres portent des blocs |
| Trop lent à travers plusieurs pays | 3.8 sous 1 token par seconde | Chaînes formées par proximité, nombre de blocs limité par modèle |
| Le registre public cesse de servir des morceaux de fichier | 1.2 reçoit le fichier entier | Autre source publique du même fichier, vérifiée par son empreinte |
| llama.cpp change son protocole | une nouvelle version casse les essais | La version reste épinglée, montée de version testée à part |

## 15. Journal

- 2026-11-27 : **J1 et J2 terminés. Tor6IA 0.3.0 et moteur 0.2.3 publiés, pipelines vertes sur Windows, Linux et macOS.** Chacun peut porter des blocs depuis l'interface. Le serveur de blocs est enfermé, plafonné en mémoire, invisible du réseau, relancé s'il tombe, et son cache ne dépasse pas ses blocs. Les groupes sont encore formés à la main : c'est J3 qui les fait former par le réseau. Décision prise en route : la ligne d'un porteur porte un secret, et les deux clés du lien en sont tirées, ce qui évite d'échanger des clés à la main.
- 2026-11-27 : phase 0 terminée. Premier plan écrit.
- 2026-11-27 : essai 0.6 fait sur une question de Tom (« peut-on découper sans télécharger ? »). Oui : plan exact dans l'en-tête, morceaux servis à la demande. Plus aucun hébergement de modèles à prévoir.
- 2026-11-27 : **J1 côté moteur fait (1.1 à 1.5).** Les blocs existent de bout en bout en ligne de commande. Trois corrections trouvées en testant : droits du dossier de cache en concurrence, contexte par défaut de 262 144 tokens, règle mémoire revue à 1,3. Moteur 0.2.0 publié. Reste l'écran dans Tor6IA (1.6).
- 2026-11-27 : **E3, E4, E5 faits, étape des essais terminée.** Deux découvertes ont changé le moteur : la table des mots doit rester sur la tête (trafic par token divisé par 3 à 4), et le relais sur le VPS devra être plafonné. Rapport dans E6.
- 2026-11-27 : **E2 fait.** Le préchargement par le cache marche pour 85 % du modèle et divise le temps de chargement par 2,5. Les 15 % restants (tenseurs de moins de 10 Mo) passent toujours par la tête : tâche 3.9 ajoutée, conditionnelle. Le plan et le téléchargement par couches existent en commandes (`plan`, `fetch`).
- 2026-11-27 : **E1 réussi, et 2.1 faite dans la foulée.** Le bac à sable existe sur Windows, Linux et macOS sans droits administrateur. La route de repli WSL est abandonnée. Trois allers-retours avec la pipeline ont été nécessaires pour Windows Server (périphérique `NUL` interdit dans la boîte).
- 2026-11-27 : plan validé par Tom. Consigne : tout faire sans s'arrêter, mettre ce fichier à jour à chaque tâche, essais sur le VPS, test final par Tom avec des gens. D4 et D10 changées en conséquence.
- 2026-11-27 : plan complet réécrit avec décisions à valider, jalons et étape d'essais. Découvert pendant la préparation : le cache des serveurs de blocs occupe 22 Go après une journée d'essais, d'où la tâche 2.5. En attente de la validation de Tom.
