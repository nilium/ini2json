// Command ini2json converts an INI file to JSON.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"math/big"
	"os"
	"strconv"

	ini "go.spiff.io/go-ini"
)

func usage() {
	fmt.Fprint(os.Stderr, `USAGE: ini2json [OPTIONS] [FILES]

Convert INI files to JSON.
If no files are passed or "-" is passed, it reads from standard input.

OPTIONS:
-s SEP    Separator for [prefix] and field names. (Default: '.')
-C TFORM  Case transformation.
            -  No case transformation.
            l  Lowercase all keys (including prefix).
            u  Uppercase all keys (including prefix).
-t TRUE   Any field without a value is assigned the value 'TRUE'.
          (Default: 'true')
-m        Merge all input files into a single JSON output.
-c        Print compact JSON output.
-r        Do not parse values (integers, floats, bools, JSON).
`)
}

func newRawValues() ini.Recorder {
	return ini.Values{}
}

func newTypedValues() ini.Recorder {
	return typedValues{}
}

func main() {
	log.SetFlags(0)

	var (
		newValues = newTypedValues
		raw       = false
		casing    = "-"
		merge     = false
		compact   = false
		rd        = &ini.Reader{
			True: "true",
		}
	)

	flag.CommandLine.Usage = usage
	// Reader flags
	flag.StringVar(&rd.Separator, "s", ".", "prefix separator")
	flag.StringVar(&casing, "C", casing, "case transformation (l to lowercase keys, u to uppercase, - to do nothing)")
	flag.StringVar(&rd.True, "t", rd.True, "true value")
	// Program flags
	flag.BoolVar(&merge, "m", false, "merge files")
	flag.BoolVar(&compact, "c", false, "compact output")
	flag.BoolVar(&raw, "r", false, "do not parse values as integers, floats, bools, or JSON")
	flag.Parse()

	if raw {
		newValues = newRawValues
	}

	args := flag.Args()
	if len(args) == 0 {
		args = []string{"-"}
	}

	switch casing {
	case "l":
		rd.Casing = ini.LowerCase
	case "u":
		rd.Casing = ini.UpperCase
	case "-":
		rd.Casing = ini.CaseSensitive
	default:
		log.Fatalf("invalid case value %+q: must be one of l, u, or -", casing)
	}

	enc := json.NewEncoder(os.Stdout)
	if !compact {
		enc.SetIndent("", "  ")
	}

	values := newValues()
	for _, path := range args {
		if err := read(values, rd, path); err != nil {
			log.Fatalf("unable to parse %v: %v", path, err)
		}

		if merge {
			continue
		}

		if err := enc.Encode(values); err != nil {
			log.Fatalf("unable to encode values from %v: %v", path, err)
		}
		values = newValues()
	}

	if !merge {
		return
	}

	if err := enc.Encode(values); err != nil {
		log.Fatalf("unable to encode final values: %v", err)
	}
}

func read(dest ini.Recorder, rd *ini.Reader, path string) error {
	var r io.Reader
	switch path {
	case "-":
		r = os.Stdin
	default:
		f, err := os.Open(path)
		if err != nil {
			return err
		}
		defer f.Close()
		r = f
	}
	return rd.Read(r, dest)
}

type typedValues map[string][]interface{}

type bigFloat big.Float

func (b *bigFloat) Float() *big.Float {
	return (*big.Float)(b)
}

func (b *bigFloat) MarshalJSON() ([]byte, error) {
	return b.Float().MarshalText()
}

func (v typedValues) Add(key, value string) {
	var jsval interface{}
	if ival, ok := new(big.Int).SetString(value, 10); ok {
		jsval = ival
	} else if fval, _, err := big.ParseFloat(value, 10, 256, big.ToNearestEven); err == nil {
		jsval = (*bigFloat)(fval)
	} else if bval, err := strconv.ParseBool(value); err == nil {
		jsval = bval
	} else if json.Unmarshal([]byte(value), &jsval) == nil {
	} else {
		jsval = value
	}
	v[key] = append(v[key], jsval)
}
