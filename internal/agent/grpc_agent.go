package agent

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/Jay-T/go-devops.git/internal/pb"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type GRPCRequestError struct {
	msg string
}

func (e *GRPCRequestError) Error() string {
	return e.msg
}

func NewGRPCRequestError(text string) error {
	return &GRPCRequestError{msg: text}
}

type GRPCAgent struct {
	*GenericAgent
	conn   *grpc.ClientConn
	client pb.MetricsAgentClient
}

func NewGRPCAgent(cfg *Config) (*GRPCAgent, error) {
	genericAgent, err := NewGenericAgent(cfg)
	if err != nil {
		return nil, err
	}
	conn, err := grpc.Dial(cfg.Address, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return &GRPCAgent{}, err
	}
	client := pb.NewMetricsAgentClient(conn)
	return &GRPCAgent{
		genericAgent,
		conn,
		client,
	}, nil
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

func (a *GRPCAgent) convertMetric(m *Metric) *pb.Metric {
	if a.Cfg.Key != "" {
		a.AddHash(m)
	}
	if m.MType == counter {
		return &pb.Metric{
			Id:    m.ID,
			Mtype: m.MType,
			Delta: *m.Delta,
			Hash:  m.Hash,
		}
	}
	return &pb.Metric{
		Id:    m.ID,
		Mtype: m.MType,
		Value: *m.Value,
		Hash:  m.Hash,
	}
}

func (a *GRPCAgent) sendData(m *Metric) error {
	log.Printf("Sending! %v+", m)

	pbMetric := a.convertMetric(m)

	req := &pb.UpdateMetricRequest{
		Metric: pbMetric,
	}

	res, err := a.client.UpdateMetric(context.Background(), req)
	if err != nil {
		log.Printf("Error during sendData, %s", err)
		return err
	}

	if res.Error != "" {
		log.Printf("server rejected metric: %s", res.Error)
		return NewGRPCRequestError(res.Error)
	}
	return nil
}

func (a *GRPCAgent) sendBulkData(mList *[]Metric) error {
	log.Println("Sending bulk!")
	var pbMetrics []*pb.Metric

	for _, m := range *mList {
		pbMetrics = append(pbMetrics, a.convertMetric(&m))
	}

	req := &pb.UpdateMetricsRequest{
		Metrics: pbMetrics,
	}

	res, err := a.client.UpdateMetrics(context.Background(), req)
	if err != nil {
		log.Printf("Error during sendData, %s", err)
		return err
	}

	if res.Error != "" {
		log.Printf("server rejected metric: %s", res.Error)
		return NewGRPCRequestError(res.Error)
	}
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
	go a.SendDataByInterval(ctx, dataChan, doneChan)
	a.StopAgent(sigChan, doneChan, cancel)
}
