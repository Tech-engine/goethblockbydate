# Go Ethereum Block By Date

[![Go Report Card](https://goreportcard.com/badge/github.com/Tech-engine/goethblockbydate)](https://goreportcard.com/report/github.com/Tech-engine/go)

Get Ethereum block number by a given date.

Works with any Ethereum based mainnet or testnet networks.

Works with [go-ethereum](https://github.com/ethereum/go-ethereum/)

This package is inspired and ported from [ethereum-block-by-date.js](https://github.com/monosux/ethereum-block-by-date)

## Installation

```
go get github.com/Tech-engine/goethblockbydate
```

## Usage

```go
package main
import (
	"github.com/Tech-engine/goethblockbydate"
	"log"
	"fmt"
)

func main() {
	// get node url
	goethblockbydate.SetNode("YOUR_INFURA_OR_ANY_OTHER_NODE_URL")
	block, err := goethblockbydate.GetDate("2021-02-05T00:00:00Z",  true)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(block.BlockNumber)
}
```
