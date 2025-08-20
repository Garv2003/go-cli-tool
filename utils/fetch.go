package utils

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sync"

	"github.com/garv2003/cli_tool/internals/models"
)

type Instance struct {
	InstanceList []models.InstanceItem
	Build        string
}

func FetchEnv(env string, urlPattern string) Instance {
	url := fmt.Sprintf(urlPattern, env)
	res, err := http.Get(url)
	if err != nil {
		return Instance{}
	}
	defer res.Body.Close()

	var raw map[string][]models.ResponseInstanceItem
	err = json.NewDecoder(res.Body).Decode(&raw)
	if err != nil {
		return Instance{}
	}

	var instances []models.InstanceItem
	var build string

	for name, items := range raw {
		for _, item := range items {
			build = item.Build
			instances = append(instances, models.InstanceItem{
				Name:     name,
				HostName: item.HostName,
				State:    item.State,
			})
		}
	}

	return Instance{
		InstanceList: instances,
		Build:        build,
	}
}

func FetchMultiple(envs []string, urlPattern string) map[string]Instance {
	var wg sync.WaitGroup
	result := make(map[string]Instance)
	lock := sync.Mutex{}

	for _, env := range envs {
		if env == "all" {
			continue
		}
		wg.Add(1)
		go func(e string) {
			defer wg.Done()
			data := FetchEnv(e, urlPattern)
			lock.Lock()
			result[e] = data
			lock.Unlock()
		}(env)
	}
	wg.Wait()
	return result
}
