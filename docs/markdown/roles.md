# Rôles et permissions : guide front

Ce document explique qui peut faire quoi dans Kanbano, et comment le front
récupère les droits de l'utilisateur courant.

**Règle d'or** : le front ne recalcule jamais les droits lui-même. Il les lit
avec `GET /api/v1/workspaces/{id}/abilities` (règles CASL), puis masque ou
désactive les actions. L'API revérifie tout côté serveur.

> **Breaking changes** par rapport à la version précédente : voir la
> [section 6](#6-migration-depuis-lancienne-version).

---

## 1. Les rôles

| Rôle | Niveau | En bref |
|---|---|---|
| `owner` | Organisation | Créateur de l'orga : tous les droits, voit tous les workspaces de l'orga |
| `admin` | Organisation | Gère les workspaces partagés avec lui, sans pouvoir les supprimer |
| `member` | Organisation | Travaille dans les workspaces partagés avec lui, en `view` ou `edit` |
| `guest` | Tâche | Personne extérieure invitée sur une ou plusieurs tâches, hors organisation |

### 1.1 Organisations

- À l'inscription, chaque utilisateur reçoit **son** organisation (« Organisation
  de {name} », renommable), dont il est `owner`. On est `owner` d'une seule
  organisation au maximum.
- Un utilisateur peut en plus être `admin` ou `member` d'**autres**
  organisations. Le front doit donc gérer **plusieurs organisations** par
  utilisateur (`GET /organisations`).
- Si l'owner supprime son organisation, il n'en reçoit pas de nouvelle : il ne
  peut plus créer de workspace (`POST /workspaces` → `403`).

### 1.2 Workspaces : privés par défaut

- Tout workspace appartient à une organisation (`organisation_id`) et est
  **privé** par défaut.
- L'**owner** voit tous les workspaces de son organisation, toujours avec tous
  les droits.
- Un **admin** ou un **member** ne voit que les workspaces **partagés avec
  lui**. Chaque partage porte un rôle `view` ou `edit`.
- Un workspace non partagé est invisible : absent de `GET /workspaces`, et
  `404 workspace not found` sur toutes ses routes (jamais `403`).

### 1.3 Ce que change le rôle `view` / `edit`

Le rôle de partage n'a pas le même effet selon le rôle dans l'organisation :

| Rôle orga | Partage `view` | Partage `edit` |
|---|---|---|
| `admin` | Gère le workspace (voir 2.3), comme en `edit` | Gère le workspace (voir 2.3) |
| `member` | Lecture seule | Modifie et réordonne colonnes et tâches, déplace les tâches, assigne. **Ne crée et ne supprime ni colonne ni tâche.** |

Pour un admin, `view` / `edit` ne change donc rien. Le rôle est quand même
conservé, et ce n'est pas lui qui compte, mais le fait que le workspace soit
partagé ou non.

### 1.4 Guests (invités externes)

Un guest est invité par email sur **une tâche précise**, avec le rôle `view`
ou `edit`. Une fois l'invitation acceptée :

- le workspace apparaît dans son `GET /workspaces` ;
- `GET /workspaces/{id}` ne lui renvoie **que ses tâches** (et les colonnes
  qui les contiennent), sans la liste des membres ;
- en `view`, il lit ses tâches ;
- en `edit`, il ne peut modifier que la **description** de ses tâches
  (`403 guests can only edit the description` pour tout autre champ) ;
- il ne crée rien, ne supprime rien, ne déplace rien, n'assigne ni n'invite
  personne ;
- toute autre tâche du workspace lui répond `404 task not found`.

### 1.5 Assignations : purement visuelles

Assigner quelqu'un à une tâche sert uniquement à afficher « Bob est sur cette
tâche ». Une assignation **ne donne aucun droit** et ne change aucun rôle.
Le champ `role` des assignés **n'existe plus**.

---

## 2. Qui peut faire quoi

