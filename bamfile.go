package meta

// BAM file operations.

import (
	"code.google.com/p/biogo.bam/bam"
	"code.google.com/p/biogo.bam/sam"
	"io"
	"log"
	"os"
	"sort"
)

// A container for a group of SAM records.
// It implements sort.Interface for sorting and searching.
type SamRecords []*sam.Record

func (sr SamRecords) Len() int      { return len(sr) }
func (sr SamRecords) Swap(i, j int) { sr[i], sr[j] = sr[j], sr[i] }

func (sr SamRecords) Search(pos int) (index int) {
	index = sort.Search(len(sr), func(i int) bool { return sr[i].Pos >= pos })
	return
}

// A struct for sorting SAM records by left cordinate.
type ByLeftCoordinate struct{ SamRecords }

func (b ByLeftCoordinate) Less(i, j int) bool {
	return b.SamRecords[i].Pos < b.SamRecords[j].Pos
}

// Read BAM file and return its header and records.
// NOT explicitly sorted.
func ReadBamFile(fileName string) (header *sam.Header, records []*sam.Record) {
	// Open bam file.
	f, err := os.Open(fileName)
	if err != nil {
		log.Panic(err)
	}
	defer f.Close()

	// Create bam reader,
	// and read the reference genomes.
	rd := 0 // ignore this now.
	reader, err := bam.NewReader(f, rd)
	if err != nil {
		log.Panic(err)
	}
	header = reader.Header()

	for {
		r, err := reader.Read()
		if err != nil {
			if err != io.EOF {
				log.Panic(err)
			}
			break
		}
		records = append(records, r)
	}

	return
}

// Separate SAM records for different reference genomes.
// Return a map of genome reference name to records.
func SeparateSamRecords(refs []*sam.Reference, records SamRecords) map[string]SamRecords {
	m := make(map[string]SamRecords)
	for _, ref := range refs {
		m[ref.Name()] = FindSorted(ref.Name(), records)
	}
	return m
}

// Find and sort SamRecords of a reference.
func FindSorted(ref string, records SamRecords) SamRecords {
	founds := SamRecords{}
	for _, r := range records {
		if r.Ref.Name() == ref {
			founds = append(founds, r)
		}
	}
	Info.Println(ref)

	sort.Sort(ByLeftCoordinate{founds})

	return founds
}
