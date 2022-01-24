package stats

import (
	"log"
	"strconv"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/rubrikinc/rubrik-sdk-for-go/rubrikcdm"
)

var (
	// Windows Volume Group Capacity Stats
	rubrikWindowsVolumeGroupCapacityLocalUsed = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "rubrik_windows_volume_group_local_used_bytes",
			Help: "Local storage consumption for Windows Volume Groups.",
		},
		[]string{
			"clusterName",
			"objectName",
			"objectID",
			"location",
		},
	)
	rubrikWindowsVolumeGroupCapacityArchiveUsed = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "rubrik_windows_volume_group_capacity_archive_used_bytes",
			Help: "Archive storage consumption for Windows Volume Groups.",
		},
		[]string{
			"clusterName",
			"objectName",
			"objectID",
			"location",
		},
	)
)

func init() {
	// Windows Volume Group storage stats
	prometheus.MustRegister(rubrikWindowsVolumeGroupCapacityLocalUsed)
	prometheus.MustRegister(rubrikWindowsVolumeGroupCapacityArchiveUsed)
}

// GetWindowsVolumeGroupapacityStats ...
func GetWindowsVolumeGroupCapacityStats(rubrik *rubrikcdm.Credentials, clusterName string) {
	reportData, err := rubrik.Get("internal", "/report?report_template=ObjectProtectionSummary&report_type=Canned", 60) // get our object protection summary report
	if err != nil {
		log.Printf("Error from stats.GetWindowsVolumeGroupCapacityStats: ", err)
		return
	}
	reports := reportData.(map[string]interface{})["data"].([]interface{})
	reportID := reports[0].(map[string]interface{})["id"]
	body := map[string]interface{}{
		"limit": 100,
		"requestFilters": map[string]interface{}{
			"objectType": "WindowsVolumeGroup",
		},
	}
	for {
		hasMore := true
		tableData, err := rubrik.Post("internal", "/report/"+reportID.(string)+"/table", body, 60) // get our first page of data for the report
		if err != nil {
			log.Printf("Error from stats.GetWindowsVolumeGroupCapacityStats: ", err)
			return
		}
		dataGrid := tableData.(map[string]interface{})["dataGrid"].([]interface{})
		hasMore = tableData.(map[string]interface{})["hasMore"].(bool)
		cursor := tableData.(map[string]interface{})["cursor"]
		columns := tableData.(map[string]interface{})["columns"].([]interface{})
		for _, v := range dataGrid {
			thisObjectID, thisObjectName, thisLocation := "null", "null", "null"
			thisLocalStorage, thisArchiveStorage := 0.0, 0.0
			for i := 0; i < len(columns); i++ {
				switch columns[i] {
				case "ObjectId":
					thisObjectID = v.([]interface{})[i].(string)
				case "ObjectLinkingId":
					thisObjectID = v.([]interface{})[i].(string)
				case "ObjectName":
					thisObjectName = v.([]interface{})[i].(string)
				case "Location":
					thisLocation = v.([]interface{})[i].(string)
				case "LocalStorage":
					thisLocalStorage, _ = strconv.ParseFloat(v.([]interface{})[i].(string), 64)
				case "ArchiveStorage":
					thisArchiveStorage, _ = strconv.ParseFloat(v.([]interface{})[i].(string), 64)
				}
			}
			rubrikWindowsVolumeGroupCapacityLocalUsed.WithLabelValues(
				clusterName,
				thisObjectName,
				thisObjectID,
				thisLocation).Set(thisLocalStorage)
			rubrikWindowsVolumeGroupCapacityArchiveUsed.WithLabelValues(
				clusterName,
				thisObjectName,
				thisObjectID,
				thisLocation).Set(thisArchiveStorage)
		}
		if !hasMore {
			return
		} else {
			body = map[string]interface{}{
				"limit":  1000,
				"cursor": cursor,
				"requestFilters": map[string]interface{}{
					"objectType": "WindowsVolumeGroup",
				},
			}
		}
	}
}
