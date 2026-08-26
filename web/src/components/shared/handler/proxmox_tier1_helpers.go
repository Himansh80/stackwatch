package handler

import (
	"fmt"
	"strconv"
)

func parseVMID(raw string) (int, error) {
	vmid, err := strconv.Atoi(raw)
	if err != nil || vmid < 1 {
		return 0, fmt.Errorf("invalid vmid")
	}
	return vmid, nil
}
