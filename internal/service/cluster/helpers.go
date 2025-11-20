package cluster

import (
	"fmt"
	"math/big"
	"reflect"
	"strings"

	"github.com/ethereum/go-ethereum/accounts/abi"
)

// decodeClusterAny attempts to extract ClusterSnapshot from a reflected struct value
func decodeClusterAny(v interface{}) (ClusterSnapshot, bool) {
	val := reflect.ValueOf(v)
	if val.Kind() == reflect.Ptr {
		val = val.Elem()
	}
	if val.Kind() != reflect.Struct {
		return ClusterSnapshot{}, false
	}
	// Expect 5 fields in order: uint32, uint64, uint64, bool, *big.Int
	if val.NumField() < 5 {
		return ClusterSnapshot{}, false
	}
	snap := ClusterSnapshot{}
	// validatorCount
	fv := val.Field(0)
	switch fv.Kind() {
	case reflect.Uint32, reflect.Uint, reflect.Uint64:
		snap.ValidatorCount = uint32(fv.Convert(reflect.TypeOf(uint64(0))).Uint())
	default:
		if fv.CanInterface() {
			if bi, ok := fv.Interface().(*big.Int); ok {
				snap.ValidatorCount = uint32(bi.Uint64())
			}
		}
	}
	// networkFeeIndex
	fv = val.Field(1)
	switch fv.Kind() {
	case reflect.Uint64, reflect.Uint, reflect.Uint32:
		snap.NetworkFeeIndex = new(big.Int).SetUint64(fv.Convert(reflect.TypeOf(uint64(0))).Uint())
	default:
		if fv.CanInterface() {
			if bi, ok := fv.Interface().(*big.Int); ok {
				snap.NetworkFeeIndex = new(big.Int).Set(bi)
			}
		}
	}
	if snap.NetworkFeeIndex == nil {
		snap.NetworkFeeIndex = big.NewInt(0)
	}
	// index
	fv = val.Field(2)
	switch fv.Kind() {
	case reflect.Uint64, reflect.Uint, reflect.Uint32:
		snap.Index = new(big.Int).SetUint64(fv.Convert(reflect.TypeOf(uint64(0))).Uint())
	default:
		if fv.CanInterface() {
			if bi, ok := fv.Interface().(*big.Int); ok {
				snap.Index = new(big.Int).Set(bi)
			}
		}
	}
	if snap.Index == nil {
		snap.Index = big.NewInt(0)
	}
	// active
	fv = val.Field(3)
	if fv.Kind() == reflect.Bool {
		snap.Active = fv.Bool()
	}
	// balance
	fv = val.Field(4)
	if fv.CanInterface() {
		if bi, ok := fv.Interface().(*big.Int); ok {
			snap.Balance = new(big.Int).Set(bi)
		}
	}
	if snap.Balance == nil {
		snap.Balance = big.NewInt(0)
	}
	return snap, true
}

// indexOfNonIndexed returns the index in vals slice corresponding to the i-th input
func indexOfNonIndexed(args abi.Arguments, i int) int {
	idx := 0
	for j, a := range args {
		if a.Indexed {
			continue
		}
		if j == i {
			return idx
		}
		idx++
	}
	return -1
}

func equalOperatorIDs(a []uint64, b []uint64) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func joinUint64(arr []uint64, sep string) string {
	if len(arr) == 0 {
		return ""
	}
	parts := make([]string, len(arr))
	for i, v := range arr {
		parts[i] = fmt.Sprintf("%d", v)
	}
	return strings.Join(parts, sep)
}
