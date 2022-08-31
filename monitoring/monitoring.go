package monitoring

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/team7mysupermon/devaten_middlewareapp/storage"
	"github.com/tidwall/gjson"
)

var (
	dbinstancemetrics    = make(map[string]*prometheus.GaugeVec)
	stopmetrics          = make(map[string]*prometheus.GaugeVec)
	runmetrics           = make(map[string]*prometheus.GaugeVec)
	mostexecutedmetrics  = make(map[string]*prometheus.GaugeVec)
	worstexecutedmetrics = make(map[string]*prometheus.GaugeVec)
	tableanalysismetrics = make(map[string]*prometheus.GaugeVec)

	databasetype       string
	starttimestamp     string
	usecasestopmetrics = make(map[string]interface{})
	stopdetails        []storage.Stop
	reportdata         []storage.ReportData
)

func Monitor() {
	go http.Handle("/metrics", promhttp.Handler())
	err := http.ListenAndServe(":9091", nil)
	if err != nil {
		log.Fatalln("Failed to serve metrics on port 9091 ")
	}
	log.Fatal(http.ListenAndServe(":9091", nil))
}

func ParseBody(body []byte, action string) {
	justString := GetPrometheusRegisteredMetrics()
	if action == "start" {
		startdataresponse := gjson.Get(string(body), "data.dataSourceList.#.databaseType").Array()
		databasetype = startdataresponse[0].String()
		instanceinfo := gjson.Get(string(body), "data.dataSourceList.#.data").Array()
		go func() {
			for key, val := range instanceinfo[0].Map() {
				fmt.Println(key, val)

				dbinstancemetrics["DBINSTANCE_"+strings.ToUpper(key)+"_GAUGE"] = prometheus.NewGaugeVec(
					prometheus.GaugeOpts{
						Name: "DBINSTANCE_" + strings.ToUpper(key) + "_GAUGE",
						Help: "",
					}, []string{
						"database",
					},
				)
				registered := strings.Contains(justString, "DBINSTANCE_"+strings.ToUpper(key)+"_GAUGE")
				if !registered {
					prometheus.MustRegister(
						dbinstancemetrics["DBINSTANCE_"+strings.ToUpper(key)+"_GAUGE"],
					)
				}
				dbinstancemetrics["DBINSTANCE_"+strings.ToUpper(key)+"_GAUGE"].With(prometheus.Labels{"database": strings.ToUpper(databasetype)}).Set(val.Float())
			}
		}()
	}

	if action == "run" {
		runinfo := gjson.Get(string(body), "data.runSituationResult.#.data").Array()
		go func() {
			for _, run := range runinfo {
				for key, val := range run.Map() {
					runmetrics["RUN_"+strings.ToUpper(key)+"_GAUGE"] = prometheus.NewGaugeVec(
						prometheus.GaugeOpts{
							Name: "RUN_" + strings.ToUpper(key) + "_GAUGE",
							Help: "",
						}, []string{
							"database",
						},
					)
					registered := strings.Contains(justString, "RUN_"+strings.ToUpper(key)+"_GAUGE")
					if !registered {
						prometheus.MustRegister(
							runmetrics["RUN_"+strings.ToUpper(key)+"_GAUGE"],
						)
					}
					runmetrics["RUN_"+strings.ToUpper(key)+"_GAUGE"].With(prometheus.Labels{"database": strings.ToUpper(databasetype)}).Set(val.Float())
				}
			}
		}()
	}

}
func CreateStopMetrics(arr []string) {
	justString := GetPrometheusRegisteredMetrics()
	go func() {
		for x := 0; x < len(arr); x++ {
			stopmetrics["STOP_"+strings.ToUpper(arr[x])+"_GAUGE"] = prometheus.NewGaugeVec(
				prometheus.GaugeOpts{
					Name: "STOP_" + strings.ToUpper(arr[x]) + "_GAUGE",
					Help: "",
				}, []string{
					"database",
					"usecase",
					"starttimestamp",
				},
			)
			registered := strings.Contains(justString, "STOP_"+strings.ToUpper(arr[x])+"_GAUGE")

			if !registered {
				prometheus.MustRegister(
					stopmetrics["STOP_"+strings.ToUpper(arr[x])+"_GAUGE"],
				)
				stopmetrics["STOP_"+strings.ToUpper(arr[x])+"_GAUGE"].With(prometheus.Labels{"database": strings.ToUpper(databasetype), "usecase": "DEMO", "starttimestamp": "starttimestamp"}).Set(0)
			}
		}
	}()

}
func RecordStopMetrics(body []byte) {
	var stopmetrics = GetStopMetricsMap()
	value10 := gjson.Get(string(body), "data").Array()
	for _, v := range value10 {
		for key, val := range v.Map() {
			err := json.Unmarshal([]byte(val.Raw), &stopdetails)
			starttimestamp = stopdetails[0].Starttimestamp
			if err != nil {
				panic(err)
			}
			var stopcolumnsmetrics = make(map[string]float64)
			for x := 0; x < len(stopdetails); x++ {
				stopData := stopdetails[x].ValueObjectList

				for y := 0; y < len(stopData); y++ {
					stopcolumnsmetrics["STOP_"+strings.ToUpper(stopData[y].ColumnName)+"_GAUGE"] = stopData[y].NewValue

				}

			}
			usecasestopmetrics[key] = stopcolumnsmetrics
		}
	}
	go func() {
		for {
			for key, element := range usecasestopmetrics {
				myMap := element.(map[string]float64)
				for columnname, value := range myMap {
					var datav = columnname
					var valuedata = value

					stopmetrics[datav].With(prometheus.Labels{"database": strings.ToUpper(databasetype), "usecase": key, "starttimestamp": starttimestamp}).Set(valuedata)
					time.Sleep(1 * time.Second)

				}
			}
		}
	}()
}

