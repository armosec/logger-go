package utilities

import (
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/armosec/armoapi-go/identifiers"
	"github.com/armosec/logger-go/system-reports/datastructures"
	"github.com/armosec/utils-go/httputils"
	"github.com/golang/glog"
)

var (
	EmptyString = []string{}
)

// TODO
// takes annotation and return the jobID, annotationObject, err
func GetJobIDByContext(jobs []byte, context string) (string, datastructures.JobsAnnotations, error) {

	var jobject datastructures.JobsAnnotations
	err := json.Unmarshal(jobs, &jobject)

	return jobject.CurrJobID, jobject, err
}

func ProcessAnnotations(reporter datastructures.IReporter, jobAnnotations interface{}, hasAnnotations bool) error {
	if !hasAnnotations {
		return fmt.Errorf("missing job annotations")
	}

	tmpstr := fmt.Sprintf("%s", jobAnnotations)
	_, jobAnnotationsObj, err := GetJobIDByContext([]byte(tmpstr), "attach")
	if err != nil {
		return fmt.Errorf("unable to parse job annotations: %v", err)
	}

	if len(jobAnnotationsObj.CurrJobID) > 0 {
		reporter.SetJobID(jobAnnotationsObj.CurrJobID)
	}

	reporter.SetParentAction(jobAnnotationsObj.ParentJobID)
	reporter.SetActionID(jobAnnotationsObj.LastActionID)
	actionID, _ := strconv.Atoi(reporter.GetActionID())
	reporter.SetActionIDN(actionID)

	return nil
}

// SendImmutableReport incase you want to send it all and just manage jobID, actionID yourself (no locking downtimes)
func SendImmutableReport(target, reporter, actionID, action, status string, jobID *string, err error) {

	lhs := datastructures.BaseReport{Reporter: reporter, ActionName: action, Target: target, JobID: *jobID, ActionID: actionID, Status: status}
	lhs.ActionIDN, _ = strconv.Atoi(actionID)
	if err != nil {
		lhs.AddError(err.Error())
		glog.Error(err.Error()) // TODO: remove log
	}
	_, *jobID, _ = lhs.Send()

}

func InitReporter(customerGUID, reporterName, actionName, wlid, eventReceiverUrl string, httpClient httputils.IHttpClient, designator *identifiers.PortalDesignator, errChan chan<- error) *datastructures.BaseReport {
	reporter := datastructures.NewBaseReport(customerGUID, reporterName, eventReceiverUrl, httpClient)
	if actionName != "" {
		reporter.SetActionName(actionName)
	}
	if wlid != "" {
		reporter.SetTarget(wlid)
	} else if designator != nil {
		reporter.SetTarget(GetTargetFromDesignator(designator))
	}
	reporter.SendAsRoutine(true, errChan)
	return reporter
}

func GetTargetFromDesignator(designator *identifiers.PortalDesignator) string {
	switch designator.DesignatorType {
	case identifiers.DesignatorWlid:
		return designator.WLID
	case identifiers.DesignatorWildWlid:
		return designator.WildWLID
	case identifiers.DesignatorAttributes:
		if designator.Attributes != nil {
			return convertMapToString(designator.Attributes)
		}
	}
	return "Unknown target"
}

func convertMapToString(smap map[string]string) string {
	str := ""
	for i := range smap {
		str += fmt.Sprintf("%s=%s;", i, smap[i])
	}
	return str
}