Légende : ✅ autorisé, ❌ refusé (code d'erreur entre parenthèses).
« admin / member » s'entend **sur un workspace partagé avec lui** : sinon, la
réponse est toujours `404`.

### 2.1 Organisation

| Action | Route | owner | admin | member |
|---|---|---|---|---|
| Lister mes organisations (avec mon rôle) | `GET /organisations` | ✅ | ✅ | ✅ |
| Voir l'organisation et ses membres | `GET /organisations/{orgId}` | ✅ | ✅ | ✅ |
| Renommer l'organisation | `PATCH /organisations/{orgId}` | ✅ | ❌ (`403`) | ❌ (`403`) |
| Supprimer l'organisation | `DELETE /organisations/{orgId}` | ✅ | ❌ (`403`) | ❌ (`403`) |
| Inviter dans l'organisation (`admin` ou `member`) | `POST /organisations/{orgId}/invitations` | ✅ | ❌ (`403`) | ❌ (`403`) |
| Voir les invitations en attente | `GET /organisations/{orgId}/invitations` | ✅ | ❌ (`403`) | ❌ (`403`) |
| Changer le rôle d'un membre (`admin` ↔ `member`) | `PATCH /organisations/{orgId}/members/{memberId}` | ✅ | ❌ (`403`) | ❌ (`403`) |
| Retirer quelqu'un de l'organisation | `DELETE /organisations/{orgId}/members/{memberId}` | ✅ | ❌ (`403`) | ❌ (`403`) |
| Voir le profil et les accès d'un membre | `GET /organisations/{orgId}/members/{memberId}/profile` | ✅ | ✅ (1) | ❌ (`403`) |

Quelqu'un qui n'appartient pas à l'organisation reçoit
`404 organisation not found` sur toutes ces routes.

(1) L'admin ne voit, dans `workspaces`, que les workspaces partagés avec lui.
L'owner les voit tous.

Détails :

- `GET /organisations` renvoie `[{ id, name, user_id, role }]`, l'organisation
  possédée en premier.
- `GET /organisations/{orgId}` renvoie `{ id, name, user_id, role, members }`.
  `role` est le rôle de l'appelant, et `members` commence par l'owner (rôle
  `owner`, `joined_at` à `null`).
- `PATCH /organisations/{orgId}` : `{ "name": "..." }`.
- `DELETE /organisations/{orgId}` supprime (soft delete) l'organisation, ses
  workspaces, ses colonnes et ses tâches.
- Invitation : `{ "email": "...", "role": "admin" | "member" }` (`member` par
  défaut). `409` si une invitation est déjà en attente pour cet email, ou si
  la personne fait déjà partie de l'organisation.
- Changement de rôle : `{ "role": "admin" | "member" }`. L'owner ne peut pas
  changer son propre rôle (`403 cannot change your own role`).
- Retirer un membre lui retire aussi ses accès aux workspaces de l'orga et ses
  assignations sur ses tâches. L'owner ne peut pas se retirer lui-même
  (`403 cannot remove yourself`).

**Invitations reçues** (tout utilisateur) :

- `GET /organisation-invitations/received` : invitations en attente ;
- `PATCH /organisation-invitations/{id}` avec
  `{ "status": "accepted" | "declined" }`.

Si la personne était guest sur des tâches de cette organisation, l'accepter
remplace ses accès guest par un partage des workspaces concernés. Le rôle
`edit` est donné s'il l'avait sur au moins une tâche du workspace, `view`
sinon.

### 2.2 Workspace

| Action | Route | owner | admin | member | guest |
|---|---|---|---|---|---|
| Lister | `GET /workspaces` | tous ceux de l'orga | partagés | partagés | ceux de ses tâches |
| Voir le détail | `GET /workspaces/{id}` | ✅ | ✅ | ✅ | ses tâches seulement |
| Créer | `POST /workspaces` | voir ci-dessous | | | |
| Renommer, changer la description | `PATCH /workspaces/{id}` | ✅ | ✅ | ❌ (`403`) | ❌ (`403`) |
| Gérer les accès (voir 2.3) | `PATCH /workspaces/{id}` ou `PUT /workspaces/{id}/members/{memberID}/role` | ✅ | ✅ (members seulement) | ❌ (`403`) | ❌ (`403`) |
| Supprimer | `DELETE /workspaces/{id}` | ✅ | ❌ (`403`) | ❌ (`403`) | ❌ (`403`) |

`POST /workspaces` n'a pas de paramètre d'organisation : le workspace est
**toujours créé dans l'organisation que possède l'appelant**. Tout utilisateur
peut donc créer un workspace, mais seulement chez lui. Quelqu'un qui est admin
d'une autre orga ne peut pas y créer de workspace. `403 organisation owner
required` seulement si l'appelant ne possède plus d'organisation (il l'a
supprimée).

Un workspace créé est privé : seul l'owner le voit tant qu'il ne l'a pas
partagé.

### 2.3 Gérer les accès à un workspace

Réservé à l'**owner** et aux **admins** avec qui le workspace est partagé
(`403 organisation owner or admin required` sinon).

- Personne ne touche aux accès de l'owner.
- Un admin ne gère que des `member`, jamais un autre admin
  (`403 cannot manage this member's access`).
- Personne ne change son propre accès (`403 cannot change your own access`).

Deux routes font la même chose :

- **plusieurs personnes d'un coup**, depuis les réglages du workspace :
  `PATCH /workspaces/{id}`

  ```json
  {
    "members": [
      { "id": "<ajouté>", "visibility": "public", "role": "edit" },
      { "id": "<modifié>", "role": "view" },
      { "id": "<retiré>", "visibility": "private" }
    ]
  }
  ```

- **une seule personne**, depuis sa fiche :
  `PUT /workspaces/{id}/members/{memberID}/role`

  ```json
  { "visibility": "public", "role": "edit" }
  ```

Règles communes :

- `visibility` : `public` partage le workspace, `private` retire l'accès.
- `role` : `view` ou `edit`. Il est **mémorisé** même quand l'accès est
  retiré : si le workspace est de nouveau partagé plus tard, la personne
  retrouve son ancien rôle.
- Les deux champs sont optionnels et indépendants. Un champ absent garde sa
  valeur actuelle, ou prend sa valeur par défaut (`private` / `view`) si la
  personne n'avait encore aucun accès à ce workspace.
- **Pour ajouter quelqu'un, il faut envoyer `"visibility": "public"`**.
  `{ "role": "edit" }` seul, sur quelqu'un sans accès, le laisse `private`.
- Chaque personne doit appartenir à l'organisation, sinon
  `404 member not found in this organisation`.
- Dans `members`, un même `id` ne peut apparaître qu'une fois (`400` sinon).
- Si une vérification échoue, **rien** n'est appliqué (ni les membres, ni
  `name` / `description`).

