package pews

import (
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"
)

type Station struct {
	Longitude int `json:"longitude"`
	Latitude  int `json:"latitude"`
}

type EarthquakeMessage struct {
	StationUpdateNeeded bool   `json:"stationUpdateNeeded"`
	Status              int8   `json:"status"`
	LastEarthquakeId    string `json:"lastEarthquakeId"`
	MMI                 []int8 `json:"mmi"`
	EarthquakeInfo      struct {
		Longitude        int      `json:"longitude"`
		Latitude         int      `json:"latitude"`
		EarthquakeId     string   `json:"earthquakeId"`
		Magnitude        int8     `json:"magnitude"`
		Depth            int8     `json:"depth"`
		Time             string   `json:"time"`
		MaxIntensity     int8     `json:"maxIntensity"`
		MaxIntensityArea []string `json:"maxIntensityArea"`
		Epicenter        string   `json:"epicenter"`
	} `json:"earthquakeInfo"`
}

type SimulationData struct {
	StartTime    time.Time // when simulation data starts(ex. 20211214081904)
	EarthquakeId uint
	Duration     time.Duration
	callTime     time.Time // when simulation started(when StartStimulation called, ex. 20230214081904)
}

var simulation *SimulationData

func init() {
}

func byteToBinaryString(b byte) string {
	// convert using bit shifting
	var binaryString string
	for i := 0; i < 8; i++ {
		binaryString += strconv.Itoa(int(b >> uint(7-i) & 1))
	}
	return binaryString
}
func byteArrayToBinaryString(byteArray []byte) string {
	var binaryString string
	for _, b := range byteArray {
		binaryString += byteToBinaryString(b)
	}
	return binaryString
}

func binaryStringToInt(binaryString string) int {
	var result int
	for i := 0; i < len(binaryString); i++ {
		result += int(binaryString[i]-'0') << uint(len(binaryString)-i-1)
	}
	return result
}

func kmaTimeString() string {
	if simulation != nil {
		timeDiff := time.Now().Unix() - simulation.callTime.Unix()
		if timeDiff > simulation.Duration.Milliseconds() {
			simulation = nil
		} else {
			return time.Unix(simulation.StartTime.Unix()+timeDiff, 0).UTC().Format("20060102150405")
		}
	}
	now := time.Now().UTC().Add(-1 * time.Second)
	return now.Format("20060102150405")
}

func GetStationList() ([]Station, error) {
	var stations []Station
	url := "https://www.weather.go.kr/pews/data/" + kmaTimeString() + ".s"
	if simulation != nil {
		url = fmt.Sprintf("https://www.weather.go.kr/pews/data/%d/%s.s", simulation.EarthquakeId, kmaTimeString())
	}
	var client http.Client
	resp, err := client.Get(url)
	if err != nil {
		return nil, err
	}
	defer func(Body io.ReadCloser) {
		_ = Body.Close()
	}(resp.Body)
	if resp.StatusCode == http.StatusOK {
		bodyBytes, err := io.ReadAll(resp.Body)
		if err != nil {
			return nil, err
		}
		// bodyBytes to binary string
		binaryString := byteArrayToBinaryString(bodyBytes)
		for i := 0; i < len(binaryString)/20*20; i += 20 {
			latitude := binaryStringToInt(binaryString[i : i+10])
			longitude := binaryStringToInt(binaryString[i+10 : i+20])
			stations = append(stations, Station{Longitude: 12000 + longitude, Latitude: 3000 + latitude})
		}
	}
	return stations, nil
}

func parseStationDataHeader(headerString string, message *EarthquakeMessage) {
	message.StationUpdateNeeded = headerString[0] == '1'
	message.Status = (func(code string) int8 {
		// it's very awful
		// why KMA does this?!!
		switch code {
		case "00":
			return 1
		case "01":
			return 4
		case "10":
			return 2
		case "11":
			return 3
		}
		return 1
	})(headerString[1:3])
	// data header is short when simulation.
	if simulation == nil {
		message.LastEarthquakeId = "20" + strconv.Itoa(binaryStringToInt(headerString[6:32]))
	}
}

