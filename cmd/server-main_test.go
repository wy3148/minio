/*
 * Minio Cloud Storage, (C) 2016, 2017 Minio, Inc.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package cmd

import (
	"errors"
	"reflect"
	"runtime"
	"testing"
)

func TestGetListenIPs(t *testing.T) {
	testCases := []struct {
		addr       string
		port       string
		shouldPass bool
	}{
		{"127.0.0.1", "9000", true},
		{"", "9000", true},
		{"", "", false},
	}
	for _, test := range testCases {
		var addr string
		// Please keep this we need to do this because
		// of odd https://play.golang.org/p/4dMPtM6Wdd
		// implementation issue.
		if test.port != "" {
			addr = test.addr + ":" + test.port
		}
		hosts, port, err := getListenIPs(addr)
		if !test.shouldPass && err == nil {
			t.Fatalf("Test should fail but succeeded %s", err)
		}
		if test.shouldPass && err != nil {
			t.Fatalf("Test should succeed but failed %s", err)
		}
		if test.shouldPass {
			if port != test.port {
				t.Errorf("Test expected %s, got %s", test.port, port)
			}
			if len(hosts) == 0 {
				t.Errorf("Test unexpected value hosts cannot be empty %#v", test)
			}
		}
	}
}

// Tests get host port.
func TestGetHostPort(t *testing.T) {
	testCases := []struct {
		addr string
		err  error
	}{
		// Test 1 - successful.
		{
			addr: ":" + getFreePort(),
			err:  nil,
		},
		// Test 2 port empty.
		{
			addr: ":0",
			err:  errEmptyPort,
		},
		// Test 3 port empty.
		{
			addr: ":",
			err:  errEmptyPort,
		},
		// Test 4 invalid port.
		{
			addr: "linux:linux",
			err:  errors.New("strconv.Atoi: parsing \"linux\": invalid syntax"),
		},
		// Test 5 port not present.
		{
			addr: "hostname",
			err:  errors.New("address hostname: missing port in address"),
		},
	}

	// Validate all tests.
	for i, testCase := range testCases {
		_, _, err := getHostPort(testCase.addr)
		if err != nil {
			if err.Error() != testCase.err.Error() {
				t.Fatalf("Test %d: Error: %s", i+1, err)
			}
		}
	}
}

// Tests finalize api endpoints.
func TestFinalizeAPIEndpoints(t *testing.T) {
	testCases := []struct {
		addr string
	}{
		{":80"},
		{":80"},
		{"127.0.0.1:80"},
		{"127.0.0.1:80"},
	}

	for i, test := range testCases {
		endPoints, err := finalizeAPIEndpoints(test.addr)
		if err != nil && len(endPoints) <= 0 {
			t.Errorf("Test case %d returned with no API end points for %s",
				i+1, test.addr)
		}
	}
}

// Tests all the expected input disks for function checkSufficientDisks.
func TestCheckSufficientDisks(t *testing.T) {
	var xlDisks []string
	if runtime.GOOS == globalWindowsOSName {
		xlDisks = []string{
			"C:\\mnt\\backend1",
			"C:\\mnt\\backend2",
			"C:\\mnt\\backend3",
			"C:\\mnt\\backend4",
			"C:\\mnt\\backend5",
			"C:\\mnt\\backend6",
			"C:\\mnt\\backend7",
			"C:\\mnt\\backend8",
			"C:\\mnt\\backend9",
			"C:\\mnt\\backend10",
			"C:\\mnt\\backend11",
			"C:\\mnt\\backend12",
			"C:\\mnt\\backend13",
			"C:\\mnt\\backend14",
			"C:\\mnt\\backend15",
			"C:\\mnt\\backend16",
			"C:\\mnt\\backend17",
		}
	} else {
		xlDisks = []string{
			"/mnt/backend1",
			"/mnt/backend2",
			"/mnt/backend3",
			"/mnt/backend4",
			"/mnt/backend5",
			"/mnt/backend6",
			"/mnt/backend7",
			"/mnt/backend8",
			"/mnt/backend9",
			"/mnt/backend10",
			"/mnt/backend11",
			"/mnt/backend12",
			"/mnt/backend13",
			"/mnt/backend14",
			"/mnt/backend15",
			"/mnt/backend16",
			"/mnt/backend17",
		}
	}
	// List of test cases fo sufficient disk verification.
	testCases := []struct {
		disks       []string
		expectedErr error
	}{
		// Even number of disks '6'.
		{
			xlDisks[0:6],
			nil,
		},
		// Even number of disks '12'.
		{
			xlDisks[0:12],
			nil,
		},
		// Even number of disks '16'.
		{
			xlDisks[0:16],
			nil,
		},
		// Larger than maximum number of disks > 16.
		{
			xlDisks,
			errXLMaxDisks,
		},
		// Lesser than minimum number of disks < 6.
		{
			xlDisks[0:3],
			errXLMinDisks,
		},
		// Odd number of disks, not divisible by '2'.
		{
			append(xlDisks[0:10], xlDisks[11]),
			errXLNumDisks,
		},
	}

	// Validates different variations of input disks.
	for i, testCase := range testCases {
		endpoints, err := parseStorageEndpoints(testCase.disks)
		if err != nil {
			t.Fatalf("Unexpected error %s", err)
		}
		if checkSufficientDisks(endpoints) != testCase.expectedErr {
			t.Errorf("Test %d expected to pass for disks %s", i+1, testCase.disks)
		}
	}
}

// Tests initializing new object layer.
func TestNewObjectLayer(t *testing.T) {
	// Tests for FS object layer.
	nDisks := 1
	disks, err := getRandomDisks(nDisks)
	if err != nil {
		t.Fatal("Failed to create disks for the backend")
	}
	defer removeRoots(disks)

	endpoints := mustGetNewEndpointList(disks...)
	obj, err := newObjectLayer(endpoints)
	if err != nil {
		t.Fatal("Unexpected object layer initialization error", err)
	}
	_, ok := obj.(*fsObjects)
	if !ok {
		t.Fatal("Unexpected object layer detected", reflect.TypeOf(obj))
	}

	// Tests for XL object layer initialization.

	// Create temporary backend for the test server.
	nDisks = 16
	disks, err = getRandomDisks(nDisks)
	if err != nil {
		t.Fatal("Failed to create disks for the backend")
	}
	defer removeRoots(disks)

	endpoints = mustGetNewEndpointList(disks...)
	obj, err = newObjectLayer(endpoints)
	if err != nil {
		t.Fatal("Unexpected object layer initialization error", err)
	}

	_, ok = obj.(*xlObjects)
	if !ok {
		t.Fatal("Unexpected object layer detected", reflect.TypeOf(obj))
	}
}
