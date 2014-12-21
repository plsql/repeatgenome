package repeatgenome

import (
    "sort"
    "fmt"
    "sync"
)

type matchSpan struct {
    start uint64
    end uint64
    repeat *Repeat
}

type matchSpans []matchSpan

func (matchSpans matchSpans) Len() int {
    return len(matchSpans)
}

func (matchSpans matchSpans) Swap(i, j int) {
    matchSpans[i], matchSpans[j] = matchSpans[j], matchSpans[i]
}

func (matchSpans matchSpans) Less(i, j int) bool {
    return matchSpans[i].start < matchSpans[j].start
}


func (rg *RepeatGenome) GetMatchSpans() map[string]matchSpans {
    // maps a seq's name to its matchSpans
    spanMap := make(map[string]matchSpans)

    for _, match := range rg.Matches {
        /*
        if _, exists := spanMap[match.SeqName]; !exists {
            spanMap[match.SeqName] = make
        */
        seqName := match.SeqName
        rep := rg.RepeatMap[match.RepeatName]
        matchSpan := matchSpan{match.SeqStart, match.SeqEnd, rep}
        spanMap[seqName] = append(spanMap[seqName], matchSpan)
    }

    for _, matchSpans := range spanMap {
        sort.Sort(matchSpans)
    }

    return spanMap
}


type KRespPair struct {
    KmerInt KmerInt
    Repeat *Repeat
}


type MRespPair struct {
    MinInt MinInt
    Repeat *Repeat
}


func (rg *RepeatGenome) SplitChromsK() (chan KRespPair, chan KRespPair) {
    spanMap := rg.GetMatchSpans()

    repChan, nonrepChan := make(chan KRespPair, 5000), make(chan KRespPair, 5000)
    var wg = new(sync.WaitGroup)

    for chromName, matchSpans := range spanMap {
        // used to determine where in seq each iteration starts
        seq := rg.chroms[chromName][chromName]
        var start uint64 = 0
        for _, matchSpan := range matchSpans {
            mStart, mEnd, rep := matchSpan.start, matchSpan.end, matchSpan.repeat
            // necessary because of overlapping matches
            if start < mStart {
                wg.Add(1)
                go rg.sendKmers(seq[start:mStart], rep, nonrepChan, wg)
            }
            wg.Add(1)
            go rg.sendKmers(seq[mStart:mEnd], rep, repChan, wg)
            start = mEnd
        }
        wg.Add(1)
        go rg.sendKmers(seq[start:], nil, nonrepChan, wg)
    }

    go func() {
        wg.Wait()
        close(repChan)
        close(nonrepChan)
    }()

    return repChan, nonrepChan
}


func (rg *RepeatGenome) sendKmers(seq TextSeq, rep *Repeat, c chan KRespPair, wg *sync.WaitGroup) {
    defer wg.Done()
    var numKmers = len(seq) - int(k) + 1
KmerLoop:
    for i := 0; i < numKmers; i++ {
        for j := int(k) + i - 1; j >= i; j-- {
            if seq[j] == byte('n') {
                i += j - i
                continue KmerLoop
            }
        }
        kmerInt := seq[i:i+int(k)].kmerInt()
        c <- KRespPair{kmerInt, rep}
        c <- KRespPair{kmerInt.revComp(), rep}
    }
}


func (rg *RepeatGenome) SplitChromsM() (chan MRespPair, chan MRespPair) {
    spanMap := rg.GetMatchSpans()

    repChan, nonrepChan := make(chan MRespPair, 5000), make(chan MRespPair, 5000)
    var wg = new(sync.WaitGroup)

    for chromName, matchSpans := range spanMap {
        // used to determine where in seq each iteration starts
        seq := rg.chroms[chromName][chromName]
        var start uint64 = 0
        for _, matchSpan := range matchSpans {
            mStart, mEnd, rep := matchSpan.start, matchSpan.end, matchSpan.repeat
            // necessary because of overlapping matches
            if start < mStart {
                wg.Add(1)
                go rg.sendMins(seq[start:mStart], rep, nonrepChan, wg)
            }
            wg.Add(1)
            go rg.sendMins(seq[mStart:mEnd], rep, repChan, wg)
            start = mEnd
        }
        wg.Add(1)
        go rg.sendMins(seq[start:], nil, nonrepChan, wg)
    }

    go func() {
        wg.Wait()
        close(repChan)
        close(nonrepChan)
    }()

    return repChan, nonrepChan
}


