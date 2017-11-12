package main

import (
	"time"
)

type Alert struct {
	Labels      map[string]interface{} `json:"labels"`
	Annotations map[string]interface{} `json:"annotations"`
	StartsAt    string                 `json:"startsAt"`
	EndsAt      string                 `json:"endsAt"`
}

type Notification struct {
	Version           string                 `json:"version"`
	GroupKey          string                 `json:"groupKey"`
	Status            string                 `json:"status"`
	Receiver          string                 `json:"receiver"`
	GroupLables       map[string]interface{} `json:"groupLabels"`
	CommonLabels      map[string]interface{} `json:"commonLabels"`
	CommonAnnotations map[string]interface{} `json:"commonAnnotations"`
	ExternalURL       string                 `json:"externalURL"`
	Alerts            []Alert                `json:"alerts"`
}

type NotificationContext struct {
	Alert         *Alert
	Notification  *Notification
	InstanceCount int
	Status        string
	ColorStart    string
	ColorEnd      string
}

func SortAlerts(alerts []Alert) (firing, resolved []Alert) {
	for _, alert := range alerts {
		tStart, _ := time.Parse(time.RFC3339, alert.StartsAt)
		tEnd, _ := time.Parse(time.RFC3339, alert.EndsAt)
		if tEnd.After(tStart) {
			resolved = append(resolved, alert)
		} else {
			firing = append(firing, alert)
		}
	}
	return
}
