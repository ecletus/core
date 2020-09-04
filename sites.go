package core

import (
	"errors"
)

var (
	StopSiteIteration = errors.New("stop site iteration")
	StopDBIteration   = errors.New("stop db iteration")
)
