package types

import (
	"encoding/json"
	"fmt"
	"testing"
)

func TestJson(t *testing.T) {
	s := Service{
		ID:          "test-service-a",
		Addr:        "192.168.1.2:1900",
		Description: "test description",
		Enabled:     false,
	}
	data, _ := json.Marshal(&s)
	fmt.Println(string(data))
}