func (rg *RepeatGenome) sendMins(seq TextSeq, rep *Repeat, c chan MRespPair, wg *sync.WaitGroup) {
    defer wg.Done()
    var numMins = len(seq) - int(m) + 1
MinLoop:
    for i := 0; i < numMins; i++ {
        for j := int(m) + i - 1; j >= i; j-- {
            if seq[j] == byte('n') {
                i += j - i
                continue MinLoop
            }
        }
        minInt := seq[i:i+int(m)].minInt()
        c <- MRespPair{minInt, rep}
        c <- MRespPair{minInt.revComp(), rep}
    }
}


func (rg *RepeatGenome) GetKmerMap() (int, int, map[KmerInt]*Repeat) {
    repChan, nonrepChan := rg.SplitChromsK()
    nonreps, reps := 0, 0
    repMap := make(map[KmerInt]*Repeat, 300000000)
    wg := new(sync.WaitGroup)
    wg.Add(2)
    go func() {
        for repPair := range repChan {
            reps++
            kmerInt, repeat := repPair.KmerInt, repPair.Repeat
            lastRepeat, exists := repMap[kmerInt]
            if !exists {
                repMap[kmerInt] = repeat
            } else if repeat != lastRepeat {
                repMap[kmerInt] = nil
            }
        }
        wg.Done()
    }()
    go func() {
        for nonrepPair := range nonrepChan {
            kmerInt := nonrepPair.KmerInt
            nonreps++
            repMap[kmerInt] = nil
        }
        wg.Done()
    }()
    wg.Wait()

    return reps, nonreps, repMap
}


func (rg *RepeatGenome) GetMinMap() (int, int, map[MinInt]*Repeat) {
    repChan, nonrepChan := rg.SplitChromsM()
    nonreps, reps := 0, 0
    repMap := make(map[MinInt]*Repeat, 300000000)
    wg := new(sync.WaitGroup)
    wg.Add(2)
    go func() {
        for repPair := range repChan {
            reps++
            minInt, repeat := repPair.MinInt, repPair.Repeat
            lastRepeat, exists := repMap[minInt]
            if !exists {
                repMap[minInt] = repeat
            } else if repeat != lastRepeat {
                repMap[minInt] = nil
            }
        }
        wg.Done()
    }()
    go func() {
        for nonrepPair := range nonrepChan {
            minInt := nonrepPair.MinInt
            nonreps++
            repMap[minInt] = nil
        }
        wg.Done()
    }()
    wg.Wait()

    return reps, nonreps, repMap
}


func (rg *RepeatGenome) KmerClassifyRead(read TextSeq, kmerMap map[KmerInt]*Repeat, wg *sync.WaitGroup, c chan *Repeat) {
    defer wg.Done()
    // the repeat we assign this read
    // nil if we don't find one
    var repeat *Repeat
    var numKmers = len(read) - int(m) + 1
KmerLoop:
    for i := 0; i < numKmers; i++ {
        for j := int(m) + i - 1; j >= i; j-- {
            if read[j] == byte('n') {
                i += j - i
                continue KmerLoop
            }
        }
        kmerInt := read[i:i+int(m)].kmerInt()
        /*
        if kmerRepeat, exists := kmerMap[kmerInt]; !exists {
            c <- kmerRepeat
        }
        */
        
        if kmerRepeat, exists := kmerMap[kmerInt]; exists {
            if repeat == nil {
                repeat = kmerRepeat
            // kmerRepeat is assumed to not be nil
            // nils in kmerMap must therefore be deleted
            } else if repeat != kmerRepeat {
                repeat = nil
                break
            }
        }
    }
    c <- repeat
    //c <- nil
}


