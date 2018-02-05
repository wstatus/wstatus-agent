package agent

import (
	"net/http"
	"net"
	"time"
	log "github.com/sirupsen/logrus"
	"bytes"
	"encoding/json"
	"syscall"
	"os/signal"
	"os"
	"github.com/zmalik/wstatus-agent/pkg/config"
	"fmt"
)

const (
	API_KEY_HEADER = "X-WStatus-Key"
)

type Check struct {
	Endpoint string `json:"endpoint,omitempty"`
	Protocol string `json:"protocol,omitempty"`
	Id       string `json:"id,omitempty"`
}

type UptimeResult struct {
	Id         string        `json:"id,omitempty"`
	Latency    time.Duration `json:"latency,omitempty"`
	StatusCode int           `json:"statusCode,omitempty"`
	Err        string        `json:"err,omitempty"`
}

type Worker struct {
	token  string
	client *http.Client
	stop   bool
}

func NewWorker(token string) *Worker {
	return &Worker{
		token: token,
	}
}
func (w *Worker) Run() {
	w.init()
	for err := w.connect(); err != nil; {
		log.Errorln(err.Error())
		log.Infof("will try again in %s", config.GetDefaultPolling())
		time.Sleep(config.GetDefaultPolling())
		err = w.connect()
	}
	for {
		check := w.fetchWorkAndSendPulse()
		if check != nil {
			result := w.Do(check)
			log.Infof("Check done : %v", result)
			w.SendResults(result)
		}
		time.Sleep(config.GetDefaultPolling())
	}
}

func (w *Worker) init() {
	if w.token == "" {
		log.Fatalf("Empty WSTATUS_TOKEN variable. Set the env variable or use the flag.")
	}
	w.client = DefaultHTTPClient
	w.stop = false
	w.configureGracefulStop()

}

func (w *Worker) connect() error {
	log.Infof("Connecting to central scheduler...")
	req, _ := http.NewRequest("GET", fmt.Sprintf("%svalidate", config.GetEndpoint()), nil)
	req.Header.Set(API_KEY_HEADER, w.token)
	req.Header.Set("Content-Type", "application/json")
	resp, err := w.client.Do(req)
	if err != nil {
		return fmt.Errorf("cannot connect to the api: %s, %s", config.GetEndpoint(), err.Error())
	}
	if resp.StatusCode != 200 {
		return fmt.Errorf("cannot connect to the api: %s, %s", config.GetEndpoint(), resp.Status)
	}
	if resp.Body != nil {
		defer resp.Body.Close()
	}
	log.Infof("Connected.")
	return nil
}

func (w *Worker) fetchWorkAndSendPulse() *Check {
	req, _ := http.NewRequest("GET", config.GetEndpoint(), nil)
	req.Header.Set(API_KEY_HEADER, w.token)
	resp, err := w.client.Do(req)
	if err != nil {
		log.Errorf("Fetching work failed: %s, %s", config.GetEndpoint(), err.Error())
		return nil
	}
	if resp.StatusCode != 200 {
		log.Errorf("Fetching work failed: %s, %s", config.GetEndpoint(), resp.Status)
		return nil
	}
	defer resp.Body.Close()
	work := &Check{}
	decoder := json.NewDecoder(resp.Body)
	err = decoder.Decode(&work)
	if err != nil {
		log.Errorf("Error formating the body. %s", err.Error())
	}
	return work
}

func (w *Worker) Do(check *Check) *UptimeResult {
	start := time.Now()
	resp, err := w.client.Head(check.Endpoint)
	if err == nil {
		defer resp.Body.Close()
		elapsed := time.Since(start)
		return &UptimeResult{
			Id:         check.Id,
			Latency:    elapsed,
			StatusCode: resp.StatusCode,
		}
	} else {
		return &UptimeResult{
			Id:  check.Id,
			Err: err.Error(),
		}
	}
}

func (w *Worker) SendResults(result *UptimeResult) {
	body, err := json.Marshal(*result)
	req, err := http.NewRequest("POST", config.GetEndpoint(), bytes.NewBuffer(body))
	req.Header.Set(API_KEY_HEADER, w.token)
	req.Header.Set("Content-Type", "application/json")
	resp, err := w.client.Do(req)

	if err != nil {
		log.Errorf("Error sending results to the endpoint: %s, %s", config.GetEndpoint(), err.Error())
	}
	if resp.Body != nil {
		defer resp.Body.Close()
	}
}

var DefaultHTTPClient = &http.Client{
	Transport: &http.Transport{
		Proxy: http.ProxyFromEnvironment,
		Dial: (&net.Dialer{
			Timeout:   10 * time.Second,
			KeepAlive: 0,
		}).Dial,
		TLSHandshakeTimeout:   5 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
		MaxIdleConnsPerHost:   1,
		DisableCompression:    true,
		DisableKeepAlives:     true,
		ResponseHeaderTimeout: 5 * time.Second,
	},
	CheckRedirect: func(req *http.Request, via []*http.Request) error {
		return http.ErrUseLastResponse
	},
	Timeout: 10 * time.Second,
}

func (w *Worker) configureGracefulStop() {
	var stopChannel = make(chan os.Signal)
	signal.Notify(stopChannel, syscall.SIGTERM)
	signal.Notify(stopChannel, syscall.SIGTRAP)
	signal.Notify(stopChannel, syscall.SIGINT)
	signal.Notify(stopChannel, syscall.SIGSTOP)
	signal.Notify(stopChannel, syscall.SIGQUIT)
	go func() {
		sig := <-stopChannel
		log.Infof("OS signal caught: %+v. Shutting down the agent.", sig)
		os.Exit(0)
	}()
}
