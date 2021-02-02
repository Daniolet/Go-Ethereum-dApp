package utils

import (
	"bufio"
	"log"
	"os"
	"strings"
)

//readStringStdin reads a string from STDIN and strips and trailing \n characters from it
func ReadStringStdin() string {
	reader := bufio.NewReader(os.Stdin) //pause the program and wait for user input
	inputVal, err := reader.ReadString('\n')
	if err != nil {
		log.Println("invalid option: ", err)
		return ""
	}

	output := strings.TrimSuffix(inputVal, "\n")
	return output
}
