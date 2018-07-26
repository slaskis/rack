package aws

import (
	"bytes"
	"encoding/json"
	"net/http"
	"time"

	"github.com/convox/logger"
	"github.com/convox/rack/helpers"
)

func (p *Provider) workerHeartbeat() {
	log := logger.New("ns=workers.heartbeat")

	defer recoverWith(func(err error) {
		helpers.Error(log, err)
	})

	p.heartbeat(log)

	for range time.Tick(1 * time.Hour) {
		p.heartbeat(log)
	}
}

func (p *Provider) heartbeat(log *logger.Logger) {
	s, err := p.SystemGet()
	if err != nil {
		log.Error(err)
		return
	}

	as, err := p.AppList()
	if err != nil {
		log.Error(err)
		return
	}

	v := map[string]interface{}{
		"id":             p.StackId,
		"app_count":      len(as),
		"instance_count": s.Count,
		"instance_type":  s.Type,
		"region":         p.Region,
		"version":        s.Version,
	}

	data, err := json.Marshal(v)
	if err != nil {
		log.Error(err)
		return
	}

	http.Post("https://metrics.convox.com/metrics/rack/heartbeat", "application/json", bytes.NewReader(data))
}
