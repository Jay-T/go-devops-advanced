package server

import (
	"encoding/json"
	"log"
	"os"
)

// producer struct fo saving metrics to file.
type producer struct {
	file    *os.File
	encoder *json.Encoder
}

// NewProducer returns new producer.
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

// Close closes producer's file.
func (p producer) Close() error {
	return p.file.Close()
}

// WriteMetric encodes MetricList before saving to file.
func (p producer) WriteMetric(MetricList *[]Metric) error {
	return p.encoder.Encode(&MetricList)
}

// consumer struct for reading metrics from file.
type consumer struct {
	file    *os.File
	decoder *json.Decoder
}

// NewConsumer returns new consumer.
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

// ReadEvents reads metrics from file, decodes them as MetricList.
func (c *consumer) ReadEvents(metrics map[string]Metric) error {
	var err error

	defer func() {
		err = c.Close()
		if err != nil {
			log.Println(err)
		}
	}()
	MetricList := []Metric{}
	if err := c.decoder.Decode(&MetricList); err != nil {
		return err
	}
	for _, i := range MetricList {
		metrics[i.ID] = i
	}
	return nil
}

// Close closes consumer's file.
func (c consumer) Close() error {
	return c.file.Close()
}
