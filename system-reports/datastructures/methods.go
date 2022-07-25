package datastructures

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/golang/glog"
)

var MAX_RETRIES int = 3
var RETRY_DELAY time.Duration = time.Second * 5

func (report *BaseReport) InitMutex() {
	report.mutex = sync.Mutex{}
}
func (report *BaseReport) getEventReceiverUrl() (string, error) {
	if len(report.eventReceiverUrl) == 0 {
		report.eventReceiverUrl = os.Getenv("CA_EVENT_RECEIVER_HTTP")
	}
	if len(report.eventReceiverUrl) == 0 {
		report.eventReceiverUrl = os.Getenv("CA_ARMO_EVENT_URL") // Deprecated
	}
	if len(report.eventReceiverUrl) == 0 {
		return "", fmt.Errorf("%s - Error: CA_EVENT_RECEIVER_HTTP is missing", report.GetReportID())
	}
	return report.eventReceiverUrl, nil
}

func (report *BaseReport) NextActionID() {
	report.ActionIDN++
	report.ActionID = report.GetNextActionId()
}
func (report *BaseReport) SimpleReportAnnotations(setParent bool, setCurrent bool) (string, string) {

	nextactionID := report.GetNextActionId()

	jobs := JobsAnnotations{LastActionID: nextactionID}
	if setParent {
		jobs.ParentJobID = report.JobID
	}
	if setCurrent {
		jobs.CurrJobID = report.JobID
	}
	jsonAsString, _ := json.Marshal(jobs)
	return string(jsonAsString), nextactionID
	//ok
}

func (report *BaseReport) GetNextActionId() string {
	return strconv.Itoa(report.ActionIDN)
}

func (report *BaseReport) AddError(er string) {
	report.mutex.Lock()
	defer report.mutex.Unlock()
	if report.Errors == nil {
		report.Errors = make([]string, 0)
	}
	report.Errors = append(report.Errors, er)
}

// The caller must read the errChan, to prevent the goroutine from waiting in memory forever
func (report *BaseReport) SendAsRoutine(progressNext bool, errChan chan<- error) {
	report.mutex.Lock()
	wg := &sync.WaitGroup{}
	report.unprotectedSendAsRoutine(errChan, progressNext, wg)
	go func(report *BaseReport) {
		wg.Wait()
		report.mutex.Unlock()
	}(report)
}

//internal send as routine without mutex lock
func (report *BaseReport) unprotectedSendAsRoutine(errChan chan<- error, progressNext bool, wg *sync.WaitGroup) {
	wg.Add(1)
	go func() {

		defer func() {
			wg.Done()
			recover()
		}()
		status, body, err := report.Send()
		if errChan != nil {
			if err != nil {
				errChan <- err
				return
			}
			if status < 200 || status >= 300 {
				err := fmt.Errorf("failed to send report. Status: %d Body:%s", status, body)
				errChan <- err
				return
			}
		}
		if progressNext {
			report.NextActionID()
		}
		if errChan != nil {
			errChan <- nil
		}
	}()
}

func (report *BaseReport) GetReportID() string {
	return fmt.Sprintf("%s::%s::%s (verbose:  %s::%s)", report.Target, report.JobID, report.ActionID, report.ParentAction, report.ActionName)
}

// Send - send http request. returns-> http status code, return message (jobID/OK), http/go error
func (report *BaseReport) Send() (int, string, error) {

	url, err := report.getEventReceiverUrl()
	if err != nil {
		return 0, "", err
	}

	url = url + SysreportEndpoint
	report.Timestamp = time.Now()
	if report.ActionID == "" {
		report.ActionID = "1"
		report.ActionIDN = 1
	}
	reqBody, err := json.Marshal(report)

	if err != nil {
		glog.Errorf("%s - Failed to marshall report object", report.GetReportID())
		return 500, "Couldn't marshall report object", err
	}
	var resp *http.Response
	var bodyAsStr string
	for i := 0; i < MAX_RETRIES; i++ {
		resp, err = http.Post(url, "application/json", bytes.NewBuffer(reqBody))
		bodyAsStr = "body could not be fetched"
		retry := err != nil
		if resp != nil {
			if resp.StatusCode < 200 || resp.StatusCode >= 300 {
				retry = true
			}
			if resp.Body != nil {
				body, err := ioutil.ReadAll(resp.Body)
				if err == nil {
					bodyAsStr = string(body)
				}
				resp.Body.Close()
			}
		}
		if !retry {
			break
		}
		//else err != nil
		e := fmt.Errorf("attempt #%d %s - Failed posting report. Url: '%s', reason: '%s' report: '%s' response: '%s'", i, report.GetReportID(), url, err.Error(), string(reqBody), bodyAsStr)
		glog.Error(e)

		if i == MAX_RETRIES-1 {
			return 500, e.Error(), err
		}
		//wait 5 secs between retries
		time.Sleep(RETRY_DELAY)
	}
	//first successful report gets it's jobID/proccessID
	if len(report.JobID) == 0 && bodyAsStr != "ok" && resp.StatusCode >= 200 && resp.StatusCode < 300 {
		report.JobID = bodyAsStr
		glog.Infof("Generated jobID: '%s'", report.JobID)
	}
	return resp.StatusCode, bodyAsStr, nil

}

