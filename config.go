package ginx

import (
	"flag"
	"fmt"
	_ "github.com/glebarez/sqlite"
	"github.com/sirupsen/logrus"
	"gopkg.in/yaml.v2"
	_ "gorm.io/driver/mysql"
	_ "gorm.io/driver/postgres"
	"gorm.io/gorm"
	"io/ioutil"
	"sync"
)

var (
	configFlag string
)

func init() {
	flag.StringVar(&configFlag, "c", "config.yaml", "配置文件")
}

type DomainConfig struct {
	FullDomain           string   `json:"fullDomain" yaml:"fullDomain"`                     // 全域名
	Host                 string   `json:"host" yaml:"host"`                                 // 不带 http:// 的域名
	Cid                  string   `json:"cid" yaml:"cid"`                                   // 租户id
	RobotsTxt            string   `json:"robotsTxt" yaml:"robotsTxt"`                       // google 的 robots.txt 文件
	HeaderScript         string   `json:"headerScript" yaml:"headerScript"`                 // 头部的 html
	FooterScript         string   `json:"footerScript" yaml:"footerScript"`                 // 底部的 html
	Template             string   `json:"template" yaml:"template"`                         // 模板名称
	Description          string   `json:"description" yaml:"description"`                   // 描述
	Keywords             string   `json:"keywords" yaml:"keywords"`                         // 关键词
	Copyright            string   `json:"copyright" yaml:"copyright"`                       // 版权信息
	RootTxt              string   `json:"rootTxt" yaml:"rootTxt"`                           //  /root.txt 文件内容
	GoogleSiteVerify     string   `json:"googleSiteVerify" yaml:"googleSiteVerify"`         // googlecfea2817ba1aa4c7.html
	GoogleSiteVerifyText string   `json:"googleSiteVerifyText" yaml:"googleSiteVerifyText"` // google 网站内容
	Spiders              []string `json:"spiders" yaml:"spiders"`                           // 爬虫
	Title                string   `json:"title" yaml:"title"`                               // 网站标签
}

type Config struct {
	DevCid    string                   `json:"devCid" yaml:"devCid"`
	Token     string                   `json:"token" yaml:"token"`
	WebConfig WebConfig                `json:"webConfig" yaml:"webConfig"`
	Domains   map[string]*DomainConfig `yaml:"domains" json:"domains"` // 域名
	DbConfig  DbConfig                 `yaml:"dbConfig" json:"dbConfig"`
}

type WebConfig struct {
	Port         string `json:"port" yaml:"port"`
	Addr         string `json:"addr" yaml:"addr"`
	StaticRoot   string `json:"staticRoot" yaml:"staticRoot"`
	TemplateRoot string `json:"templateRoot" yaml:"templateRoot"`
	Token        string `json:"token" yaml:"token"`
}

type DbConfig struct {
	Driver string `json:"driver" yaml:"driver"`

	Host     string `json:"host" yaml:"host"`
	Port     string `json:"port" yaml:"port"`
	Username string `json:"username" yaml:"username"`
	Password string `json:"password" yaml:"password"`
	Database string `json:"database" yaml:"database"`
}

func (d *DbConfig) ToDsn() string {
	switch d.Driver {
	case "sqlite3":
		if d.Database == "" {
			return `file::memory:?cache=shared`
		}
		return d.Database
	case "mysql":
		return fmt.Sprintf(`%s:%s@tcp(%s:%s)/%s?charset=utf8mb4&parseTime=True&loc=Local`, d.Username, d.Password, d.Host, d.Port, d.Database)
	case "postgres":
		return fmt.Sprintf(`host=%s user=%s password=%s dbname=%s port=%s sslmode=disable TimeZone=Asia/Shanghai`,
			d.Host, d.Username, d.Password, d.Database, d.Port)
	default:
		return ""
	}
}

var globalConfig *Config

func InitConfig() *Config {
	if !flag.Parsed() {
		flag.Parse()
	}

	var cfg Config
	data, err := ioutil.ReadFile(configFlag)
	if err != nil {
		panic(fmt.Errorf("读取配置失败: %v", err))
	}
	err = yaml.Unmarshal(data, &cfg)
	if err != nil {
		panic(fmt.Errorf("解析配置失败: %v", err))
	}
	globalConfig = &cfg

	initDb()
	return globalConfig
}

var (
	mutex sync.RWMutex
)

func GlobalConfig() *Config {
	mutex.RLock()
	defer mutex.RUnlock()

	return globalConfig
}

func WithAccount(ctx *Context) func(db2 *gorm.DB) {
	return func(db *gorm.DB) {
		db.Where("cid = ?", ctx.AccountId)
	}
}

func ReloadConfig() {
	var cfg Config
	data, err := ioutil.ReadFile(configFlag)
	if err != nil {
		logrus.Errorf("读取配置失败: %v", err)
		return
	}
	err = yaml.Unmarshal(data, &cfg)
	if err != nil {
		logrus.Errorf("解析配置失败: %v", err)
		return
	}
	mutex.Lock()
	defer mutex.Unlock()
	globalConfig = &cfg
}
