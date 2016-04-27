package main

import (
	"github.com/drone/routes"
	"log"
    "encoding/json"
	"net/http"
	"strconv"
	"strings"
)

//To Hold Json generated by calling getUUID()
type UUID struct{
	Uuids []string `json:"uuids"`
}

//To Hold the json response when calling CouchDB's isstudentpresent api
type HandleGenericCouchJSON struct{
	Total_rows int `json:"total_rows"`
	Offset int `json:"offset"`
	Rows []struct{
			Id string `json:"id"`
			Key int `json:"key"`
			Value int `json:"value"`
		}`json:"rows"`
}

//To Hold the json response when calling CouchDB's bluetoothid api. Also handle classlist JSOn
type HandleBluetoothIdCouchJSON struct{
	Total_rows int `json:"total_rows"`
	Offset int `json:"offset"`
	Rows []struct{
			Id string `json:"id"`
			Key int `json:"key"`
			Value string `json:"value"`
		}`json:"rows"`
}

//To handle JSON returned when getting a particular bluetoothid DB record.
type BluetoothIdFull struct{
	ClassId int `json:"classid"`
	BluetoothId string `json:"bluetoothid"`
	Rev string `json:"_rev"`
	Id string `json:"_id"`
}

//To handle JSON returned when getting a particular classlist DB record.
type ClassListFull struct{
	ClassNumber int `json:"classnumber"`
	ClassName string `json:"classname"`
	Rev string `json:"_rev"`
	Id string `json:"_id"`
}

//To store details of student name will be received from my GetStudentName API
type StudentName struct{
	Id string `json:"id"`
	Key int `json:"key"`
	Value string `json:"value"`
}

var uniqueid UUID
var attendanceList HandleGenericCouchJSON
var studentName StudentName
var studentNameList []StudentName
var BaseUrl string
var NewDesignDoc string
var handleBluetoothIdCouchJSON HandleBluetoothIdCouchJSON
var bluetoothIdFull BluetoothIdFull
var handleClassListCouchJSON HandleBluetoothIdCouchJSON
var classListFull ClassListFull

//Get Unique UUID from CouchDB
func getUUID() string{
	response,_ := http.Get("https://couchdb-80f683.smileupps.com/_uuids")
	defer response.Body.Close()
	decoder := json.NewDecoder(response.Body)
	err := decoder.Decode(&uniqueid)
	if err != nil {
		panic(err)
	}
	return uniqueid.Uuids[0]
}

func GetAttendanceList(rw http.ResponseWriter, r *http.Request) {
	params := r.URL.Query()
	classid := params.Get(":classid")
	Url:=BaseUrl+"/class"+classid+"/_design/classattendance/_view/isstudentpresent"
	response,_ := http.Get(Url)
	defer response.Body.Close()
	decoder := json.NewDecoder(response.Body)
	decoder.Decode(&attendanceList)
	for i:=0; i<len(attendanceList.Rows); i++ {
		key:=strconv.Itoa(attendanceList.Rows[i].Key)
		UrlGet:="http://localhost:3000/studentname/"+key
		response,_ := http.Get(UrlGet)
		defer response.Body.Close()
		decoder := json.NewDecoder(response.Body)
		decoder.Decode(&studentName)
		studentNameList = append(studentNameList, studentName)
	}
	//Send the attendance list
	a, _ := json.Marshal(studentNameList)
	rw.WriteHeader(http.StatusOK)
	rw.Write([]byte(a))
}


func ClearAttendanceList(rw http.ResponseWriter, r *http.Request) {
	params := r.URL.Query()
	classid := params.Get(":classid")
	//Delete
	Url:= BaseUrl+"/class"+classid
	request, _ := http.NewRequest("DELETE", Url, nil)
	client := &http.Client{}
	client.Do(request)

	//Create
	Url= BaseUrl+"/class"+classid
	request, _ = http.NewRequest("PUT", Url, nil)
	client = &http.Client{}
	client.Do(request)

	//Insert Design Doc
	Url= BaseUrl+"/class"+classid+"/_design/classattendance"

	request, _ = http.NewRequest("PUT", Url, strings.NewReader(NewDesignDoc))
	client = &http.Client{}
	client.Do(request)
	
	rw.WriteHeader(http.StatusOK)
	rw.Write([]byte(""))
}

