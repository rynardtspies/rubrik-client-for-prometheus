package jobs

import (
	"log"
	"strconv"
	"strings"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/rubrikinc/rubrik-sdk-for-go/rubrikcdm"
)

var (
	// Failed job details
	rubrikFailedConfigJob = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "rubrik_failed_configuration_job",
			Help: "Information for failed Rubrik configuration jobs.",
		},
		[]string{
			"clusterName",
			"objectName",
			"objectID",
			"objectType",
			"eventSeverity",
			"eventDate",
		},
	)
)

func init() {
	// failed job details
	prometheus.MustRegister(rubrikFailedConfigJob)
}

// GetFailedJobs ...
//func GetFailedJobs(rubrik *rubrikcdm.Credentials, clusterName string, objectType string) {
func GetFailedConfigJobs(rubrik *rubrikcdm.Credentials, clusterName string) {
	clusterVersion, err := rubrik.ClusterVersion()

	if err != nil {
		log.Printf("Error from jobs.GetFailedJobs: ", err)
		return
	}
	clusterMajorVersion, err := strconv.ParseInt(strings.Split(clusterVersion, ".")[0], 10, 64)
	if err != nil {
		log.Printf("Error from jobs.GetFailedJobs: ", err)
		return
	}
	clusterMinorVersion, err := strconv.ParseInt(strings.Split(clusterVersion, ".")[1], 10, 64)
	if err != nil {
		log.Printf("Error from jobs.GetFailedJobs: ", err)
		return
	}
	if (clusterMajorVersion == 5 && clusterMinorVersion < 2) || clusterMajorVersion < 5 { // cluster version is older than 5.1
		//eventData, err := rubrik.Get("internal", "/event_series?status=Failure&event_type=Backup&object_type="+objectType, 60)
		eventData, err := rubrik.Get("internal", "/event_series?status=Failure&event_type=Configuration", 60)
		if err != nil {
			log.Printf("Error from jobs.GetFailedJobs: ", err)
			return
		}
		if eventData != nil || eventData.(map[string]interface{})["data"] != nil {
			for _, v := range eventData.(map[string]interface{})["data"].([]interface{}) {
				thisEventSeriesID := v.(map[string]interface{})["eventSeriesId"]
				eventSeriesData, err := rubrik.Get("internal", "/event_series/"+thisEventSeriesID.(string), 60)
				if err != nil {
					log.Printf("Error from jobs.GetFailedJobs: ", err)
					return
				}
				hasFailedEvent := false
				for _, w := range eventSeriesData.(map[string]interface{})["eventDetailList"].([]interface{}) {
					thisEventStatus := w.(map[string]interface{})["status"]
					if thisEventStatus == "Failure" {
						hasFailedEvent = true
					}
				}
				if hasFailedEvent == true {
					thisObjectName := v.(map[string]interface{})["objectInfo"].(map[string]interface{})["objectName"]
					thisObjectID := v.(map[string]interface{})["objectInfo"].(map[string]interface{})["objectId"]
					//thisLocation := v.(map[string]interface{})["location"]
					var thisObjectType string
					if eventSeriesData.(map[string]interface{})["objectType"] == nil {
						thisObjectType = "null"
					} else {
						thisObjectType = eventSeriesData.(map[string]interface{})["objectType"].(string)
					}
					var thisEventSeverity string
					if eventSeriesData.(map[string]interface{})["eventSeverity"] == nil {
						thisEventSeverity = "null"
					} else {
						thisEventSeverity = eventSeriesData.(map[string]interface{})["eventSeverity"].(string)
					}
					thisEventDate := v.(map[string]interface{})["time"]
					rubrikFailedConfigJob.WithLabelValues(
						clusterName,
						thisObjectName.(string),
						thisObjectID.(string),
						thisObjectType,
						thisEventSeverity,
						//thisLocation.(string),
						//thisStartTime,
						//thisEndTime,
						//thisLogicalSize,
						//thisDuration,
						thisEventDate.(string)).Set(1)
				}
			}
		}
	} else { // cluster version is 5.2 or newer
		var yesterday = time.Now().AddDate(0, 0, -1).Format("2006-01-02T15:04:05.000Z")
		//eventData, err := rubrik.Get("v1", "/event/latest?limit=9999&event_status=Failure&event_type=Backup&object_type="+objectType+"&before_date="+yesterday, 60)
		eventData, err := rubrik.Get("v1", "/event/latest?limit=9999&event_type=Configuration&event_status=Failure&before_date="+yesterday, 60)
		if err != nil {
			log.Printf("Error from jobs.GetFailedJobs: ", err)
			return
		}
		if eventData != nil || eventData.(map[string]interface{})["data"] != nil {
			for _, v := range eventData.(map[string]interface{})["data"].([]interface{}) {
				thisEventSeriesID := v.(map[string]interface{})["latestEvent"].(map[string]interface{})["eventSeriesId"]
				eventSeriesData, err := rubrik.Get("v1", "/event_series/"+thisEventSeriesID.(string), 60)
				if err != nil {
					log.Printf("Error from jobs.GetFailedJobs: ", err)
					return
				}
				hasFailedEvent := false
				for _, w := range eventSeriesData.(map[string]interface{})["eventDetailList"].([]interface{}) {
					thisEventStatus := w.(map[string]interface{})["eventStatus"]
					if thisEventStatus == "Failure" {
						hasFailedEvent = true
					}
				}
				if hasFailedEvent == true {
					var thisObjectName string
					if eventSeriesData.(map[string]interface{})["objectName"] == nil {
						thisObjectName = "null"
					} else {
						thisObjectName = eventSeriesData.(map[string]interface{})["objectName"].(string)
					}
					//thisObjectID := eventSeriesData.(map[string]interface{})["objectId"]
					var thisObjectID string
					if eventSeriesData.(map[string]interface{})["objectId"] == nil {
						thisObjectID = "null"
					} else {
						thisObjectID = eventSeriesData.(map[string]interface{})["objectId"].(string)
					}
					//thisLocation := eventSeriesData.(map[string]interface{})["location"]
					var thisObjectType string
					if eventSeriesData.(map[string]interface{})["objectType"] == nil {
						thisObjectType = "null"
					} else {
						thisObjectType = eventSeriesData.(map[string]interface{})["objectType"].(string)
					}
					var thisEventSeverity string
					if v.(map[string]interface{})["latestEvent"].(map[string]interface{})["eventSeverity"] == nil {
						thisEventSeverity = "null"
					} else {
						thisEventSeverity = v.(map[string]interface{})["latestEvent"].(map[string]interface{})["eventSeverity"].(string)
					}

					var thisEventDate string
					if v.(map[string]interface{})["latestEvent"].(map[string]interface{})["time"] == nil {
						thisEventDate = "null"
					} else {
						thisEventDate = v.(map[string]interface{})["latestEvent"].(map[string]interface{})["time"].(string)
					}
					rubrikFailedConfigJob.WithLabelValues(
						clusterName,
						thisObjectName,
						thisObjectID,
						thisObjectType,
						thisEventSeverity,
						//thisLocation.(string),
						//thisStartTime,
						//thisEndTime,
						//thisLogicalSize,
						//thisDuration,
						thisEventDate).Set(1)
				}
			}
		}
	}
}
