package datastructures

import (
	"bytes"
	_ "embed"
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/francoispqt/gojay"
)

func TestBaseReportStructure(t *testing.T) {
	// a := BaseReport{Reporter: "unit-test", Target: "unit-test-framework", JobID: "id", ActionID: "id2"}
	// timestamp := a.Timestamp

	// a.Send()
	// if timestamp == a.Timestamp {
	// 	t.Errorf("Expecting different timestamp when sending a notification, received %v", a)
	// }

}

func TestFirstBaseReportStructure(t *testing.T) {
	// a := BaseReport{Reporter: "unit-test", Target: "unit-test-framework"}
	// id := a.JobID
	// a.Send()
	// if id != a.JobID {
	// 	t.Errorf("Expecting to have proccessID generated from 1st report, received %v", a)
	// }

}
func BaseReportDiff(lhs, rhs *BaseReport) {
	if strings.Compare(lhs.JobID, rhs.JobID) != 0 {
		fmt.Printf("jobID: %v != %v\n", lhs.JobID, rhs.JobID)
	}
	if strings.Compare(lhs.Status, rhs.Status) != 0 {
		fmt.Printf("Status: %v != %v\n", lhs.Status, rhs.Status)
	}
	if strings.Compare(lhs.Reporter, rhs.Reporter) != 0 {
		fmt.Printf("Reporter: %v != %v\n", lhs.Reporter, rhs.Reporter)
	}
	if strings.Compare(lhs.Target, rhs.Target) != 0 {
		fmt.Printf("Target: %v != %v\n", lhs.Target, rhs.Target)
	}
	if strings.Compare(lhs.ActionID, rhs.ActionID) != 0 {
		fmt.Printf("ActionID: %v != %v\n", lhs.ActionID, rhs.ActionID)
	}
	if strings.Compare(lhs.ActionName, rhs.ActionName) != 0 {
		fmt.Printf("ActionName: %v != %v\n", lhs.ActionName, rhs.ActionName)
	}
	if strings.Compare(lhs.ParentAction, rhs.ParentAction) != 0 {
		fmt.Printf("%v != %v\n", lhs.ParentAction, rhs.ParentAction)
	}
	if lhs.Timestamp.Unix() != rhs.Timestamp.Unix() {
		fmt.Printf("Timestamp: %v != %v\n", lhs.Timestamp, rhs.Timestamp)
	}
	if lhs.ActionIDN != rhs.ActionIDN {
		fmt.Printf("ActionIDN: %v != %v\n", lhs.ActionIDN, rhs.ActionIDN)
	}
	if !reflect.DeepEqual(rhs.Errors, lhs.Errors) {
		fmt.Printf("Errors: %v != %v\n", lhs.Errors, rhs.Errors)
	}

}
func TestUnMarshallingSuccess(t *testing.T) {
	lhs := BaseReport{Reporter: "unit-test", Target: "unit-test-framework", JobID: "1", ActionID: "1", Status: "testing", ActionName: "Testing", ActionIDN: 1}
	rhs := &BaseReport{}
	lhs.AddError("1")
	lhs.AddError("2")
	lhs.Timestamp = time.Now()
	bolB, _ := json.Marshal(lhs)
	r := bytes.NewReader(bolB)

	er := gojay.NewDecoder(r).DecodeObject(rhs)
	if er != nil {
		t.Errorf("marshalling failed due to: %v", er.Error())
	}
	if !IsEqual(&lhs, rhs) {
		BaseReportDiff(&lhs, rhs)
		fmt.Printf("%+v\n", lhs)
		t.Errorf("%v", rhs)
	}

}

