package main

import (
	"fmt"
	jupag "github.com/verzth/go-jup-ag"
)

func main() {
	client := jupag.NewJupag()
	prcs, e := client.Price(jupag.PriceParams{
		IDs: "JitoSOL,SOL",
	})

	if e != nil {
		panic(e)
	}

	for _, prc := range prcs {
		fmt.Printf("%s [%s]: %.06f %s\n", prc.ID, prc.MintSymbol, prc.Price, prc.VsTokenSymbol)
	}
}
