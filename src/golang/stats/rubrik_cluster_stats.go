package stats

import (
	"log"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/rubrikinc/rubrik-sdk-for-go/rubrikcdm"
)

var (
	// Cluster stats
	rubrikClusterInfo = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "rubrik_cluster_info",
			Help: "Rubrik Cluster info.",
		},
		[]string{
			"clusterName",
			"id",
			"version",
			"apiVersion",
			"timezone",
			"location",
		},
	)
)

func init() {
	// job stats
	prometheus.MustRegister(rubrikClusterInfo)
}

// GetClusterStats ...
func GetClusterStats(rubrik *rubrikcdm.Credentials, clusterName string) {
	clusterData, err := rubrik.Get("v1", "/cluster/me", 60)
	if err != nil {
		log.Printf("Error from stats.GetClusterStats: ", err)
		return
	}
	//cluster := clusterData.(map[string]interface{})["data"].([]interface{})
	id := clusterData.(map[string]interface{})["id"]
	version := clusterData.(map[string]interface{})["version"]
	apiVersion := clusterData.(map[string]interface{})["apiVersion"]
	timezone := clusterData.(map[string]interface{})["timezone"].(map[string]interface{})["timezone"]
	location := clusterData.(map[string]interface{})["geolocation"].(map[string]interface{})["address"]

	rubrikClusterInfo.WithLabelValues(
		clusterName,
		id.(string),
		version.(string),
		apiVersion.(string),
		timezone.(string),
		location.(string)).Set(1)

}
