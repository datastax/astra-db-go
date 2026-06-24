#!/usr/bin/env bash
set -e

REPO_ROOT=$(git rev-parse --show-toplevel 2>/dev/null || pwd)
OUT_DIR="$REPO_ROOT/astra/serdes/testdata/fuzz/FuzzStdlibDrift"

TMP_DIR=$(mktemp -d)
trap 'rm -rf "$TMP_DIR"' EXIT

echo "Downloading dvyukov/go-fuzz-corpus/json/corpus..."
(
  cd "$TMP_DIR"
  git clone --quiet --depth 1 --filter=blob:none --sparse https://github.com/dvyukov/go-fuzz-corpus.git go-fuzz-corpus-master
  cd go-fuzz-corpus-master
  git sparse-checkout set json/corpus
)

rm -rf "$OUT_DIR"
mkdir -p "$OUT_DIR"

cat << 'EOF' > "$TMP_DIR/convert.go"
package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
)

func main() {
	in := os.Args[1]
	out := os.Args[2]

	files, err := os.ReadDir(in)
	if err != nil {
		log.Fatalf("Failed to read corpus: %v\n", err)
	}

	count := 0
	for _, f := range files {
		data, err := os.ReadFile(filepath.Join(in, f.Name()))
		if err != nil {
		  log.Fatalf("Failed to read file %s: %v\n", f.Name(), err)
		}

		outPath := filepath.Join(out, f.Name())
		outData := fmt.Sprintf("go test fuzz v1\n[]byte(%q)\n", data)

		if err := os.WriteFile(outPath, []byte(outData), 0644); err != nil {
			log.Fatalf("Failed to write file %s: %v\n", outPath, err)
		}
		count++
	}
	fmt.Printf("Successfully migrated %d files to %s\n", count, out)
}
EOF

go run "$TMP_DIR/convert.go" "$TMP_DIR/go-fuzz-corpus-master/json/corpus" "$OUT_DIR"
