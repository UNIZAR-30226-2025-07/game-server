package main

import (
	"fmt"

	"galaxy.io/server/galaxy/operations"
)

func main() {
	fmt.Println("Hello Nix!")
  a := []operations.OperationType{
    operations.OpJoin,
  }
  fmt.Printf("{}", a);
}