/*
func (rg *RepeatGenome) MinClassifyRead(read TextSeq, minMap map[MinInt]*Repeat, wg *sync.WaitGroup, c chan *Repeat) {
    defer wg.Done()
    // the repeat we assign this read
    // nil if we don't find one
    var repeat *Repeat
    var numMins = len(read) - int(m) + 1
MinLoop:
    for i := 0; i < numMins; i++ {
        for j := int(m) + i - 1; j >= i; j-- {
            if read[j] == byte('n') {
                i += j - i
                continue MinLoop
            }
        }
        minInt := read[i:i+int(m)].minInt()
        if minRepeat, exists := minMap[minInt]; exists {
            if repeat == nil {
                repeat = minRepeat
            // minRepeat is assumed to not be nil
            // nils in minMap must therefore be deleted
            } else if repeat != minRepeat {
                repeat = nil
                break
            }
        }
    }
    c <- repeat
    //c <- nil
}
*/

type ReadSAMRepeat struct {
    ReadSAM ReadSAM
    Repeat *Repeat
}


func (rg *RepeatGenome) MinClassifyRead(readSAM ReadSAM, minMap map[MinInt]*Repeat, wg *sync.WaitGroup, c chan ReadSAMRepeat) {
    defer wg.Done()
    read := readSAM.TextSeq
    // the repeat we assign this read
    // nil if we don't find one
    var repeat *Repeat
    var numMins = len(read) - int(m) + 1
MinLoop:
    for i := 0; i < numMins; i++ {
        for j := int(m) + i - 1; j >= i; j-- {
            if read[j] == byte('n') {
                i += j - i
                continue MinLoop
            }
        }
        minInt := read[i:i+int(m)].minInt()
        if minRepeat, exists := minMap[minInt]; exists {
            if repeat == nil {
                repeat = minRepeat
            // minRepeat is assumed to not be nil
            // nils in minMap must therefore be deleted
            } else if repeat != minRepeat {
                repeat = nil
                break
            }
        } else {
            minInt.print()
        }
    }
    c <- ReadSAMRepeat{readSAM, repeat}
}


func (rg *RepeatGenome) MinClassifyReadVerb(readSAM ReadSAM, minMap map[MinInt]*Repeat, wg *sync.WaitGroup, c chan ReadSAMRepeat) {
    defer wg.Done()
    read := readSAM.TextSeq
    fmt.Println(read)
    fmt.Println()
    // the repeat we assign this read
    // nil if we don't find one
    var repeat *Repeat
    var numMins = len(read) - int(m) + 1
MinLoop:
    for i := 0; i < numMins; i++ {
        for j := int(m) + i - 1; j >= i; j-- {
            if read[j] == byte('n') {
                i += j - i
                continue MinLoop
            }
        }
        minInt := read[i:i+int(m)].minInt()
        if minRepeat, exists := minMap[minInt]; exists {
            if repeat == nil {
                repeat = minRepeat
                fmt.Println("recognized:")
                fmt.Print("\t"); minInt.print(); fmt.Println()
                fmt.Printf("\t\t%s\n", repeat.Name)
            // minRepeat is assumed to not be nil
            // nils in minMap must therefore be deleted
            } else if repeat != minRepeat {
                fmt.Println("conflict:")
                fmt.Print("\t"); minInt.print(); fmt.Println()
                fmt.Printf("\t\t%s\n", repeat.Name)
                repeat = nil
                fmt.Println("BREAK")
                break
            }
        } else {
            fmt.Print("\tunrecognized: ")
            minInt.print()
            fmt.Println()
        }
    }
    c <- ReadSAMRepeat{readSAM, repeat}
    //c <- nil
}
