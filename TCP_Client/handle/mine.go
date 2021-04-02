package handle

import (
	"encoding/binary"
	"fmt"
	"hash"
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"golang.org/x/crypto/sha3"
)

const (
	datasetInitBytes   = 1 << 30 // Bytes in dataset at genesis
	datasetGrowthBytes = 1 << 23 // Dataset growth per epoch
	cacheInitBytes     = 1 << 24 // Bytes in cache at genesis
	cacheGrowthBytes   = 1 << 17 // Cache growth per epoch

	mixBytes       = 128 // Width of mix
	hashBytes      = 64  // Hash length in bytes
	hashWords      = 16  // Number of 32 bit ints in a hash
	datasetParents = 256 // Number of parents of each dataset element
	cacheRounds    = 3   // Number of rounds in cache production
	loopAccesses   = 64  // Number of accesses in hashimoto loop
)

func (w *Worker) Miner(abort chan bool) {
	refreshIntv, err := time.ParseDuration("20s")
	if err != nil {
		fmt.Println("the reshreshIntv err:", err)
		return
	}
	reshreshTimer := time.NewTimer(refreshIntv)
	ethash := New(Config{
		CacheDir:       "ethash",
		PowMode:        ModeNormal,
		CachesOnDisk:   3,
		CachesLockMmap: false,
	}, nil, false)

	for {
		select {
		case <-reshreshTimer.C:
			w.NewTcpGetWork()
		default:
			w.mine(abort, ethash)
		}
	}
}

func (w Worker) mine(abort chan bool, ethash *Ethash) {

	var (
		hash   = (common.HexToHash(w.Work.Hash)).Bytes() //矿池发过来的
		height = w.Work.Seed                             //随机数
		cache  = ethash.cache(height)                    //这里需要区块高度
	)
	var nonce uint64
	nonce = 0
search:
	for {
		select {
		case <-abort:
			break search
		default:
			// Compute the PoW value of this nonce
			digest, result := hashimotoLight(cacheSize(height), cache.cache, hash, nonce)
			if new(big.Int).SetBytes(result).Cmp(w.Work.Target) <= 0 {
				fmt.Println("send share success")
				i := w.Work.Hash

				fmt.Println("cacheSize:", cacheSize(height), "nonce:", nonce, "height", uint64(height))
				fmt.Println("hash:", hash)
				fmt.Println("result:", result)
				//fmt.Println("cache:", cache)
				w.NewTcpSubmitWork(digest, nonce, i)
			}
			nonce++
		}
	}
}

func hashimotoLight(size uint64, cache []uint32, hash []byte, nonce uint64) ([]byte, []byte) {
	keccak512 := makeHasher(sha3.NewLegacyKeccak512())

	lookup := func(index uint32) []uint32 {
		rawData := generateDatasetItem(cache, index, keccak512)

		data := make([]uint32, len(rawData)/4)
		for i := 0; i < len(data); i++ {
			data[i] = binary.LittleEndian.Uint32(rawData[i*4:])
		}
		return data
	}
	return hashimoto(hash, nonce, size, lookup)
}

type hasher func(dest []byte, data []byte)

func makeHasher(h hash.Hash) hasher {
	// sha3.state supports Read to get the sum, use it to avoid the overhead of Sum.
	// Read alters the state but we reset the hash before every operation.
	type readerHash interface {
		hash.Hash
		Read([]byte) (int, error)
	}
	rh, ok := h.(readerHash)
	if !ok {
		panic("can't find Read method on hash")
	}
	outputLen := rh.Size()
	return func(dest []byte, data []byte) {
		rh.Reset()
		rh.Write(data)
		rh.Read(dest[:outputLen])
	}
}

func generateDatasetItem(cache []uint32, index uint32, keccak512 hasher) []byte {
	// Calculate the number of theoretical rows (we use one buffer nonetheless)
	rows := uint32(len(cache) / hashWords)

	// Initialize the mix
	mix := make([]byte, hashBytes)

	binary.LittleEndian.PutUint32(mix, cache[(index%rows)*hashWords]^index)
	for i := 1; i < hashWords; i++ {
		binary.LittleEndian.PutUint32(mix[i*4:], cache[(index%rows)*hashWords+uint32(i)])
	}
	keccak512(mix, mix)

	// Convert the mix to uint32s to avoid constant bit shifting
	intMix := make([]uint32, hashWords)
	for i := 0; i < len(intMix); i++ {
		intMix[i] = binary.LittleEndian.Uint32(mix[i*4:])
	}
	// fnv it with a lot of random cache nodes based on index
	for i := uint32(0); i < datasetParents; i++ {
		parent := fnv(index^i, intMix[i%16]) % rows
		fnvHash(intMix, cache[parent*hashWords:])
	}
	// Flatten the uint32 mix into a binary one and return
	for i, val := range intMix {
		binary.LittleEndian.PutUint32(mix[i*4:], val)
	}
	keccak512(mix, mix)
	return mix
}

func hashimoto(hash []byte, nonce uint64, size uint64, lookup func(index uint32) []uint32) ([]byte, []byte) {
	// Calculate the number of theoretical rows (we use one buffer nonetheless)
	rows := uint32(size / mixBytes)

	// Combine header+nonce into a 64 byte seed
	seed := make([]byte, 40)
	copy(seed, hash)
	binary.LittleEndian.PutUint64(seed[32:], nonce)

	seed = crypto.Keccak512(seed)
	seedHead := binary.LittleEndian.Uint32(seed)

	// Start the mix with replicated seed
	mix := make([]uint32, mixBytes/4)
	for i := 0; i < len(mix); i++ {
		mix[i] = binary.LittleEndian.Uint32(seed[i%16*4:])
	}
	// Mix in random dataset nodes
	temp := make([]uint32, len(mix))

	for i := 0; i < loopAccesses; i++ {
		parent := fnv(uint32(i)^seedHead, mix[i%len(mix)]) % rows
		for j := uint32(0); j < mixBytes/hashBytes; j++ {
			copy(temp[j*hashWords:], lookup(2*parent+j))
		}
		fnvHash(mix, temp)
	}
	// Compress mix
	for i := 0; i < len(mix); i += 4 {
		mix[i/4] = fnv(fnv(fnv(mix[i], mix[i+1]), mix[i+2]), mix[i+3])
	}
	mix = mix[:len(mix)/4]

	digest := make([]byte, common.HashLength)
	for i, val := range mix {
		binary.LittleEndian.PutUint32(digest[i*4:], val)
	}
	return digest, crypto.Keccak256(append(seed, digest...))
}

func fnv(a, b uint32) uint32 {
	return a*0x01000193 ^ b
}

func fnvHash(mix []uint32, data []uint32) {
	for i := 0; i < len(mix); i++ {
		mix[i] = mix[i]*0x01000193 ^ data[i]
	}
}
