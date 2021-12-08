package jobs

import (
	"log"
	"strconv"
	"strings"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/rubrikinc/rubrik-sdk-for-go/rubrikcdm"
)

var SUPPORTED_EVENTS []string = []string{
	"Mssql.LogBackupFailed",
	"Mssql.LogBackupSucceeded",
	"Snapshot.BackupFromLocationFailed",
	"Snapshot.BackupFromLocationSucceeded",
	"Snapshot.BackupFailed",
	"Snapshot.BackupSucceeded",
}

var last_collection_time string

//var oldest_failure time.Time
var first_run bool = true

var (
	// Failed job details
	rubrikBackupJobFailCount = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "rubrik_backup_job_failures",
			Help: "Count of failed Rubrik Backup jobs by object and event.",
		},
		[]string{
			"clusterName",
			"objectName",
			"objectType",
			"backupLocation",
			"eventName",
		},
	)
)

var (
	rubrikEventName = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "rubrik_event_name",
			Help: "Rubrik events encountered",
		},
		[]string{
			"clusterName",
			"objectType",
			"eventName",
			"eventStatus",
		},
	)
)

func init() {
	// failed job details
	prometheus.MustRegister(rubrikBackupJobFailCount)
	prometheus.MustRegister(rubrikEventName)
}

// GetFailedJobs ...
//func GetFailedJobs(rubrik *rubrikcdm.Credentials, clusterName string, objectType string) {
func GetBackupJobStatus(rubrik *rubrikcdm.Credentials, clusterName string) {
	clusterVersion, err := rubrik.ClusterVersion()
	loc, _ := time.LoadLocation("UTC")

	if last_collection_time == "" {
		last_collection_time = time.Now().In(loc).AddDate(0, 0, -1).Format("2006-01-02T15:04:05.000Z")
	}

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
	if (clusterMajorVersion == 5 && clusterMinorVersion < 2) || clusterMajorVersion < 5 {
		//Unsupported
		log.Printf("Unsupported version")
	} else { // cluster version is 5.2 or newer
		//eventData, err := rubrik.Get("v1", "/event/latest?limit=9999&event_status=Failure&event_type=Backup&object_type="+objectType+"&before_date="+yesterday, 60)
		var beforeCall = time.Now().In(loc).Format("2006-01-02T15:04:05.000Z")
		//log.Printf("Looking for events from " + last_collection_time + " until " + beforeCall)

		var restcall string
		if first_run == true {
			//Look for failures in the past 24 hours
			//log.Printf("FIRST RUN")
			restcall = "/event/latest?limit=9999&event_type=Backup&event_status=Failure&order_by_time=asc&before_date=" + last_collection_time
			//eventData, err := rubrik.Get("v1", "/event/latest?limit=9999&event_type=Backup&event_status=Failure&order_by_time=asc&before_date="+last_collection_time, 60)
		} else {
			//Look for all events since the last run time
			//eventData, err := rubrik.Get("v1", "/event/latest?limit=9999&event_type=Backup&order_by_time=asc&before_date="+last_collection_time, 60)
			restcall = "/event/latest?limit=9999&event_type=Backup&order_by_time=asc&before_date=" + last_collection_time
		}

		//eventData, err := rubrik.Get("v1", "/event/latest?limit=9999&event_type=Backup&event_status=Failure&order_by_time=asc&before_date="+last_collection_time, 60)
		//eventData, err := rubrik.Get("v1", "/event/latest?limit=9999&event_type=Backup&order_by_time=asc&before_date="+last_collection_time, 60)
		eventData, err := rubrik.Get("v1", restcall, 60)
		if err != nil {
			log.Printf("Error from jobs.GetFailedJobs: ", err)
			return
		}
		last_collection_time = beforeCall

		if eventData != nil || eventData.(map[string]interface{})["data"] != nil {
			for _, v := range eventData.(map[string]interface{})["data"].([]interface{}) {
				thisEventSeriesID := v.(map[string]interface{})["latestEvent"].(map[string]interface{})["eventSeriesId"]
				thisEventStatus := v.(map[string]interface{})["latestEvent"].(map[string]interface{})["eventStatus"]
				thisObjectName := v.(map[string]interface{})["latestEvent"].(map[string]interface{})["objectName"]

				//thisObjectID := v.(map[string]interface{})["latestEvent"].(map[string]interface{})["objectId"]
				//thisEventDate := v.(map[string]interface{})["latestEvent"].(map[string]interface{})["time"].(string)

				hasActiveFailedEvent := false
				switch eventStatus := thisEventStatus; eventStatus {
				//Classify as Success
				case "Success":
					//Not a considered a failure event, obv
					hasActiveFailedEvent = false
				//Classify as Failure
				case "Failure", "Canceled", "Canceling":
					hasActiveFailedEvent = true
				//Classify as NOOP
				case "Running", "Queued":
					//Not a considered a failure event
					//log.Printf("Skipping running event")
					hasActiveFailedEvent = false
				//Suspected to be not needed/relevant
				case "TaskSuccess", "Info", "Warning":
					hasActiveFailedEvent = false
				default:
					log.Printf("Skipping unknown event - " + eventStatus.(string))
					hasActiveFailedEvent = false
				}

				var thisObjectType string
				if v.(map[string]interface{})["latestEvent"].(map[string]interface{})["objectType"] == nil {
					thisObjectType = "null"
				} else {
					thisObjectType = v.(map[string]interface{})["latestEvent"].(map[string]interface{})["objectType"].(string)
				}

				var thisEventName string
				//var sanitizedEventName string
				if v.(map[string]interface{})["latestEvent"].(map[string]interface{})["eventName"] == nil {
					thisEventName = "null"
				} else {
					thisEventName = v.(map[string]interface{})["latestEvent"].(map[string]interface{})["eventName"].(string)

				}

				rubrikEventName.WithLabelValues(
					clusterName,
					thisObjectType,
					thisEventName,
					thisEventStatus.(string)).Add(1)

				if hasActiveFailedEvent == false {
					continue
				}

				//If not a "supported" event name, continue to next event, do not store metric
				if stringInSlice(thisEventName, SUPPORTED_EVENTS) == false {
					continue
				}

				//By being below the SUPPORTED_EVENTS check, unnecessary calls to API ar avoided
				eventSeriesData, err := rubrik.Get("v1", "/event_series/"+thisEventSeriesID.(string), 60)
				if err != nil {
					log.Printf("Error from jobs.GetFailedJobs: ", err)
					return
				}

				var thisLocation string
				if v.(map[string]interface{})["latestEvent"].(map[string]interface{})["location"] == nil {
					thisLocation = "null"
				} else {
					thisLocation = eventSeriesData.(map[string]interface{})["location"].(string)
				}

				sanitizedEventName := cleanEventName(thisEventName)

				// if first_run == true {
				// 	//Check for successes after the above failure
				// 	var beforeDate = v.(map[string]interface{})["latestEvent"].(map[string]interface{})["time"].(string)
				// 	var objectType = v.(map[string]interface{})["latestEvent"].(map[string]interface{})["objectType"].(string)
				// 	var objectName = v.(map[string]interface{})["latestEvent"].(map[string]interface{})["objectName"].(string)
				// 	//var eventType = v.(map[string]interface{})["latestEvent"].(map[string]interface{})["eventType"].(string)
				// 	var objectID = v.(map[string]interface{})["latestEvent"].(map[string]interface{})["objectId"].(string)

				// 	log.Printf("Looking for success")
				// 	successData, err := rubrik.Get("v1", "/event/latest?limit=9999&event_type=Backup&event_status=Success&object_ids="+objectID+"&object_type="+objectType+"&object_name="+objectName+"&before_date="+beforeDate, 60)
				// 	if err != nil {
				// 		log.Printf("Error from jobs.GetFailedJobs.SuccessCheck: ", err)
				// 		return
				// 	}

				// 	if successData.(map[string]interface{})["data"] != nil {
				// 		for _, z := range successData.(map[string]interface{})["data"].([]interface{}) {
				// 			thisSuccessStatus := z.(map[string]interface{})["latestEvent"].(map[string]interface{})["eventStatus"]
				// 			thisSuccessEventName := z.(map[string]interface{})["latestEvent"].(map[string]interface{})["eventName"].(string)
				// 			thisSuccessEventName = cleanEventName(thisSuccessEventName)

				// 			if thisSuccessStatus == "Success" {
				// 				if thisSuccessEventName == sanitizedEventName {
				// 					log.Printf("Found a success for " + thisSuccessEventName + " for " + objectID)
				// 					hasActiveFailedEvent = false
				// 					break
				// 				}
				// 			}
				// 		}
				// 	}

				// }

				//Debugging/Validation
				// var status string
				// if hasActiveFailedEvent == true {
				// 	status = "Active Failure or In Progress"
				// } else {
				// 	status = "Success"
				// }
				// log.Printf(thisEventDate + " - " + sanitizedEventName + " - " + thisLocation + " -- " + status + "(" + thisEventName + ") -- " + thisEventSeriesID.(string))

				if hasActiveFailedEvent == true {
					rubrikBackupJobFailCount.WithLabelValues(
						clusterName,
						thisObjectName.(string),
						thisObjectType,
						thisLocation,
						sanitizedEventName).Add(1)
				} else {
					// rubrikBackupJobStatus.WithLabelValues(
					// 	clusterName,
					// 	thisObjectName.(string),
					// 	//thisObjectID.(string),
					// 	thisObjectType,
					// 	//thisEventSeverity,
					// 	thisLocation,
					// 	sanitizedEventName).Set(0)
				}
				//}

				//}
			}
		}
	}
	//End "first run" status
	first_run = false
}

func cleanEventName(eventName string) string {

	//var orig string = eventName
	eventName = strings.ReplaceAll(eventName, "Failed", "")
	eventName = strings.ReplaceAll(eventName, "Succeeded", "")
	// eventName = strings.ReplaceAll(eventName, "Canceled", "")
	// eventName = strings.ReplaceAll(eventName, "Started", "")
	// eventName = strings.ReplaceAll(eventName, "Begin", "")
	// eventName = strings.ReplaceAll(eventName, "End", "")
	// eventName = strings.ReplaceAll(eventName, "Throttling", "")
	// eventName = strings.ReplaceAll(eventName, "Initializing", "")

	// if eventName != orig {
	// 	log.Printf(orig + "->" + eventName)
	// }
	return eventName
}

func stringInSlice(a string, list []string) bool {
	for _, b := range list {
		if b == a {
			return true
		}
	}
	return false
}
