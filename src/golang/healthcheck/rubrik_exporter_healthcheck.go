package healthcheck

import (
	"log"
	"os"
	"regexp"

	"github.com/rubrikinc/rubrik-sdk-for-go/rubrikcdm"
)

func init() {
	// failed job details

}

// GetFailedJobs ...
//func GetFailedJobs(rubrik *rubrikcdm.Credentials, clusterName string, objectType string) {
func GetHealthCheck(rubrik *rubrikcdm.Credentials, clusterName string) {
	clusterVersion, err := rubrik.ClusterVersion()

	if clusterVersion != "" {
	}
	if err != nil {
		log.Printf("Error from healthcheck.GetHealthCheck: ", err)
		matched, notFound := regexp.MatchString("Incorrect username/password", err.Error())
		if matched {
			if notFound != nil {
			}
			log.Printf("Invalid Username/Password")
			os.Exit(127)

		}
		return
	}

}
