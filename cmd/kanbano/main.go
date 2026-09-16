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

	fmt.Println("kanbano-cli — type 'help' for the list of commands, 'exit' to quit.")

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
			fmt.Printf("unknown command: %q — type 'help'\n", cmd)
		}
	}
}

func printHelp() {
	fmt.Println(`Available commands:
  help                          shows this help
  migrate-up                    applies pending migrations
  migrate-down                  rolls back the last migration
  migrate-status                shows the current migration version
  migrate-create <name>         creates a new pair of migration files
  install-hooks                 configures git to use .githooks
  routes                        lists all registered HTTP routes
  scaffold <name> [layers]      generates handler/repository/model/routes for <name>
  exit                          quits the CLI`)
}

func runMigrate(action string) {
	dbURL := os.Getenv("DATABASE_URL_MIGRATE")
	if dbURL == "" {
		fmt.Println("DATABASE_URL_MIGRATE is not set (check your .env)")
		return
	}
	// Local dev CLI: dbURL comes from the operator's .env, action is an internal constant.
	//nolint:gosec // internal tool, not exposed over the network
	run(exec.CommandContext(context.Background(), "migrate", "-path", "internal/db/migrations", "-database", dbURL, action))
}

func migrateCreate(args []string) {
	if len(args) != 1 {
		fmt.Println("usage: migrate-create <name>")
		return
	}
	// Local dev CLI: args[0] comes from the operator's terminal input.
	//nolint:gosec // internal tool, not exposed over the network
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

// routeGroup returns the first path segment after /api/v1/, e.g.
// "/api/v1/organisation/invitations/{id}" -> "organisation".
func routeGroup(route string) string {
	trimmed := strings.TrimPrefix(route, "/api/v1/")
	trimmed = strings.TrimPrefix(trimmed, "/")
	if trimmed == "" {
		return "/"
	}
	if before, _, ok := strings.Cut(trimmed, "/"); ok {
		return before
	}
	return trimmed
}

const (
	colorReset  = "\033[0m"
	colorGet    = "\033[32m"  // green
	colorPost   = "\033[33m"  // yellow
	colorPut    = "\033[34m"  // blue
	colorPatch  = "\033[36m"  // cyan
	colorDelete = "\033[31m"  // red
	colorHeader = "\033[1;4m" // bold underline
	colorGray   = "\033[90m"  // gray
	colorParam  = "\033[35m"  // magenta
)

// colorPath grays out the path and highlights {xxx} parameters in magenta.
func colorPath(path string) string {
	segments := strings.Split(path, "/")
	for i, seg := range segments {
		if strings.HasPrefix(seg, "{") && strings.HasSuffix(seg, "}") {
			segments[i] = colorParam + seg + colorGray
		}
	}
	return colorGray + strings.Join(segments, "/") + colorReset
}

// colorBodyField formats a body field: name in light cyan, type in gray,
// and a red asterisk if required.
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
	// Local dev CLI: name/layers come from the operator's terminal input.
	//nolint:gosec // internal tool, not exposed over the network
	run(exec.CommandContext(context.Background(), "bash", "scripts/scaffold.sh", name, layers))
}

func run(c *exec.Cmd) {
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr
	c.Stdin = os.Stdin
	if err := c.Run(); err != nil {
		fmt.Printf("error: %v\n", err)
	}
}
