package pool

import (
	"go.sia.tech/siad/modules"
)

const (
	// Names of the various persistent files in the pool.
	dbFilename   = modules.PoolDir + ".db"
	logFile      = modules.PoolDir + ".log"
	yiilogFile   = "yii.log"
	settingsFile = modules.PoolDir + ".json"
	// MajorVersion is the significant version of the pool module
	MajorVersion = 0
	// MinorVersion is the minor version of the pool module
	MinorVersion = 3
	// SiaCoinID is the coin id used by yiimp to associate various records
	// with Siacoin
	SiaCoinID = 1316
	// SiaCoinSymbol is the coin symbol used by yiimp to associate various records
	// with Siacoin
	SiaCoinSymbol = "SC"
	// SiaCoinAlgo is the algo used by yiimp to associate various records
	// with blake2b mining
	SiaCoinAlgo = "blake2b"
)
