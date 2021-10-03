package goethblockbydate

import (
	"context"
	"errors"
	"log"
	"math"
	"math/big"
	"strconv"
	"sync"

	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/nleeper/goment"
)

type newBlock struct {
	timestamp uint64
	number    *big.Int
}

type newBlockWrapper struct {
	Date        *goment.Goment
	BlockNumber *big.Int
	Timestamp   uint64
}

var savedBlocks map[string]*newBlock = make(map[string]*newBlock)

var checkedBlocks map[uint64][]*big.Int = make(map[uint64][]*big.Int)

var latestBlock, firstBlock *newBlock

var blockTime float64

var nodeUrl string

var requests uint

var mutex = &sync.Mutex{}

/* utils */
func contains(s *[]*big.Int, item *big.Int) bool {
	for _, v := range *s {
		if v.Cmp(item) == 0 {
			return true
		}
	}

	return false
}

func SetNode(url string) {
	nodeUrl = url
}

func setBoundaries() error {
	var err error

	latestBlock, err = getBlockWrapper(nil)
	if err != nil {
		return err
	}

	firstBlock, err = getBlockWrapper(big.NewInt(1))
	if err != nil {
		return err
	}

	deltaTimestamp, _ := new(big.Float).SetString(strconv.Itoa(int(latestBlock.timestamp - firstBlock.timestamp)))

	lastestBlockNumber := big.NewInt(0).Div(latestBlock.number, big.NewInt(1))
	lastestBlockNumberFloat := new(big.Float).SetInt(lastestBlockNumber)
	blockTime, _ = big.NewFloat(0).Quo(deltaTimestamp, lastestBlockNumberFloat).Float64()
	return nil
}

func GetDate(dateStr string, after bool) (*newBlockWrapper, error) {

	if nodeUrl == "" {
		return nil, errors.New("Node URL is required")
	}
	date, err := goment.New(dateStr, "YYYY-MM-DDTHH:mm:ssZ")

	if err != nil {
		return nil, err
	}

	date = date.UTC()

	if firstBlock == nil || latestBlock == nil || blockTime == 0 {
		err = setBoundaries()
		if err != nil {
			return nil, err
		}
	}

	if date.IsBefore(goment.Unix(int64(firstBlock.timestamp))) {
		return returnWrapper(date, big.NewInt(1)), nil
	}

	if date.IsSameOrAfter(goment.Unix(int64(latestBlock.timestamp))) {
		return returnWrapper(date, latestBlock.number), nil
	}

	mutex.Lock()
	checkedBlocks[uint64(date.ToUnix())] = []*big.Int{}
	mutex.Unlock()

	blockTimestamp, err := goment.Unix(int64(firstBlock.timestamp))
	if err != nil {
		return nil, err
	}

	diff := date.Diff(blockTimestamp, "seconds")

	skip := int(math.Ceil(float64(diff) / float64(blockTime)))

	blockNumber, _ := new(big.Int).SetString(strconv.Itoa(skip), 10)

	predictedBlock, err := getBlockWrapper(blockNumber)

	if err != nil {
		return nil, err
	}

	var betterBlock *big.Int
	betterBlock, err = findBetter(date, predictedBlock, after, blockTime)
	if err != nil {
		return nil, err
	}

	return returnWrapper(date, betterBlock), nil
}

