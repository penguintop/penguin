package property

import (
	"fmt"
	"testing"
	"time"
)

func TestRFC3339ToUTC(t *testing.T) {
	timeStr := "2019-09-21T07:44:30"
	time, _ := RFC3339ToUTC(timeStr)
	fmt.Println("RFC3339ToUTC:", time)
}

func TestUTCToRFC3339(t *testing.T) {
	timeStr := UTCToRFC3339(uint64(time.Now().Unix()))
	fmt.Println("UTCToRFC3339:", timeStr)
}