// ======================================== SEND WRAPPER =======================================

// SendError - wrap AddError
func (report *BaseReport) SendError(err error, sendReport bool, initErrors bool, errChan chan<- error) {
	report.mutex.Lock() // +

	if report.Errors == nil {
		report.Errors = make([]string, 0)
	}
	if err != nil {
		e := fmt.Sprintf("Action: %s, Error: %s", report.ActionName, err.Error())
		report.Errors = append(report.Errors, e)
	}
	report.Status = JobFailed // TODO - Add flag?

	if sendReport {
		wg := &sync.WaitGroup{}
		report.unprotectedSendAsRoutine(errChan, true, wg)
		go func(report *BaseReport) {
			wg.Wait()
			if initErrors {
				report.Errors = make([]string, 0)
			}
			report.mutex.Unlock() // -
		}(report)
	} else {
		if errChan != nil {
			go func() { errChan <- nil }()
		}
		if initErrors {
			report.Errors = make([]string, 0)
		}
		report.mutex.Unlock() // -
	}
}

func (report *BaseReport) SendWarning(warnMsg string, sendReport bool, initWarnings bool, errChan chan<- error) {
	report.mutex.Lock() // +
	if report.Errors == nil {
		report.Errors = make([]string, 0)
	}
	if len(warnMsg) != 0 {
		e := fmt.Sprintf("Action: %s, Error: %s", report.ActionName, warnMsg)
		report.Errors = append(report.Errors, e)
	}
	report.Status = JobWarning

	if sendReport {
		wg := &sync.WaitGroup{}
		report.unprotectedSendAsRoutine(errChan, true, wg)
		go func(report *BaseReport) {
			wg.Wait()
			if initWarnings {
				report.Errors = make([]string, 0)
			}
			report.mutex.Unlock() // -
		}(report)
	} else {
		if errChan != nil {
			go func() { errChan <- nil }()
		}
		if initWarnings {
			report.Errors = make([]string, 0)
		}
		report.mutex.Unlock() // -
	}
}

func (report *BaseReport) SendAction(actionName string, sendReport bool, errChan chan<- error) {
	report.mutex.Lock()
	report.doSetActionName(actionName)
	if sendReport {
		wg := &sync.WaitGroup{}
		report.unprotectedSendAsRoutine(errChan, true, wg)
		go func(report *BaseReport) {
			wg.Wait()
			report.mutex.Unlock() // -
		}(report)
	} else {
		if errChan != nil {
			go func() { errChan <- nil }()
		}
		report.mutex.Unlock() // -
	}
}

func (report *BaseReport) SendStatus(status string, sendReport bool, errChan chan<- error) {
	report.mutex.Lock()
	report.doSetStatus(status)
	if sendReport {
		wg := &sync.WaitGroup{}
		report.unprotectedSendAsRoutine(errChan, true, wg)
		go func(report *BaseReport) {
			wg.Wait()
			report.mutex.Unlock() // -
		}(report)
	} else {
		if errChan != nil {
			go func() { errChan <- nil }()
		}
		report.mutex.Unlock() // -
	}
}

func (report *BaseReport) SendDetails(details string, sendReport bool, errChan chan<- error) {
	report.mutex.Lock()
	report.doSetDetails(details)
	if sendReport {
		wg := &sync.WaitGroup{}
		report.unprotectedSendAsRoutine(errChan, true, wg)
		go func(report *BaseReport) {
			wg.Wait()
			report.mutex.Unlock() // -
		}(report)
	} else {
		if errChan != nil {
			go func() { errChan <- nil }()
		}
		report.mutex.Unlock() // -
	}
}

