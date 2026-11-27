# Plan de travail : des blocs choisis par chacun, pour le monde entier

But : chaque personne choisit un modèle et un nombre de blocs dans l'interface de Tor6IA, sa machine ne télécharge que ces blocs, et des inconnus du monde entier portent ensemble des modèles trop gros pour une seule machine.

Ce fichier est la référence. Il est mis à jour à chaque tâche terminée : case cochée, date, résultat mesuré, décision prise.

**Prochaine tâche : 1.1**

Dernière mise à jour : 2026-11-27

## Comment lire ce plan

- `[ ]` à faire, `[~]` en cours, `[x]` fait, `[!]` bloqué.
- Chaque tâche dit **Fini quand** : un résultat vérifiable, jamais « le code est écrit ».
- Chaque phase commence par un **essai** court. Si l'essai échoue, la phase change de route avant d'avoir coûté cher.
- Les durées sont des jours de travail, en fourchette. Elles seront corrigées après chaque essai.
- **À vérifier** signale une hypothèse que personne n'a encore mesurée.

## Vue d'ensemble

| Phase | Ce qu'elle apporte | Durée | Dépend de |
|---|---|---|---|
| 0. Socle | Un modèle réparti entre machines de confiance | fait | |
| 1. Bac à sable | Porter un bloc pour un inconnu sans risque | 5 à 8 j | 0 |
| 2. Blocs téléchargeables | Chaque machine ne télécharge que sa partie | 3 à 5 j | 0 |
| 3. Choix des blocs | « Pour ta machine, 3 blocs conseillés » dans l'interface | 2 à 3 j | 2 |
| 4. Relais et groupes | Le réseau assemble les machines tout seul | 8 à 12 j | 1, 2 |
| 5. Paiement par bloc | Chacun payé pour sa part, triche détectée | 4 à 6 j | 4 |
| 6. Ouverture | Activé pour tout le monde | 3 à 5 j | 1 à 5 |

Total : 25 à 39 jours de travail. Les phases 1 et 2 ne dépendent pas l'une de l'autre et peuvent avancer en alternance.

## Phase 0 : socle (fait)

- [x] 0.1 Répartir un modèle entre plusieurs serveurs de blocs avec llama.cpp. *2026-11-27 : Gemma 4 12B en 3 blocs, 3,3 tokens/s sur un portable sans carte graphique.*
- [x] 0.2 Mesurer le coût du délai réseau. *2026-11-27 : coupé en deux, 3,3 tokens/s à 0 ms, 2,9 à 10 ms, 2,5 à 25 ms, 1,9 à 50 ms dans chaque sens.*
- [x] 0.3 Une commande pour essayer, testée sur Windows, Linux et macOS à chaque version.
- [x] 0.4 Branché dans Tor6IA 0.2.0 : groupe réglé à la main, découpage affiché, paiement à la machine de tête.
- [x] 0.5 Tor6IA 0.2.1 : une demande attend une machine, et chaque modèle affiche ses demandes en attente.

Limites laissées par cette phase, reprises plus bas : serveur de blocs sans protection (phase 1), la tête envoie les couches aux autres (phase 2), adresses données à la main (phase 4).

## Phase 1 : bac à sable

Le verrou. Un serveur de blocs exécute les ordres de calcul d'une autre machine, dans un programme C++ sans identification ni chiffrement. Tant que ce programme peut toucher aux fichiers ou au réseau de la machine, il ne doit servir que des gens de confiance.

- [ ] **1.1 Essai : choisir la technique par système.** 1 à 2 j.
  Candidats : bubblewrap et seccomp sous Linux, AppContainer avec jeton restreint sous Windows, `sandbox-exec` sous macOS. Solution de repli commune : compiler le serveur de blocs en WebAssembly.
  **Fini quand** : sur chaque système, un serveur de blocs enfermé sert un vrai modèle, et un programme de test lancé à sa place échoue à lire un fichier personnel, à ouvrir une connexion sortante et à lancer un autre programme. Sans droits administrateur.
  **Si ça échoue sous Windows** : porter des blocs y passe par WebAssembly ou WSL, et la durée de la phase est revue.
- [ ] **1.2 Limites de ressources et arrêt garanti.** 1 j.
  Plafond de mémoire égal aux blocs choisis, plafond de processeur, et arrêt du serveur de blocs quand le moteur s'arrête, même tué de force (Job Object sous Windows, `PR_SET_PDEATHSIG` sous Linux).
  **Fini quand** : moteur tué de force, aucun de ses processus ne reste. Corrige la limite connue du README.
