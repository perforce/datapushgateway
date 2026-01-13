// Tool to generate bcrypt password hashes for use in pushgateway format basic auth config files.
// This has been replaced by the vmaught format config file in the vmauth project, but is left here for posterity.
package main

import (
	"fmt"
	"log"

	"golang.org/x/crypto/bcrypt"
)

func main() {
	// Prompt the user for a password
	var password string
	fmt.Print("Enter Password: ")
	fmt.Scanln(&password)

	// Generate a bcrypt hash
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		log.Fatal(err)
	}

	// Print the bcrypt hash
	fmt.Println(string(hashedPassword))
}
