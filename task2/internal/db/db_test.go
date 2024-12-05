package db_test

import (
	"fmt"
	"raftdb/internal/db"
	"testing"

	"github.com/stretchr/testify/assert"
)

type InStruct struct {
	Command  string
	Key      string
	Value    string
	OldValue string
}

type OutStruct struct {
	Key     string
	Value   any
	Success bool
	Error   error
}

func executeTestCases(t *testing.T, testCases []struct {
	In  []InStruct
	Out OutStruct
}) {
	for _, testCase := range testCases {
		out := OutStruct{}
		for _, in := range testCase.In {
			success, err := db.ProcessWrite(in.Command, in.Key, &in.Value, &in.OldValue)
			out.Success = success
			out.Error = err

			if err != nil {
				break
			}
		}

		if out.Error == nil {
			out.Key = testCase.Out.Key
			out.Value, out.Success = db.Get(out.Key)
		}

		assert.Equal(t, testCase.Out, out)
	}
}

func TestSimpleOperations(t *testing.T) {
	testCases := []struct {
		In  []InStruct
		Out OutStruct
	}{
		{
			In: []InStruct{
				{Command: "CREATE", Key: "1", Value: "2"},
			},
			Out: OutStruct{Key: "1", Value: "2", Success: true},
		},
		{
			In: []InStruct{
				{Command: "CREATE", Key: "2", Value: "3"},
				{Command: "UPDATE", Key: "2", Value: "5"},
			},
			Out: OutStruct{Key: "2", Value: "5", Success: true},
		},
		{
			In: []InStruct{
				{Command: "CREATE", Key: "3", Value: "4"},
				{Command: "CAS", Key: "3", OldValue: "4", Value: "5"},
			},
			Out: OutStruct{Key: "3", Value: "5", Success: true},
		},
		{
			In: []InStruct{
				{Command: "CREATE", Key: "4", Value: "5"},
				{Command: "DELETE", Key: "4"},
			},
			Out: OutStruct{Key: "4", Value: nil, Success: false},
		},
		{
			In: []InStruct{
				{Command: "CREATE", Key: "4", Value: "5"},
				{Command: "CREATE", Key: "4", Value: "5"},
			},
			Out: OutStruct{Success: false, Error: fmt.Errorf("Key '4' already exists")},
		},
	}

	executeTestCases(t, testCases)
}
