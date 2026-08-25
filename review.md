# Review Go — kanbano-api

Review du diff courant (`git diff`, dev), skill `golang` (xobotyi/cc-foundry).

## Bugs / correction

### `requireWorkspace` masque les vraies erreurs serveur en 404
`internal/handler/common.go:52-56`

```go
exists, err := workspaceRepo.Exists(r.Context(), workspaceID, userID)
if err != nil || !exists {
    http.Error(w, "workspace not found", http.StatusNotFound)
    return userID, workspaceID, false
}
```

Une erreur DB (`err != nil`, ex. pool épuisé, timeout) répond `404 workspace not found` au lieu de `500`. C'est trompeur pour l'appelant et invisible en observabilité (pas de `serverError`/log). C'était déjà le cas avant (pattern dupliqué dans chaque handler), mais la centralisation dans `requireWorkspace` est l'occasion de corriger une bonne fois :

```go
exists, err := workspaceRepo.Exists(r.Context(), workspaceID, userID)
if err != nil {
    serverError(w, err)
    return userID, workspaceID, false
}
if !exists {
    http.Error(w, "workspace not found", http.StatusNotFound)
    return userID, workspaceID, false
}
```

### `parseTaskContext` ne vérifie pas que `columnId` appartient au workspace
`internal/handler/task.handler.go:69-83`

`requireWorkspace` vérifie que le workspace appartient au user, mais `columnID` est ensuite simplement parsé depuis l'URL sans vérifier `h.columnRepo.Exists(ctx, columnID, workspaceID)`. Un user authentifié peut donc passer l'ID d'une colonne d'un **autre** workspace (le sien ou celui d'un tiers) dans `.../workspaces/{id}/columns/{columnId}/tasks` et créer/modifier/reorder/supprimer une tâche hors du workspace ciblé par l'URL — faille d'autorisation IDOR. Pré-existant (pas introduit par ce diff), mais toujours présent après le refactor ; `Reorder` fait bien ce contrôle pour `TargetColumnID` (ligne 162), ce qui montre l'incohérence.

### Assertion de type non protégée sur les claims JWT
`internal/middleware/auth.go:41-42`

```go
claims := token.Claims.(jwt.MapClaims)
userID := claims["sub"].(string)
```

Deux type assertions sans `comma-ok`. Si `sub` est absent du token ou d'un type inattendu, panic (500 non géré, crash potentiel du goroutine si pas de recover en amont). Le skill impose la forme `comma-ok` systématique :

```go
userID, ok := claims["sub"].(string)
if !ok {
    http.Error(w, "invalid token", http.StatusUnauthorized)
    return
}
```

## Style / conventions

### Messages d'erreur mélangeant FR et EN
Le diff traduit une partie des messages HTTP (`"workspace not found"`, `"invalid request body"`, etc.) mais `internal/handler/task.handler.go` garde des messages non touchés en anglais correctement, alors que **les commentaires de code restent en français** (`common.go:29`, `:42`) et le message `"colonne cible introuvable dans ce workspace"` a été traduit mais pas systématiquement partout — à vérifier qu'il ne reste plus de message FR oublié ailleurs dans le repo (`state.repository.go`, autres handlers non touchés par ce diff). Choisir une convention unique (tout le texte API en anglais, ou tout en français) et l'appliquer au repo entier, pas fichier par fichier.

### Erreur ignorée sur `appMiddleware.InitJWKS`
Correction bienvenue dans le diff (`main.go`) — bien géré maintenant avec `log.Fatalf`.

### `err := ... ; if err != nil` remplacé par `err = ...` pour éviter le shadowing implicite
Bon changement dans les repositories (`column.repository.go`, `task.repository.go`, `workspace.repository.go`, `state.repository.go`) : le passage de `if err := tx.Exec(...); err != nil` vers une déclaration `err` explicite en dehors du `if` est cohérent avec la règle *"Always verify assignments in inner blocks use `=` (not `:=`) when targeting outer-scope variables"* — ça élimine le risque de shadowing sur `err` dans ces fonctions à transaction. Bon point.

### `respondCreated` et `requireWorkspace` : bonne réduction de duplication
`common.go` — factorisation logique et conforme au principe *consumer-side simplicity*. Une remarque mineure : `respondCreated[T any]` est un générique pour 4 lignes de logique très simple ; ça reste justifié ici vu la répétition (4 handlers), donc ok — pas d'abus d'abstraction.

### `ColumnHandler.parseWorkspaceContext` : indirection à évaluer
`internal/handler/column.handler.go:61-64`

```go
func (h *ColumnHandler) parseWorkspaceContext(w http.ResponseWriter, r *http.Request) (workspaceID uuid.UUID, ok bool) {
    _, workspaceID, ok = requireWorkspace(w, r, h.workspaceRepo)
    return workspaceID, ok
}
```

Wrapper qui ne fait qu'ignorer `userID` et renvoyer `requireWorkspace(w, r, h.workspaceRepo)`. Sachant qu'aucune méthode de `ColumnHandler` n'utilise `userID`, c'est un choix défendable pour la lisibilité des call sites, mais à la limite de l'abstraction inutile — un simple `_, workspaceID, ok := requireWorkspace(...)` inline dans chaque méthode (4 lignes) ferait la même chose sans indirection. À garder si le style d'équipe préfère nommer l'intention, sinon à supprimer.

### `serverError` : bon nettoyage
`log.Println("erreur serveur:", err)` → `log.Printf("internal server error: %v", err)` : conforme aux conventions de format Go (`%v` plutôt que concat via `Println`).

## Manques

### Aucun test
`find . -name '*_test.go'` ne retourne rien sur tout le repo. Les repositories contiennent de la logique de réordonnancement non triviale (`Reorder` dans `column.repository.go` et `task.repository.go`, avec branches selon `position < oldPosition`) — candidats évidents à des tests de table (`references/testing.md`), même via une DB de test réelle (le skill recommande *live services over mocks* pour ce type de requêtes SQL).

### `parseWorkspaceContext`/`requireWorkspace` : pas de vérification `err` vs `!exists` différenciée nulle part dans le diff
Voir bug ci-dessus — s'applique à tous les appelants (`ColumnHandler`, `TaskHandler`, `WorkspaceHandler.Get/Update/Delete` via `handleRepoError`, qui lui gère correctement `errors.Is(err, pgx.ErrNoRows)` vs erreur serveur — bon pattern à répliquer dans `requireWorkspace`).

## Résumé priorisé

1. **[Bug/sécurité]** Corriger `requireWorkspace` pour distinguer erreur DB (500) de "not found" (404).
2. **[Bug/sécurité — IDOR]** Vérifier que `columnId` appartient bien au `workspaceId` dans `parseTaskContext`.
3. **[Robustesse]** Sécuriser les type assertions sur les claims JWT dans `auth.go`.
4. **[Cohérence]** Unifier la langue des messages d'erreur HTTP et des commentaires sur tout le repo.
5. **[Dette]** Ajouter des tests de table pour la logique de `Reorder` (colonnes et tâches).
