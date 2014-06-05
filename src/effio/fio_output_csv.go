package effio

// encoding/csv doesn't strip whitespace and does a fair bit of
// work to handle strings & quoting which are totally unnecessary
// overhead for these files so skip it

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"os"
	"strconv"
	"strings"
)

// the input is ints but plotinum uses float64 so might as well
// start there and avoid the type conversions later
type LatRec struct {
	time float64 // time offset from beginning of fio run
	perf float64 // latency value
}

type LatRecs []LatRec

/*
time, perf, ??, block
3, 205274611861, 0, 4096
16, 205274624691, 0, 4096
*/
func LoadCSV(filename string) LatRecs {
	fmt.Printf("Parsing file: '%s' ... ", filename)

	fd, err := os.Open(filename)
	if err != nil {
		fmt.Printf(" Failed.\nCould not open file '%s' for read: %s\n", filename, err)
		return LatRecs{}
	}
	defer fd.Close()

	records := make(LatRecs, 0)
	var time, perf float64
	bfd := bufio.NewReader(fd)
	var lno int = 0
	for {
		line, _, err := bfd.ReadLine()
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Fatalf("\nRead from file '%s' failed: %s", filename, err)
		}
		lno++

		if lno%10000 == 0 {
			fmt.Printf(".")
		}

		// fio always uses ", " instead of "," as far as I can tell
		r := strings.SplitN(string(line), ", ", 4)
		// probably an impartial record at the end of the file
		if len(r) < 4 || r[0] == "" || r[1] == "" {
			continue
		}

		time, err = strconv.ParseFloat(r[0], 64)
		if err != nil {
			log.Fatalf("\nParsing time integer failed in file '%s' at line %d: %s", filename, lno, err)
		}
		perf, err = strconv.ParseFloat(r[1], 64)
		if err != nil {
			log.Fatalf("\nParsing perf integer failed in file '%s' at line %d: %s", filename, lno, err)
		}
		// r[2:3] are unused, 2 is reserved, 3 is block size

		// check for broken data
		if perf > 10e6 {
			log.Fatalf("Failed.\nInvalid perf data in file '%s', too big! raw: '%s', float: '%f'\n", filename, r[1], perf)
		}

		lr := LatRec{time, perf}
		records = append(records, lr)
	}
	fmt.Println(" Done.")

	return records
}

// implement some plotinum interfaces
func (lrs LatRecs) Len() int {
	return len(lrs)
}

func (lrs LatRecs) XY(i int) (float64, float64) {
	return lrs[i].time, lrs[i].perf
}

func (lrs LatRecs) Value(i int) float64 {
	return lrs[i].perf
}

func (lrs LatRecs) Values(i int) (vals []float64) {
	for _, l := range lrs {
		vals = append(vals, l.perf)
	}
	return
}

// reduces the number of data points to sz by taking the mean across buckets
func (lrs LatRecs) Histogram(sz int) (out LatRecs) {
	if sz > len(lrs) {
		log.Fatalf("Error: Histogram(%d) is smaller than the dataset of length %d.", sz, len(lrs))
	}

	bktsz := len(lrs) / sz
	log.Printf("Bucket size for %d/%d is %d\n", len(lrs), sz, bktsz)

	var total, time float64
	var count int = 0
	for _, v := range lrs {
		if count == 0 {
			time = v.time
			total = 0.0
		}

		total += v.perf
		count++

		if count == bktsz {
			val := total / float64(count)
			out = append(out, LatRec{time, val})
			count = 0
			continue
		}
	}
	return
}
