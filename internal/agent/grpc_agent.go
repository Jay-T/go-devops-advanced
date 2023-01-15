package agent

import (
	"context"
	"log"
	"time"

	"github.com/Jay-T/go-devops.git/internal/pb"
	"github.com/Jay-T/go-devops.git/internal/utils/metric"
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
	interceptor := getClientInterceptor(genericAgent.localAddress)
	conn, err := grpc.Dial(cfg.Address, grpc.WithTransportCredentials(insecure.NewCredentials()), grpc.WithUnaryInterceptor(interceptor))
	if err != nil {
		return nil, err
	}

	client := pb.NewMetricsAgentClient(conn)

	return &GRPCAgent{
		genericAgent,
		conn,
		client,
	}, nil
}

func (a *GRPCAgent) sendData(ctx context.Context, m *metric.Metric) error {
	pbMetric := m.ConvertMetricToPB(a.Cfg.Key)

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

func (a *GRPCAgent) sendBulkData(ctx context.Context, mList *[]metric.Metric) error {
	var pbMetrics []*pb.Metric

	for _, m := range *mList {
		pbMetrics = append(pbMetrics, m.ConvertMetricToPB(a.Cfg.Key))
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
	var mList []metric.Metric

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
func (a *GRPCAgent) Run(ctx context.Context, doneChan chan<- struct{}) {
	dataChan := make(chan Data)
	syncChan := make(chan time.Time)

	// GenericAgent methods
	go a.RunTicker(ctx, syncChan)
	go a.NewMetric(ctx, dataChan)
	go a.GetDataByInterval(ctx, dataChan, syncChan)
	go a.GetMemDataByInterval(ctx, dataChan, syncChan)
	go a.GetCPUDataByInterval(ctx, dataChan)

	// GRPCAgent method
	go a.SendDataByInterval(ctx, dataChan, doneChan)
}