Pour afficher les accès actuels sur un workspace : `GET /workspaces/{id}`,
champ `members`. Chaque entrée a `role` (`view` / `edit`) et
`organisation_role` (`admin` / `member`). L'owner n'est **jamais** dans
`members` : il voit tous les workspaces sans y être partagé. Côté réglages,
un admin ne doit proposer de modifier que les entrées
`organisation_role: "member"` (et pas lui-même).

Pour afficher les accès actuels d'une personne sur chaque workspace :
`GET /organisations/{orgId}/members/{memberId}/profile`, champ `workspaces`
(`visibility` et `role` pour chacun).

### 2.4 Colonnes

| Action | Route | owner | admin | member `edit` | member `view` | guest |
|---|---|---|---|---|---|---|
| Lister les noms | `GET /workspaces/{id}/columns/names` | ✅ | ✅ | ✅ | ✅ | ❌ (`403`) |
| Créer | `POST /workspaces/{id}/columns` | ✅ | ✅ | ❌ (`403`) | ❌ (`403`) | ❌ (`403`) |
| Renommer / réordonner | `PATCH /workspaces/{id}/columns/{columnId}` | ✅ | ✅ | ✅ | ❌ (`403`) | ❌ (`403`) |
| Supprimer | `DELETE /workspaces/{id}/columns/{columnId}` | ✅ | ✅ | ❌ (`403`) | ❌ (`403`) | ❌ (`403`) |

### 2.5 Tâches

Toutes les routes sont préfixées par `/workspaces/{id}/columns/{columnId}`.