// ============================================ SET ============================================

func (report *BaseReport) SetReporter(reporter string) {
	report.mutex.Lock()
	defer report.mutex.Unlock()
	report.doSetReporter(reporter)
}
func (report *BaseReport) doSetReporter(reporter string) {
	report.Reporter = strings.ToTitle(reporter)
}

func (report *BaseReport) SetStatus(status string) {
	report.mutex.Lock()
	defer report.mutex.Unlock()
	report.doSetStatus(status)
}
func (report *BaseReport) doSetStatus(status string) {
	report.Status = status
}

func (report *BaseReport) SetActionName(actionName string) {
	report.mutex.Lock()
	defer report.mutex.Unlock()
	report.doSetActionName(actionName)
}
func (report *BaseReport) doSetActionName(actionName string) {
	report.ActionName = actionName
}

func (report *BaseReport) SetDetails(details string) {
	report.mutex.Lock()
	defer report.mutex.Unlock()
	report.doSetDetails(details)
}
func (report *BaseReport) doSetDetails(details string) {
	report.Details = details
}

func (report *BaseReport) SetTarget(target string) {
	report.mutex.Lock()
	defer report.mutex.Unlock()
	report.doSetTarget(target)
}
func (report *BaseReport) doSetTarget(target string) {
	report.Target = target
}

func (report *BaseReport) SetActionID(actionID string) {
	report.mutex.Lock()
	defer report.mutex.Unlock()
	report.doSetActionID(actionID)
}
func (report *BaseReport) doSetActionID(actionID string) {
	report.ActionID = actionID
}

func (report *BaseReport) SetJobID(jobID string) {
	report.mutex.Lock()
	defer report.mutex.Unlock()
	report.doSetJobID(jobID)
}
func (report *BaseReport) doSetJobID(jobID string) {
	report.JobID = jobID
}

func (report *BaseReport) SetParentAction(parentAction string) {
	report.mutex.Lock()
	defer report.mutex.Unlock()
	report.doSetParentAction(parentAction)
}
func (report *BaseReport) doSetParentAction(parentAction string) {
	report.ParentAction = parentAction
}

func (report *BaseReport) SetCustomerGUID(customerGUID string) {
	report.mutex.Lock()
	defer report.mutex.Unlock()
	report.doSetCustomerGUID(customerGUID)
}
func (report *BaseReport) doSetCustomerGUID(customerGUID string) {
	report.CustomerGUID = customerGUID
}

func (report *BaseReport) SetActionIDN(actionIDN int) {
	report.mutex.Lock()
	defer report.mutex.Unlock()
	report.doSetActionIDN(actionIDN)
}
func (report *BaseReport) doSetActionIDN(actionIDN int) {
	report.ActionIDN = actionIDN
	report.ActionID = strconv.Itoa(report.ActionIDN)
}

func (report *BaseReport) SetTimestamp(timestamp time.Time) {
	report.mutex.Lock()
	defer report.mutex.Unlock()
	report.doSetTimestamp(timestamp)
}
func (report *BaseReport) doSetTimestamp(timestamp time.Time) {
	report.Timestamp = timestamp
}

// ============================================ GET ============================================
func (report *BaseReport) GetActionName() string {
	return report.ActionName
}

func (report *BaseReport) GetStatus() string {
	return report.Status
}

func (report *BaseReport) GetErrorList() []string {
	return report.Errors
}

func (report *BaseReport) GetTarget() string {
	return report.Target
}

func (report *BaseReport) GetReporter() string {
	return report.Reporter
}

func (report *BaseReport) GetActionID() string {
	return report.ActionID
}

func (report *BaseReport) GetJobID() string {
	return report.JobID
}

func (report *BaseReport) GetParentAction() string {
	return report.ParentAction
}

func (report *BaseReport) GetCustomerGUID() string {
	return report.CustomerGUID
}

func (report *BaseReport) GetActionIDN() int {
	return report.ActionIDN
}

func (report *BaseReport) GetTimestamp() time.Time {
	return report.Timestamp
}

func (report *BaseReport) GetDetails() string {
	return report.Details
}
