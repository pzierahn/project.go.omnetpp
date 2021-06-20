package provider

import (
	"github.com/pzierahn/project.go.omnetpp/simple"
	"log"
)

func (prov *provider) distributeSlots() {

	prov.mu.RLock()

	log.Printf("distributeSlots: slots=%d requesters=%d", prov.freeSlots, len(prov.requests))

	for cId, req := range prov.requests {

		if prov.freeSlots == 0 {
			break
		}

		assign := simple.MathMinUint32(prov.freeSlots, req)
		prov.allocate[cId] <- assign
	}

	prov.mu.RUnlock()

	return
}
