package agent

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"google.golang.org/grpc"
)

type GRPCAgent struct {
	*GenericAgent
	conn *grpc.ClientConn
}

// func (a *GRPCAgent) sendData(m *Metric) error {
// 	var url string
// 	if a.Cfg.Key != "" {
// 		a.AddHash(m)
// 	}

// 	mSer, err := json.Marshal(*m)
// 	if err != nil {
// 		return err
// 	}
// 	url = fmt.Sprintf("http://%s/update/", a.Cfg.Address)

// 	if a.Encryptor != nil {
// 		mSer, err = a.Encryptor.encrypt(mSer)
// 		if err != nil {
// 			return err
// 		}
// 	}

// 	req, err := http.NewRequest("POST", url, bytes.NewBuffer(mSer))
// 	if err != nil {
// 		return err
// 	}

// 	req.Header.Add("Content-Type", "application/json")
// 	if a.localAddress != "" {
// 		req.Header.Add("X-Real-Ip", a.localAddress)
// 	}

// 	resp, err := a.client.Do(req)
// 	if err != nil {
// 		return err
// 	}

// 	statusOK := resp.StatusCode >= 200 && resp.StatusCode < 300
// 	if !statusOK {
// 		return NewDecryptError(fmt.Sprintf("Non-OK HTTP status: %d", resp.StatusCode))
// 	}

// 	err = resp.Body.Close()
// 	if err != nil {
// 		return err
// 	}
// 	return nil
// }

func (a *GRPCAgent) sendData(m *Metric) error {
	return nil
}

func (a *GRPCAgent) sendBulkData(mList *[]Metric) error {
	return nil
}

func (a *GRPCAgent) combineAndSend(dataChan chan<- Data, doneChan chan<- struct{}, finFlag bool) {
	var mList []Metric

	func() {
		a.Lock()
		defer a.Unlock()

		for _, m := range a.Metrics {
			err := a.sendData(&m)
			if err != nil {
				log.Printf("metric: %s, error: %s", m.ID, err)
			}
			mList = append(mList, m)
			if m.ID == "PollCount" {
				PollCount = 0
			}
		}
	}()

	if finFlag {
		doneChan <- struct{}{}
	}
	if PollCount == 0 {
		dataChan <- Data{name: "PollCount", counterValue: 0}
	}
	if len(mList) > 0 {
		err := a.sendBulkData(&mList)
		if err != nil {
			log.Print(err)
		}
	}
}

// SendDataByInterval gorouting sends data to server every specified interval.
func (a *GRPCAgent) SendDataByInterval(ctx context.Context, dataChan chan<- Data, doneChan chan<- struct{}) {
	log.Printf("Sending data with interval: %s", a.Cfg.ReportInterval)
	log.Printf("Sending data to: %s", a.Cfg.Address)

	ticker := time.NewTicker(a.Cfg.ReportInterval)
	for {
		select {
		case <-ticker.C:
			a.combineAndSend(dataChan, doneChan, false)
		case <-ctx.Done():
			log.Println("Received cancel command. Sending processed data.")
			a.combineAndSend(dataChan, doneChan, true)

			log.Println("Context has been canceled successfully.")
			return
		}
	}
}

func (a *GRPCAgent) Run() {
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan,
		syscall.SIGINT,
		syscall.SIGTERM,
		syscall.SIGQUIT)
	dataChan := make(chan Data)
	syncChan := make(chan time.Time)
	doneChan := make(chan struct{})

	ctx, cancel := context.WithCancel(context.Background())
	go a.RunTicker(ctx, syncChan)
	go a.NewMetric(ctx, dataChan)
	go a.GetDataByInterval(ctx, dataChan, syncChan)
	go a.GetMemDataByInterval(ctx, dataChan, syncChan)
	go a.GetCPUDataByInterval(ctx, dataChan)
	// go a.SendDataByInterval(ctx, dataChan, doneChan)
	a.StopAgent(sigChan, doneChan, cancel)
}