func parseStationDataBody(bodyString string, stationLength int, message *EarthquakeMessage) {
	for i := 0; i < stationLength; i++ {
		mmiConvertArray := []int8{1, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 10, 1, 1, 1}
		message.MMI = append(message.MMI, mmiConvertArray[binaryStringToInt(bodyString[i*4:i*4+4])])
	}
	if message.Status == 2 || message.Status == 3 {
		// very disgusting
		var earthquakeInfoBinaryString = bodyString[len(bodyString)-600:]
		message.EarthquakeInfo.Latitude = 3000 + binaryStringToInt(earthquakeInfoBinaryString[0:10])
		message.EarthquakeInfo.Longitude = 12000 + binaryStringToInt(earthquakeInfoBinaryString[10:20])
		message.EarthquakeInfo.Magnitude = int8(binaryStringToInt(earthquakeInfoBinaryString[20:27]))
		message.EarthquakeInfo.Depth = int8(binaryStringToInt(earthquakeInfoBinaryString[27:36]))
		message.EarthquakeInfo.Time = strconv.Itoa(binaryStringToInt(earthquakeInfoBinaryString[36:69])+32400) + "000"
		message.EarthquakeInfo.EarthquakeId = "20" + strconv.Itoa(binaryStringToInt(earthquakeInfoBinaryString[69:95]))
		message.EarthquakeInfo.MaxIntensity = int8(binaryStringToInt(earthquakeInfoBinaryString[95:99]))
		// It is impossible to observe maximum seismic intensity in all regions (except for the end of the earth).
		// If the maximum intensity is "I", it should be treated as an exception by KMA.
		if earthquakeInfoBinaryString[99:116] == "11111111111111111" {
			message.EarthquakeInfo.MaxIntensityArea = []string{}
		} else {
			// This order should never be changed!
			areaNames := []string{"서울", "부산", "대구", "인천", "광주", "대전", "울산", "세종", "경기", "강원", "충북", "충남", "전북", "전남", "경북", "경남", "제주"}
			for i := 99; i < 116; i++ {
				if earthquakeInfoBinaryString[i] == '1' {
					message.EarthquakeInfo.MaxIntensityArea = append(message.EarthquakeInfo.MaxIntensityArea, areaNames[i-99])
				}
			}
		}
	}
}

// StartSimulation id: earthquake id, startTime: simulation data start time, duration: simulation data duration
func StartSimulation(data SimulationData) {
	simulation = &data
	simulation.callTime = time.Now().UTC().Add(-1 * time.Second)
}

func GetStationData(stationLength int) (*EarthquakeMessage, error) {
	var message EarthquakeMessage
	var client http.Client
	headerSize := 32
	url := "https://www.weather.go.kr/pews/data/" + kmaTimeString() + ".b"
	if simulation != nil {
		url = "https://www.weather.go.kr/pews/data/2021007178/" + kmaTimeString() + ".b"
		headerSize = 8
	}
	resp, err := client.Get(url)
	if err != nil {
		return nil, err
	}
	defer func(Body io.ReadCloser) {
		_ = Body.Close()
	}(resp.Body)
	if resp.StatusCode == http.StatusOK {
		bodyBytes, err := io.ReadAll(resp.Body)
		if err != nil {
			return nil, err
		}
		// bodyBytes to binary string
		binaryString := byteArrayToBinaryString(bodyBytes)
		header := binaryString[:headerSize]
		body := binaryString[headerSize:]
		parseStationDataHeader(header, &message)
		if message.StationUpdateNeeded {
			_, _ = GetStationList()
		}
		parseStationDataBody(body, stationLength, &message)
		if message.Status == 2 || message.Status == 3 {
			message.EarthquakeInfo.Epicenter = strings.Trim(string(bodyBytes[len(bodyBytes)-60:]), "\x00\x20")
		}
	}
	return &message, nil
}
