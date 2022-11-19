package main

import (
	"fmt"
	"log"
	"net/http"
)

func (a Agent) sendDataOld(m *Metric) error {
	var url string
	switch m.MType {
	case gauge:
		url = fmt.Sprintf("http://%s/update/%s/%s/%f", a.cfg.Address, m.MType, m.ID, *m.Value)
	case counter:
		url = fmt.Sprintf("http://%s/update/%s/%s/%d", a.cfg.Address, m.MType, m.ID, *m.Delta)
	}
	resp, err := http.Post(url, "text/plain", nil)
	if err != nil {
		log.Println(err)
		return err
	}
	defer resp.Body.Close()
	return err
}
