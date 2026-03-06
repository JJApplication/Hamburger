/*
Create: 2023/3/6
Project: Niphotri
Github: https://github.com/landers1037
Copyright Renj
*/

package common

import (
	"fmt"
	"os"

	"Hamburger/exp/trojan/log"
	"github.com/AlecAivazis/survey/v2"
	"gopkg.in/yaml.v3"
)

// init server config

type Result struct {
	Name       string `yaml:"name"`
	LocalAddr  string `yaml:"local_addr"`
	LocalPort  int    `yaml:"local_port"`
	RemoteAddr string `yaml:"remote_addr"`
	RemotePort int    `yaml:"remote_port"`
}

type cf struct {
	Result   `yaml:",inline"`
	RunType  string   `yaml:"run_type"`
	Password []string `yaml:"password"`
	SSL      struct {
		Cert string `yaml:"cert" survey:"cert"`
		Key  string `yaml:"key" survey:"key"`
	} `yaml:"ssl"`
}

func Init() {
	var qs = []*survey.Question{
		{
			Name:      "name",
			Prompt:    &survey.Input{Message: "What is config filename?"},
			Validate:  survey.Required,
			Transform: survey.Title,
		},
		{
			Name:   "localAddr",
			Prompt: &survey.Input{Message: "server listen address:"},
		},
		{
			Name:   "localPort",
			Prompt: &survey.Input{Message: "server listen port:"},
		},
		{
			Name:   "remoteAddr",
			Prompt: &survey.Input{Message: "https server address:"},
		},
		{
			Name:   "remotePort",
			Prompt: &survey.Input{Message: "https server port:"},
		},
	}

	var res Result
	var conf cf
	if err := survey.Ask(qs, &res); err != nil {
		log.Error("init error", err.Error())
		return
	}

	conf.Password = []string{""}
	survey.AskOne(&survey.Input{Message: "server password:"}, &conf.Password[0], survey.WithValidator(survey.Required))
	survey.AskOne(&survey.Input{Message: "server ssl certificate:"}, &conf.SSL.Cert, survey.WithValidator(survey.Required))
	survey.AskOne(&survey.Input{Message: "server ssl privatekey:"}, &conf.SSL.Key, survey.WithValidator(survey.Required))
	done := false
	survey.AskOne(&survey.Confirm{Message: "Finish init?", Default: true}, &done)
	if !done {
		log.Warn("initialize paused")
		return
	}

	// merge
	conf.RunType = "server"
	conf.Result = res
	log.Infof("init config: %s.yml", res.Name)
	data, err := yaml.Marshal(conf)
	if err != nil {
		log.Errorf("dump data to yaml error: %s", err.Error())
		return
	}
	if err = os.WriteFile(fmt.Sprintf("%s.yml", res.Name), data, 0644); err != nil {
		log.Errorf("create file error: %s", err.Error())
	}
}
