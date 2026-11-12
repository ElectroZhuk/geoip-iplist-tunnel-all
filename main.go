package main

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"sort"
)

type field struct {
	number int
	wire   int
	bytes  []byte
	value  uint64
	raw    []byte
}

func readField(data []byte, off *int) (field, error) {
	var f field
	start := *off
	if start >= len(data) {
		return f, errors.New("unexpected end of protobuf message")
	}

	tag, n := binary.Uvarint(data[*off:])
	if n <= 0 {
		return f, errors.New("invalid protobuf tag")
	}
	*off += n

	f.number = int(tag >> 3)
	f.wire = int(tag & 7)
	if f.number == 0 {
		return f, errors.New("invalid protobuf field number 0")
	}

	switch f.wire {
	case 0:
		v, n := binary.Uvarint(data[*off:])
		if n <= 0 {
			return f, errors.New("invalid protobuf varint")
		}
		*off += n
		f.value = v
	case 1:
		if len(data)-*off < 8 {
			return f, errors.New("truncated fixed64 field")
		}
		*off += 8
	case 2:
		l, n := binary.Uvarint(data[*off:])
		if n <= 0 {
			return f, errors.New("invalid protobuf length")
		}
		*off += n
		if l > uint64(len(data)-*off) {
			return f, errors.New("truncated length-delimited field")
		}
		f.bytes = data[*off : *off+int(l)]
		*off += int(l)
	case 5:
		if len(data)-*off < 4 {
			return f, errors.New("truncated fixed32 field")
		}
		*off += 4
	default:
		return f, fmt.Errorf("unsupported protobuf wire type %d", f.wire)
	}

	f.raw = data[start:*off]
	return f, nil
}

type cidr struct {
	ip     []byte
	prefix uint64
	raw    []byte
}

func parseCIDR(data []byte) (cidr, error) {
	var c cidr
	var hasIP, hasPrefix bool

	for off := 0; off < len(data); {
		f, err := readField(data, &off)
		if err != nil {
			return c, fmt.Errorf("CIDR: %w", err)
		}
		switch {
		case f.number == 1 && f.wire == 2:
			if hasIP {
				return c, errors.New("CIDR contains duplicate IP field")
			}
			c.ip = append([]byte(nil), f.bytes...)
			hasIP = true
		case f.number == 2 && f.wire == 0:
			if hasPrefix {
				return c, errors.New("CIDR contains duplicate prefix field")
			}
			c.prefix = f.value
			hasPrefix = true
		default:
			return c, fmt.Errorf("CIDR contains unsupported field %d (wire %d)", f.number, f.wire)
		}
	}

	if !hasIP || !hasPrefix {
		return c, errors.New("CIDR is missing IP or prefix")
	}
	if len(c.ip) != 4 {
		return c, fmt.Errorf("expected IPv4 CIDR, got %d-byte address", len(c.ip))
	}
	if c.prefix > 32 {
		return c, fmt.Errorf("invalid IPv4 prefix length %d", c.prefix)
	}
	c.raw = append([]byte(nil), data...)
	return c, nil
}

func parseGeoIP(data []byte) (string, []cidr, error) {
	var code string
	var cidrs []cidr
	var hasCode bool

	for off := 0; off < len(data); {
		f, err := readField(data, &off)
		if err != nil {
			return "", nil, fmt.Errorf("GeoIP entry: %w", err)
		}
		switch {
		case f.number == 1 && f.wire == 2:
			if hasCode {
				return "", nil, errors.New("GeoIP entry contains duplicate country_code")
			}
			code = string(f.bytes)
			hasCode = true
		case f.number == 2 && f.wire == 2:
			c, err := parseCIDR(f.bytes)
			if err != nil {
				return "", nil, fmt.Errorf("group %q: %w", code, err)
			}
			cidrs = append(cidrs, c)
		case f.number == 3 && f.wire == 0:
			if f.value != 0 {
				return "", nil, fmt.Errorf("group %q uses reverse_match; refusing lossy merge", code)
			}
		default:
			return "", nil, fmt.Errorf("group %q contains unsupported field %d (wire %d)", code, f.number, f.wire)
		}
	}

	if !hasCode || code == "" {
		return "", nil, errors.New("GeoIP entry has no country_code")
	}
	return code, cidrs, nil
}

