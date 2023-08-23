// represent a basic report
package datastructures

import (
	"fmt"
	"reflect"
	"strings"
	"sync"
	"time"

	"github.com/armosec/utils-go/httputils"
)

const (
	defaultSystemReportEndpoint = "/k8s/sysreport"
)

var (
	systemReportEndpoint = &sysEndpoint{}
)

type sysEndpoint struct {
	value string
	mu    sync.RWMutex
}

func (e *sysEndpoint) IsEmpty() bool {
	return e.Get() == ""
}

func (e *sysEndpoint) Set(value string) {
	e.mu.Lock()
	e.value = value
	e.mu.Unlock()
}

func (e *sysEndpoint) Get() string {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.value
}

// SetOrDefault sets the system report endpoint to the provided value for valid
// inputs, or to a default value on invalid ones
//
// An empty input is considered invalid, and would thus be set to a default endpoint
func (e *sysEndpoint) SetOrDefault(value string) {
	if value == "" {
		value = defaultSystemReportEndpoint
	}
	e.Set(value)
}

// GetOrDefault returns the value of the current system report endpoint if it
// is set. If not set, it sets the value to a default and returns the newly set
// value
func (e *sysEndpoint) GetOrDefault() string {
	current := e.Get()
	if current == "" {
		e.SetOrDefault("")
	}
	return e.Get()
}

// SetSystemReportEndpoint sets the system report endpoint to the provided
// value or a default
//
// Deprecated: the SetSysetmReportEndpoint shouldn’t be the setter for an
// unexported variable. Use the sysEndpoint.SetOrDefault() method instead.
func SetSystemReportEndpoint(endpoint string) {
	systemReportEndpoint.SetOrDefault(endpoint)
}

// GetSystemReportEndpoint gets the system report endpoint or sets a default
// and returns it
//
// Deprecated: the GetSysetmReportEndpoint shouldn’t be the setter for an
// unexported variable. Use the sysEndpoint.GetOrDefault() method instead.
func GetSystemReportEndpoint() string {
	return systemReportEndpoint.GetOrDefault()
}

// JobsAnnotations job annotation
type JobsAnnotations struct {
	/* jobID: context   eg. if a certain job has multiple stages
	  eg. attach namespace>attach wlid in ns
	  so obj when pod is cached should look like:
	  {
		  jobID#1: {
			"attach namespace"
		  }
	  }
	  - SHOULD BE RETHINK
	*/
	// JobIDsContex map[string]string `json:"jobIDsContex,omitempty"`
	CurrJobID    string `json:"jobID"`       //simplest case (for now till we have a better idea)
	ParentJobID  string `json:"parentJobID"` //simplest case (for now till we have a better idea)
	LastActionID string `json:"actionID"`    //simplest case (for now till we have a better idea) used to pass as defining ordering between multiple components
}

//BaseReport : represents the basic reports from various actions eg. attach and so on
//
// ("reporter": "auditlog processor", //the name of your k8s component
//  "target": "<scope> auditlogs", // eg. if you know its cluster & ns you can say: hipstershop/dev auditlogs
//  "status":  <use constants defined above eg. JobStarted>
//  "action": "<the action itself- eg. fetching logs from s3",
//  "errors": <fill if u encountered any>
//  "actionID" & "actionIDN" - numerical representation - eg if it's the first step then it should be 1, it also allow "forks" to happen
// 	"jobID": event receiver will fill that for you
// 	"parentAction": used like if you have like autoattach right? namespaces is the parent job but every wl up has attach but it's parent is the autoattach task
// 	"timestamp": <s.e>
// 	"customerGUID": s.e
// }

// Statuses type
type StatusType string

const (
	JobSuccess string = "success"
	JobFailed  string = "failure"
	JobWarning string = "warning"
	JobStarted string = "started"
	JobDone    string = "done"
)

type BaseReport struct {
	CustomerGUID     string                `json:"customerGUID"` // customerGUID as declared in environment
	Reporter         string                `json:"reporter"`     // component reporting the event
	Target           string                `json:"target"`       // wlid, cluster,etc. - which component this event is applicable on
	Status           string                `json:"status"`       // Action scope: Before action use "started", after action use "failure/success". Reporter scope: Before action use "started", after action use "done".
	ActionName       string                `json:"action"`       // Stage action. short description of the action to-be-done. When defining an action
	Errors           []string              `json:"errors,omitempty"`
	ActionID         string                `json:"actionID"`               // Stage counter of the E2E process. initialize at 1. The number is increased when sending job report
	ActionIDN        int                   `json:"numSeq"`                 // The ActionID in number presentation
	JobID            string                `json:"jobID"`                  // UID received from the eventReceiver after first report (the initializing is part of the first report)
	ParentAction     string                `json:"parentAction,omitempty"` // Parent JobID
	Details          string                `json:"details,omitempty"`      // Details of the action
	Timestamp        time.Time             `json:"timestamp"`              //
	mutex            sync.Mutex            `json:"-"`                      // ignore
	eventReceiverUrl string                `json:"-"`                      // event receiver url
	httpClient       httputils.IHttpClient `json:"-"`                      // http client
}

