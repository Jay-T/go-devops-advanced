package server

import (
	"encoding/json"
	"os"
)

type producer struct {
	file    *os.File
	encoder *json.Encoder
}

func NewProducer(filename string, flags int) (*producer, error) {
	file, err := os.OpenFile(filename, flags, 0777)
	if err != nil {
		return nil, err
	}

	return &producer{
		file:    file,
		encoder: json.NewEncoder(file),
	}, nil
}

func (p producer) Close() error {
	return p.file.Close()
}

func (p producer) WriteMetric(MetricList *[]Metric) error {
	return p.encoder.Encode(&MetricList)
}

type consumer struct {
	file    *os.File
	decoder *json.Decoder
}

func NewConsumer(filename string, flags int) (*consumer, error) {
	file, err := os.OpenFile(filename, flags, 0777)
	if err != nil {
		return nil, err
	}

	return &consumer{
		file:    file,
		decoder: json.NewDecoder(file),
	}, nil
}

func (c *consumer) ReadEvents() error {
	defer c.Close()
	MetricList := []Metric{}
	if err := c.decoder.Decode(&MetricList); err != nil {
		return err
	}
	for _, i := range MetricList {
		metrics[i.ID] = i
	}
	return nil
}

func (c consumer) Close() error {
	return c.file.Close()
}
