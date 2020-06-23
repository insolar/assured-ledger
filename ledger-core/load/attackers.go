package load

import (
	"github.com/skudasov/loadgen"
	"log"
)

func AttackerFromName(name string) loadgen.Attack {
	switch name {
	case "create_contract_test":
		return loadgen.WithCSVMonitor(new(CreateContractTestAttack))
	case "get_contract_test":
		return loadgen.WithCSVMonitor(new(GetContractTestAttack))
	case "set_contract_test":
		return loadgen.WithCSVMonitor(new(SetContractTestAttack))
	default:
		log.Fatalf("unknown attacker type: %s", name)
		return nil
	}
}
