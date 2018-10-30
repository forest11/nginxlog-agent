package config

import (
	"fmt"
	"io/ioutil"
	"strings"
	"sync"
	"time"

	"gopkg.in/yaml.v2"
)

//FalconMetrics openfalcon 数据类型
type FalconMetrics struct {
	Endpoint    string      `json:"endpoint"`
	Metric      string      `json:"metric"`
	Timestamp   int64       `json:"timestamp"`
	Step        int64       `json:"step"`
	Value       interface{} `json:"value"`
	CounterType string      `json:"counterType"`
	Tags        string      `json:"tags"`
}

// Metrics 定义数据类型
type Metrics struct {
	CountTotal        int64
	Status_2xx        int64
	Status_3xx        int64
	Status_4xx        int64
	Status_5xx        int64
	StatusOther       int64
	ParseErrors       int64
	RequstTimeltOne   int64
	RequstTimeltThree int64
	RequstTimegtThree int64
	rw                sync.RWMutex
}

// NameSpace 配置内容
type NameSpace struct {
	Endpoint    string            `yaml:"endpoint"`
	Format      string            `yaml:"format"`
	SourceFiles []string          `yaml:"source_files"`
	Labels      map[string]string `yaml:"labels"`
}

// AppConfig 配置
type AppConfig struct {
	NameSpace []NameSpace `yaml:"namespace"`
}

// NewMetrics 新建对象
func NewMetrics() *Metrics {
	m := &Metrics{
		CountTotal:        0,
		Status_2xx:        0,
		Status_3xx:        0,
		Status_4xx:        0,
		Status_5xx:        0,
		StatusOther:       0,
		ParseErrors:       0,
		RequstTimeltOne:   0,
		RequstTimeltThree: 0,
		RequstTimegtThree: 0,
		rw:                sync.RWMutex{},
	}
	return m
}

//NewFalconMetrics 新建对象
func NewFalconMetrics() *FalconMetrics {
	return &FalconMetrics{
		Timestamp:   time.Now().Unix(),
		Step:        60,
		CounterType: "GAUGE",
	}
}

//Init FalconMetrics进行赋值
func (f *FalconMetrics) Init(endpoint, metric string, value interface{}, tags map[string]string) {
	f.Endpoint = endpoint
	f.Metric = metric
	f.Value = value
	var tag string
	for k, v := range tags {
		tag += fmt.Sprintf("%s=%s,", k, v)
	}
	tag = strings.Trim(tag, ",")
	f.Tags = tag
}

//CountTotalInc CountTotal数量+1
func (m *Metrics) CountTotalInc() {
	m.rw.Lock()
	defer m.rw.Unlock()

	m.CountTotal++
}

//ParseErrorsInc BytesTotal数量+1
func (m *Metrics) ParseErrorsInc() {
	m.rw.Lock()
	defer m.rw.Unlock()
	m.ParseErrors++
}

//StatusAdd status计数
func (m *Metrics) StatusAdd(status int64) {
	m.rw.Lock()
	defer m.rw.Unlock()

	switch {
	case status >= 0 && status < 300:
		m.Status_2xx++
	case status >= 300 && status < 400:
		m.Status_3xx++
	case status >= 400 && status < 500:
		m.Status_4xx++
	case status >= 500 && status < 600:
		m.Status_5xx++
	default:
		m.StatusOther++
	}
}

//RequestTimeAdd requst响应计数
func (m *Metrics) RequestTimeAdd(req float64) {
	m.rw.Lock()
	defer m.rw.Unlock()

	switch {
	case req >= 0 && req < 1:
		m.RequstTimeltOne++
	case req >= 1 && req < 3:
		m.RequstTimeltThree++
	default:
		m.RequstTimegtThree++
	}
}

// InitConfig 初始化配置文件
func InitConfig(file string) (app AppConfig, err error) {
	buf, err := ioutil.ReadFile(file)
	if err != nil {
		return
	}

	err = yaml.Unmarshal(buf, &app)
	if err != nil {
		return
	}
	return
}