- [ ] **1.3 Aucune écoute sur le réseau.** 1 j.
  Le serveur de blocs n'écoute que sur la machine elle-même. C'est le moteur qui parle au reste du monde et vérifie qui est en face.
  **Fini quand** : un balayage de ports depuis une autre machine ne trouve rien, et le groupe fonctionne toujours.
- [ ] **1.4 Essais d'abus.** 1 à 2 j.
  Ordres malformés, tailles absurdes, demande de mémoire géante, connexion coupée au milieu.
  **Fini quand** : le serveur de blocs plante ou refuse, le bac à sable tient, le moteur le signale et le relance. Ces essais tournent dans la pipeline.
- [ ] **1.5 Perte de vitesse mesurée.** 0,5 j.
  **Fini quand** : tokens par seconde avec et sans bac à sable, sur les trois systèmes, écrits ici. Plus de 30 % de perte rouvre le choix de 1.1.

## Phase 2 : blocs téléchargeables

Aujourd'hui la machine de tête lit le modèle entier et envoie leurs couches aux autres. Pour que chacun ne télécharge que sa part, les blocs doivent exister comme fichiers séparés.

- [ ] **2.1 Essai : comment un serveur de blocs peut charger ses couches depuis son disque.** 1 j.
  À vérifier : le cache du serveur de blocs (`-c`) est indexé par l'empreinte de chaque couche. Si on le remplit à l'avance, la tête n'envoie plus rien.
  **Fini quand** : un modèle se charge sur deux serveurs de blocs avec moins de 1 % de ses octets transférés, mesuré avec `proof/delayproxy`.
  **Si ça échoue** : étudier le découpage du fichier GGUF par `gguf-split`.
- [ ] **2.2 Format d'un modèle prédécoupé.** 1 j.
  Un manifeste par modèle : liste des blocs, couches de chaque bloc, taille, empreinte SHA-256. Taille de bloc visée : environ 1 Go.
  **Fini quand** : `tor6ia-engine pack --model gemma4:12b` produit les blocs et le manifeste, et deux exécutions donnent les mêmes empreintes.
- [ ] **2.3 Hébergement des blocs.** 0,5 à 1 j.
  Un hébergeur de fichiers public et gratuit pour des modèles ouverts. Pas le serveur du coordinateur.
  **Fini quand** : les blocs de Gemma 4 12B sont téléchargeables par une adresse stable, empreintes vérifiées.
- [ ] **2.4 Télécharger seulement ses blocs.** 1 j.
  `tor6ia-engine fetch --model gemma4:12b --blocks 3-5`, avec reprise après coupure et vérification d'empreinte.
  **Fini quand** : une machine qui porte 3 blocs sur 8 a téléchargé 3 huitièmes du modèle, pas plus, et sert ces blocs.
- [ ] **2.5 Rôle de la machine de tête.** 0,5 j.
  À vérifier : la tête a besoin du fichier entier sur son disque, pas en mémoire.
  **Fini quand** : la règle est écrite ici avec la mémoire et le disque mesurés pour la tête d'un modèle de 20 Go.

## Phase 3 : choix des blocs dans l'interface

- [ ] **3.1 Répartition par parts choisies.** 0,5 j.
  À vérifier : `--tensor-split` de llama.cpp s'applique aux serveurs de blocs distants.
  **Fini quand** : avec les parts 1, 3 et 2, les couches affichées suivent ces proportions.
- [ ] **3.2 Le conseil.** 0,5 j.
  Nombre de blocs conseillé à partir de la mémoire libre mesurée, avec la même prudence que la règle actuelle de Tor6IA (1,5 fois la taille, 2 Go laissés au système).
  **Fini quand** : sur trois machines de mémoires différentes, le nombre conseillé se charge sans que la machine se mette à ramer.
- [ ] **3.3 L'écran « Porter des blocs » dans Tor6IA.** 1 à 2 j.
  Choix du modèle, « pour ta machine : 3 blocs conseillés », choix de 1 au maximum possible, téléchargement bloc par bloc, puis « 3 blocs chargés, 3 en service ».
  **Fini quand** : une personne qui n'a jamais vu le projet porte des blocs sans aide, et l'affichage correspond à ce que le moteur rapporte.

## Phase 4 : relais et groupes formés par le réseau

- [ ] **4.1 Essai : combien d'octets par token entre deux blocs.** 0,5 j.
  **Fini quand** : octets par token et par frontière de bloc mesurés pour Gemma 4 12B et 31B. Ce chiffre fixe le débit que le relais doit tenir et son coût.
- [ ] **4.2 Choisir où tourne le relais.** 0,5 j.
  Pas sur le serveur actuel, partagé avec d'autres sites.
  **Fini quand** : serveur choisi, coût mensuel et débit garanti écrits ici.