| Action | Route | owner | admin | member `edit` | member `view` | guest `edit` | guest `view` |
|---|---|---|---|---|---|---|---|
| Créer | `POST /tasks` | ✅ | ✅ | ❌ | ❌ | ❌ | ❌ |
| Modifier (nom, tag, statut…) | `PATCH /tasks/{taskId}` | ✅ | ✅ | ✅ | ❌ | ❌ | ❌ |
| Modifier la description | `PATCH /tasks/{taskId}` | ✅ | ✅ | ✅ | ❌ | ✅ | ❌ |
| Réordonner, déplacer vers une autre colonne | `PATCH /tasks/{taskId}` (`position`, `targetColumnId`) | ✅ | ✅ | ✅ | ❌ | ❌ | ❌ |
| Supprimer | `DELETE /tasks/{taskId}` | ✅ | ✅ | ❌ | ❌ | ❌ | ❌ |
| Voir les assignés | `GET /tasks/{taskId}/assignees` | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ |
| Assigner / désassigner | `POST /tasks/{taskId}/assignees`, `DELETE /tasks/{taskId}/assignees/{memberId}` | ✅ | ✅ | ✅ | ❌ | ❌ | ❌ |
| Inviter un guest, voir / gérer les guests | `/tasks/{taskId}/guests…` (voir 2.6) | ✅ | ✅ | ❌ | ❌ | ❌ | ❌ |

Tous les refus sont des `403`, sauf pour un guest sur une tâche qui ne lui a
pas été partagée : `404 task not found`.

- `POST /tasks/{taskId}/assignees` : `{ "member_id": "..." }`. La personne
  doit avoir accès au workspace
  (`404 member not found in this workspace` sinon).
- `GET /tasks/{taskId}/assignees` renvoie
  `[{ id, name, email, avatar, is_guest }]` : les membres assignés, puis les
  guests qui ont accepté une invitation sur la tâche (`is_guest: true`).

### 2.6 Guests

Gestion réservée à l'**owner** et aux **admins** avec qui le workspace est
partagé (`403 organisation owner or admin required` sinon). Routes préfixées
par `/workspaces/{id}/columns/{columnId}/tasks/{taskId}` :

| Action | Route | Body |
|---|---|---|
| Inviter | `POST /guests` | `{ "email": "...", "role": "view" \| "edit" }` (`view` par défaut) |
| Lister les invitations en attente | `GET /guests` | inclut les invitations expirées (`expires_at` passé), pour pouvoir les renvoyer |
| Changer le rôle d'un guest | `PUT /guests/{userId}` | `{ "role": "view" \| "edit" }` |
| Retirer un guest | `DELETE /guests/{userId}` | |

- Une invitation **expire au bout de 7 jours**.
- Réinviter le même email remplace une invitation expirée.
- `409` si une invitation est déjà en attente pour cet email, si la personne
  est déjà guest sur la tâche, ou si elle fait déjà partie de l'organisation.
- **Renvoyer** une invitation : `POST /guest-invitations/{id}/resend`. Seul
  l'auteur de l'invitation peut le faire, et seulement s'il peut encore
  inviter sur ce workspace. L'email est renvoyé, et l'invitation est
  prolongée de 7 jours à partir de maintenant, qu'elle ait expiré ou non.

Côté invité :

- `GET /guest-invitations/received` : invitations en attente et non
  expirées ;
- `PATCH /guest-invitations/{id}` avec `{ "status": "accepted" | "declined" }`.
  `410 invitation expired` si elle a expiré (l'auteur doit la renvoyer),
  `409 you already have access to this task` si la personne est déjà guest
  de la tâche **ou a rejoint l'organisation entre-temps** (un membre de
  l'orga ne peut pas être guest : il peut seulement refuser l'invitation).
- Une invitation déjà acceptée ou refusée → `404 invitation not found`.
- `POST /guest-invitations/{id}/resend` par quelqu'un d'autre que l'auteur →
  `404 invitation not found`.

---

## 3. Récupérer les droits côté front

### 3.1 Abilities CASL (recommandé)

`GET /api/v1/workspaces/{id}/abilities` renvoie les règles à passer
directement à `createMongoAbility` de `@casl/ability`.

| Sujet | Actions possibles | Condition |
|---|---|---|
| `Workspace` | `read`, `update`, `delete`, `share` (gérer les accès) | `{ id }` |
| `Column` | `read`, `create`, `update`, `delete` | `{ workspace_id }` |
| `Task` | `read`, `create`, `update`, `delete`, `assign`, `invite` (guests) | `{ workspace_id }`, ou `{ id }` pour un guest |