func TestUnMarshallingPartial(t *testing.T) {
	lhs := BaseReport{Reporter: "unit-test", Target: "unit-test-framework", JobID: "1", ActionID: "1", Status: "testing", ActionName: "Testing", ActionIDN: 1}
	rhs := &BaseReport{}

	lhs.Timestamp = time.Now()
	bolB, _ := json.Marshal(lhs)
	r := bytes.NewReader(bolB)

	er := gojay.NewDecoder(r).DecodeObject(rhs)
	if er != nil {
		t.Errorf("marshalling failed due to: %v", er.Error())
	}
	if !IsEqual(&lhs, rhs) {
		BaseReportDiff(&lhs, rhs)
		fmt.Printf("%+v\n", lhs)
		t.Errorf("%v", rhs)
	}

}

func TestSetAction(t *testing.T) {
	lhs := BaseReport{Reporter: "unit-test", Target: "unit-test-framework",
		JobID: "1", ActionID: "1", Status: "testing", ActionName: "Testing", ActionIDN: 1}
	newAct := "blabla"
	lhs.SetActionName(newAct)

	if lhs.ActionName != newAct {
		t.Errorf("wrong action name after set: %s", lhs.ActionName)
	}
}

func TestSendDeadlock(t *testing.T) {
	done := make(chan interface{})

	go func() {
		snapshotNum := 0
		reporter := NewBaseReport("a-user-guid", "my-reporter")
		reporter.SetDetails("testing reporter")
		reporter.SetActionName("testing action")

		err1 := fmt.Errorf("dummy error")
		reporter.SendError(err1, true, false, nil)
		reporter.SendError(err1, false, false, nil)
		reporter.SendError(err1, false, false, nil)
		reporter.SendError(err1, true, false, nil)
		snapshotNum++
		compareSnapshot(snapshotNum, t, reporter)

		errChan := make(chan error)
		reporter.SendError(err1, true, true, errChan)
		e := <-errChan
		assert.Error(t, e)
		done <- 0
		snapshotNum++
		compareSnapshot(snapshotNum, t, reporter)

		errChan1 := make(chan error)
		err2 := fmt.Errorf("dummy error1")
		reporter.SendError(err2, false, false, errChan1)
		e = <-errChan1
		assert.NoError(t, e)
		done <- 1
		snapshotNum++
		compareSnapshot(snapshotNum, t, reporter)

		reporter.SetJobID("job-id")
		reporter.SetParentAction("parent-action")
		reporter.SetActionName("testing action2")
		reporter.SetActionIDN(20)
		reporter.SetStatus("testing status")
		reporter.SetTarget("testing target")
		reporter.SetReporter("testing reporter v2")
		reporter.SetCustomerGUID("new-customer-guid")
		reporter.SetTimestamp(time.Date(2020, time.January, 1, 0, 0, 0, 0, time.UTC))
		snapshotNum++
		compareSnapshot(snapshotNum, t, reporter)

		reporter.SendAsRoutine(true, nil)
		errChan2 := make(chan error)
		reporter.SendAsRoutine(true, errChan2)
		e = <-errChan2
		assert.Error(t, e)
		done <- 2

		errChan3 := make(chan error)
		reporter.SendError(nil, false, true, errChan3)
		e = <-errChan3
		assert.NoError(t, e)
		done <- 3
		snapshotNum++
		compareSnapshot(snapshotNum, t, reporter)

		reporter.SendStatus("status", true, nil)
		reporter.SendStatus("status", false, nil)
		snapshotNum++
		compareSnapshot(snapshotNum, t, reporter)

		errChan4 := make(chan error)
		reporter.SendStatus("status", true, errChan4)
		e = <-errChan4
		assert.Error(t, e)
		done <- 4

		errChan5 := make(chan error)
		reporter.SendStatus("status", false, errChan5)
		e = <-errChan5
		assert.NoError(t, e)
		done <- 5

		reporter.SendAction("action", true, nil)
		reporter.SendAction("action", false, nil)

		errChan6 := make(chan error)
		reporter.SendAction("action", true, errChan6)
		e = <-errChan6
		assert.Error(t, e)
		done <- 6

		errChan7 := make(chan error)
		reporter.SendAction("action", false, errChan7)
		e = <-errChan7
		assert.NoError(t, e)
		done <- 7
		snapshotNum++
		compareSnapshot(snapshotNum, t, reporter)

		reporter.SendDetails("details", true, nil)
		reporter.SendDetails("details", false, nil)

		errChan8 := make(chan error)
		reporter.SendDetails("details", true, errChan8)
		e = <-errChan8
		assert.Error(t, e)
		done <- 8

		errChan9 := make(chan error)
		reporter.SendDetails("details", false, errChan9)
		e = <-errChan9
		assert.NoError(t, e)
		done <- 9
		snapshotNum++
		compareSnapshot(snapshotNum, t, reporter)

		reporter.SendWarning("warning", true, false, nil)
		reporter.SendWarning("warning", false, false, nil)
		reporter.SendWarning("warning", false, false, nil)
		reporter.SendWarning("warning", true, false, nil)
		snapshotNum++
		compareSnapshot(snapshotNum, t, reporter)

		errChan10 := make(chan error)
		reporter.SendWarning("warning", true, false, errChan10)
		e = <-errChan10
		assert.Error(t, e)
		done <- 10

		errChan11 := make(chan error)
		reporter.SendWarning("warning", false, false, errChan11)
		e = <-errChan11
		assert.NoError(t, e)
		done <- 11
		snapshotNum++
		compareSnapshot(snapshotNum, t, reporter)

		errChan12 := make(chan error)
		reporter.SendWarning("warning", false, true, errChan12)
		e = <-errChan12
		assert.NoError(t, e)
		done <- 12
		snapshotNum++
		compareSnapshot(snapshotNum, t, reporter)

	}()

	expectedMsgs := 12
	for i := 0; i <= expectedMsgs+1; i++ {
		select {
		case <-time.After(300 * time.Millisecond):
			if i <= expectedMsgs {
				t.Fatalf("Deadlock detected message %d did not arrive", i)
			}
		case id := <-done:
			if i > expectedMsgs {
				t.Errorf("unexpected message %d id:%d", i, id)
			}
		}
	}
}

