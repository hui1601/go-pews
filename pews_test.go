package pews_test

import (
	"fmt"
	pews "go-pews"
	"testing"
	"time"
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
	stations, err := pews.GetStationList()
	if err != nil {
		t.Error(err)
	}
	if len(stations) <= 100 {
		t.Errorf("stations length too short! it's likely more than 100.\n:Result: %d", len(stations))
	}
	data, err := pews.GetStationData(len(stations))
	if err != nil {
		t.Error(err)
	}

	if len(data.MMI) != len(stations) {
		t.Errorf("mmi length is not matched with stations length.\n:Result: %d", len(data.MMI))
	}

	if data.LastEarthquakeId == "" {
		t.Errorf("last earthquake id is empty.")
	}

	if data.Status == 0 {
		t.Errorf("status is not set.")
	}
}

func parseKMATimeString(timeString string) time.Time {
	t, _ := time.Parse("20060102150405", timeString)
	loc, _ := time.LoadLocation("Asia/Seoul")
	return t.In(loc)
}

func TestStartSimulation(t *testing.T) {
	simulationData := new(pews.SimulationData)
	simulationData.StartTime = time.Unix(1639469954, 0)
	simulationData.Duration = time.Minute * 7
	// 2021 Jeju earthquake
	simulationData.EarthquakeId = 2021007178
	pews.StartSimulation(*simulationData)
	simulationTime := time.Unix(1639469954, 0)
	earthquakeAlertTime := parseKMATimeString("20211214081931")
	earthquakeInfoTime := parseKMATimeString("20211214082327")
	stationList, err := pews.GetStationList()
	if err != nil {
		t.Error(err)
	}
	if len(stationList) <= 100 {
		t.Errorf("stations length too short! it's likely more than 100.\n:Result: %d", len(stationList))
	}
	for {
		fmt.Println(simulationTime.String(), earthquakeAlertTime.String(), earthquakeInfoTime.String())
		message, err := pews.GetStationData(len(stationList))
		if err != nil {
			t.Error(err)
		}
		if simulationTime.Before(earthquakeAlertTime) && message.Status != pews.PhaseNormal {
			t.Errorf("status is not set to 1(Normal).\n:Result: %d", message.Status)
		} else if simulationTime.After(earthquakeAlertTime) && simulationTime.Before(earthquakeInfoTime) && message.Status != pews.PhaseAlert {
			t.Errorf("status is not set to 2(Alert).\n:Result: %d", message.Status)
		} else if simulationTime.After(earthquakeInfoTime) && message.Status != pews.PhaseInfo {
			t.Errorf("status is not set to 3(Info).\n:Result: %d", message.Status)
		}
		simulationTime = simulationTime.Add(time.Second)
		time.Sleep(time.Second)
	}
}
