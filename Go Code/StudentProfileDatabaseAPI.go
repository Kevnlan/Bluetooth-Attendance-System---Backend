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
//To Hold the json response when calling CouchDB's studentregistered, studentpassword and isstudentpresent api
type HandleGenericCouchJSON struct{
	Total_rows int `json:"total_rows"`
	Offset int `json:"offset"`
	Rows []struct{
			Id string `json:"id"`
			Key int `json:"key"`
			Value string `json:"value"`
		}`json:"rows"`
}
//To Hold the json response when calling CouchDB's getbluetoothid api
type HandleBluetoothIdCouchJSON struct{
	Total_rows int `json:"total_rows"`
	Offset int `json:"offset"`
	Rows []struct{
			Id string `json:"id"`
			Key int `json:"key"`
			Value string `json:"value"`
		}`json:"rows"`
}
//To handle JSON reurned from calling myo own CheckStudentValid API
type ValidStudent struct{

	Status string `json:"status"`
}
//To store the details of the Student to be inserted in RegisterStudent() 
type Student struct{
	StudentId int `json:"studentid"`
	Password string `json:"password"`
}
//To handle JSON returned when getting a particular student's record. Used in doDelete()
type StudentFull struct{
	StudentId int `json:"studentid"`
	Password string `json:"password"`
	Rev string `json:"_rev"`
	Id string `json:"_id"`
}
//To store the list of classes student has enrolled that is returned when calling my studentenrolled api
type StudentEnrolled struct{
	Id string `json:"id"`
	Key int `json:"key"`
	Value []int `json:"value"`
}
//To build the Json to send back after successful register
type RegisterSuccess struct{
	DeviceId string `json:"deviceid"`
	BluetoothIds []string `json:"bluetoothids"`
}

var uniqueid UUID
var hasStudentRegistered HandleGenericCouchJSON
var validStudent ValidStudent
var BaseUrl string
var student Student
var studentFull StudentFull
var studentPasswordData HandleGenericCouchJSON
var isStudentPresent HandleGenericCouchJSON
var studentEnrolled StudentEnrolled
var handleBluetoothIdCouchJson HandleBluetoothIdCouchJSON
var registerSuccess RegisterSuccess

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

//Helper Funtion does GET to check if student registered
func doGetRegister(id string){
	var Url string
	Url=BaseUrl+"/studentprofile/_design/studentdetails/_view/studentregistered?key=+"+string(id)
	response,_ := http.Get(Url)
	defer response.Body.Close()
	decoder := json.NewDecoder(response.Body)
	err := decoder.Decode(&hasStudentRegistered)
	if err != nil {
		panic(err)
	}
}

//Helper Funtion does GET to get student password
func doGetPassword(id string){
	var Url string
	Url=BaseUrl+"/studentprofile/_design/studentdetails/_view/studentpassword?key="+string(id)
	response,_ := http.Get(Url)
	defer response.Body.Close()
	decoder := json.NewDecoder(response.Body)
	err := decoder.Decode(&studentPasswordData)
	if err != nil {
		panic(err)
	}
}

//Helper Funtion does GET to check if student is valid(ie enrolled in college)
func doGetValidStudent(id string){
	response,_ := http.Get("http://localhost:3000/checkstudentvalid/"+id)
	defer response.Body.Close()
	decoder := json.NewDecoder(response.Body)
	err := decoder.Decode(&validStudent)
	if err != nil {
		panic(err)
	}
}

//Helper Function to GET the list of classes student has enrolled for
func doGetEnrolled(id string){
	response,_ := http.Get("http://localhost:3000/studentenrolled/"+id)
	defer response.Body.Close()
	decoder := json.NewDecoder(response.Body)
	err := decoder.Decode(&studentEnrolled)
	if err != nil {
		panic(err)
	}
}

//Helper Funtion does PUT to Insert a new student record into the table
func doPut(body string,uuid string){
    Url:= BaseUrl+"/studentprofile/"+uuid
	request, _ := http.NewRequest("PUT", Url, strings.NewReader(body))
	client := &http.Client{}
	client.Do(request)
}

//Helper Funtion does PUT to insert StudentID into a class's Attendance Table
func doPutAttendance(classid string,body string,uuid string){
    Url:= BaseUrl+"/class"+classid+"/"+uuid
	request, _ := http.NewRequest("PUT", Url, strings.NewReader(body))
	client := &http.Client{}
	client.Do(request)
}

//Helper Funtion does DELETE to delete a student record from tbale
func doDelete(id string){
	response,_ := http.Get("https://couchdb-80f683.smileupps.com/studentprofile/"+id)
	defer response.Body.Close()
	decoder := json.NewDecoder(response.Body)
	decoder.Decode(&studentFull)

	Url:= BaseUrl+"/studentprofile/"+id+"?rev="+studentFull.Rev
	request, _ := http.NewRequest("DELETE", Url, nil)
	client := &http.Client{}
	client.Do(request)
}

