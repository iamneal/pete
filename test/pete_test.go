package test

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path"
	"runtime"
	"testing"
)

func TestMain(m *testing.M) {
	// get the file that this function is in.
	_, filepath, _, ok := runtime.Caller(0)
	if !ok {
		panic(fmt.Sprintf("could not get runtime info"))
	}
	// remove the last element (the filename)
	filepath = filepath[:len(filepath)-len(path.Base(filepath))]
	// the package we are going to build is right above our test package
	input := path.Clean(path.Join(filepath, "../"))
	// create a binary in our test directory
	output := path.Join(filepath, "testpete")

	// create a command that will build the package above ours, and put the binary in our test package
	command := exec.CommandContext(context.Background(), "go", "build", "-o", output, input)
	// we want output/errors to display to the terminal
	command.Stderr = os.Stderr
	command.Stdout = os.Stdout

	fmt.Printf("args: %+v\n", command.Args)

	err := command.Run()
	if err != nil {
		panic(fmt.Sprintf("error running command: %v", err))
	}
	// run our tests, they can now use the binary locally
	os.Exit(m.Run())
}
