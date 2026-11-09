package utils

import (
	"errors"
	"testing"

	smserrors "go.gh.ink/smsutils/v3/errors"
	"go.gh.ink/smsutils/v3/model"
)

func TestValidateVars(t *testing.T) {
	for _, vars := range []model.Vars{{nil}, {{Key: "", Value: "x"}}, {{Key: "code", Value: "1"}, {Key: "code", Value: "2"}}} {
		if err := ValidateVars(vars, true); !errors.Is(err, smserrors.ErrMessageInvalid) {
			t.Fatalf("vars=%v err=%v", vars, err)
		}
	}
	positional := model.Vars{{Value: "second"}, {Value: "first"}}
	if err := ValidateVars(positional, false); err != nil {
		t.Fatal(err)
	}
	if positional[0].Value != "second" || positional[1].Value != "first" {
		t.Fatal("slice was modified")
	}
	if err := ValidateVars(nil, true); err != nil {
		t.Fatal("parameterless templates must remain valid", err)
	}
}
