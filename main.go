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
	"golang.org/x/xerrors"
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
	buildKit := flag.Bool("buildkit", false, "BuildKit")
	flag.Parse()

	if *imageName == "" {
		log.Print("-i <image name> is must be required")
		invalidArgs()
	}

	stageNames, err := readStageNames(*dockerfilePath)
	if err != nil {
		return xerrors.Errorf("read stage names: %w", err)
	}

	cachedStages := make([]string, 0, len(stageNames))
	fmt.Println("set -ex")
	if *buildKit {
		fmt.Printf("docker build -t %s --cache-from=%s --build-arg BUILDKIT_INLINE_CACHE=1 %s\n", fmt.Sprintf("%s:%s", *imageName, stageNames[len(stageNames)-1]), strings.Join(stageNames, ","), *buildContext)
		for i, stageName := range stageNames {
			if i == len(stageNames)-1 {
				break
			}
			fmt.Printf("docker build -t %s --target=%s --build-arg BUILDKIT_INLINE_CACHE=1 %s &\n", fmt.Sprintf("%s:%s", *imageName, stageName), stageName, *buildContext)
		}
		fmt.Println("wait")
		if *push {
			for _, stageName := range stageNames {
				fmt.Printf("docker push %s &\n", fmt.Sprintf("%s:%s", *imageName, stageName))
			}
			fmt.Println("wait")
		}
	} else {
		for _, stageName := range stageNames {
			cachedStage := fmt.Sprintf("%s:%s", *imageName, stageName)
			cachedStages = append(cachedStages, cachedStage)
			fmt.Printf("docker pull %s || true\n", cachedStage)
			fmt.Printf("docker build -t %s --target=%s --cache-from=%s %s\n", cachedStage, stageName, strings.Join(cachedStages, ","), *buildContext)
			if *push {
				fmt.Printf("docker push %s\n", cachedStage)
			}
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
			return nil, xerrors.Errorf("open Dockerfile: %w", err)
		}
		defer f.Close()
		r = f
	}
	result, err := parser.Parse(r)
	if err != nil {
		return nil, xerrors.Errorf("parse Dockerfile: %w", err)
	}

	stages, _, err := instructions.Parse(result.AST)
	if err != nil {
		return nil, xerrors.Errorf("parse instructions: %w", err)
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