//
// ("reporter": "auditlog processor", //the name of your k8s component
//  "target": "<scope> auditlogs", // eg. if you know its cluster & ns you can say: hipstershop/dev auditlogs
//  "status":  <use constants defined above eg. JobStarted>
//  "action": "<the action itself- eg. fetching logs from s3",
//  "errors": <fill if u encountered any>
//  "actionID" & "actionIDN" - numerical representation - eg if it's the first step then it should be 1, it also allow "forks" to happen
// 	"jobID": event receiver will fill that for you
// 	"parentAction": parent ID of the action
// 	"timestamp": <s.e>
// 	"customerGUID": s.e
// }

// NewBaseReport return pointer to new BaseReport obj
func NewBaseReport(customerGUID, reporter, eventReceiverUrl string, httpClient httputils.IHttpClient) *BaseReport {
	return &BaseReport{
		CustomerGUID:     customerGUID,
		Reporter:         reporter,
		Status:           JobStarted,
		ActionName:       fmt.Sprintf("Starting %s", reporter),
		ActionID:         "1",
		ActionIDN:        1,
		eventReceiverUrl: eventReceiverUrl,
		httpClient:       httpClient,
	}
}

// IReporter reporter interface
type IReporter interface {
	// createReport() BaseReport

	/*
		send the report
		@Output:
		int: http Status Code,
		string: message: can be jobID for successful 1st report or "OK" from 2nd report onwards or "body could not be fetched"
		error: error from event receiver
	*/
	Send() (int, string, error) //send logic here
	GetReportID() string
	/* a multiple errors can occur but these error are not critical,
	errorString will be added to a vector of errors so the error flow until the critical error will be clear
	*/
	AddError(errorString string)
	GetNextActionId() string
	NextActionID()
	/*
		SimpleReportAnnotations - create an object that can be passed on as annotation and serialize it.

		This objects can be shared between the different microservices processing the same workload.

		thus this will save the jobID,it's latest actionID.
		@Input:
		setParent- set parentJobID to the jobID
		setCurrent - set the jobID to the current jobID

		@returns:
		 jsonAsString, nextActionID
	*/
	SimpleReportAnnotations(setParent bool, setCurrent bool) (string, string)

	/*
		SendAsRoutine
		@input:
		collector []string - leave as empty (a way to hold all previous failed reports and send them in bulk)
		progressNext bool - increase actionID, sometimes u send parallel jobs that have the same order - (vuln scanning a cluster for eg. all wl scans have the same order)
		errChan - chan to allow the goroutine to return the errors inside
	*/
	SendAsRoutine(bool, chan<- error) //goroutine wrapper

	// set methods
	SendAction(action string, sendReport bool, errChan chan<- error)
	SendError(err error, sendReport bool, initErrors bool, errChan chan<- error)
	SendStatus(status string, sendReport bool, errChan chan<- error)
	SendDetails(details string, sendReport bool, errChan chan<- error)
	SendWarning(warning string, sendReport bool, initWarnings bool, errChan chan<- error)

	// set methods
	SetReporter(string)
	SetStatus(string)
	SetActionName(string)
	SetTarget(string)
	SetActionID(string)
	SetJobID(string)
	SetParentAction(string)
	SetTimestamp(time.Time)
	SetActionIDN(int)
	SetCustomerGUID(string)
	SetDetails(string)

	// get methods
	GetReporter() string
	GetStatus() string
	GetActionName() string
	GetTarget() string
	GetErrorList() []string
	GetActionID() string
	GetJobID() string
	GetParentAction() string
	GetTimestamp() time.Time
	GetActionIDN() int
	GetCustomerGUID() string
	GetDetails() string
}

// IsEqual are two IReporter objects equal
func IsEqual(lhs, rhs IReporter) bool {
	if strings.Compare(lhs.GetJobID(), rhs.GetJobID()) != 0 ||
		strings.Compare(lhs.GetStatus(), rhs.GetStatus()) != 0 ||
		strings.Compare(lhs.GetReporter(), rhs.GetReporter()) != 0 ||
		strings.Compare(lhs.GetTarget(), rhs.GetTarget()) != 0 ||
		strings.Compare(lhs.GetActionID(), rhs.GetActionID()) != 0 ||
		strings.Compare(lhs.GetActionName(), rhs.GetActionName()) != 0 ||
		strings.Compare(lhs.GetParentAction(), rhs.GetParentAction()) != 0 ||
		lhs.GetActionIDN() != rhs.GetActionIDN() ||

		lhs.GetTimestamp().Unix() != rhs.GetTimestamp().Unix() ||
		!reflect.DeepEqual(rhs.GetErrorList(), lhs.GetErrorList()) {
		return false
	}

	return true
}
