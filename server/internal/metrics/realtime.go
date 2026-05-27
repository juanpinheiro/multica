package metrics

import (
	"github.com/prometheus/client_golang/prometheus"

	"github.com/multica-ai/multica/server/internal/realtime"
)

type RealtimeCollector struct {
	metrics *realtime.Metrics

	connectsTotal      *prometheus.Desc
	disconnectsTotal   *prometheus.Desc
	activeConnections  *prometheus.Desc
	slowEvictionsTotal *prometheus.Desc
	messagesSentTotal  *prometheus.Desc
	messagesDropped    *prometheus.Desc
}

func NewRealtimeCollector(m *realtime.Metrics) *RealtimeCollector {
	return &RealtimeCollector{
		metrics: m,

		connectsTotal:      newRealtimeDesc("connects_total", "Total realtime WebSocket connections opened."),
		disconnectsTotal:   newRealtimeDesc("disconnects_total", "Total realtime WebSocket connections closed."),
		activeConnections:  newRealtimeDesc("active_connections", "Current realtime WebSocket connections."),
		slowEvictionsTotal: newRealtimeDesc("slow_evictions_total", "Total realtime clients evicted for slow consumption."),
		messagesSentTotal:  newRealtimeDesc("messages_sent_total", "Total realtime messages sent."),
		messagesDropped:    newRealtimeDesc("messages_dropped_total", "Total realtime messages dropped."),
	}
}

func newRealtimeDesc(name, help string) *prometheus.Desc {
	return prometheus.NewDesc("multica_realtime_"+name, help, nil, nil)
}

func (c *RealtimeCollector) Describe(ch chan<- *prometheus.Desc) {
	for _, desc := range []*prometheus.Desc{
		c.connectsTotal,
		c.disconnectsTotal,
		c.activeConnections,
		c.slowEvictionsTotal,
		c.messagesSentTotal,
		c.messagesDropped,
	} {
		ch <- desc
	}
}

func (c *RealtimeCollector) Collect(ch chan<- prometheus.Metric) {
	if c.metrics == nil {
		return
	}
	m := c.metrics
	ch <- prometheus.MustNewConstMetric(c.connectsTotal, prometheus.CounterValue, float64(m.ConnectsTotal.Load()))
	ch <- prometheus.MustNewConstMetric(c.disconnectsTotal, prometheus.CounterValue, float64(m.DisconnectsTotal.Load()))
	ch <- prometheus.MustNewConstMetric(c.activeConnections, prometheus.GaugeValue, float64(m.ActiveConnections.Load()))
	ch <- prometheus.MustNewConstMetric(c.slowEvictionsTotal, prometheus.CounterValue, float64(m.SlowEvictionsTotal.Load()))
	ch <- prometheus.MustNewConstMetric(c.messagesSentTotal, prometheus.CounterValue, float64(m.MessagesSentTotal.Load()))
	ch <- prometheus.MustNewConstMetric(c.messagesDropped, prometheus.CounterValue, float64(m.MessagesDroppedTotal.Load()))
}

