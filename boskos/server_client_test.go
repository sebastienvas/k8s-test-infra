package main

import (
	"fmt"
	"reflect"
	"testing"
	"time"

	"net/http/httptest"

	"k8s.io/test-infra/boskos/client"
	"k8s.io/test-infra/boskos/common"
	"k8s.io/test-infra/boskos/ranch"
	"sort"
)

// json does not serialized time with nanosecond precision
func now() time.Time {
	format := "2006-01-02 15:04:05.000"
	now, _ := time.Parse(format, time.Now().Format(format))
	return now
}

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
		userData := common.UserData{"test": "new"}

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
		if !reflect.DeepEqual(updatedResource.UserData, userData) {
			t.Errorf("info should match. Expected \n%v, received \n%v", userData, updatedResource.UserData)
		}
	}
}

func TestAcquireByState(t *testing.T) {
	newState := "newState"
	owner := "owner"
	fakeNow := now()
	updateTime := func() time.Time { return fakeNow }
	var testcases = []struct {
		name, state         string
		resources, expected []common.Resource
		err                 error
	}{
		{
			name:  "noState",
			state: "state1",
			resources: []common.Resource{
				common.NewResource("test", "type", common.Dirty, "", time.Time{}),
			},
			err: fmt.Errorf("resource not found"),
		},
		{
			name:  "existing",
			state: "state1",
			resources: []common.Resource{
				common.NewResource("test1", "type1", common.Dirty, "", time.Time{}),
				common.NewResource("test2", "type2", "state1", "", time.Time{}),
				common.NewResource("test3", "type3", "state1", "", time.Time{}),
				common.NewResource("test4", "type4", common.Dirty, "", time.Time{}),
			},
			expected: []common.Resource{
				common.NewResource("test2", "type2", newState, owner, fakeNow),
				common.NewResource("test3", "type3", newState, owner, fakeNow),
			},
		},
		{
			name:  "alreadyOwned",
			state: "state1",
			resources: []common.Resource{
				common.NewResource("test1", "type1", common.Dirty, "", time.Time{}),
				common.NewResource("test2", "type2", "state1", "foo", time.Time{}),
				common.NewResource("test3", "type3", "state1", "foo", time.Time{}),
				common.NewResource("test4", "type4", common.Dirty, "", time.Time{}),
			},
			err: fmt.Errorf("resources already used by another user"),
		},
	}
	fmt.Println("test")
	for _, tc := range testcases {
		r := MakeTestRanch(tc.resources)
		r.UpdateTime = updateTime
		boskos := makeTestBoskos(r)
		client := client.NewClient(owner, boskos.URL)

		receivedRes, err := client.AcquireByState(tc.state, newState)
		boskos.Close()
		if !reflect.DeepEqual(err, tc.err) {
			t.Errorf("tc: %s - errors don't match, expected %v, received\n %v", tc.name, err, tc.err)
			continue
		}
		sort.Sort(common.ResourceByName(receivedRes))
		if !reflect.DeepEqual(receivedRes, tc.expected) {
			t.Errorf("tc: %s - resources should match. Expected \n%v, received \n%v", tc.name, tc.expected[1].UserData, receivedRes[1].UserData)
		}
	}
}