Il n'y a plus d'action `manage` ni de règle `inverted`.

Exemple pour un **member `edit`** :

```json
[
  { "action": "read",   "subject": "Workspace", "conditions": { "id": "w1" } },
  { "action": "read",   "subject": "Column",    "conditions": { "workspace_id": "w1" } },
  { "action": "read",   "subject": "Task",      "conditions": { "workspace_id": "w1" } },
  { "action": "update", "subject": "Column",    "conditions": { "workspace_id": "w1" } },
  { "action": "update", "subject": "Task",      "conditions": { "workspace_id": "w1" } },
  { "action": "assign", "subject": "Task",      "conditions": { "workspace_id": "w1" } }
]
```

Exemple pour un **guest**, en `edit` sur `t42` et en `view` sur `t43` :

```json
[
  { "action": "read",   "subject": "Workspace", "conditions": { "id": "w1" } },
  { "action": "read",   "subject": "Task", "conditions": { "id": "t42" } },
  { "action": "update", "subject": "Task", "fields": ["description"], "conditions": { "id": "t42" } },
  { "action": "read",   "subject": "Task", "conditions": { "id": "t43" } }
]
```

Côté Angular :

```ts
import { createMongoAbility, subject } from '@casl/ability';

const rules = await firstValueFrom(
  this.http.get<RawRule[]>(`/api/v1/workspaces/${workspaceId}/abilities`),
);
const ability = createMongoAbility(rules);

// Les tâches renvoyées par l'API n'ont que `column_id` : ajouter
// `workspace_id` avant de tester.
const t = subject('Task', { ...task, workspace_id: workspaceId });

ability.can('update', t);                  // modifier la tâche (tous champs)
ability.can('update', t, 'description');   // modifier la description (guest edit inclus)
ability.can('delete', t);
ability.can('assign', t);
ability.can('invite', t);
ability.can('create', subject('Column', { workspace_id: workspaceId }));
ability.can('share', subject('Workspace', { id: workspaceId }));
ability.can('delete', subject('Workspace', { id: workspaceId }));
```

Attention au champ : pour un guest `edit`, `ability.can('update', t)` sans
champ vaut `true` avec CASL (la règle existe pour au moins un champ). Pour
afficher l'édition complète d'une tâche, tester un champ autre que
`description`, par exemple `ability.can('update', t, 'name')`.

Le drag & drop (réordonner, changer de colonne) suit `update` sur `Task`,
testé avec un autre champ que `description`.

Recharger les abilities à chaque entrée dans un workspace et après tout
changement de rôle.

### 3.2 Ce que les abilities ne couvrent pas

- **Créer un workspace** : rôle `owner` dans l'organisation
  (`GET /organisations` ou `GET /roles?organisation_id=...`).
- **Écrans de l'organisation** (renommer, supprimer, inviter, changer un
  rôle, retirer quelqu'un) : `role === 'owner'` dans
  `GET /organisations/{orgId}`.
- **Fiche d'accès d'un membre** : `role` vaut `owner` ou `admin`.
- **Choix des personnes dans l'écran de partage** : un admin ne peut
  sélectionner que des `member` (pas l'owner, pas d'autres admins, pas
  lui-même). L'owner peut sélectionner tout le monde sauf lui-même.

### 3.3 Autres routes utiles