func appendVarint(dst []byte, v uint64) []byte {
	for v >= 0x80 {
		dst = append(dst, byte(v)|0x80)
		v >>= 7
	}
	return append(dst, byte(v))
}

func appendBytesField(dst []byte, number int, value []byte) []byte {
	dst = appendVarint(dst, uint64(number<<3|2))
	dst = appendVarint(dst, uint64(len(value)))
	return append(dst, value...)
}

func cidrKey(c cidr) string {
	return hex.EncodeToString(c.ip) + "/" + fmt.Sprint(c.prefix)
}

func main() {
	if len(os.Args) != 3 {
		fmt.Fprintln(os.Stderr, "usage: geoip-merge INPUT.dat OUTPUT.dat")
		os.Exit(2)
	}

	input, err := os.ReadFile(os.Args[1])
	if err != nil {
		fatal(err)
	}
	if len(input) == 0 {
		fatal(errors.New("source file is empty"))
	}

	var groups int
	var metadata [][]byte
	unique := make(map[string]cidr)

	for off := 0; off < len(input); {
		f, err := readField(input, &off)
		if err != nil {
			fatal(fmt.Errorf("GeoIPList: %w", err))
		}

		if f.number == 1 && f.wire == 2 {
			code, cidrs, err := parseGeoIP(f.bytes)
			if err != nil {
				fatal(err)
			}
			groups++
			for _, c := range cidrs {
				unique[cidrKey(c)] = c
			}
			fmt.Printf("Read group %-32s %d CIDRs\n", code, len(cidrs))
		} else {
			// Preserve top-level metadata/unknown fields from the source file.
			metadata = append(metadata, append([]byte(nil), f.raw...))
		}
	}

	if groups == 0 || len(unique) == 0 {
		fatal(errors.New("no GeoIP groups or IPv4 CIDRs found"))
	}

	keys := make([]string, 0, len(unique))
	for k := range unique {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	var geoIP []byte
	geoIP = appendBytesField(geoIP, 1, []byte("all"))
	for _, k := range keys {
		geoIP = appendBytesField(geoIP, 2, unique[k].raw)
	}

	var output []byte
	for _, raw := range metadata {
		output = append(output, raw...)
	}
	output = appendBytesField(output, 1, geoIP)

	// Validate the generated GeoIPList before writing it.
	var outputGroups, outputCIDRs int
	for off := 0; off < len(output); {
		f, err := readField(output, &off)
		if err != nil {
			fatal(fmt.Errorf("output validation: %w", err))
		}
		if f.number == 1 && f.wire == 2 {
			code, cidrs, err := parseGeoIP(f.bytes)
			if err != nil {
				fatal(fmt.Errorf("output validation: %w", err))
			}
			if code != "all" {
				fatal(fmt.Errorf("output validation: unexpected category %q", code))
			}
			outputGroups++
			outputCIDRs = len(cidrs)
		}
	}
	if outputGroups != 1 || outputCIDRs != len(unique) {
		fatal(fmt.Errorf("output validation failed: groups=%d ranges=%d unique=%d", outputGroups, outputCIDRs, len(unique)))
	}
	if outputCIDRs == 0 {
		fatal(errors.New("output contains no CIDRs"))
	}

	if err := os.WriteFile(os.Args[2], output, 0644); err != nil {
		fatal(err)
	}

	// Ensure the written bytes are exactly the validated bytes.
	written, err := os.ReadFile(os.Args[2])
	if err != nil || !bytes.Equal(written, output) {
		fatal(errors.New("written output verification failed"))
	}

	fmt.Println("Conversion successful")
	fmt.Printf("Source groups: %d\n", groups)
	fmt.Printf("Unique IPv4 CIDRs: %d\n", len(unique))
	fmt.Println("Output category: all")
	fmt.Printf("Output file: %s (%d bytes)\n", os.Args[2], len(output))
}

func fatal(err error) {
	fmt.Fprintln(os.Stderr, "ERROR:", err)
	os.Exit(1)
}
