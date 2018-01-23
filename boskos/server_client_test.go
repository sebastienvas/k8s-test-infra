package main

import (
	"net/http/httptest"

	"reflect"
	"testing"

	"k8s.io/test-infra/boskos/common"
	"k8s.io/test-infra/boskos/ranch"
	"k8s.io/test-infra/boskos/client"
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
			name: "noInfo",
			resource: common.Resource{
				Type:  "Type",
				Name:  "test",
				State: "dirty",
			},
		},
		{
			name: "existingInfo",
			resource: common.Resource{
				Type:  "Type",
				Name:  "test",
				State: "dirty",
				UserData: common.ResourceInfo{
					Info: common.TypedContent{
						Type:    "old",
						Content: "old",
					},
				},
			},
		},
	}
	for _, tc := range testcases {
		r := MakeTestRanch([]common.Resource{tc.resource}, nil)
		boskos := makeTestBoskos(r)
		owner := "owner"
		client := client.NewClient(owner, boskos.URL)
		info := &common.ResourceInfo{
			Info: common.TypedContent{
				Type:    "New",
				Content: "New",
			},
		}

		newState := "acquired"
		receivedRes, err := client.Acquire(tc.resource.Type, tc.resource.State, newState)
		if err != nil {
			t.Error("unable to acquire resource")
			continue
		}
		if receivedRes.State != newState || receivedRes.Owner != owner {
			t.Errorf("resource should match. Expected \n%v, received \n%v", tc.resource, receivedRes)
		}
		if err := client.UpdateOne(receivedRes.Name, receivedRes.State, info); err != nil {
			t.Errorf("unable to update resource. %v", err)
		}
		boskos.Close()
		updatedResource, err := r.Storage.GetResource(tc.resource.Name)
		if err != nil {
			t.Error("unable to list resources")
		}
		if ! reflect.DeepEqual(updatedResource.UserData, *info) {
			t.Errorf("info should match. Expected \n%v, received \n%v", *info, updatedResource.UserData)
		}
	}
}

func TestGetConfig(t *testing.T) {
	var testcases = []struct {
		name, configName     string
		exists bool
		configs []common.ResourceConfig
	}{
		{
			name: "exists",
			exists: true,
			configName: "test",
			configs: []common.ResourceConfig{
				{
					Needs: common.ResourceNeeds{"type1": 1, "type2": 2},
					Config: common.TypedContent{
						Type: "type3",
						Content: "content",
					},
					Name:  "test",

				},
			},
		},
		{
			name: "noConfig",
			exists: false,
			configName: "test",
		},
		{
			name: "existsMultipleConfigs",
			exists: true,
			configName: "test1",
			configs: []common.ResourceConfig{
				{
					Needs: common.ResourceNeeds{"type1": 1, "type2": 2},
					Config: common.TypedContent{
						Type: "type3",
						Content: "content",
					},
					Name:  "test",

				},
				{
					Needs: common.ResourceNeeds{"type1": 1, "type2": 2},
					Config: common.TypedContent{
						Type: "type3",
						Content: "content",
					},
					Name:  "test1",

				},
			},
		},
		{
			name: "noExistMultipleConfigs",
			exists: false,
			configName: "test2",
			configs: []common.ResourceConfig{
				{
					Needs: common.ResourceNeeds{"type1": 1, "type2": 2},
					Config: common.TypedContent{
						Type: "type3",
						Content: "content",
					},
					Name:  "test",

				},
				{
					Needs: common.ResourceNeeds{"type1": 1, "type2": 2},
					Config: common.TypedContent{
						Type: "type3",
						Content: "content",
					},
					Name:  "test1",

				},
			},
		},
	}
	for _, tc := range testcases {
		r := MakeTestRanch(nil, tc.configs)
		boskos := makeTestBoskos(r)
		client := client.NewClient("owner", boskos.URL)
		config, err := client.GetConfig(tc.configName)
		if !tc.exists {
			if err == nil {
				t.Error("client should return an error")
			}
		} else {
			if config.Name != tc.configName {
				t.Error("config name should match")
			}
		}
	}
}