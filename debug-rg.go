package repeatgenome

import (
    "fmt"
    "unsafe"
)

func (rg *RepeatGenome) printSample(numMins, numKmers int) {
    for i := 0; i < len(rg.SortedMins); i += len(rg.SortedMins) / numMins {
        rg.SortedMins[i].print(); fmt.Println()
        for _, fullKmer := range rg.getMinsFullKmers(uint64(i))[:numKmers] {
            fmt.Print("\t")
            (*(*KmerInt)(unsafe.Pointer(&fullKmer))).print()
            lca_ID := *(*uint16)(unsafe.Pointer(&fullKmer[12]))
            fmt.Printf("\t%s\n", rg.ClassTree.NodesByID[lca_ID].Name)
        }
    }
}