| Route | Renvoie |
|---|---|
| `GET /roles?organisation_id={id}` | `{ "role": "owner" \| "admin" \| "member" }` (`404` si l'appelant n'est pas dans l'orga) |
| `GET /roles?workspace_id={id}` | `{ "role": "view" \| "edit" }` : `edit` si l'appelant peut modifier colonnes et tâches, toujours `view` pour un guest (`403` sans accès) |
| `GET /roles?workspace_id={id}&task_id={id}` | `{ "role": "view" \| "edit" }` : pour un guest, son rôle d'invitation (`edit` = description seulement) |
| `GET /workspaces/{id}/tasks/roles` | `[{ task_id, column_id, role }]` : chaque tâche visible avec le rôle de l'appelant |

`GET /roles` sans `organisation_id` ni `workspace_id` → `400`.

---

## 4. Erreurs liées aux droits

| Statut | Signification pour le front |
|---|---|
| `404` | La ressource n'existe pas **ou** l'appelant n'y a pas accès (workspace non partagé, organisation dont il ne fait pas partie, tâche non partagée à un guest). Retirer la ressource de l'état local et revenir à la liste. |
| `403` | La ressource est visible, mais le rôle ne permet pas l'action. Normalement évité si l'UI suit les abilities. |
| `409` | Invitation en doublon, ou personne déjà membre / déjà guest. |
| `410` | Invitation guest expirée : demander à l'auteur de la renvoyer. |

Messages `403` possibles :

- `organisation owner required`
- `organisation owner or admin required`
- `edit access required`
- `guests can only edit the description`
- `guest access is limited to shared tasks`
- `cannot change your own access` / `cannot manage this member's access`
- `cannot change your own role` / `cannot remove yourself`

---

## 5. Récap par rôle

**Owner** : tout, sur tous les workspaces de son organisation. Seul à pouvoir
créer ou supprimer un workspace, et à gérer l'organisation (renommer,
supprimer, inviter, changer un rôle, retirer quelqu'un).

**Admin** (workspace partagé avec lui, `view` ou `edit`) : renomme le
workspace, crée, modifie et supprime colonnes et tâches, assigne, invite des
guests, gère les accès des `member`. Ne supprime ni le workspace ni l'orga, ne
touche ni l'owner ni un autre admin, ni son propre accès.

**Member `edit`** : modifie, réordonne et déplace colonnes et tâches, assigne.
Ne crée et ne supprime rien, n'invite personne.

**Member `view`** : lecture seule.

**Guest `edit`** : voit ses tâches, en modifie la description.

**Guest `view`** : voit ses tâches.

---

## 6. Migration depuis l'ancienne version

| Avant | Maintenant |
|---|---|
| `GET /organisation` | `GET /organisations` (liste) + `GET /organisations/{orgId}` |
| `POST /organisation/invitations` | `POST /organisations/{orgId}/invitations` |
| `GET /organisation/invitations/sent` | `GET /organisations/{orgId}/invitations` |
| `GET /organisation/invitations/received` | `GET /organisation-invitations/received` |
| `PATCH /organisation/invitations/{id}` | `PATCH /organisation-invitations/{id}` |
| `GET /organisation/members/{id}/profile` | `GET /organisations/{orgId}/members/{memberId}/profile` (owner et admin seulement) |
| `GET /roles` (rôle orga implicite) | `GET /roles?organisation_id={id}` (`400` sans paramètre) |
| — | `PATCH` / `DELETE /organisations/{orgId}` |
| — | `PATCH` / `DELETE /organisations/{orgId}/members/{memberId}` |
| — | `PUT` / `DELETE .../tasks/{taskId}/guests/{userId}` |
| — | `POST /guest-invitations/{id}/resend` |

Changements de comportement :

- **Plusieurs organisations** par utilisateur : l'orga courante doit être
  choisie côté front, et `organisation_id` est présent sur chaque workspace.
- On ne crée un workspace **que dans l'organisation qu'on possède**. Le
  créateur n'a plus de droits
  particuliers : il n'y a plus de notion de « créateur » dans les permissions.
- **Seul l'owner supprime** un workspace (avant : le créateur ou un admin).
- **Member `edit`** : ne crée et ne supprime plus ni colonnes ni tâches.
- **Admin** : gère tout workspace partagé avec lui, même en `view`.
- **Assignations** : plus de champ `role`, plus aucun droit donné. Les
  guests apparaissent dans les assignés (`is_guest`).
- **Guests** : ne voient plus le board entier, seulement leurs tâches. En
  `edit`, ils ne modifient que la description et ne peuvent plus supprimer.
  Les invitations expirent au bout de 7 jours.
- **Abilities** : actions `read` / `create` / `update` / `delete` / `share` /
  `assign` / `invite` au lieu de `manage`, plus de règles `inverted`, `fields`
  pour les guests.
- Les actions réservées à l'owner répondent maintenant `403` (et non plus
  `404`) à un membre de l'organisation.
