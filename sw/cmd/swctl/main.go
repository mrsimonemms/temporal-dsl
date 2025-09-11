package main

import (
	"context"
	"flag"
	"fmt"
	"log"

	"github.com/your-org/sw/internal/loader"
	"github.com/your-org/sw/internal/runtime"
	"github.com/your-org/sw/internal/tasks/http"
	"github.com/your-org/sw/internal/tasks/run"
)

func main() {
	file := flag.String("f", "", "path to workflow (yaml/json)")
	flag.Parse()

	if *file == "" {
		log.Fatal("missing -f workflow file")
	}

	wf, err := loader.FromFile(*file)
	if err != nil {
		log.Fatalf("workflow invalid: %v", err)
	}

	reg := runtime.NewRegistry(
		http.New(),
		run.New(),
	)
	engine := runtime.NewEngine(reg)

	out, err := engine.Run(context.Background(), wf, map[string]any{})
	if err != nil {
		log.Fatalf("run error: %v", err)
	}
	fmt.Printf("Output: %#v\n", out)
}
