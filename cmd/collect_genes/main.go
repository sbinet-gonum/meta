package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"strings"

	"github.com/alecthomas/kingpin"
)

func main() {
	var sampleFile string
	var appendix string
	var outfile string
	app := kingpin.New("collect_genes", "Calculate correlation across multiple samples")
	app.Version("v0.1")
	sampleFileArg := app.Arg("sample-file", "sample file").Required().String()
	outFileArg := app.Arg("out-file", "output file").Required().String()
	kingpin.MustParse(app.Parse(os.Args[1:]))
	sampleFile = *sampleFileArg
	outfile = *outFileArg

	collectorMap := make(map[string]*Collector)

	samples := readSamples(sampleFile)
	for _, sample := range samples {
		corrFile := sample + appendix
		corrChan := readCorrResults(corrFile)
		for corrResults := range corrChan {
			geneID := corrResults.GeneID
			_, found := collectorMap[geneID]
			if !found {
				collectorMap[geneID] = NewCollector()
			}
			collectorMap[geneID].Add(corrResults)
		}
	}

	w, err := os.Create(outfile)
	if err != nil {
		log.Panic(err)
	}
	defer w.Close()

	w.WriteString("l,m,v,n,t,g\n")
	for geneID, collector := range collectorMap {
		results := collector.Results()
		for _, res := range results {
			w.WriteString(fmt.Sprintf("%d,%g,%g,%d,%s,%s\n",
				res.Lag, res.Value, res.Variance, res.Count, res.Type, geneID))
		}
	}
}

func readSamples(filename string) []string {
	f, err := os.Open(filename)
	if err != nil {
		log.Panic(err)
	}
	defer f.Close()

	rd := bufio.NewReader(f)
	var results []string
	for {
		line, err := rd.ReadString('\n')
		if err != nil {
			if err != io.EOF {
				log.Panic(err)
			}
			break
		}
		results = append(results, strings.TrimSpace(line))
	}
	return results
}

func readCorrResults(filename string) chan CorrResults {
	c := make(chan CorrResults)
	go func() {
		f, err := os.Open(filename)
		if err != nil {
			log.Panic(err)
		}
		defer f.Close()
		decoder := json.NewDecoder(f)
		for {
			var rec CorrResults
			if err := decoder.Decode(&rec); err != nil {
				if err != io.EOF {
					log.Panic(err)
				}
				break
			}
			c <- rec
		}
	}()
	return c
}