func findBetter(date *goment.Goment, predictedBlock *newBlock, after bool, blockTime float64) (*big.Int, error) {

	ok, err := isBetterBlock(date, predictedBlock, after)

	if (err == nil) && ok {
		return predictedBlock.number, nil
	}

	if err != nil {
		return nil, err
	}

	blockTimestamp, _ := goment.Unix(int64(predictedBlock.timestamp))
	diff := date.Diff(blockTimestamp, "seconds")

	if blockTime == 0 {
		blockTime = 1
	}

	skip := int(math.Ceil(float64(diff) / blockTime))

	if skip == 0 {
		if diff < 0 {
			skip = -1
		} else {
			skip = 1
		}
	}
	n, _ := new(big.Int).SetString(strconv.Itoa(skip), 10)

	var nxtBlk *big.Int
	nxtBlk, err = getNextBlock(date, predictedBlock.number, n)

	if err != nil {
		return nil, err
	}

	nextPredictedBlock, err := getBlockWrapper(nxtBlk)

	if err != nil {
		return nil, err
	}

	deltaTimestamp, _ := new(big.Float).SetString(strconv.Itoa(int(predictedBlock.timestamp - nextPredictedBlock.timestamp)))

	deltaBlockNum := big.NewInt(0).Sub(predictedBlock.number, nextPredictedBlock.number)

	deltaBlockNumFloat := new(big.Float).SetInt(deltaBlockNum)
	blockTimeCalc := big.NewFloat(0).Quo(deltaTimestamp, deltaBlockNumFloat)
	blockTimeCalc.Abs(blockTimeCalc)

	blockTimeCalcFloat, _ := blockTimeCalc.Float64()

	var betterBlock *big.Int

	betterBlock, err = findBetter(date, nextPredictedBlock, after, blockTimeCalcFloat)
	if err != nil {
		return nil, err
	}
	return betterBlock, nil

}

func isBetterBlock(date *goment.Goment, predictedBlock *newBlock, after bool) (bool, error) {
	blockTime, _ := goment.Unix(int64(predictedBlock.timestamp))

	if after {
		if blockTime.IsBefore(date) {
			return false, nil
		}

		previousBlock, err := getBlockWrapper(big.NewInt(0).Sub(predictedBlock.number, big.NewInt(1)))

		if err != nil {
			return true, err
		}

		prevBlockTime, _ := goment.Unix(int64(previousBlock.timestamp))

		if blockTime.IsSameOrAfter(date) && prevBlockTime.IsBefore(date) {
			return true, nil
		}
	} else {
		if blockTime.IsSameOrAfter(date) {
			return false, nil
		}

		nextBlock, err := getBlockWrapper(predictedBlock.number.Add(predictedBlock.number, big.NewInt(1)))

		if err != nil {
			return true, err
		}

		nextBlockTime, _ := goment.Unix(int64(nextBlock.timestamp))
		if blockTime.IsBefore(date) && nextBlockTime.IsSameOrAfter(date) {
			return true, nil
		}
	}
	return false, nil
}

func getNextBlock(date *goment.Goment, currentBlockNumber *big.Int, skip *big.Int) (*big.Int, error) {
	var nextBlockNumber = big.NewInt(0).Add(currentBlockNumber, skip)
	if blockSlice, ok := checkedBlocks[uint64(date.ToUnix())]; ok {
		if contains(&blockSlice, nextBlockNumber) {
			if skip.Cmp(big.NewInt(0)) == -1 {
				skip.Sub(skip, big.NewInt(1))
			} else {
				skip.Add(skip, big.NewInt(1))
			}

			nextBlk, err := getNextBlock(date, currentBlockNumber, skip)
			if err != nil {
				return nil, err
			} else {
				return nextBlk, nil
			}
		}
	}

	mutex.Lock()
	checkedBlocks[uint64(date.ToUnix())] = append(checkedBlocks[uint64(date.ToUnix())], nextBlockNumber)
	mutex.Unlock()

	if nextBlockNumber.Cmp(big.NewInt(1)) == -1 {
		nextBlockNumber = big.NewInt(1)
	}

	return nextBlockNumber, nil
}

func returnWrapper(date *goment.Goment, blockNumber *big.Int) *newBlockWrapper {
	return &newBlockWrapper{
		Date:        date,
		BlockNumber: blockNumber,
		Timestamp:   savedBlocks[blockNumber.String()].timestamp,
	}
}

func getBlockWrapper(blockNumber *big.Int) (*newBlock, error) {
	client, err := ethclient.Dial(nodeUrl)

	if err != nil {
		log.Fatal("Something went wrong")
		return nil, err
	}

	ctx := context.Background()

	blockNumberStr := blockNumber.String()
	if block, ok := savedBlocks[blockNumberStr]; ok {
		return block, nil
	}

	if block, err := client.BlockByNumber(ctx, blockNumber); err == nil {
		mutex.Lock()
		savedBlocks[blockNumberStr] = &newBlock{
			timestamp: block.Time(),
			number:    block.Number(),
		}
		mutex.Unlock()
	}

	requests++
	return savedBlocks[blockNumberStr], nil
}
