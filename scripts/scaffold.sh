#!/usr/bin/env bash
# Scaffold de fichiers Go avec le bon package.
#   scripts/scaffold.sh <nom> [layers=handler,repository,model,routes]
set -eu

name="${1:-}"
layers="${2:-handler,repository,model,routes}"

if [ -z "$name" ]; then
	echo "Usage: make scaffold name=<nom> [layers=handler,repository,model,routes]" >&2
	exit 1
fi

IFS=','
for layer in $layers; do
	case "$layer" in
		handler)        dir=internal/handler;    file="$name.handler.go";    pkg=handler ;;
		repository|repo) dir=internal/repository; file="$name.repository.go"; pkg=repository ;;
		model|models)   dir=internal/models;     file="$name.go";            pkg=models ;;
		routes|route)   dir=internal/routes;     file="$name.routes.go";     pkg=routes ;;
		*) echo "couche inconnue: $layer" >&2; exit 1 ;;
	esac
	mkdir -p "$dir"
	if [ -e "$dir/$file" ]; then
		echo "existe deja, ignore: $dir/$file"
	else
		printf 'package %s\n' "$pkg" > "$dir/$file"
		echo "cree: $dir/$file"
	fi
done
