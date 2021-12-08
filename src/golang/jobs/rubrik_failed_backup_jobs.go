package jobs

import (
	"log"
	"strconv"
	"strings"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/rubrikinc/rubrik-sdk-for-go/rubrikcdm"
)

//var last_collection_time string

var (
	// Failed job details
	rubrikFailedBackupJob = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "rubrik_failed_backup_job",
			Help: "Information for failed Rubrik Backup jobs.",
		},
		[]string{
			"clusterName",
			"objectName",
			"objectID",
			"objectType",
			"eventSeverity",
			"location",
			"eventDate",
			"eventName",
			"eventSeriesID",
		},
	)
)

var (
	// Failed job details
	rubrikFailedBackupJobActive = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "rubrik_failed_backup_job_active",
			Help: "Information if a backup job failure is currently in a failed state.",
		},
		[]string{
			"clusterName",
			"objectName",
			"objectType",
			"objectID",
			//"location",
			//"eventName",
		},
	)
)

func init() {
	// failed job details
	prometheus.MustRegister(rubrikFailedBackupJob)
	prometheus.MustRegister(rubrikFailedBackupJobActive)
}

// GetFailedJobs ...
//func GetFailedJobs(rubrik *rubrikcdm.Credentials, clusterName string, objectType string) {
func GetFailedBackupJobs(rubrik *rubrikcdm.Credentials, clusterName string) {
	clusterVersion, err := rubrik.ClusterVersion()

	//if last_collection_time == "" {}

	//Remove metrics older than 24 hours

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
		eventData, err := rubrik.Get("internal", "/event_series?status=Failure&event_type=Backup", 60)
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
					//thisObjectID := v.(map[string]interface{})["objectInfo"].(map[string]interface{})["objectId"]
					thisLocation := v.(map[string]interface{})["location"]
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
					// var thisStartTime string
					// if v.(map[string]interface{})["startTime"] == nil {
					// 	thisStartTime = "null"
					// } else {
					// 	thisStartTime = v.(map[string]interface{})["startTime"].(string)
					// }
					// var thisEndTime string
					// if v.(map[string]interface{})["endTime"] == nil {
					// 	thisEndTime = "null"
					// } else {
					// 	thisEndTime = v.(map[string]interface{})["endTime"].(string)
					// }
					// var thisLogicalSize string
					// if v.(map[string]interface{})["objectLogicalSize"] == nil {
					// 	thisLogicalSize = "null"
					// } else {
					// 	thisLogicalSize = strconv.FormatFloat(v.(map[string]interface{})["objectLogicalSize"].(float64), 'f', -1, 64)
					// }
					// var thisDuration string
					// if v.(map[string]interface{})["duration"] == nil {
					// 	thisDuration = "null"
					// } else {
					// 	thisDuration = v.(map[string]interface{})["duration"].(string)
					// }
					thisEventDate := v.(map[string]interface{})["eventDate"]
					rubrikFailedBackupJob.WithLabelValues(
						clusterName,
						thisObjectName.(string),
						//thisObjectID.(string),
						thisObjectType,
						thisEventSeverity,
						thisLocation.(string),
						//thisStartTime,
						//thisEndTime,
						//thisLogicalSize,
						//thisDuration,
						thisEventDate.(string)).Set(1)
				}
			}
		}
	} else { // cluster version is 5.2 or newer
		//tzData, err := rubrik.Get("v1", "/cluster/me", 60)
		//timeZone := tzData.(map[string]interface{})["timezone"].(map[string]interface{})["timezone"].(string)
		//tz, err := time.LoadLocation(timeZone)
		//timeLayout := "2006-01-02T15:04:05.000Z"

		var yesterday = time.Now().AddDate(0, 0, -1).Format("2006-01-02T15:04:05.000Z")
		//log.Printf("Looking for errors since " + yesterday)
		//eventData, err := rubrik.Get("v1", "/event/latest?limit=9999&event_status=Failure&event_type=Backup&object_type="+objectType+"&before_date="+yesterday, 60)
		eventData, err := rubrik.Get("v1", "/event/latest?limit=9999&event_type=Backup&event_status=Failure&before_date="+yesterday, 60)
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
					thisObjectName := eventSeriesData.(map[string]interface{})["objectName"]
					thisObjectID := eventSeriesData.(map[string]interface{})["objectId"]
					thisLocation := eventSeriesData.(map[string]interface{})["location"]
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
					var thisEventName string
					if v.(map[string]interface{})["latestEvent"].(map[string]interface{})["eventName"] == nil {
						thisEventName = "null"
					} else {
						thisEventName = v.(map[string]interface{})["latestEvent"].(map[string]interface{})["eventName"].(string)
					}
					//log.Printf("Saving rubrikFailedBackupJob metric")
					rubrikFailedBackupJob.WithLabelValues(
						clusterName,
						thisObjectName.(string),
						thisObjectID.(string),
						thisObjectType,
						thisEventSeverity,
						thisLocation.(string),
						thisEventDate,
						thisEventName,
						thisEventSeriesID.(string)).Set(1)
					//}

					//Check for successes after the above failure
					var beforeDate = v.(map[string]interface{})["latestEvent"].(map[string]interface{})["time"].(string)
					var objectType = v.(map[string]interface{})["latestEvent"].(map[string]interface{})["objectType"].(string)
					var objectName = v.(map[string]interface{})["latestEvent"].(map[string]interface{})["objectName"].(string)
					//var eventType = v.(map[string]interface{})["latestEvent"].(map[string]interface{})["eventType"].(string)
					var objectID = v.(map[string]interface{})["latestEvent"].(map[string]interface{})["objectId"].(string)

					successData, err := rubrik.Get("v1", "/event/latest?limit=9999&event_type=Backup&event_status=Success&object_ids="+objectID+"&object_type="+objectType+"&object_name="+objectName+"&before_date="+beforeDate, 60)
					if err != nil {
						log.Printf("Error from jobs.GetFailedJobs.SuccessCheck: ", err)
						return
					}
					hasSuccessEvent := false
					if successData.(map[string]interface{})["data"] != nil {
						for _, z := range successData.(map[string]interface{})["data"].([]interface{}) {
							thisSuccessStatus := z.(map[string]interface{})["latestEvent"].(map[string]interface{})["eventStatus"]
							if thisSuccessStatus == "Success" {
								hasSuccessEvent = true
							}
						}
					}

					var successStatus float64
					if hasSuccessEvent {
						//found failure
						//log.Printf("Found a success")
						successStatus = 0
					} else {
						//found failure
						//log.Printf("No success after failure")
						successStatus = 1
					}
					//log.Printf("Saving rubrikFailedBackupJobActive metric: " + objectID)
					rubrikFailedBackupJobActive.WithLabelValues(
						clusterName,
						objectName,
						objectType,
						objectID).Set(successStatus)

				}
			}
		}
	}
}
