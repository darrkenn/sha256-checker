package main

import (
	"fmt"
	"log"

	"github.com/gin-gonic/gin"
)

func main() {
	fmt.Println("Starting sha256 checker")

	r := gin.Default()

	err := r.Run(":3119")
	if err != nil {
		log.Fatal(err)
		return
	}
}