func CreateClass(rw http.ResponseWriter, r *http.Request) {
	params := r.URL.Query()
	classid := params.Get(":classid")
	classname := params.Get(":classname")
	bluetoothid := params.Get(":bluetoothid")
	//Insert into classlist
	Url:= BaseUrl+"/classlist/"+getUUID()
	body:=`{"classnumber":`+classid+`,"classname":"`+classname+`"}`
	request, _ := http.NewRequest("PUT", Url, strings.NewReader(body))
	client := &http.Client{}
	client.Do(request)

	//Create DB
	Url= BaseUrl+"/class"+classid
	request, _ = http.NewRequest("PUT", Url, nil)
	client = &http.Client{}
	client.Do(request)

	//Insert Design Doc
	Url= BaseUrl+"/class"+classid+"/_design/classattendance"

	request, _ = http.NewRequest("PUT", Url, strings.NewReader(NewDesignDoc))
	client = &http.Client{}
	client.Do(request)
	
	//Insert into bluetooth id database
	Url= BaseUrl+"/bluetoothid/"+getUUID()
	body=`{"classid":`+classid+`,"bluetoothid":"`+bluetoothid+`"}`
	request, _ = http.NewRequest("PUT", Url, strings.NewReader(body))
	client = &http.Client{}
	client.Do(request)

	rw.WriteHeader(http.StatusOK)
	rw.Write([]byte(""))
}

func DeleteClass(rw http.ResponseWriter, r *http.Request) {
	params := r.URL.Query()
	classid := params.Get(":classid")

	//Delete from classlist
	//Go get the record id
	Url:=BaseUrl+"/classlist/_design/getclassdata/_view/classexists?key="+string(classid)
	response,_ := http.Get(Url)
	defer response.Body.Close()
	decoder := json.NewDecoder(response.Body)
	decoder.Decode(&handleClassListCouchJSON)
	//Using record id get rev
	response,_ = http.Get("https://couchdb-80f683.smileupps.com/classlist/"+handleClassListCouchJSON.Rows[0].Id)
	defer response.Body.Close()
	decoder = json.NewDecoder(response.Body)
	decoder.Decode(&classListFull)
	//Delete using rev
	Url= BaseUrl+"/classlist/"+handleClassListCouchJSON.Rows[0].Id+"?rev="+classListFull.Rev
	request, _ := http.NewRequest("DELETE", Url, nil)
	client := &http.Client{}
	client.Do(request)

	//Delete DB
	Url= BaseUrl+"/class"+classid
	request, _ = http.NewRequest("DELETE", Url, nil)
	client = &http.Client{}
	client.Do(request)

	//Delet Entry in bluetoothid database
	//Go get the record id
	Url=BaseUrl+"/bluetoothid/_design/getbluetoothid/_view/bluetoothid?key="+string(classid)
	response,_ = http.Get(Url)
	defer response.Body.Close()
	decoder = json.NewDecoder(response.Body)
	decoder.Decode(&handleBluetoothIdCouchJSON)
	//Using record id get rev
	response,_ = http.Get("https://couchdb-80f683.smileupps.com/bluetoothid/"+handleBluetoothIdCouchJSON.Rows[0].Id)
	defer response.Body.Close()
	decoder = json.NewDecoder(response.Body)
	decoder.Decode(&bluetoothIdFull)
	//Delete using rev
	Url= BaseUrl+"/bluetoothid/"+handleBluetoothIdCouchJSON.Rows[0].Id+"?rev="+bluetoothIdFull.Rev
	request, _ = http.NewRequest("DELETE", Url, nil)
	client = &http.Client{}
	client.Do(request)
	
	rw.WriteHeader(http.StatusOK)
	rw.Write([]byte(""))
}

func main(){

	//BASE URL for CouchDB. Curling this URL should give a welcome message
	BaseUrl="https://admin:9631aa6374e6@couchdb-80f683.smileupps.com"

	NewDesignDoc=`{"_id": "_design/classattendance","views": {"isstudentpresent": {"map": "function(doc){ emit(doc.studentid, doc.studentid)}"}}}`

	//REST Config begins
			mux := routes.New()
			mux.Get("/classattendance/:classid", GetAttendanceList)
			mux.Del("/clearclassattendance/:classid", ClearAttendanceList)
			mux.Del("/deleteclass/:classid", DeleteClass)
			mux.Post("/createclass/:classid/:classname/:bluetoothid", CreateClass)
			http.Handle("/", mux)
			log.Println("REST has been set up: "+strconv.Itoa(3002))
			log.Println("Listening...")
			http.ListenAndServe(":"+strconv.Itoa(3002), nil)
	//REST Config end
}