- [ ] **4.3 Le relais.** 2 à 3 j.
  Chaque machine garde une connexion sortante vers le relais, qui fait passer les données entre la tête et les blocs. Identité par la clé du compte, débit plafonné par compte.
  **Fini quand** : un groupe de trois machines derrière trois box différentes répond, sans aucun port ouvert.
- [ ] **4.4 Connexion directe quand c'est possible.** 1 à 2 j.
  Même réseau local ou port joignable : les machines se parlent sans passer par le relais.
  **Fini quand** : deux machines du même réseau échangent en direct, vérifié par le compteur d'octets du relais qui reste à zéro.
- [ ] **4.5 Registre des blocs dans le coordinateur.** 1 à 2 j.
  Qui porte quels blocs de quel modèle. Chaque modèle affiche « blocs couverts : 6 sur 8, il manque 2 machines ».
  **Fini quand** : l'interface de Tor6IA affiche ce compte en direct, et il est juste quand des machines arrivent et partent.
- [ ] **4.6 Assemblage d'une chaîne.** 1 à 2 j.
  Pour une demande, le coordinateur choisit une machine par bloc, la plus rapide disponible, et une tête qui a le fichier.
  **Fini quand** : une demande sur un modèle que personne ne porte en entier reçoit sa réponse, calculée par une chaîne que le réseau a formée seul.
- [ ] **4.7 Reprise quand une machine part.** 1 à 2 j.
  Chaque bloc doit avoir au moins deux porteurs avant que le modèle soit annoncé disponible. Une machine part : la chaîne se reforme avec l'autre porteur.
  **Fini quand** : machine débranchée au milieu d'une réponse, la réponse se termine, délai de reprise mesuré.
- [ ] **4.8 Essai réel.** 1 j.
  **Fini quand** : au moins trois vraies machines, sur au moins deux connexions Internet différentes, portent un modèle qu'aucune ne tient seule. Vitesse mesurée et écrite ici.

## Phase 5 : paiement et triche, par bloc

- [ ] **5.1 Paiement partagé dans le registre.** 1 à 2 j.
  Une réponse paie plusieurs comptes, chacun selon ses couches et les tokens calculés. Toujours signé par celui qui paie, toujours à somme nulle.
  **Fini quand** : `tor6ia audit` vérifie un registre contenant des paiements partagés, et refuse une part modifiée.
- [ ] **5.2 Détecter une machine qui renvoie n'importe quoi.** 2 à 3 j.
  Deux porteurs du même bloc reçoivent la même entrée : leurs sorties doivent concorder. Tests à réponse connue passés par une chaîne précise.
  **Fini quand** : une machine modifiée pour renvoyer du bruit est exclue en moins de dix demandes, sans exclure une machine honnête sur mille demandes.
- [ ] **5.3 Ce que voit chaque machine.** 0,5 j.
  Chaque porteur voit passer les données internes du calcul. Le chiffrement de bout en bout actuel ne couvre pas les blocs.
  **Fini quand** : c'est écrit clairement dans l'interface et la documentation, et le mode « groupe de confiance » reste disponible pour qui veut garder ses questions entre proches.

## Phase 6 : ouverture

- [ ] **6.1 Essai avec des volontaires.** 2 à 3 j. **Fini quand** : dix machines de volontaires portent un modèle pendant une semaine, pannes et vitesses notées ici.
- [ ] **6.2 Limites et réglages par défaut.** 0,5 à 1 j. **Fini quand** : débit, mémoire et nombre de blocs par défaut sont fixés à partir des mesures de 6.1.
- [ ] **6.3 Activé pour tous.** 0,5 à 1 j. **Fini quand** : une installation neuve de Tor6IA propose de porter des blocs, et la pipeline le teste sur les trois systèmes.

## Ce qui peut faire dérailler le plan

| Risque | Signe | Réponse prévue |
|---|---|---|
| Pas de bac à sable sans droits administrateur sous Windows | 1.1 échoue | WebAssembly ou WSL pour porter des blocs sous Windows |
| Le cache des couches ne permet pas le préchargement | 2.1 échoue | Découpage du fichier par `gguf-split` |
| Trop d'octets par token pour un relais abordable | 4.1 donne un chiffre trop haut | Connexion directe obligatoire, le relais ne sert qu'à se trouver |
| Trop lent à travers plusieurs pays | 4.8 sous 1 token par seconde | Chaînes formées par proximité, modèles limités en nombre de blocs |
| llama.cpp change son protocole | une nouvelle version casse les essais | La version reste épinglée, montée de version testée à part |

## Journal

- 2026-11-27 : plan écrit. Phase 0 terminée le même jour. Rien de commencé au-delà.
