package modules

import (
	"go.sia.tech/siad/types"
)

const (
	// PoolDir names the directory that contains the pool persistence.
	PoolDir = "pool"
)

type (
	// PoolInternalSettings contains a list of settings that can be changed.
	PoolInternalSettings struct {
		PoolNetworkPort int              `json:"networkport"`
		PoolName        string           `json:"name"`
		PoolID          uint64           `json:"poolid"`
		PoolDBName      string           `json:"dbname"`
		PoolWallet      types.UnlockHash `json:"poolwallet"`
		PoolWebUrl      string           `json:"poolweburl"`
	}
	// A Pool accepts incoming target solutions, tracks the share (an attempted solution),
	// checks to see if we have a new block, and if so, pays all the share submitters,
	// proportionally based on their share of the solution (minus a percentage to the
	// pool operator )
	Pool interface {
		// InternalSettings returns the pool's internal settings, including
		// potentially private or sensitive information.
		InternalSettings() PoolInternalSettings

		// SetInternalSettings sets the parameters of the pool.
		SetInternalSettings(PoolInternalSettings) error

		// Close closes the Pool.
		Close() error
	}
)