func GetStopMetricsMap() map[string]*prometheus.GaugeVec {

	stopmetrics["STOP_SQL_PER_SEC_GAUGE"] = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "STOP_SQL_PER_SEC_GAUGE",
			Help: "",
		}, []string{
			"databse",
			"usecase",
		},
	)
	return stopmetrics
}
func GetPrometheusRegisteredMetrics() string {
	scientists := []string{
		"Einstein",
	}
	mfs, err := prometheus.DefaultGatherer.Gather()
	if err != nil {
		panic(err)
	}

	for _, mf := range mfs {
		scientists = append(scientists, mf.GetName())
	}
	justString := strings.Join(scientists, " ")
	return justString
}
func RecordReport(body []byte) {
	fmt.Println("this is in report file")
	report := gjson.Get(string(body), "list")
	err := json.Unmarshal([]byte(report.Raw), &reportdata)
	if err != nil {
		panic(err)
	}
	go func() {
		for {
			for x := 0; x < len(reportdata); x++ {
				mostExecuteddata := reportdata[x].MostExecuted
				starttimestamp := reportdata[x].Starttimestamp
				usecaseIdentifier := reportdata[x].UsecaseIdentifier
				//
				for y := 0; y < len(mostExecuteddata); y++ {

					queryid := mostExecuteddata[y].QueryId
					fmt.Println(queryid)
					res := strings.Split(mostExecuteddata[y].Colvalues, ",")
					for j := 0; j < len(res); j++ {
						medata := strings.Split(res[j], "|")
						mcolname := "MOSTEXECUTE_" + strings.ToUpper(medata[0]) + "_GAUGE"
						mcolval := medata[1]
						mostexecutedmetrics[mcolname] = prometheus.NewGaugeVec(
							prometheus.GaugeOpts{
								Name: mcolname,
								Help: "",
							}, []string{
								"database",
								"usecase",
								"queryid",
								"startimestamp",
							},
						)
						justString := GetPrometheusRegisteredMetrics()

						registered := strings.Contains(justString, mcolname)

						if !registered {

							prometheus.MustRegister(
								mostexecutedmetrics[mcolname],
							)
						}
						if s, err := strconv.ParseFloat(mcolval, 64); err == nil {
							//go func() {

							mostexecutedmetrics[mcolname].With(prometheus.Labels{"database": strings.ToUpper(databasetype), "usecase": usecaseIdentifier, "queryid": queryid, "startimestamp": starttimestamp}).Set(s)
							time.Sleep(1 * time.Second)
							//}()
						}

					}

				}
				//	}()
				wrostExecuteddata := reportdata[x].WrostExecuted
				//go func() {
				for i := 0; i < len(wrostExecuteddata); i++ {

					queryid := wrostExecuteddata[i].QueryId

					res1 := strings.Split(wrostExecuteddata[i].Colvalues, ",")

					for k := 0; k < len(res1); k++ {
						wedata := strings.Split(res1[k], "|")
						wcolname := "WROSTEXECUTE_" + strings.ToUpper(wedata[0]) + "_GAUGE"
						wcolval := wedata[1]
						worstexecutedmetrics[wcolname] = prometheus.NewGaugeVec(
							prometheus.GaugeOpts{
								Name: wcolname,
								Help: "",
							}, []string{
								// Which user has requested the operation?
								"database",
								"usecase",
								"queryid",
								"startimestamp",
								// Of what type is the operation?

							},
						)
						justString := GetPrometheusRegisteredMetrics()

						registered := strings.Contains(justString, wcolname)

						if !registered {

							prometheus.MustRegister(
								worstexecutedmetrics[wcolname],
							)
						}
						if s, err := strconv.ParseFloat(wcolval, 64); err == nil {
							//go func() {

							worstexecutedmetrics[wcolname].With(prometheus.Labels{"database": strings.ToUpper(databasetype), "usecase": usecaseIdentifier, "queryid": queryid, "startimestamp": starttimestamp}).Set(s)
							time.Sleep(1 * time.Second)
							//}()
						}

					}

				}
				//
			}
		}
	}()
}
func TableanalysisReport(body []byte) {

	tablereport := gjson.Get(string(body), "data").Array()
	//go func() {
	for _, v := range tablereport {
		reportval := v
		for key, val := range reportval.Map() {
			tablecolumn := key
			colname := "TABAALEANALYSISDATA_" + strings.ToUpper(tablecolumn) + "_GAUGE"
			tableval := val.Float()
			tableanalysismetrics[colname] = prometheus.NewGaugeVec(
				prometheus.GaugeOpts{
					Name: colname,
					Help: "",
				}, []string{
					"database",
					"tablename",
				},
			)
			justString := GetPrometheusRegisteredMetrics()

			registered := strings.Contains(justString, colname)

			if !registered {

				prometheus.MustRegister(
					tableanalysismetrics[colname],
				)

			}
			//if s, err := strconv.ParseFloat(val., 64); err == nil {

			fmt.Println(reportval.Map()["TABLE_NAME"].String())

			tableanalysismetrics[colname].With(prometheus.Labels{"database": strings.ToUpper(databasetype), "tablename": reportval.Map()["TABLE_NAME"].String()}).Set(tableval)

			time.Sleep(1 * time.Second)

		}

	}

	//}()

}
