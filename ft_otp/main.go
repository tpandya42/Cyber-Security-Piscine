package main

import (
	"fmt"
	"os"
)

func parseArgs(args []string) (string, error) {
	
	if len(args) < 1 {
		return "", fmt.Errorf("invalid number of arguments")
	}
	
	var mode string

	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "-g":
			handleG(args[i + 1])
		case "-k":
			handleK(args[i + 1])
		default:
			fmt.Errorf("Invalid Input")
		}
	}
	return "", nil
}

func main(){
	// Parse and validate input
	mode, _ := parseArgs(os.Args[1:])
	// Generate the key
	// Generate random 6 digit otp based on the key
}
