package pews_test

import (
	pews "go-pews"
	"testing"
)

func TestGetStationList(t *testing.T) {
	stations, err := pews.GetStationList()
	if err != nil {
		t.Error(err)
	}
	if len(stations) <= 100 {
		t.Errorf("stations length too short! it's likely more than 100.\n:Result: %d", len(stations))
	}
}

func TestGetStationData(t *testing.T) {

}
