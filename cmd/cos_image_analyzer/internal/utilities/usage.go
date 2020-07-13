package utilities

import (
	"flag"
	"fmt"
	"os"
)

// Custom usage function. See -h flag
func printUsage() {
	fmt.Println("NAME\ncos_image_analyzer - finds all meaningful differences of two COS Images")
	fmt.Print("(binary, package, commit, and release notes differences)\n\n")
	fmt.Printf("SYNOPSIS\n%s [OPTION] argument1 argument2\n\nDESCRIPTION\n", os.Args[0])
	fmt.Print("Default: input arguments are two local filesystem paths to root directiory of COS images\n\n")
	flag.PrintDefaults()
	fmt.Println("\nOUPUT\n(stdout) terminal ouput - All differences printed to the terminal")
}
