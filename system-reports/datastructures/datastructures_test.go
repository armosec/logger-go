package datastructures

import (
	"bytes"
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
		reporter := NewBaseReport("auser", "myreporter")
		reporter.SetDetails("someDetails")

		errChan := make(chan error)
		err1 := fmt.Errorf("dummy error")
		reporter.SendError(err1, true, true, errChan)
		e := <-errChan
		assert.Error(t, e)
		done <- 0

		errChan1 := make(chan error)
		err2 := fmt.Errorf("dummy error1")
		reporter.SendError(err2, false, false, errChan1)
		e = <-errChan1
		assert.NoError(t, e)
		done <- 1

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

		errChan12 := make(chan error)
		reporter.SendWarning("warning", false, true, errChan12)
		e = <-errChan12
		assert.NoError(t, e)
		done <- 11

	}()

	for i := 0; i < 12; i++ {
		select {
		case <-time.After(1 * time.Second):
			if i != 12 {
				t.Fatalf("Deadlock detected message %d did not arrived", i)
			}
		case <-done:
			if i == 12 {
				t.Errorf("unexpected message %d ", i)
			}
		}
	}
}
