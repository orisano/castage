package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"strings"

	"github.com/moby/buildkit/frontend/dockerfile/instructions"
	"github.com/moby/buildkit/frontend/dockerfile/parser"
	"github.com/pkg/errors"
)

func main() {
	log.SetFlags(0)
	log.SetPrefix("castage: ")

	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {
	dockerfilePath := flag.String("f", "Dockerfile", "Dockerfile path")
	imageName := flag.String("i", "", "image name (required)")
	buildContext := flag.String("p", ".", "build context")
	push := flag.Bool("push", false, "push")
	flag.Parse()

	if *imageName == "" {
		log.Print("-i <image name> is must be required")
		invalidArgs()
	}

	stageNames, err := readStageNames(*dockerfilePath)
	if err != nil {
		return errors.Wrap(err, "failed to read stage names")
	}

	cachedStages := make([]string, 0, len(stageNames))
	fmt.Println("set -ex")
	for _, stageName := range stageNames {
		cachedStage := fmt.Sprintf("%s:%s-cache", *imageName, stageName)
		cachedStages = append(cachedStages, cachedStage)
		fmt.Printf("docker pull %s || true\n", cachedStage)
		fmt.Printf("docker build -t %s --target=%s --cache-from=%s %s\n", cachedStage, stageName, strings.Join(cachedStages, ","), *buildContext)
		if *push {
			fmt.Printf("docker push %s\n", cachedStage)
		}
	}

	return nil
}

func readStageNames(dockerfilePath string) ([]string, error) {
	var r io.Reader
	if dockerfilePath == "-" {
		r = os.Stdin
	} else {
		f, err := os.Open(dockerfilePath)
		if err != nil {
			return nil, errors.Wrap(err, "failed to open Dockerfile")
		}
		defer f.Close()
		r = f
	}
	result, err := parser.Parse(r)
	if err != nil {
		return nil, errors.Wrap(err, "failed to parse Dockerfile")
	}

	stages, _, err := instructions.Parse(result.AST)
	if err != nil {
		return nil, errors.Wrap(err, "failed to parse instructions")
	}

	stageNames := make([]string, 0, len(stages))
	for _, stage := range stages {
		if stage.Name == "" {
			continue
		}
		stageNames = append(stageNames, stage.Name)
	}
	return stageNames, nil
}

func invalidArgs() {
	flag.Usage()
	os.Exit(2)
}