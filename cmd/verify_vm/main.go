package main

import (
	"fmt"
	"time"

	"github.com/antibyte/retroterm/pkg/tinybasic"
)

func main() {
	fmt.Println("Starting VM Verification...")

	// 1. Initialize Interpreter
	machine := tinybasic.NewTinyBASIC(nil)

	// 2. Set up a simple program that uses a loop (heavy VM usage)
	// 10 FOR I = 1 TO 1000
	// 20 A = A + 1
	// 30 NEXT I
	// 40 PRINT A
	machine.Execute("10 FOR I = 1 TO 1000")
	machine.Execute("20 A = A + 1")
	machine.Execute("30 NEXT I")
	machine.Execute("40 PRINT A")

	fmt.Println("Program loaded.")

	// 3. Enable Bytecode
	// We need to access private field useBytecode? No, usually exposed or default true?
	// vm.go: useBytecode: true by default (line 477 in original view).
	// We can invoke RUN command which uses bytecode if enabled.

	// 4. Run
	start := time.Now()
	// RUN command is async, but for this test we might want to wait?
	// or use machine.Execute("RUN") and wait for output.

	// We'll use the internal run method logic via Execute("RUN")
	// and listen to OutputChan.

	go func() {
		machine.Execute("RUN")
	}()

	fmt.Println("Running...")

	timeout := time.After(5 * time.Second)
	for {
		select {
		case msg := <-machine.OutputChan:
			fmt.Printf("Output: %v\n", msg)
			if msg.Content == "1000" { // Expected output
				fmt.Printf("Success! Output matched expected value.\n")
				fmt.Printf("Time taken: %v\n", time.Since(start))
				return
			}
			if msg.Content == "OKR" || msg.Content == "OK" {
				// Finished
			}
		case <-timeout:
			fmt.Println("Timeout waiting for output.")
			return
		}
	}
}
