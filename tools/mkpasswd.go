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
