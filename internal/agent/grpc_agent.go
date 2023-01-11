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

// GRPCAgent struct describes format of GRPC agent based on GenericAgent.
type GRPCAgent struct {
	*GenericAgent
	conn   *grpc.ClientConn
	client pb.MetricsAgentClient
}

// NewGRPCAgent returns GRPCAgent for work.
func NewGRPCAgent(cfg *Config) (*GRPCAgent, error) {
	genericAgent, err := NewGenericAgent(cfg)
	if err != nil {
		return nil, err
	}
	newGRPCAgent := &GRPCAgent{
		genericAgent,
		nil,
		nil,
	}

	conn, err := grpc.Dial(cfg.Address, grpc.WithTransportCredentials(insecure.NewCredentials()), grpc.WithUnaryInterceptor(newGRPCAgent.clientInterceptor))
	if err != nil {
		return &GRPCAgent{}, err
	}
	client := pb.NewMetricsAgentClient(conn)

	newGRPCAgent.conn = conn
	newGRPCAgent.client = client

	return newGRPCAgent, nil
}

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

func (a *GRPCAgent) sendData(ctx context.Context, m *Metric) error {
	pbMetric := a.convertMetric(m)

	req := &pb.UpdateMetricRequest{
		Metric: pbMetric,
	}

	res, err := a.client.UpdateMetric(ctx, req)
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

func (a *GRPCAgent) sendBulkData(ctx context.Context, mList *[]Metric) error {
	var pbMetrics []*pb.Metric

	for _, m := range *mList {
		pbMetrics = append(pbMetrics, a.convertMetric(&m))
	}

	req := &pb.UpdateMetricsRequest{
		Metrics: pbMetrics,
	}

	res, err := a.client.UpdateMetrics(ctx, req)
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

func (a *GRPCAgent) combineAndSend(ctx context.Context, dataChan chan<- Data, doneChan chan<- struct{}, finFlag bool) {
	var mList []Metric

	func() {
		a.Lock()
		defer a.Unlock()

		for _, m := range a.Metrics {
			err := a.sendData(ctx, &m)
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
		err := a.sendBulkData(ctx, &mList)
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
			a.combineAndSend(ctx, dataChan, doneChan, false)
		case <-ctx.Done():
			log.Println("Received cancel command. Sending processed data.")
			a.combineAndSend(ctx, dataChan, doneChan, true)

			log.Println("Context has been canceled successfully.")
			return
		}
	}
}

// Run begins the agent work.
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