//Main RegisterStudent function - REST
//Accepts a StudentId, password and checks if student is already registered. Else checks if student is valid. If valid, Insters into CouchDB.
//Returns the unique device id and the list of PI MAC addresses to store
func RegisterStudent(rw http.ResponseWriter, r *http.Request) {
	params := r.URL.Query()
	id := params.Get(":id")
	pass:= params.Get(":pass")
	//Check if student has already registered
	doGetRegister(id)
	if len(hasStudentRegistered.Rows)>0 {
		a:=`{"error":"Already Registered"}`
		rw.Write([]byte(a))
	}
	//Check if student is a valid student enrolled in college
	if len(hasStudentRegistered.Rows)==0 {
		doGetValidStudent(id)
		if validStudent.Status=="yes"{
			//To-Do Do JSON Unmarshall
			student.StudentId,_=strconv.Atoi(id)
			student.Password=pass
			a, _ := json.Marshal(student)
			uuid:=getUUID()
			doPut(string([]byte(a)),uuid)
			//Get list of classes student is enrolled in
			doGetEnrolled(id)
			//Get the BluetoothIds for all the classes student has enrolled in
			registerSuccess.DeviceId=uuid
			registerSuccess.BluetoothIds=nil
			for i:=0; i<len(studentEnrolled.Value);i++ {
				UrlGet:=BaseUrl+"/bluetoothid/_design/getbluetoothid/_view/bluetoothid?key="+strconv.Itoa(studentEnrolled.Value[i])
				response,_ := http.Get(UrlGet)
				defer response.Body.Close()
				decoder := json.NewDecoder(response.Body)
				decoder.Decode(&handleBluetoothIdCouchJson)
				registerSuccess.BluetoothIds=append(registerSuccess.BluetoothIds,handleBluetoothIdCouchJson.Rows[0].Value)
			}
			rw.WriteHeader(http.StatusCreated)
			b,_:=json.Marshal(registerSuccess)
			rw.Write([]byte(b))
		}
		if validStudent.Status=="no" {
			a:=`{"error":"Not Valid Student"}`
			rw.Write([]byte(a))
		}	
	}
}

//Main DeleteStudent function - REST
//Accepts a StudentId, password and checks if id and password match. If they do, delete a student record
func DeleteStudent(rw http.ResponseWriter, r *http.Request) {
	params := r.URL.Query()
	id := params.Get(":id")
	pass:= params.Get(":pass")
	//Get Password to check if its legit delete request
	doGetPassword(id)
	if len(studentPasswordData.Rows)==0 {
		a:=`{"error":"Not Exist"}`
		rw.Write([]byte(a))
	}
	if len(studentPasswordData.Rows)>0 {
		if studentPasswordData.Rows[0].Value==pass {
			doDelete(studentPasswordData.Rows[0].Id)
			rw.WriteHeader(http.StatusOK)
			rw.Write([]byte(""))
		}
	}
	if len(studentPasswordData.Rows)==0 {
		rw.Write([]byte(`{"error":"Student does not exist"}`))
	}

}

//Main MarkPresent function - REST
//Accepts a StudentId, deviceId and classId. Check if valid deviceId. If yes, check if student already present. IF no, mark student as present
func MarkPresent(rw http.ResponseWriter, r *http.Request) {
	params := r.URL.Query()
	id := params.Get(":id")
	deviceid:= params.Get(":deviceid")
	classid:= params.Get(":classid")
	doGetRegister(id)
	if len(hasStudentRegistered.Rows)>0 {
		//Check If Student is not Proxying - Check deviceid
		if deviceid==hasStudentRegistered.Rows[0].Id {
			//Do duplication Checks 
			response,_ := http.Get(BaseUrl+"/+class"+classid+"/_design/classattendance/_view/isstudentpresent?key="+id)
			defer response.Body.Close()
			decoder := json.NewDecoder(response.Body)
			decoder.Decode(&isStudentPresent)
			if len(isStudentPresent.Rows)>0 {
				a:=`{"error":"Student already present"}`
				rw.Write([]byte(a))
			}
			if len(isStudentPresent.Rows)==0 {
				//To-Do check if student is allowed to enter the class
				doPutAttendance(classid,`{"studentid":`+id+`}`,getUUID())
				a:=`{"success":"Student is marked present"}`
				rw.Write([]byte(a))	
			}
		}
		if deviceid!=hasStudentRegistered.Rows[0].Id {
			a:=`{"error":"Student is cheating. StudentId and DeviceId dont match"}`
			rw.Write([]byte(a))	
		}
	}
	if len(hasStudentRegistered.Rows)==0 {
		a:=`{"error":"Student not registered"}`
		rw.Write([]byte(a))		
	}
}


func main(){

	//BASE URL for CouchDB. Curling this URL should give a welcome message
	BaseUrl="https://admin:9631aa6374e6@couchdb-80f683.smileupps.com"

	//REST Config begins
			mux := routes.New()
			mux.Post("/registerstudent/:id/:pass",RegisterStudent)
			mux.Post("/markpresent/:id/:deviceid/:classid",MarkPresent)
			mux.Del("/deletestudent/:id/:pass",DeleteStudent)
			http.Handle("/", mux)
			log.Println("REST has been set up: "+strconv.Itoa(3001))
			log.Println("Listening...")
			http.ListenAndServe(":"+strconv.Itoa(3001), nil)
	//REST Config end
}