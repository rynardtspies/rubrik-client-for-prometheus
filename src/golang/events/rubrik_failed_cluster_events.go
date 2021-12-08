package events

import (
	"log"
	"strconv"
	"strings"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/rubrikinc/rubrik-sdk-for-go/rubrikcdm"
)

var last_collection_time string
var first_run bool = true

var (
	// Failed Event details
	rubrikFailedClusterEvent = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "rubrik_failed_cluster_event",
			Help: "Information for failed Rubrik cluster Events.",
		},
		[]string{
			"clusterName",
			"objectName",
			//"objectID",
			"objectType",
			//"eventSeverity",
			"eventName",
			//"eventDate",
		},
	)
)

func init() {
	// failed Event details
	prometheus.MustRegister(rubrikFailedClusterEvent)
}

// GetFailedClusterEvents ...
//func GetFailedEvents(rubrik *rubrikcdm.Credentials, clusterName string, objectType string) {
func GetFailedClusterEvents(rubrik *rubrikcdm.Credentials, clusterName string) {
	clusterVersion, err := rubrik.ClusterVersion()

	loc, _ := time.LoadLocation("UTC")

	if last_collection_time == "" {
		last_collection_time = time.Now().In(loc).AddDate(0, 0, -1).Format("2006-01-02T15:04:05.000Z")
	}

	if err != nil {
		log.Printf("Error from events.GetFailedEvents: ", err)
		return
	}
	clusterMajorVersion, err := strconv.ParseInt(strings.Split(clusterVersion, ".")[0], 10, 64)
	if err != nil {
		log.Printf("Error from events.GetFailedEvents: ", err)
		return
	}
	clusterMinorVersion, err := strconv.ParseInt(strings.Split(clusterVersion, ".")[1], 10, 64)
	if err != nil {
		log.Printf("Error from events.GetFailedEvents: ", err)
		return
	}
	if (clusterMajorVersion == 5 && clusterMinorVersion < 2) || clusterMajorVersion < 5 { // cluster version is older than 5.1
		//eventData, err := rubrik.Get("internal", "/event_series?status=Failure&event_type=Backup&object_type="+objectType, 60)
		//Unsupported
	} else { // cluster version is 5.2 or newer
		//var yesterday = time.Now().AddDate(0, 0, -1).Format("2006-01-02T15:04:05.000Z")
		//eventData, err := rubrik.Get("v1", "/event/latest?limit=9999&event_status=Failure&event_type=Backup&object_type="+objectType+"&before_date="+yesterday, 60)

		var beforeCall = time.Now().In(loc).Format("2006-01-02T15:04:05.000Z")
		eventData, err := rubrik.Get("v1", "/event/latest?limit=9999&object_type=Cluster&event_status=Failure&before_date="+last_collection_time, 60)
		if err != nil {
			log.Printf("Error from Events.GetFailedClusterEvents: ", err)
			return
		}

		last_collection_time = beforeCall

		if eventData != nil || eventData.(map[string]interface{})["data"] != nil {
			for _, v := range eventData.(map[string]interface{})["data"].([]interface{}) {
				thisEventSeriesID := v.(map[string]interface{})["latestEvent"].(map[string]interface{})["eventSeriesId"]
				eventSeriesData, err := rubrik.Get("v1", "/event_series/"+thisEventSeriesID.(string), 60)
				if err != nil {
					log.Printf("Error from Events.GetFailedClusterEvents: ", err)
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
					// var thisObjectID string
					// if eventSeriesData.(map[string]interface{})["objectId"] == nil {
					// 	thisObjectID = "null"
					// } else {
					// 	thisObjectID = eventSeriesData.(map[string]interface{})["objectId"].(string)
					// }
					//thisLocation := eventSeriesData.(map[string]interface{})["location"]
					var thisObjectType string
					if eventSeriesData.(map[string]interface{})["objectType"] == nil {
						thisObjectType = "null"
					} else {
						thisObjectType = eventSeriesData.(map[string]interface{})["objectType"].(string)
					}
					// var thisEventSeverity string
					// if v.(map[string]interface{})["latestEvent"].(map[string]interface{})["eventSeverity"] == nil {
					// 	thisEventSeverity = "null"
					// } else {
					// 	thisEventSeverity = v.(map[string]interface{})["latestEvent"].(map[string]interface{})["eventSeverity"].(string)
					// }

					var thisEventName string
					if v.(map[string]interface{})["latestEvent"].(map[string]interface{})["eventName"] == nil {
						thisEventName = "null"
					} else {
						thisEventName = v.(map[string]interface{})["latestEvent"].(map[string]interface{})["eventName"].(string)
					}

					// var thisEventDate string
					// if v.(map[string]interface{})["latestEvent"].(map[string]interface{})["time"] == nil {
					// 	thisEventDate = "null"
					// } else {
					// 	thisEventDate = v.(map[string]interface{})["latestEvent"].(map[string]interface{})["time"].(string)
					// }
					rubrikFailedClusterEvent.WithLabelValues(
						clusterName,
						thisObjectName,
						//thisObjectID,
						thisObjectType,
						//thisEventSeverity,
						thisEventName).Add(1)
				}
			}
		}
	}
}
