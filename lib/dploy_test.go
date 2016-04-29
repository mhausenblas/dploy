package dploy

import "testing"

// Tests

func TestInit(t *testing.T) {

}

// Examples

func ExampleInit_output() {
	Init("/tmp/")
	// Output: /tmp/dploy.app with following content:
	//  marathon_url: http://localhost:8080
	//  app_name: CHANGEME
}
