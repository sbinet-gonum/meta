package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"github.com/mingzhi/biogo/feat/gff"
	"github.com/mingzhi/biogo/seq"
	"github.com/mingzhi/gomath/stat/correlation"
	"github.com/mingzhi/gomath/stat/desc/meanvar"
	"github.com/mingzhi/ncbiftp/genomes/profiling"
	"github.com/mingzhi/ncbiftp/taxonomy"
	"log"
	"math"
	"os"
)

func main() {
	var piFile string
	var genomeFile string
	var gffFile string
	var outFile string
	var maxl int
	var pos int
	var codonTableID string
	// Parse arguments.
	flag.IntVar(&maxl, "maxl", 100, "max length of correlations")
	flag.IntVar(&pos, "pos", 4, "position")
	flag.StringVar(&codonTableID, "codon", "11", "codon table ID")
	flag.Parse()
	if flag.NArg() < 4 {
		log.Fatalln("Usage: go run calc_cr.go <pi file> <genome file> <gff file> <out file>")
	}
	piFile = flag.Arg(0)
	genomeFile = flag.Arg(1)
	gffFile = flag.Arg(2)
	outFile = flag.Arg(3)

	// Obtain codon table for identifying four-fold degenerate sites.
	codonTable := taxonomy.GeneticCodes()[codonTableID]
	// Profiling genome using reference sequence and protein feature data.
	genome := readGenome(genomeFile)
	gffs := readGff(gffFile)
	profile := profiling.ProfileGenome(genome, gffs, codonTable)

	// Read pi.
	piArr := readPi(piFile)
	numChunck := 1000
	lenChunck := len(piArr) / numChunck
	piChuncks := [][]Pi{}
	for i := 0; i < numChunck; i++ {
		pis := piArr[i*lenChunck : (i+1)*lenChunck]
		piChuncks = append(piChuncks, pis)
	}
	posType := convertPosType(pos)
	/*
		genePiMap := make(map[string][]Pi)
		for _, pi := range piArr {
			pos := pi.Position
			geneName := profile[pos].Gene
			genePiMap[geneName] = append(genePiMap[geneName], pi)
		}
	*/
	covMVs := make([]*meanvar.MeanVar, maxl)
	for i := range covMVs {
		covMVs[i] = meanvar.New()
	}
	for _, pis := range piChuncks {
		covs := CalcCr(pis, profile, posType, maxl)
		for i := range covs {
			n := covs[i].GetN()
			v := covs[i].GetResult()
			if n > 10 && !math.IsNaN(v) {
				covMVs[i].Increment(v)
			}
		}
	}

	w, err := os.Create(outFile)
	if err != nil {
		log.Fatalln(err)
	}
	defer w.Close()

	for i := 0; i < len(covMVs); i++ {
		c := covMVs[i]
		w.WriteString(fmt.Sprintf("%d\t%g\t%g\t%d\n", i, c.Mean.GetResult(), c.Var.GetResult(), c.Mean.GetN()))
	}
}

func readGenome(filename string) []byte {
	f, err := os.Open(filename)
	if err != nil {
		panic(err)
	}
	defer f.Close()

	rd := seq.NewFastaReader(f)
	ss, err := rd.ReadAll()
	if err != nil {
		panic(err)
	}

	return ss[0].Seq
}

func readGff(filename string) []*gff.Record {
	f, err := os.Open(filename)
	if err != nil {
		panic(err)
	}
	defer f.Close()

	rd := gff.NewReader(f)
	ss, err := rd.ReadAll()
	if err != nil {
		panic(err)
	}

	records := []*gff.Record{}
	for _, s := range ss {
		if s.Feature == "CDS" {
			records = append(records, s)
		}
	}

	return records
}

func readPi(filename string) []Pi {
	piArr := []Pi{}
	f, err := os.Open(filename)
	if err != nil {
		log.Fatalln(err)
	}
	defer f.Close()
	decoder := json.NewDecoder(f)
	var pi Pi
	for decoder.More() {
		err := decoder.Decode(&pi)
		if err != nil {
			log.Fatal(err)
		}
		piArr = append(piArr, pi)
	}
	return piArr
}

func convertPosType(pos int) byte {
	var p byte
	switch pos {
	case 0:
		p = profiling.NonCoding
	case 1:
		p = profiling.FirstPos
		break
	case 2:
		p = profiling.SecondPos
		break
	case 3:
		p = profiling.ThirdPos
		break
	case 4:
		p = profiling.FourFold
		break
	default:
		p = profiling.Coding
	}

	return p
}

type Pi struct {
	Genome   string
	Position int
	Pi       float64
}

type Covariance interface {
	GetN() int
	GetResult() float64
	MeanX() float64
	MeanY() float64
	Increment(x, y float64)
}

// Calculate covariance of rates.
func CalcCr(pis []Pi, profile []profiling.Pos, posType byte, maxl int) (covs []Covariance) {
	corrs := make([]Covariance, maxl)
	for i := 0; i < maxl; i++ {
		corrs[i] = correlation.NewBivariateCovariance(false)
	}

	for i := 0; i < len(pis); i++ {
		pos1 := profile[pis[i].Position-1]
		if checkPosType(posType, pos1.Type) {
			for j := i; j < len(pis); j++ {
				pos2 := profile[pis[j].Position-1]

				distance := pis[j].Position - pis[i].Position
				if distance >= maxl {
					break
				}

				if checkPosType(posType, pos2.Type) {
					corrs[distance].Increment(pis[i].Pi, pis[j].Pi)
				}
			}
		}

	}

	covs = corrs
	return
}

func checkPosType(t, t1 byte) bool {
	isFirstPos := t1 == profiling.FirstPos
	isSecondPos := t1 == profiling.SecondPos
	isThirdPos := t1 == profiling.ThirdPos
	isFourFold := t1 == profiling.FourFold

	if t == profiling.Coding {
		if isFirstPos || isSecondPos || isThirdPos || isFourFold {
			return true
		}
		return false
	}

	if t == profiling.ThirdPos {
		if isThirdPos || isFourFold {
			return true
		}
		return false
	}

	return t == t1
}
