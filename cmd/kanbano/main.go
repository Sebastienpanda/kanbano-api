package main

import (
	"bufio"
	"context"
	"fmt"
	"kanbano-api/internal/routes"
	"net/http"
	"os"
	"os/exec"
	"sort"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/joho/godotenv"
)

func main() {
	_ = godotenv.Load()

	fmt.Println("kanbano-cli — tape 'help' pour la liste des commands, 'exit' pour quitter.")

	scanner := bufio.NewScanner(os.Stdin)
	for {
		fmt.Print("kanbano> ")
		if !scanner.Scan() {
			fmt.Println()
			return
		}

		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		args := strings.Fields(line)
		cmd, rest := args[0], args[1:]

		switch cmd {
		case "help", "?":
			printHelp()
		case "exit", "quit":
			return
		case "migrate-up":
			runMigrate("up")
		case "migrate-down":
			runMigrate("down")
		case "migrate-status":
			runMigrate("version")
		case "migrate-create":
			migrateCreate(rest)
		case "install-hooks":
			installHooks()
		case "routes":
			printRoutes()
		case "scaffold":
			scaffold(rest)
		default:
			fmt.Printf("commande inconnue : %q — tape 'help'\n", cmd)
		}
	}
}

func printHelp() {
	fmt.Println(`Commands disponibles :
  help                          affiche cette aide
  migrate-up                    applique les migrations en attente
  migrate-down                  annule la dernière migration
  migrate-status                affiche la version de migration courante
  migrate-create <name>         crée une nouvelle paire de fichiers de migration
  install-hooks                 configure git pour utiliser .githooks
  routes                        liste toutes les routes HTTP enregistrées
  scaffold <name> [layers]      génère handler/repository/model/routes pour <name>
  exit                          quitte le CLI`)
}

func runMigrate(action string) {
	dbURL := os.Getenv("DATABASE_URL_MIGRATE")
	if dbURL == "" {
		fmt.Println("DATABASE_URL_MIGRATE n'est pas défini (vérifie ton .env)")
		return
	}
	// CLI dev local : dbURL vient du .env de l'opérateur, action est une constante interne.
	//nolint:gosec // outil interne, pas exposé réseau
	run(exec.CommandContext(context.Background(), "migrate", "-path", "internal/db/migrations", "-database", dbURL, action))
}

func migrateCreate(args []string) {
	if len(args) != 1 {
		fmt.Println("usage: migrate-create <name>")
		return
	}
	// CLI dev local : args[0] vient de la saisie de l'opérateur au terminal.
	//nolint:gosec // outil interne, pas exposé réseau
	run(exec.CommandContext(context.Background(), "migrate", "create", "-ext", "sql", "-dir", "internal/db/migrations", "-seq", args[0]))
}

func installHooks() {
	run(exec.CommandContext(context.Background(), "git", "config", "core.hooksPath", ".githooks"))
}

type routeEntry struct {
	method string
	path   string
}

func printRoutes() {
	r := routes.SetupRouter(nil, nil)

	groups := make(map[string][]routeEntry)
	var groupOrder []string

	_ = chi.Walk(r, func(method, route string, _ http.Handler, _ ...func(http.Handler) http.Handler) error {
		group := routeGroup(route)
		if _, exists := groups[group]; !exists {
			groupOrder = append(groupOrder, group)
		}
		groups[group] = append(groups[group], routeEntry{method: method, path: route})
		return nil
	})

	sort.Strings(groupOrder)

	for i, group := range groupOrder {
		if i > 0 {
			fmt.Println()
		}
		fmt.Printf("%s%s%s\n", colorHeader, group, colorReset)
		for _, e := range groups[group] {
			fields := bodyFieldsFor(e.method, e.path)
			if len(fields) > 0 {
				fmt.Printf("  %s %s %s→%s\n", colorMethod(e.method), colorPath(e.path), colorGray, colorReset)
				for _, f := range fields {
					fmt.Printf("      %s\n", colorBodyField(f))
				}
			} else {
				fmt.Printf("  %s %s\n", colorMethod(e.method), colorPath(e.path))
			}
		}
	}
}

// routeGroup renvoie le premier segment du chemin après /api/v1/, ex.
// "/api/v1/organisation/invitations/{id}" -> "organisation".
func routeGroup(route string) string {
	trimmed := strings.TrimPrefix(route, "/api/v1/")
	trimmed = strings.TrimPrefix(trimmed, "/")
	if trimmed == "" {
		return "/"
	}
	if idx := strings.Index(trimmed, "/"); idx != -1 {
		return trimmed[:idx]
	}
	return trimmed
}

const (
	colorReset  = "\033[0m"
	colorGet    = "\033[32m"  // vert
	colorPost   = "\033[33m"  // jaune
	colorPut    = "\033[34m"  // bleu
	colorPatch  = "\033[36m"  // cyan
	colorDelete = "\033[31m"  // rouge
	colorHeader = "\033[1;4m" // gras souligné
	colorGray   = "\033[90m"  // gris
	colorParam  = "\033[35m"  // magenta
)

// colorPath grise le chemin et met en évidence les paramètres {xxx} en magenta.
func colorPath(path string) string {
	segments := strings.Split(path, "/")
	for i, seg := range segments {
		if strings.HasPrefix(seg, "{") && strings.HasSuffix(seg, "}") {
			segments[i] = colorParam + seg + colorGray
		}
	}
	return colorGray + strings.Join(segments, "/") + colorReset
}

// colorBodyField formatte un champ de body : nom en cyan clair, type en gris,
// et un astérisque rouge s'il est requis.
func colorBodyField(f bodyField) string {
	marker := ""
	if f.required {
		marker = colorDelete + "*" + colorReset
	}
	return fmt.Sprintf("%s%s%s %s(%s)%s%s", colorParam, f.name, colorReset, colorGray, f.typ, colorReset, marker)
}

func colorMethod(method string) string {
	var color string
	switch method {
	case http.MethodGet:
		color = colorGet
	case http.MethodPost:
		color = colorPost
	case http.MethodPut:
		color = colorPut
	case http.MethodPatch:
		color = colorPatch
	case http.MethodDelete:
		color = colorDelete
	default:
		return fmt.Sprintf("%-7s", method)
	}
	return fmt.Sprintf("%s%-7s%s", color, method, colorReset)
}

func scaffold(args []string) {
	if len(args) < 1 {
		fmt.Println("usage: scaffold <name> [layers]")
		return
	}
	name := args[0]
	layers := "handler,repository,model,routes"
	if len(args) > 1 {
		layers = args[1]
	}
	// CLI dev local : name/layers viennent de la saisie de l'opérateur au terminal.
	//nolint:gosec // outil interne, pas exposé réseau
	run(exec.CommandContext(context.Background(), "bash", "scripts/scaffold.sh", name, layers))
}

func run(c *exec.Cmd) {
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr
	c.Stdin = os.Stdin
	if err := c.Run(); err != nil {
		fmt.Printf("erreur : %v\n", err)
	}
}
