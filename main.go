package main

import "github.com/demo/ledger-service/pkg"

func main() {
	if err := ledger.StartServer(); err != nil {
		panic(err)
	}
}
