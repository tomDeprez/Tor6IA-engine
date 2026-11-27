# Plan de travail : des blocs choisis par chacun, pour le monde entier

But : chaque personne choisit un modèle et un nombre de blocs dans l'interface de Tor6IA, sa machine ne télécharge que ces blocs, et des inconnus du monde entier portent ensemble des modèles trop gros pour une seule machine.

Ce fichier est la référence. Il est mis à jour à chaque tâche terminée : case cochée, date, résultat mesuré, décision prise.

**État : plan validé par Tom le 2026-11-27. Travail en cours.**

**Prochaine tâche : E3**

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
| E. Essais | Lire un rapport : chaque hypothèse du plan est confirmée ou remplacée | aucune | 3,5 à 4,5 j | 4,5 j |
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
- [ ] **E3 Peut-on imposer quelle machine porte quelles couches ?** 0,5 j.
  **Fini quand** : avec trois serveurs de blocs et les parts 1, 3 et 2, les couches affichées par le moteur suivent exactement le plan demandé, avec `--tensor-split` ou `--override-tensor`.
- [ ] **E4 Combien d'octets et d'allers-retours par token entre deux blocs ?** 0,5 j.
  Pour Gemma 4 12B et 31B, avec 2, 4 et 8 blocs.
  **Fini quand** : le tableau est écrit ici. Il donne le débit que le relais doit tenir, donc son coût (D4), et le nombre de blocs au-delà duquel la vitesse tombe sous 1 token par seconde (D9).
- [ ] **E5 De quoi la machine de tête a-t-elle besoin ?** 0,5 j.
  À vérifier : le fichier entier sur son disque, ou seulement l'en-tête et sa part.
  **Fini quand** : mémoire et disque mesurés pour la tête d'un modèle de 20 Go. Si la tête doit avoir tout le fichier, c'est écrit dans l'interface : « pour lancer une conversation sur ce modèle, 20 Go de disque ».
- [ ] **E6 Rapport des essais.** 0,5 j.
  **Fini quand** : ce fichier contient les résultats, les durées corrigées de J1 à J5 et les décisions qui ont changé. Tom revalide avant J1.

## 7. Jalon J1 : les blocs à la carte, entre machines de confiance

Tout ce qui se voit dans l'interface, sans encore ouvrir aux inconnus. Dépend de E2, E3, E5.

