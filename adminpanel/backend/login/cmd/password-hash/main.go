package main

import (
	"flag"
	"fmt"
	"log"
	"strings"

	"github.com/ivankudzin/tgapp/adminpanel/backend/login/internal/security"
)

func main() {
	password := flag.String("password", "", "plain password")
	flag.Parse()

	if strings.TrimSpace(*password) == "" {
		log.Fatal("use -password to pass plain password")
	}

	hash, err := security.HashPassword(*password)
	if err != nil {
		log.Fatalf("hash password: %v", err)
	}
	fmt.Println(hash)
}
