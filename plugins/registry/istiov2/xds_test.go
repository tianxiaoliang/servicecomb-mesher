package pilotv2

import (
	"fmt"
	"testing"
	"time"

	apiv2 "github.com/envoyproxy/go-control-plane/envoy/api/v2"
	// apiv2core "github.com/envoyproxy/go-control-plane/envoy/api/v2/core"
	// apiv2endpoint "github.com/envoyproxy/go-control-plane/envoy/api/v2/endpoint"
	// apiv2route "github.com/envoyproxy/go-control-plane/envoy/api/v2/route"
)

var (
	ValidXdsClient *XdsClient
	TestClusters   []apiv2.Cluster
)

func TestNewXdsClient(t *testing.T) {
	client, err := NewXdsClient(ValidPilotAddr, nil, nodeInfo, KubeConfig)

	if err != nil {
		t.Errorf("Failed to create xds client: %s", err.Error())
	}

	ValidXdsClient = client
}

func TestCDS(t *testing.T) {
	clusters, err := ValidXdsClient.CDS()
	if err != nil {
		t.Errorf("Failed to get clusters by CDS: %s", err.Error())
	}

	fmt.Printf("Got %d clusters\n", len(clusters))
	TestClusters = clusters
}

func TestEDS(t *testing.T) {
	if len(TestClusters) == 0 { // With istio, there should always be clusters
		t.Errorf("No clusters found")
	}

	loadAssignment, err := ValidXdsClient.EDS(TestClusters[0].Name)
	if err != nil {
		t.Errorf("Failed to get endpoints by EDS: %s", err.Error())
	}

	if loadAssignment == nil {
		t.Errorf("Failed to get load assginment with EDS: %s", err.Error())
	}
}

func TestRDS(t *testing.T) {
	targetClusterName := ""
	for _, c := range TestClusters {
		info := ParseClusterName(c.Name)
		if info != nil {
			targetClusterName = c.Name
			break
		}
	}

	if targetClusterName == "" {
		fmt.Println("We don't find a valid cluster")
	}

	_, err := ValidXdsClient.RDS(targetClusterName)
	if err != nil {
		t.Errorf("Failed to get routers: %s", err.Error())
	}
}

func TestLDS(t *testing.T) {
	listeners, err := ValidXdsClient.LDS()
	if err != nil {
		t.Errorf("Failed to get listeners with LDS: %s", err.Error())
	}

	fmt.Printf("%d listeners found\n", len(listeners))
}

func TestNonce(t *testing.T) {
	nowStr := time.Now().String()
	ValidXdsClient.setNonce(TypeCds, nowStr)
	ValidXdsClient.setNonce(TypeEds, nowStr)
	ValidXdsClient.setNonce(TypeRds, nowStr)
	ValidXdsClient.setNonce(TypeLds, nowStr)

	cdsNonce := ValidXdsClient.getNonce(TypeCds)
	if cdsNonce != nowStr {
		t.Errorf("Failed to test nonce: %s should be equal to %s", cdsNonce, nowStr)
	}

	edsNonce := ValidXdsClient.getNonce(TypeEds)
	if edsNonce != nowStr {
		t.Errorf("Failed to test nonce: %s should be equal to %s", edsNonce, nowStr)
	}

	ldsNonce := ValidXdsClient.getNonce(TypeLds)
	if ldsNonce != nowStr {
		t.Errorf("Failed to test nonce: %s should be equal to %s", ldsNonce, nowStr)
	}

	rdsNonce := ValidXdsClient.getNonce(TypeRds)
	if rdsNonce != nowStr {
		t.Errorf("Failed to test nonce: %s should be equal to %s", rdsNonce, nowStr)
	}
}

func TestVersionInfo(t *testing.T) {
	nowStr := time.Now().String()
	ValidXdsClient.setVersionInfo(TypeCds, nowStr)
	ValidXdsClient.setVersionInfo(TypeEds, nowStr)
	ValidXdsClient.setVersionInfo(TypeRds, nowStr)
	ValidXdsClient.setVersionInfo(TypeLds, nowStr)

	cdsVersionInfo := ValidXdsClient.getVersionInfo(TypeCds)
	if cdsVersionInfo != nowStr {
		t.Errorf("Failed to test VersionInfo: %s should be equal to %s", cdsVersionInfo, nowStr)
	}

	edsVersionInfo := ValidXdsClient.getVersionInfo(TypeEds)
	if edsVersionInfo != nowStr {
		t.Errorf("Failed to test VersionInfo: %s should be equal to %s", edsVersionInfo, nowStr)
	}

	ldsVersionInfo := ValidXdsClient.getVersionInfo(TypeLds)
	if ldsVersionInfo != nowStr {
		t.Errorf("Failed to test VersionInfo: %s should be equal to %s", ldsVersionInfo, nowStr)
	}

	rdsVersionInfo := ValidXdsClient.getVersionInfo(TypeRds)
	if rdsVersionInfo != nowStr {
		t.Errorf("Failed to test VersionInfo: %s should be equal to %s", rdsVersionInfo, nowStr)
	}
}

func TestGetSubsetTags(t *testing.T) {
	var targetClusterInfo *XdsClusterInfo = nil
	for _, c := range TestClusters {
		if info := ParseClusterName(c.Name); info != nil && info.Subset != "" {
			targetClusterInfo = info
			break
		}
	}

	if targetClusterInfo == nil {
		fmt.Println("No tagged services in test environment, skip")
	} else {
		tags, err := ValidXdsClient.GetSubsetTags(targetClusterInfo.Namespace, targetClusterInfo.ServiceName, targetClusterInfo.Subset)
		if err != nil {
			t.Errorf("Failed to get subset tags: %s", err.Error())
		} else if len(tags) == 0 {
			t.Errorf("Should not return empty tags")
		}
	}
}

func TestGetAdsResClient(t *testing.T) {
	_, err := getAdsResClient(ValidXdsClient)

	if err != nil {
		t.Errorf("Failed to get ads resource client: %s", err.Error())
	}
}

func TestParseClusterName(t *testing.T) {
	validClusterName := "inbound|3030||client.default.svc.cluster.local"

	clusterInfo := ParseClusterName(validClusterName)

	if clusterInfo == nil {
		t.Errorf("Failed to parse cluster name: %s, should return cluster info", validClusterName)
	}
	if clusterInfo.Direction != "inbound" {
		t.Errorf("Failed to parse cluster name: %s, direction should be inbound", validClusterName)
	}
	if clusterInfo.Port != "3030" {
		t.Errorf("Failed to parse cluster name: %s, port should be 3030", validClusterName)
	}
	if clusterInfo.ServiceName != "client" {
		t.Errorf("Failed to parse cluster name: %s, servicename should be client", validClusterName)
	}

	invalidClusterName := "BlackHoleCluster"
	clusterInfo = ParseClusterName(invalidClusterName)
	if clusterInfo != nil {
		t.Errorf("Failed to parse cluster name: %s, should return nil", validClusterName)
	}

	invalidClusterName = "outbound|9080|v2|black"
	clusterInfo = ParseClusterName(invalidClusterName)
	if clusterInfo != nil {
		t.Errorf("Failed to parse cluster name: %s, should return nil", validClusterName)
	}
}