- [~] **1.1 Le plan d'un modèle dans le moteur.** *Fait le 2026-11-27 : `tor6ia-engine plan --model gemma4:31b` décrit un modèle jamais téléchargé (18,67 Go, 60 couches de 257 à 314 Mo, 1 102 Mo hors couches) en lisant 15,8 Mo, soit 0,08 % du fichier. Plan identique depuis le disque et depuis le registre. Reste : le regroupement en blocs d'environ 1 Go et le test sur les 5 modèles du catalogue.*
  `tor6ia-engine plan --model gemma4:31b` lit l'en-tête par morceaux depuis le registre et donne les couches, leur taille, et le découpage en blocs d'environ 1 Go. Sortie lisible et sortie JSON.
  Fichiers : `gguf.go` (lecture de l'en-tête), `plan.go`.
  **Fini quand** : le plan d'un modèle jamais téléchargé s'affiche en ayant reçu moins de 20 Mo, identique à celui lu dans le fichier complet. Testé sur les 5 modèles du catalogue.
- [~] **1.2 Télécharger seulement ses blocs.** *Fait le 2026-11-27 : `tor6ia-engine fetch --model M --layers 3-5` prend les morceaux sur le disque si le modèle y est, sinon au registre, avec reprise. Reste : l'adressage par blocs et la progression pour l'interface.*
  `tor6ia-engine fetch --model gemma4:31b --blocks 3-5` prend les morceaux du fichier public, avec reprise après coupure et progression.
  À vérifier : le registre ne donne qu'une empreinte pour le fichier entier. L'intégrité d'un morceau repose sur HTTPS vers le registre officiel, sauf si le cache vérifie l'empreinte de chaque couche (E2).
  **Fini quand** : une machine qui porte 3 blocs sur 20 a téléchargé 3 vingtièmes du modèle, pas plus, et sert ces blocs.
- [ ] **1.3 Un serveur de blocs qui annonce et tient un nombre de blocs.** 1 j.
  `tor6ia-engine block --model gemma4:31b --blocks 3-5` : il ne charge que ça, avec un plafond de mémoire.
  **Fini quand** : la mémoire utilisée correspond aux blocs choisis à 15 % près, mesurée.
- [ ] **1.4 Le conseil.** 0,5 j.
  Nombre de blocs conseillé à partir de la mémoire libre mesurée et de la carte graphique, avec la prudence déjà utilisée dans Tor6IA (1,5 fois la taille, 2 Go laissés au système).
  **Fini quand** : sur trois machines de mémoires différentes (8, 16 et 32 Go), le nombre conseillé se charge sans que la machine se mette à ramer.
- [ ] **1.5 La tête assemble selon les blocs annoncés.** 1 j.
  Elle reçoit « machine A : blocs 1 à 3, machine B : blocs 4 à 9 » et place les couches exactement ainsi (E3). Elle refuse de démarrer s'il manque un bloc et dit lequel.
  **Fini quand** : le découpage affiché est celui demandé, et un bloc manquant donne « il manque les blocs 10 à 12 ».
- [ ] **1.6 L'écran « Porter des blocs » dans Tor6IA.** 1,5 à 2 j.
  Choix du modèle, « pour ta machine : 3 blocs conseillés sur 20 », choix de 1 au maximum possible, téléchargement bloc par bloc, puis « 3 blocs chargés, 3 en service ». La barre du modèle montre les blocs couverts et ceux qui manquent.
  Fichiers : `ui/index.html`, `split.go` dans Tor6IA.
  **Fini quand** : une personne qui n'a jamais vu le projet porte des blocs sans aide, et l'affichage correspond à ce que le moteur rapporte.
- [ ] **1.7 Démonstration J1.** 0,5 j.
  **Fini quand** : sur ta machine, trois porteurs de blocs choisis à la main (2, 5 et 13 blocs) font répondre Gemma 4 31B, modèle qu'elle ne tient pas seule en mémoire. Vitesse notée ici. Tor6IA 0.3.0 publiée.

## 8. Jalon J2 : le bac à sable

Le verrou de l'ouverture. Dépend de E1.

- [x] **2.1 Le serveur de blocs enfermé, sur les trois systèmes.** *Fait le 2026-11-27 avec E1 : `tor6ia-engine block` et la démo enferment par défaut, `--unlocked` rend l'ancien comportement. Les évasions sont testées dans la pipeline sur les trois systèmes.*
  Avec la technique choisie en E1. Il ne voit que son dossier de blocs. Aucun autre fichier, aucun autre programme, aucune connexion sortante.
  **Fini quand** : les trois tentatives d'évasion de E1 échouent dans la pipeline, sur Windows, Linux et macOS.
- [~] **2.2 Limites de ressources et arrêt garanti.** 0,5 j restant. *Fait le 2026-11-27 : les programmes lancés par le moteur s'arrêtent avec lui, même tué de force (job sous Windows, `PR_SET_PDEATHSIG` sous Linux). Reste : le plafond de mémoire branché sur les blocs choisis, l'équivalent sous macOS, et le test dans la pipeline.*
  Plafond de mémoire et de processeur. Le serveur de blocs s'arrête avec le moteur, même tué de force (Job Object sous Windows, `PR_SET_PDEATHSIG` sous Linux).
  **Fini quand** : moteur tué de force, aucun de ses processus ne reste. Corrige la limite connue du README.
- [ ] **2.3 Aucune écoute sur le réseau.** 0,5 j.
  Le serveur de blocs n'écoute que sur la machine elle-même. Le moteur parle au reste du monde et vérifie la clé de celui qui est en face.
  **Fini quand** : un balayage de ports depuis une autre machine ne trouve rien, et le groupe fonctionne toujours.
- [ ] **2.4 Essais d'abus.** 0,5 à 1 j.
  Ordres malformés, tailles absurdes, demande de mémoire géante, connexion coupée au milieu.
  **Fini quand** : le serveur de blocs plante ou refuse, le bac à sable tient, le moteur le signale et le relance. Ces essais tournent dans la pipeline.
- [ ] **2.5 Le cache de blocs sous contrôle.** 0,5 j.
  Taille limitée, blocs inutilisés supprimés, tout effacé quand on arrête de porter un modèle ou qu'on désinstalle. Aujourd'hui il grossit sans limite (22 Go après une journée d'essais).
  **Fini quand** : le dossier ne dépasse jamais la taille des blocs portés, vérifié après dix changements de modèle.
- [ ] **2.6 Perte de vitesse mesurée.** 0,5 j.
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

- 2026-11-27 : phase 0 terminée. Premier plan écrit.
- 2026-11-27 : essai 0.6 fait sur une question de Tom (« peut-on découper sans télécharger ? »). Oui : plan exact dans l'en-tête, morceaux servis à la demande. Plus aucun hébergement de modèles à prévoir.
- 2026-11-27 : **E2 fait.** Le préchargement par le cache marche pour 85 % du modèle et divise le temps de chargement par 2,5. Les 15 % restants (tenseurs de moins de 10 Mo) passent toujours par la tête : tâche 3.9 ajoutée, conditionnelle. Le plan et le téléchargement par couches existent en commandes (`plan`, `fetch`).
- 2026-11-27 : **E1 réussi, et 2.1 faite dans la foulée.** Le bac à sable existe sur Windows, Linux et macOS sans droits administrateur. La route de repli WSL est abandonnée. Trois allers-retours avec la pipeline ont été nécessaires pour Windows Server (périphérique `NUL` interdit dans la boîte).
- 2026-11-27 : plan validé par Tom. Consigne : tout faire sans s'arrêter, mettre ce fichier à jour à chaque tâche, essais sur le VPS, test final par Tom avec des gens. D4 et D10 changées en conséquence.
- 2026-11-27 : plan complet réécrit avec décisions à valider, jalons et étape d'essais. Découvert pendant la préparation : le cache des serveurs de blocs occupe 22 Go après une journée d'essais, d'où la tâche 2.5. En attente de la validation de Tom.
