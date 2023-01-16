package server

import (
	"encoding/json"
	"os"
	"strings"
	"testing"

	"github.com/Jay-T/go-devops.git/internal/utils/metric"
	"github.com/stretchr/testify/assert"
)

func TestNewConsumer(t *testing.T) {
	filename := "/tmp/test"
	flags := os.O_WRONLY | os.O_CREATE | os.O_TRUNC
	_, err := NewConsumer(filename, flags)
	assert.NoError(t, err, "NewConsumer returned error unexpectedly.")
}

func TestReadEvents(t *testing.T) {
	data := strings.NewReader(`[{"id": "CPUutilization2", "type": "gauge", "value": 23.100000000116708, "hash": "aa7b6302546b009d298ac318508237e30369e1f8dd81e3b9441a1c54b6608d2c"}, {"id": "OtherSys", "type": "gauge", "value": 967657, "hash": "9a0374370dfe7e896563dc95182d3137dee1427f6bec1053730132df7f39638c"}, {"id": "MCacheSys", "type": "gauge", "value": 15600, "hash": "4a23e40c21a66f4573fd2c5e2f8f63efb3ca576b145d61b6ba88411e91e2b855"}]`)

	c := &consumer{
		decoder: json.NewDecoder(data),
	}

	metrics := make(map[string]metric.Metric)
	err := c.ReadEvents(metrics)

	assert.NoError(t, err, "Function returned error unexpectedly.")
}
