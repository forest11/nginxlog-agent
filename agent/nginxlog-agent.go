package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/astaxie/beego/logs"

	"github.com/forest11/nginxlog-agent/config"
	"github.com/forest11/nginxlog-agent/log"
	"github.com/forest11/nginxlog-agent/tailf"
	"github.com/satyrius/gonx"
)

const (
	falconURL = "http://127.0.0.1:1988/v1/push"
	logPath   = "/var/log/nginxlog-agent.log"
)

func main() {
	var confFile = flag.String("f", "log_agent.yaml", "log config format yaml")
	flag.Parse()

	cfg, err := config.InitConfig(*confFile)
	if err != nil {
		panic(fmt.Sprintf("init config failed, err:%v", err))
	}

	err = log.InitLog(logPath, "info")
	if err != nil {
		panic(fmt.Sprintf("init logger failed, err:%v", err))
	}

	clearChan := make(chan struct{}, len(cfg.NameSpace))

	for _, ns := range cfg.NameSpace {
		go processNamespace(ns, clearChan)
	}

	t := time.NewTicker(time.Second * 60)
	defer t.Stop()
	for v := range t.C {
		logs.Info("time:%v start push data", v)
		for i := 0; i < len(cfg.NameSpace); i++ {
			clearChan <- struct{}{} //每个namespace对应的线程，都需要获取定时器
		}
	}
}

func processNamespace(nsCfg config.NameSpace, clearChan chan struct{}) {
	parser := gonx.NewParser(nsCfg.Format)

	for _, f := range nsCfg.SourceFiles {
		t, err := tailf.NewFollower(f)
		if err != nil {
			panic(err)
		}

		t.OnError(func(err error) {
			panic(err)
		})

		go processSourceFile(nsCfg, t, parser, clearChan)
	}
}

func processSourceFile(cfg config.NameSpace, t tailf.Follower, parser *gonx.Parser, clearChan chan struct{}) {
	metrics := config.NewMetrics()

	go func() {
		for {
			<-clearChan
			push(cfg, metrics)
			metrics = config.NewMetrics()
		}
	}()

	for line := range t.Lines() {
		entry, err := parser.ParseString(line.Text)
		if err != nil {
			logs.Error("error while parsing line '%s': %s\n", line.Text, err)
			metrics.ParseErrorsInc()
			continue
		}

		metrics.CountTotalInc()

		if status, err := entry.Field("status"); err == nil {
			value, err := strconv.ParseInt(status, 0, 64)
			if err == nil {
				metrics.StatusAdd(value)
			}
		}

		if req, err := entry.Field("request_time"); err == nil {
			value, err := strconv.ParseFloat(req, 64)
			if err == nil {
				metrics.RequestTimeAdd(value)
			}
		}

	}
}

func push(cfg config.NameSpace, metrics *config.Metrics) {
	var data []config.FalconMetrics

	countTotal := config.NewFalconMetrics()
	countTotal.Init(cfg.Endpoint, "countTotal", metrics.CountTotal, cfg.Labels)
	data = append(data, *countTotal)

	status2xx := config.NewFalconMetrics()
	status2xx.Init(cfg.Endpoint, "status_2xx", metrics.Status_2xx, cfg.Labels)
	data = append(data, *status2xx)

	status3xx := config.NewFalconMetrics()
	status3xx.Init(cfg.Endpoint, "status_3xx", metrics.Status_3xx, cfg.Labels)
	data = append(data, *status3xx)

	status4xx := config.NewFalconMetrics()
	status4xx.Init(cfg.Endpoint, "status_4xx", metrics.Status_4xx, cfg.Labels)
	data = append(data, *status4xx)

	status5xx := config.NewFalconMetrics()
	status5xx.Init(cfg.Endpoint, "status_5xx", metrics.Status_5xx, cfg.Labels)
	data = append(data, *status5xx)

	statusOther := config.NewFalconMetrics()
	statusOther.Init(cfg.Endpoint, "status_other", metrics.StatusOther, cfg.Labels)
	data = append(data, *statusOther)

	parseErrors := config.NewFalconMetrics()
	parseErrors.Init(cfg.Endpoint, "parse_errors", metrics.ParseErrors, cfg.Labels)
	data = append(data, *parseErrors)

	requstTimeltOne := config.NewFalconMetrics()
	requstTimeltOne.Init(cfg.Endpoint, "requstTime_lt_1s", metrics.RequstTimeltOne, cfg.Labels)
	data = append(data, *requstTimeltOne)

	requstTimeltThree := config.NewFalconMetrics()
	requstTimeltThree.Init(cfg.Endpoint, "requstTime_lt_3s", metrics.RequstTimeltThree, cfg.Labels)
	data = append(data, *requstTimeltThree)

	requstTimegtThree := config.NewFalconMetrics()
	requstTimegtThree.Init(cfg.Endpoint, "requstTime_gt_3s", metrics.RequstTimegtThree, cfg.Labels)
	data = append(data, *requstTimegtThree)

	dataBytes, _ := json.Marshal(data)
	//fmt.Printf("%s\n", dataBytes)
	httpPost(dataBytes)
}

func httpPost(msg []byte) {
	bodyReader := bytes.NewBuffer(msg)
	rep, err := http.Post(falconURL, "application/json", bodyReader)
	if err != nil {
		logs.Error("http post err: %s", err)
	}
	defer rep.Body.Close()
}
