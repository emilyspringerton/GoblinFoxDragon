// Source [9] - RandomX Miner Implementation
package pow

import "sync"

// RandomXVM simulates the ASIC-resistant virtual machine
type RandomXVM struct {
	scratchpad []byte
	registers  [18]uint64
	program    []byte
}

type Miner struct {
	workerCount int
	minerAddr   string
	difficulty  uint64
	stopChan    chan struct{}
	wg          sync.WaitGroup
	mu          sync.Mutex
}
