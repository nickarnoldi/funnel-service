package flags

import (
	"flag"
	"fmt"

	"github.com/nickarnoldi/funnel-service/internal/enum"
)

func Network(flagSet *flag.FlagSet, flagName ...string) *FlagNetwork {
	var name string = "network"
	if len(flagName) > 0 {
		name = flagName[0]
	}

	var network FlagNetwork
	flagSet.Var(&network, name, "Network to sync data (primary, secondary)")
	return &network
}

func (fs *FlagSet) Network(flagName ...string) *FlagNetwork {
	return Network(fs.FlagSet, flagName...)
}

type FlagNetwork struct {
	Enum  enum.NetworkEnum
	Valid bool
}

func (network FlagNetwork) String() string {
	return enum.Network[network.Enum]
}

func (network *FlagNetwork) Set(value string) error {
	networkID, found := enum.Index(enum.Network, value)
	if !found {
		network.Valid = false
		return fmt.Errorf("invalid network: %s", value)
	}
	network.Enum = enum.NetworkEnum(networkID)
	network.Valid = true
	return nil
}
