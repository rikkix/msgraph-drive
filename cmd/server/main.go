package main

import (
	"flag"
	"fmt"
	"html/template"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"time"

	"gopkg.in/yaml.v3"

	drive "github.com/iochen/msgraph-drive"
)

type Config struct {
	TenantID      string `yaml:"tenant"`
	ApplicationID string `yaml:"application"`
	ClientSecret  string `yaml:"secret"`
	DriveID       string `yaml:"drive"`
	View          string `yaml:"view"`
	Listen        string `yaml:"listen"`
}

type DrvSrv struct {
	Drive *drive.Drive
	Tpl   *template.Template
}

type DataItem struct {
	Name         string
	ReadableSize string
	Date         time.Time
	ReadableDate string
	IsFolder     bool
}

type Data struct {
	Parent  string
	Current string
	Items   []DataItem
}

func main() {
	confPath := flag.String("conf", "config.yaml", "config file")
	flag.Parse()
	if len(os.Args) > 1 {
		if os.Args[1] == "new" {
			confFile, err := genConfig(&Config{Listen: ":8086"})
			if err != nil {
				log.Fatalln(err)
			}
			if err := ioutil.WriteFile(*confPath, confFile, 0644); err != nil {
				log.Fatalln(err)
			}
			return
		}
	}
	confFile, err := ioutil.ReadFile(*confPath)
	if err != nil {
		log.Fatalln(err)
	}
	conf, err := loadConfig(confFile)
	if err != nil {
		log.Fatalln(err)
	}
	cli, err := drive.NewGraphClient(conf.TenantID, conf.ApplicationID, conf.ClientSecret)
	if err != nil {
		log.Fatalln(err)
	}
	drvH := NewDrvHandler(cli.GetDrive(conf.DriveID))
	drvH.Tpl, err = template.New("index.html").ParseFiles(conf.View)
	http.Handle("/", drvH)
	go func() {
		exitCh := make(chan os.Signal)
		signal.Notify(exitCh, os.Kill, os.Interrupt)
		for range exitCh {
			fmt.Println("Bye!")
			os.Exit(0)
		}
	}()
	err = http.ListenAndServe(conf.Listen, nil)
	if err != nil {
		log.Fatalln(err)
	}
}

func genConfig(conf *Config) ([]byte, error) {
	return yaml.Marshal(conf)
}

func loadConfig(c []byte) (*Config, error) {
	conf := &Config{}
	err := yaml.Unmarshal(c, conf)
	return conf, err
}

func NewDrvHandler(drv *drive.Drive) *DrvSrv {
	return &DrvSrv{
		Drive: drv,
	}
}

func (ds *DrvSrv) ServeHTTP(resp http.ResponseWriter, req *http.Request) {
	path := req.URL.Path
	items, err := ds.Drive.ListChildren(path)
	if err != nil {
		switch err.(type) {
		case *drive.ReqError:
			if err.(*drive.ReqError).Err.Code == "itemNotFound" {
				resp.WriteHeader(404)
				resp.Write([]byte("Item Not Found."))
				return
			}
		}
		resp.WriteHeader(500)
		resp.Write([]byte("Server Error"))
		return
	}

	if len(items) == 0 {
		item, err := ds.Drive.Item(path)
		if err != nil {
			resp.Write([]byte(err.Error()))
			return
		}
		http.Redirect(resp, req, item.DownloadURL, http.StatusTemporaryRedirect)
		return
	}

	parent := path
	if strings.HasSuffix(parent, "/") {
		parent = parent[:len(parent)-1]
	}
	parent = filepath.Dir(parent)
	data := Data{
		Parent:  parent,
		Current: path,
		Items:   []DataItem{},
	}
	for i := range items {
		data.Items = append(data.Items, DataItem{
			Name:         items[i].Name,
			ReadableSize: size2readable(items[i].Size),
			Date:         items[i].LastMod,
			ReadableDate: date2readable(items[i].LastMod),
			IsFolder:     items[i].IsFolder(),
		})
	}
	err = ds.Tpl.Execute(resp, data)
	if err != nil {
		log.Println(err)
	}
}

func date2readable(date time.Time) string {
	sub := time.Now().Sub(date)
	hours := sub.Hours()
	minutes := sub.Minutes()
	switch {
	case hours < 1:
		switch {
		case minutes < 1:
			return "recently"
		default:
			return fmt.Sprintf("%.0f minute(s) ago", minutes)
		}
	case hours < 24:
		return fmt.Sprintf("%.0f hour(s) ago", hours)
	default:
		return fmt.Sprintf("%.0f day(s) ago", hours/24)
	}
}

func size2readable(size int64) string {
	const unit = 1024
	if size < unit {
		return fmt.Sprintf("%d B", size)
	}
	div, exp := int64(unit), 0
	for n := size / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %ciB",
		float64(size)/float64(div), "KMGTPE"[exp])
}
