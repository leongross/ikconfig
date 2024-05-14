package main

import (
	"fmt"
	"os"

	"github.com/leongross/extract-ikconfig/pkg/ikconfig"
)

func main() {
	file := os.Args[1]

	if _, err := os.Stat(file); err != nil {
		fmt.Printf("file %q not found: %v\n", file, err)
		os.Exit(1)
	}

	kernel, err := ikconfig.NewKernelConfig(file, ikconfig.KERNEL_COMPRESSION_TYPE_BZIP2)
	if err != nil {
		fmt.Printf("error creating kernel config: %v\n", err)
		os.Exit(1)
	}

	kc, err := kernel.ParseKernelConfig()
	if err != nil {
		fmt.Printf("error parsing kernel config: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("kernel config: %v\n", kc)
}
