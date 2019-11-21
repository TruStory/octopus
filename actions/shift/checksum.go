package main

import (
	"fmt"
	"strings"
)

var gen = []int{0x3b6a57b2, 0x26508e6d, 0x1ea119fa, 0x3d4233dd, 0x2a1462b3}

const charset = "qpzry9x8gf2tvdw0s3jn54khce6mua7l"

func fixChecksum(cosmosPrefixedBech string) (string, error) {
	// Locating the position of 1 in the string
	one := strings.LastIndexByte(cosmosPrefixedBech, '1')
	if one < 1 || one+7 > len(cosmosPrefixedBech) {
		panic("invalid index of 1")
	}

	// The human-readable part (hrp) is everything before the last '1'.
	hrp := cosmosPrefixedBech[:one]
	data := cosmosPrefixedBech[one+1:]
	dataBz, err := toBytes(data)
	if err != nil {
		panic(err)
	}

	truhrp := Bech32PrefixAccAddr + strings.TrimPrefix(hrp, "cosmos") // cosmosvaloper => truvaloper

	truChecksum, err := toChars(bech32Checksum(truhrp, dataBz[:len(dataBz)-6]))
	if err != nil {
		return "", err
	}

	truPrefixedBech := truhrp + strings.TrimSuffix(data, cosmosPrefixedBech[len(data)-6:]) + truChecksum

	return truPrefixedBech, nil
}

// toChars converts the byte slice 'data' to a string where each byte in 'data'
// encodes the index of a character in 'charset'.
func toChars(data []byte) (string, error) {
	result := make([]byte, 0, len(data))
	for _, b := range data {
		if int(b) >= len(charset) {
			return "", fmt.Errorf("invalid data byte: %v", b)
		}
		result = append(result, charset[b])
	}
	return string(result), nil
}

// toBytes converts each character in the string 'chars' to the value of the
// index of the correspoding character in 'charset'.
func toBytes(chars string) ([]byte, error) {
	decoded := make([]byte, 0, len(chars))
	for i := 0; i < len(chars); i++ {
		index := strings.IndexByte(charset, chars[i])
		if index < 0 {
			return nil, fmt.Errorf("invalid character not part of "+
				"charset: %v", chars[i])
		}
		decoded = append(decoded, byte(index))
	}
	return decoded, nil
}

// For more details on the checksum calculation, please refer to BIP 173.
func bech32Checksum(hrp string, data []byte) []byte {
	// Convert the bytes to list of integers, as this is needed for the
	// checksum calculation.
	integers := make([]int, len(data))
	for i, b := range data {
		integers[i] = int(b)
	}
	values := append(bech32HrpExpand(hrp), integers...)
	values = append(values, []int{0, 0, 0, 0, 0, 0}...)
	polymod := bech32Polymod(values) ^ 1
	var res []byte
	for i := 0; i < 6; i++ {
		res = append(res, byte((polymod>>uint(5*(5-i)))&31))
	}
	return res
}

// For more details on HRP expansion, please refer to BIP 173.
func bech32HrpExpand(hrp string) []int {
	v := make([]int, 0, len(hrp)*2+1)
	for i := 0; i < len(hrp); i++ {
		v = append(v, int(hrp[i]>>5))
	}
	v = append(v, 0)
	for i := 0; i < len(hrp); i++ {
		v = append(v, int(hrp[i]&31))
	}
	return v
}

// For more details on the polymod calculation, please refer to BIP 173.
func bech32Polymod(values []int) int {
	chk := 1
	for _, v := range values {
		b := chk >> 25
		chk = (chk&0x1ffffff)<<5 ^ v
		for i := 0; i < 5; i++ {
			if (b>>uint(i))&1 == 1 {
				chk ^= gen[i]
			}
		}
	}
	return chk
}