//go:embed fixtures/report1_snapshot.json
var report1_snapshot []byte

//go:embed fixtures/report2_snapshot.json
var report2_snapshot []byte

//go:embed fixtures/report3_snapshot.json
var report3_snapshot []byte

//go:embed fixtures/report4_snapshot.json
var report4_snapshot []byte

//go:embed fixtures/report5_snapshot.json
var report5_snapshot []byte

//go:embed fixtures/report6_snapshot.json
var report6_snapshot []byte

//go:embed fixtures/report7_snapshot.json
var report7_snapshot []byte

//go:embed fixtures/report8_snapshot.json
var report8_snapshot []byte

//go:embed fixtures/report9_snapshot.json
var report9_snapshot []byte

//go:embed fixtures/report10_snapshot.json
var report10_snapshot []byte

//go:embed fixtures/report11_snapshot.json
var report11_snapshot []byte

func compareSnapshot(id int, t *testing.T, r *BaseReport) {
	r.mutex.Lock()
	rStr, _ := json.MarshalIndent(r, "", "\t")
	/*
		uncomment to update expected snapshots
		os.WriteFile(fmt.Sprintf("./fixtures/report%d_snapshot.json", id), rStr, 0666)
	*/
	r.mutex.Unlock()
	var expected []byte
	switch id {
	case 1:
		expected = report1_snapshot
	case 2:
		expected = report2_snapshot
	case 3:
		expected = report3_snapshot
	case 4:
		expected = report4_snapshot
	case 5:
		expected = report5_snapshot
	case 6:
		expected = report6_snapshot
	case 7:
		expected = report7_snapshot
	case 8:
		expected = report8_snapshot
	case 9:
		expected = report9_snapshot
	case 10:
		expected = report10_snapshot
	case 11:
		expected = report11_snapshot
	default:
		t.Fatalf("Unknown snapshot id: %d", id)
	}
	assert.Equal(t, string(expected), string(rStr), "Snapshot id: %d is different than expected", id)
}
