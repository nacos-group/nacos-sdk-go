package naming_client

import (
	"testing"

	"github.com/nacos-group/nacos-sdk-go/v2/model"
)

func TestWithGroup(t *testing.T) {
	o := &discoverOptions{}
	WithGroup("testGroup")(o)
	if o.group != "testGroup" {
		t.Errorf("WithGroup() failed, expected: %s, got: %s", "testGroup", o.group)
	}
}

func TestWithService(t *testing.T) {
	o := &discoverOptions{}
	WithService("testService")(o)
	if o.service != "testService" {
		t.Errorf("WithService() failed, expected: %s, got: %s", "testService", o.service)
	}
}

func TestWithVersion(t *testing.T) {
	o := &discoverOptions{}
	WithVersion("v1")(o)
	if len(o.versions) != 1 || o.versions[0] != "v1" {
		t.Errorf("WithVersion() failed, expected: %s, got: %v", "v1", o.versions)
	}
}

func TestWithCluster(t *testing.T) {
	o := &discoverOptions{}
	WithCluster("testCluster")(o)
	if len(o.clusters) != 1 || o.clusters[0] != "testCluster" {
		t.Errorf("WithCluster() failed, expected: %s, got: %v", "testCluster", o.clusters)
	}
}

func TestWithChoose(t *testing.T) {
	o := &discoverOptions{}
	choose := func(instances ...model.Instance) model.Instance {
		return instances[0]
	}
	WithChoose(choose)(o)
	if o.choose == nil {
		t.Errorf("WithChoose() failed, expected: not nil, got: nil")
	}
}

func TestCheckMeta(t *testing.T) {
	o := &discoverOptions{
		versions: []string{"v1", "v2"},
	}
	if !o.CheckMeta(map[string]string{"version": "v1"}) {
		t.Errorf("CheckMeta() failed, expected: true, got: false")
	}
	if o.CheckMeta(map[string]string{"version": "v3"}) {
		t.Errorf("CheckMeta() failed, expected: false, got: true")
	}
}

func TestVersionCheck(t *testing.T) {
	o := discoverOptions{
		versions: []string{"v1", "v2"},
	}
	if !o.VersionCheck("v1") {
		t.Errorf("VersionCheck() failed, expected: true, got: false")
	}
	if o.VersionCheck("v3") {
		t.Errorf("VersionCheck() failed, expected: false, got: true")
	}
}

func TestChoose(t *testing.T) {
	o := discoverOptions{}
	instances := []model.Instance{
		{InstanceId: "1", Weight: 1, Metadata: map[string]string{"version": "v1"}},
		{InstanceId: "2", Weight: 2, Metadata: map[string]string{"version": "v2"}},
	}
	selectedInstance := o.Choose(instances...)
	if selectedInstance.InstanceId == "" {
		t.Errorf("Choose() failed, expected: not nil, got: nil")
	}
}
