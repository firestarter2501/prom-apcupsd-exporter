package metric

import (
	"local/apcupsd_exporter/apcupsd"
	"local/apcupsd_exporter/model"
	"os/exec"
	"time"

	promLog "github.com/prometheus/common/log"
)

var defaultFactory = NewFactory()

// CollectorOtps ..
type CollectorOtps struct {
	ApcupsdAddr         string
	ApcaccessPath       string
	ApcaccessFloodLimit time.Duration
	CollectInterval     time.Duration
	Factory             *Factory
}

// Collector ..
type Collector struct {
	Opts CollectorOtps

	started      bool
	collectCh    chan CollectOpts
	currModel    *model.Model
	lastOutput   *apcupsd.Output
	lastOutputTs int64
	metrics      []*Metric
}

// NewCollector ..
func NewCollector(opts CollectorOtps) *Collector {
	if opts.Factory == nil {
		opts.Factory = defaultFactory
	}
	return &Collector{
		Opts: opts,

		collectCh:  make(chan CollectOpts),
		currModel:  model.NewModel(),
		lastOutput: apcupsd.NewOutput(""),
	}
}

// Start method
func (c *Collector) Start() {
	if !c.started {
		c.started = true

		go c.listenCollect()
		go c.loopCollect()
	}
}

// GetModel method
func (c *Collector) GetModel() *model.Model {
	return c.currModel
}

// GetLastOutput method
func (c *Collector) GetLastOutput() *apcupsd.Output {
	return c.lastOutput
}

// Collect method
func (c *Collector) Collect(opts CollectOpts) {
	promLog.Infoln("Collect")

	c.collectCh <- opts
}

func (c *Collector) loopCollect() {
	for {
		c.Collect(CollectOpts{
			PreventFlood: true,
		})
		time.Sleep(c.Opts.CollectInterval)
	}
}

func (c *Collector) listenCollect() {
	for {
		if opts, ok := <-c.collectCh; ok {
			c.collect(opts)
		} else {
			return
		}
	}
}

func (c *Collector) collect(opts CollectOpts) {
	promLog.Infoln("collect()")

	c.updateOutput(opts)
	c.updateModel(opts)
	c.updateMetrics(opts)

	if opts.OnComplete != nil {
		opts.OnComplete <- true
	}
}

func (c *Collector) updateOutput(opts CollectOpts) {
	promLog.Infoln("updating apcupsd output..")

	ts := time.Now().UnixNano()
	if opts.PreventFlood && ts-c.lastOutputTs < int64(c.Opts.ApcaccessFloodLimit) {
		return
	}
	c.lastOutputTs = ts

	cmdResult, err := exec.Command(c.Opts.ApcaccessPath, "status", c.Opts.ApcupsdAddr).Output()
	if err != nil {
		promLog.Errorln("apcaccess exited with error")
		promLog.Errorln("  Error:", err.Error())
		promLog.Errorln("  Result:", string(cmdResult))
		cmdResult = []byte{}
	}

	c.lastOutput = apcupsd.NewOutput(string(cmdResult))
	c.lastOutput.Parse()
}

func (c *Collector) updateModel(opts CollectOpts) {
	promLog.Infoln("updating model..")

	c.currModel.Update(model.NewStateFromOutput(c.lastOutput))

	for field, diff := range c.currModel.ChangedFields {
		promLog.Infof("field changed '%s'\n  OLD: %#v\n  NEW: %#v\n", field, diff[0], diff[1])
	}
}

func (c *Collector) updateMetrics(opts CollectOpts) {
	promLog.Infoln("updating metrics..")

	metrics, metricsChanged := c.Opts.Factory.GetMetrics()
	if metricsChanged {
		promLog.Infoln("metrics changed: unregistering old")
		for _, metric := range c.metrics {
			metric.Unregister()
		}
	}

	c.metrics = metrics
	for _, metric := range c.metrics {
		metric.Update(c.currModel)
	}

	promLog.Infoln("metrics updated")
}

// CollectOpts ..
type CollectOpts struct {
	PreventFlood bool
	OnComplete   chan bool
}
