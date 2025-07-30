package models

type ResponseInstanceItem struct {
	Build    string `json:"build"`
	HostName string `json:"hostName"`
	Id       string `json:"id"`
	State    string `json:"state"`
}

type InstanceItem struct {
	Name     string
	HostName string
	State    string
}
