package server

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
)

func ExampleGetAllMetricHandler() {
	url := "http://localhost:8080/"

	resp, err := http.Get(url)
	if err != nil {
		fmt.Printf("%s", err.Error())
	}
	defer resp.Body.Close()
}

func ExampleService_SetMetricHandler() {
	m := &Metric{}

	mSer, err := json.Marshal(m)
	if err != nil {
		fmt.Println(err)
	}

	url := "http://localhost:8080/update/"

	resp, err := http.Post(url, "application/json", bytes.NewBuffer(mSer))
	if err != nil {
		fmt.Printf("%s", err.Error())
	}
	defer resp.Body.Close()
}

func ExampleService_SetMetricListHandler() {
	m := &[]Metric{}

	mSer, err := json.Marshal(m)
	if err != nil {
		fmt.Println(err)
	}

	url := "http://localhost:8080/updates/"

	resp, err := http.Post(url, "application/json", bytes.NewBuffer(mSer))
	if err != nil {
		fmt.Printf("%s", err.Error())
	}
	defer resp.Body.Close()
}

func ExampleService_GetMetricHandler() {
	m := &Metric{}

	mSer, err := json.Marshal(m)
	if err != nil {
		fmt.Println(err)
	}

	url := "http://localhost:8080/value/"

	resp, err := http.Post(url, "application/json", bytes.NewBuffer(mSer))
	if err != nil {
		fmt.Printf("%s", err.Error())
	}
	defer resp.Body.Close()
}

func ExampleService_SetMetricOldHandler() {
	metricName := "Alloc"
	metricValue := 0.2345

	url := fmt.Sprintf("http://localhost:8080/update/gauge/%s/%f", metricName, metricValue)

	resp, err := http.Post(url, "application/json", nil)
	if err != nil {
		fmt.Printf("%s", err.Error())
	}
	defer resp.Body.Close()
}

func ExampleService_SetMetricOldHandler_second() {
	metricName := "PollCount"
	metricValue := 2

	url := fmt.Sprintf("http://localhost:8080/update/counter/%s/%d", metricName, metricValue)

	resp, err := http.Post(url, "application/json", nil)
	if err != nil {
		fmt.Printf("%s", err.Error())
	}
	defer resp.Body.Close()
}

func ExampleGetMetricOldHandler() {
	metricName := "PollCount"

	url := fmt.Sprintf("http://localhost:8080/value/%s", metricName)

	resp, err := http.Get(url)
	if err != nil {
		fmt.Printf("%s", err.Error())
	}
	defer resp.Body.Close()
}

func ExampleService_PingDBHandler() {
	url := "http://localhost:8080/ping"

	resp, err := http.Get(url)
	if err != nil {
		fmt.Printf("%s", err.Error())
	}
	defer resp.Body.Close()
}
