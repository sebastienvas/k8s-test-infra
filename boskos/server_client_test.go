package main

import (
	"reflect"
	"testing"
	"time"

	"net/http/httptest"

	"k8s.io/test-infra/boskos/client"
	"k8s.io/test-infra/boskos/common"
	"k8s.io/test-infra/boskos/ranch"
)

func makeTestBoskos(r *ranch.Ranch) *httptest.Server {
	handler := NewBoskosHandler(r)
	return httptest.NewServer(handler)
}

func TestAcquireUpdate(t *testing.T) {
	var testcases = []struct {
		name     string
		resource common.Resource
	}{
		{
			name:     "noInfo",
			resource: common.NewResource("test", "type", common.Dirty, "", time.Time{}),
		},
		{
			name: "existingInfo",
			resource: common.Resource{
				Type:     "type",
				Name:     "test",
				State:    "dirty",
				UserData: common.UserData{"test": "old"},
			},
		},
	}
	for _, tc := range testcases {
		r := MakeTestRanch([]common.Resource{tc.resource})
		boskos := makeTestBoskos(r)
		owner := "owner"
		client := client.NewClient(owner, boskos.URL)
		userData := &common.UserData{"test": "new"}

		newState := "acquired"
		receivedRes, err := client.Acquire(tc.resource.Type, tc.resource.State, newState)
		if err != nil {
			t.Error("unable to acquire resource")
			continue
		}
		if receivedRes.State != newState || receivedRes.Owner != owner {
			t.Errorf("resource should match. Expected \n%v, received \n%v", tc.resource, receivedRes)
		}
		if err := client.UpdateOne(receivedRes.Name, receivedRes.State, userData); err != nil {
			t.Errorf("unable to update resource. %v", err)
		}
		boskos.Close()
		updatedResource, err := r.Storage.GetResource(tc.resource.Name)
		if err != nil {
			t.Error("unable to list resources")
		}
		if !reflect.DeepEqual(updatedResource.UserData, *userData) {
			t.Errorf("info should match. Expected \n%v, received \n%v", *userData, updatedResource.UserData)
		}
	}
}
