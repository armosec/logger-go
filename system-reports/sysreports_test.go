package systemReport

import (
	"encoding/json"
	"strconv"
	"sync"
	"testing"

	"github.com/armosec/logger-go/system-reports/datastructures"
	"github.com/armosec/logger-go/system-reports/utilities"
)

func TestJobsAnnotation(t *testing.T) {
	a := datastructures.JobsAnnotations{CurrJobID: "test-job", LastActionID: "1"}

	marshal, err := json.Marshal(a)
	if err != nil {
		t.Errorf("unable to stringify job annotation: %v", a)
	}

	jobid, obj, err := utilities.GetJobIDByContext(marshal, "test")
	if err != nil {
		t.Errorf("unable to parse json job annotation: %v", a)
	}

	if jobid != "test-job" || a.CurrJobID != obj.CurrJobID || a.LastActionID != obj.LastActionID || a.ParentJobID != obj.ParentJobID {
		t.Error("unable to parse job annotation correctly")
	}

}

func TestBaseReportTestConcurrentErrorAdding(t *testing.T) {
	a := &datastructures.BaseReport{Reporter: "unit-test", Target: "unit-test-framework", Status: "started", JobID: "processid1", ActionID: "1"}
	var wg sync.WaitGroup
	for j := 0; j < 10; j++ {

		for i := 0; i < 4; i++ {
			wg.Add(1)
			go func(i int, wg *sync.WaitGroup) {
				defer wg.Done()
				s := strconv.Itoa(i)
				a.AddError(s)
			}(i, &wg)
		}
		wg.Wait()

		if len(a.Errors) != 4 {
			t.Errorf("an inconsistency error occurred at round %d, expected 4 errors and got %v", j, a)
		}
		a.Errors = nil

	}
}
