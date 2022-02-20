package main

import (
	"crypto/md5"
	"encoding/hex"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

var selfName string

func main() {
	selfName = filepath.Base(os.Args[0])
	log.SetFlags(log.Lmsgprefix)
	log.SetPrefix(selfName + ": ")

	verbose := false
	vlog := log.New(os.Stdout, selfName+" debug: ", log.Lmsgprefix)
	flag.BoolVar(&verbose, "v", false, "verbose logging")
	flag.Parse()

	if flag.NArg() < 1 {
		fmt.Printf(`%[1]s usage:

	%[1]s [-v] PATH

OPTIONS:
	-v	Enable verbose logging

ARGUMENTS:
	PATH	A valid file path to the Go code to be run
`, selfName)
		os.Exit(2)
	}
	if !verbose {
		vlog.SetOutput(io.Discard)
	}

	filename := flag.Arg(0)
	code, err := os.ReadFile(filename)
	if err != nil {
		log.Fatalf("Could not read supplied filepath %q: %v", filename, err)
	}
	contents := string(code)
	shebangSplit := strings.SplitN(contents, "\n", 2)
	if len(shebangSplit) > 1 && strings.HasPrefix(shebangSplit[0], "#!") {
		contents = shebangSplit[1]
	}

	vlog.Println("Creating temporary compilation directory")
	codeDir := makeCompilationDir(filename)
	vlog.Printf("Temporary compilation directory created: %q\n", codeDir)

	binaryName := filepath.Base(filename)
	if idx := strings.LastIndex(binaryName, "."); idx >= 0 {
		binaryName = binaryName[:idx]
	}
	vlog.Printf("Using %q as script binary name\n", binaryName)

	vlog.Printf("Checking contents of temporary compilation directory %q\n", codeDir)
	dirContents, err := os.ReadDir(codeDir)
	if err != nil {
		log.Fatalf("Could not read the contents of temporary compilation directory %q: %v", codeDir, err)
	}

	skipCopy := false
	skipInit := false
	skipCompile := false
	containsBinary := false
	for _, c := range dirContents {
		vlog.Printf("Found file in temporary compilation directory: %q\n", c.Name())
		switch c.Name() {
		case "main.go":
			h := md5.New()
			if _, err = h.Write([]byte(contents)); err != nil {
				log.Fatalf("Could not compute checksum for script contents: %v", err)
			}
			contentsSum := hex.EncodeToString(h.Sum(nil))
			existingSum := getChecksum(filepath.Join(codeDir, c.Name()))
			if contentsSum == existingSum {
				vlog.Println("Checksum of existing 'main.go' in temporary compilation directory matched the supplied file, skipping copying")
				skipCopy = true
				if containsBinary {
					skipCompile = true
				}
			}
		case "go.mod":
			skipInit = true
		case binaryName:
			containsBinary = true
			if skipCopy {
				skipCompile = true
			}
		default:
		}
	}

	if !skipCopy {
		if err := os.WriteFile(filepath.Join(codeDir, "main.go"), []byte(contents), 0666); err != nil {
			log.Fatalf("Could not write script contents to compilation directory %q: %v", codeDir, err)
		}
	} else {
		vlog.Println("Skipping copy for script contents")
	}
	if err := os.Chdir(codeDir); err != nil {
		log.Fatalf("Could not navigate to temporary compilation directory %q: %v", codeDir, err)
	}
	if !skipInit {
		if output, err := exec.Command("go", "mod", "init", binaryName).CombinedOutput(); err != nil {
			log.Fatalf("Could not initialise Go module in temporary compilation directory %q: %v\nOutput:\n%s\n", codeDir, err, output)
		}
	} else {
		vlog.Println("Skipping Go module initialisation for temporary compilation directory")
	}
	if !skipCompile {
		if output, err := exec.Command("go", "mod", "tidy").CombinedOutput(); err != nil {
			log.Fatalf("Could not tidy Go module in temporary compilation directory %q: %v\nOutput:\n%s\n", codeDir, err, output)
		}
		if output, err := exec.Command("go", "build").CombinedOutput(); err != nil {
			log.Fatalf("Could not build script in temporary compilation directory %q: %v\nOutput:\n%s\n", codeDir, err, output)
		}
		containsBinary = true
	} else {
		vlog.Println("Skipping compilation")
	}
	if !containsBinary {
		log.Fatalf("Could not locate compiled binary to execute in temporary compilation directory %q", codeDir)
	}

	binaryRun := exec.Command("./" + binaryName)
	binaryRun.Stdin = os.Stdin
	binaryRun.Stdout = os.Stdout
	binaryRun.Stderr = os.Stderr

	vlog.Println("Executing compiled binary")
	err = binaryRun.Run()
	if err != nil {
		log.Fatalf("Error occurred during script execution: %v", err)
	}

}

func getChecksum(filename string) string {
	f, err := os.Open(filepath.Clean(filename))
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()

	h := md5.New()
	if _, err := io.Copy(h, f); err != nil {
		log.Fatal(err)
	}

	return hex.EncodeToString(h.Sum(nil))
}

func makeCompilationDir(filename string) string {
	envDir := filepath.Join(os.TempDir(), selfName)
	stats, err := os.Stat(envDir)
	if os.IsNotExist(err) {
		err = os.Mkdir(envDir, 0777)
		if err != nil {
			log.Fatalf("Could not create temporary compilation directory %q: %v", envDir, err)
		}
	}
	if err != nil {
		log.Fatalf("Could not stat temporary compilation directory %q: %v", envDir, err)
	}
	if stats != nil && !stats.IsDir() {
		log.Fatalf("Temporary compilation directory %q already exists and is a file", envDir)
	}

	cleanedFilename := filepath.Clean(filename)
	filenameChecksum := hex.EncodeToString(md5.New().Sum([]byte(cleanedFilename)))
	codeDir := filepath.Join(envDir, filenameChecksum)
	err = os.MkdirAll(codeDir, 0777)
	if err != nil {
		log.Fatalf("Could not create compilation directory %q for file: %v", codeDir, err)
	}

	return codeDir
}
