package telemetry

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"time"

	pr "github.com/influxdata/influxdb/prometheus"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/common/expfmt"
)

// Pusher pushes metrics to a prometheus push gateway.
type Pusher struct {
	URL    string
	Gather prometheus.Gatherer
	Client *http.Client
}

// NewPusher sends usage metrics to a prometheus push gateway.
func NewPusher(g prometheus.Gatherer) *Pusher {
	return &Pusher{
		//URL: "https://0czqg3djc8.execute-api.us-east-1.amazonaws.com/prod",
		URL: "http://localhost:8080/metrics/job/influxdb",
		Gather: &pr.Filter{
			Gatherer: g,
			Matcher:  telemetryMatcher,
		},
		Client: &http.Client{
			Transport: http.DefaultTransport,
			Timeout:   10 * time.Second,
		},
	}
}

// Push POSTs prometheus metrics in protobuf delimited format to a push gateway.
func (p *Pusher) Push(ctx context.Context) error {
	resps := make(chan (error))
	go func() {
		resps <- p.push(ctx)
	}()

	select {
	case err := <-resps:
		return err
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (p *Pusher) push(ctx context.Context) error {
	r, err := p.encode()
	if err != nil {
		return err
	}

	// when there are no metrics to send, then, no need to POST.
	if r == nil {
		return nil
	}

	req, err := http.NewRequest(http.MethodPost, p.URL, r)
	if err != nil {
		return err
	}

	req = req.WithContext(ctx)

	req.Header.Set("Content-Type", string(expfmt.FmtProtoDelim))

	res, err := p.Client.Do(req)
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	if err != nil {
		return err
	}

	defer res.Body.Close()
	if res.StatusCode != http.StatusAccepted {
		body, _ := ioutil.ReadAll(res.Body)
		return fmt.Errorf("unable to POST metrics; received status %s: %s", http.StatusText(res.StatusCode), body)
	}
	return nil
}

func (p *Pusher) encode() (io.Reader, error) {
	mfs, err := p.Gather.Gather()
	if err != nil {
		return nil, err
	}

	if len(mfs) == 0 {
		return nil, nil
	}

	buf := &bytes.Buffer{}
	enc := expfmt.NewEncoder(buf, expfmt.FmtProtoDelim)
	for _, mf := range mfs {
		if err := enc.Encode(mf); err != nil {
			return nil, err
		}
	}

	return buf, nil
}